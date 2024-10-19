package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/oauth2"
	"tornberg.me/facet-search/pkg/index"
)

var (
	totalItems = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "slaskfinder_items_total",
		Help: "The total number of items in index",
	})
)

func (ws *WebServer) HandlePopularOverride(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {
		defaultHeaders(w, true, "0")
		sort := index.SortOverride{}
		err := json.NewDecoder(r.Body).Decode(&sort)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ws.Sorting.AddPopularOverride(&sort)

		w.WriteHeader(http.StatusOK)
		return
	}

	sort := ws.Sorting.GetPopularOverrides()
	if sort == nil {
		http.Error(w, "Sort not found", http.StatusNotFound)
		return
	}
	defaultHeaders(w, true, "120")
	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(sort)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) HandleFieldSort(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {
		defaultHeaders(w, true, "0")
		sort := index.SortOverride{}
		err := json.NewDecoder(r.Body).Decode(&sort)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ws.Sorting.SetFieldSortOverride(&sort)
		w.WriteHeader(http.StatusOK)
		return
	}

	sort := ws.Sorting.FieldSort
	if sort == nil {
		http.Error(w, "Sort not found", http.StatusNotFound)
		return
	}
	defaultHeaders(w, true, "120")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(sort)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) HandleStaticPositions(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {
		defaultHeaders(w, true, "0")
		sort := index.StaticPositions{}
		err := json.NewDecoder(r.Body).Decode(&sort)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ws.Sorting.SetStaticPositions(sort)
		w.WriteHeader(http.StatusOK)
		return
	}

	sort := ws.Sorting.GetStaticPositions()
	if sort == nil {
		http.Error(w, "Sort not found", http.StatusNotFound)
		return
	}
	defaultHeaders(w, true, "120")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(sort)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) AddItem(w http.ResponseWriter, r *http.Request) {
	items := AddItemRequest{}
	err := json.NewDecoder(r.Body).Decode(&items)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ws.Index.UpsertItems(items)
	totalItems.Set(float64(len(ws.Index.Items)))
	w.WriteHeader(http.StatusOK)
}

func (ws *WebServer) Save(w http.ResponseWriter, r *http.Request) {
	err := ws.Db.SaveIndex(ws.Index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

// type CategoryUpdateRequest struct {
// 	Ids     []uint                 `json:"ids"`
// 	Updates []index.CategoryUpdate `json:"updates"`
// }

// func (ws *WebServer) UpdateCategories(w http.ResponseWriter, r *http.Request) {
// 	update := CategoryUpdateRequest{}
// 	err := json.NewDecoder(r.Body).Decode(&update)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusBadRequest)
// 		return
// 	}

// 	ws.Index.UpdateCategoryValues(update.Ids, update.Updates)
// 	w.WriteHeader(http.StatusAccepted)
// }

func generateStateOauthCookie() string {
	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)

	return state
}

var secretKey = []byte("slask-62541337!-banansecret")

func createToken(username string, name string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"username": username,
			"name":     name,
			"role":     "admin",
			"exp":      time.Now().Add(time.Hour * 24).Unix(),
		})

	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (ws *WebServer) Login(w http.ResponseWriter, r *http.Request) {
	oauthState := generateStateOauthCookie()
	url := ws.OAuthConfig.AuthCodeURL(oauthState, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

type UserData struct {
	Name          string `json:"name"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Id            string `json:"id"`
	Picture       string `json:"picture"`
}

const tokenCookieName = "sf-admin"
const apiKey = "Basic YXBpc2xhc2tlcjptYXN0ZXJzbGFza2VyMjAwMA=="

func (ws *WebServer) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   tokenCookieName,
		Value:  "",
		MaxAge: -1,
	})
	w.WriteHeader(http.StatusOK)
}

func (ws *WebServer) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != apiKey {
			cookie, err := r.Cookie(tokenCookieName)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
				return secretKey, nil
			})
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if !token.Valid {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func getUserData(token *oauth2.Token) (*UserData, error) {
	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		return nil, err
	}
	var userData UserData
	err = json.NewDecoder(resp.Body).Decode(&userData)
	if err != nil {
		return nil, err
	}
	return &userData, nil
}

func (ws *WebServer) AuthCallback(w http.ResponseWriter, r *http.Request) {

	token, err := ws.OAuthConfig.Exchange(context.Background(), r.FormValue("code"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userData, err := getUserData(token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ownToken, err := createToken(userData.Email, userData.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     tokenCookieName,
		Value:    ownToken,
		Path:     "/",
		MaxAge:   3600,
		SameSite: http.SameSiteStrictMode,
	})

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func (ws *WebServer) User(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(tokenCookieName)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if !token.Valid {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	defaultHeaders(w, true, "0")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(claims)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// func (ws *WebServer) RagData(w http.ResponseWriter, r *http.Request) {
// 	defaultHeaders(w, false, "240")
// 	w.WriteHeader(http.StatusOK)
// 	ws.Index.Lock()
// 	defer ws.Index.Unlock()
// 	sorting := ws.Index.Sorting.GetSort("popular")
// 	var base *facet.BaseField
// 	for i, id := range *sorting {
// 		item, ok := ws.Index.Items[id]
// 		if !ok {
// 			continue
// 		}
// 		fmt.Fprintf(w, "%d;%s, %s, pris %d (%s)", item.Id, item.Title, strings.ReplaceAll(item.BulletPoints, "\n", ", "), item.GetPrice()/100, item.Url)

// 		for id, field := range item.Fields {
// 			f, ok := ws.Index.Facets[id]
// 			base = f.GetBaseField()
// 			if ok && !base.HideFacet {
// 				fmt.Fprintf(w, "%s %v, ", base.Name, field)
// 			}
// 		}
// 		w.Write([]byte("\n"))
// 		if i > 500 {
// 			break
// 		}
// 	}

// }

func (ws *WebServer) AdminHandler() *http.ServeMux {

	srv := http.NewServeMux()
	srv.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		defaultHeaders(w, false, "0")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	srv.HandleFunc("/login", ws.Login)
	srv.HandleFunc("/logout", ws.Logout)
	srv.HandleFunc("/user", ws.User)
	//srv.HandleFunc("GET /rag", ws.RagData)
	srv.HandleFunc("/auth_callback", ws.AuthCallback)
	srv.HandleFunc("/add", ws.AuthMiddleware(ws.AddItem))
	//srv.HandleFunc("/get/{id}", ws.AuthMiddleware(ws.GetItem))
	//srv.HandleFunc("PUT /key-values", ws.UpdateCategories)
	srv.HandleFunc("/save", ws.AuthMiddleware(ws.Save))
	srv.HandleFunc("/sort/popular", ws.AuthMiddleware(ws.HandlePopularOverride))
	srv.HandleFunc("/sort/static", ws.AuthMiddleware(ws.HandleStaticPositions))
	//srv.HandleFunc("/sort/{id}/partial", ws.ReOrderSort)
	srv.HandleFunc("/sort/fields", ws.AuthMiddleware(ws.HandleFieldSort))
	return srv
}
