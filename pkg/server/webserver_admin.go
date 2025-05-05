package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/oauth2"
)

var (
	totalItems = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "slaskfinder_items_total",
		Help: "The total number of items in index",
	})
)

func (ws *WebServer) HandlePopularRules(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {
		defaultHeaders(w, r, false, "0")
		jsonArray := types.JsonTypes{}
		err := json.NewDecoder(r.Body).Decode(&jsonArray)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		sort := make(types.ItemPopularityRules, 0, len(jsonArray))
		for _, item := range jsonArray {
			v, ok := item.(types.ItemPopularityRule)
			if !ok {
				continue
			}
			sort = append(sort, v)
		}
		types.CurrentSettings.PopularityRules = &sort
		//ws.Sorting.SetPopularityRules(&sort)

		w.WriteHeader(http.StatusOK)
		return
	}

	sort := types.CurrentSettings.PopularityRules
	if sort == nil {
		http.Error(w, "rules not found", http.StatusNotFound)
		return
	}
	defaultHeaders(w, r, true, "0")
	w.WriteHeader(http.StatusOK)

	jsonArray := make(types.JsonTypes, 0, len(*sort))
	for _, v := range *sort {
		j, ok := v.(types.JsonType)
		if !ok {
			continue
		}
		jsonArray = append(jsonArray, j)
	}

	err := json.NewEncoder(w).Encode(jsonArray)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) HandlePopularOverride(w http.ResponseWriter, r *http.Request) {
	if ws.Sorting == nil {
		log.Printf("Sorting not initialized in handlePopularOverride")
		http.Error(w, "Sorting not initialized", http.StatusInternalServerError)
		return
	}
	if r.Method == "POST" {
		defaultHeaders(w, r, true, "0")
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
	defaultHeaders(w, r, true, "120")
	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(sort)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// func (ws *WebServer) HandleFieldSort(w http.ResponseWriter, r *http.Request) {
// 	if ws.Sorting == nil {
// 		log.Printf("Sorting not initialized in handleFieldSort")
// 		http.Error(w, "Sorting not initialized", http.StatusInternalServerError)
// 		return
// 	}
// 	if r.Method == "POST" {
// 		defaultHeaders(w, r, true, "0")
// 		sort := index.SortOverride{}
// 		err := json.NewDecoder(r.Body).Decode(&sort)
// 		if err != nil {
// 			http.Error(w, err.Error(), http.StatusBadRequest)
// 			return
// 		}
// 		ws.Sorting.SetFieldSortOverride(&sort)
// 		w.WriteHeader(http.StatusOK)
// 		return
// 	}

// 	sort := ws.Sorting.FieldSort
// 	if sort == nil {
// 		http.Error(w, "Sort not found", http.StatusNotFound)
// 		return
// 	}
// 	defaultHeaders(w, r, true, "120")
// 	w.WriteHeader(http.StatusOK)
// 	err := json.NewEncoder(w).Encode(sort)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 	}
// }

// func (ws *WebServer) HandleStaticPositions(w http.ResponseWriter, r *http.Request) {
// 	if ws.Sorting == nil {
// 		log.Printf("Sorting not initialized in handleFieldSort")
// 		http.Error(w, "Sorting not initialized", http.StatusInternalServerError)
// 		return
// 	}
// 	if r.Method == "POST" {
// 		defaultHeaders(w, r, true, "0")
// 		sort := index.StaticPositions{}
// 		err := json.NewDecoder(r.Body).Decode(&sort)
// 		if err != nil {
// 			http.Error(w, err.Error(), http.StatusBadRequest)
// 			return
// 		}
// 		err = ws.Sorting.SetStaticPositions(sort)
// 		if err != nil {
// 			http.Error(w, err.Error(), http.StatusBadRequest)
// 		}
// 		w.WriteHeader(http.StatusOK)
// 		return
// 	}

// 	sort := ws.Sorting.GetStaticPositions()
// 	if sort == nil {
// 		http.Error(w, "Sort not found", http.StatusNotFound)
// 		return
// 	}
// 	defaultHeaders(w, r, true, "120")
// 	w.WriteHeader(http.StatusOK)
// 	err := json.NewEncoder(w).Encode(sort)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 	}
// }

func (ws *WebServer) AddItem(w http.ResponseWriter, r *http.Request) {
	items := AddItemRequest{}
	err := json.NewDecoder(r.Body).Decode(&items)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	toUpdate := make([]types.Item, len(items))
	for i, item := range items {
		toUpdate[i] = &item
	}
	ws.Index.UpsertItems(toUpdate)
	totalItems.Set(float64(len(ws.Index.Items)))
	toUpdate = nil
	w.WriteHeader(http.StatusOK)
}

func (ws *WebServer) Save(w http.ResponseWriter, _ *http.Request) {
	err := ws.Db.SaveIndex(ws.Index)
	if err != nil {
		log.Printf("Error saving index: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

type CategoryUpdateRequest struct {
	Ids     []uint                 `json:"ids"`
	Updates []types.CategoryUpdate `json:"updates"`
}

func (ws *WebServer) UpdateCategories(w http.ResponseWriter, r *http.Request) {
	update := CategoryUpdateRequest{}
	err := json.NewDecoder(r.Body).Decode(&update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ws.Index.UpdateCategoryValues(update.Ids, update.Updates)
	w.WriteHeader(http.StatusAccepted)
}

func generateStateOauthCookie() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return ""
	}
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

func (ws *WebServer) Logout(w http.ResponseWriter, _ *http.Request) {
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
		MaxAge:   7 * 86400,
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

	defaultHeaders(w, r, true, "0")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(claims)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

//func (ws *WebServer) RagData(w http.ResponseWriter, r *http.Request) {
//	defaultHeaders(w, false, "240")
//	w.WriteHeader(http.StatusOK)
//	ws.Index.Lock()
//	defer ws.Index.Unlock()
//	sorting := ws.Index.Sorting.GetSort("popular")
//
//	for i, id := range *sorting {
//		item, ok := ws.Index.Items[id]
//		if !ok {
//			continue
//		}
//		fmt.Fprintf(w, (*item).ToString())
//
//		w.Write([]byte("\n"))
//		if i > 500 {
//			break
//		}
//	}
//
//}

func (ws *WebServer) HandleUpdateFields(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, r, true, "0")
	w.WriteHeader(http.StatusOK)
	tmpFields := make(map[string]*FieldData)
	err := json.NewDecoder(r.Body).Decode(&tmpFields)
	for key, field := range tmpFields {
		facet, ok := ws.Index.Facets[field.Id]
		if ok {
			base := facet.GetBaseField()
			if base != nil {
				if field.Name != "" {
					base.Name = field.Name
				}
				if field.Description != "" {
					base.Description = field.Description
				}
			}
		}
		existing, found := ws.FieldData[key]
		if found {
			if existing.Created == 0 {
				existing.Created = time.Now().UnixMilli()
			}
			existing.Purpose = field.Purpose
			if field.Name != "" {
				existing.Name = field.Name
			}
			if field.Description != "" {
				existing.Description = field.Description
			}
			existing.Type = field.Type
			existing.LastSeen = time.Now().UnixMilli()
		} else {
			field.LastSeen = time.Now().UnixMilli()
			field.Created = time.Now().UnixMilli()
			ws.FieldData[key] = field
		}
	}
	ws.Db.SaveJsonFile(ws.FieldData, "fields.jz")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) GetFields(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, r, true, "0")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(ws.FieldData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) GetField(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, r, true, "0")
	fieldId := r.PathValue("id")
	field, ok := ws.FieldData[fieldId]
	if !ok {
		http.Error(w, "Field not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(field)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) CleanFields(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, r, true, "0")
	ws.Index.Lock()
	defer ws.Index.Unlock()
	for _, field := range ws.FieldData {
		field.ItemCount = 0
	}
	for _, itemP := range ws.Index.Items {
		item, ok := itemP.(*index.DataItem)
		if ok && !item.IsDeleted() {
			for _, field := range ws.FieldData {
				_, found := item.Fields[field.Id]
				if found {
					field.ItemCount++
				}
			}
		}
	}
	cleanFields := make(map[string]*FieldData)
	for key, field := range ws.FieldData {
		if field.ItemCount > 0 {
			cleanFields[key] = field
			facet, ok := ws.Index.Facets[field.Id]
			if ok {
				base := facet.GetBaseField()
				if base != nil {
					base.Name = field.Name
					base.Description = field.Description
					if slices.Index(field.Purpose, "Key Specification") == -1 {
						base.KeySpecification = false
					} else {
						base.KeySpecification = true
					}
				}
			} else {
				log.Printf("Field %s not found in index", key)
			}
		}
	}
	ws.FieldData = cleanFields
	err := ws.Db.SaveJsonFile(ws.FieldData, "fields.jz")
	// err := json.NewEncoder(w).Encode(cleanFields)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
}

func (ws *WebServer) UpdateFacetsFromFields(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, r, true, "0")
	w.WriteHeader(http.StatusOK)
	//toDelete := make([]uint, 0)
	for _, field := range ws.FieldData {
		facet, ok := ws.Index.Facets[field.Id]
		if ok {
			base := facet.GetBaseField()
			if base != nil {
				base.Name = field.Name
				base.Description = field.Description
				if slices.Index(field.Purpose, "do not show") != -1 {
					base.HideFacet = true
				}
				if slices.Index(field.Purpose, "UL Benchmarking") != -1 {
					base.Type = "fps"
				}
				if slices.Index(field.Purpose, "Key Specification") == -1 {
					base.KeySpecification = false
				} else {
					base.KeySpecification = true
				}
			}
			if field.ItemCount < 5 {
				log.Printf("Useless index field %s %d, count: %d", field.Name, field.Id, field.ItemCount)
				// 	toDelete = append(toDelete, field.Id)
			}
		}
	}
	// for _, id := range toDelete {
	// 	delete(ws.Index.Facets, id)
	// }

	err := ws.Db.SaveFacets(ws.Index.Facets)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) CreateFacetFromField(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, r, true, "0")
	fieldId := r.PathValue("id")
	field, ok := ws.FieldData[fieldId]
	if !ok {
		http.Error(w, "Field not found", http.StatusNotFound)
		return
	}
	_, found := ws.Index.Facets[field.Id]
	if found {
		http.Error(w, "Facet already exists", http.StatusBadRequest)
		return
	}
	baseField := &types.BaseField{
		Name:        field.Name,
		Description: field.Description,
		Id:          field.Id,
		Priority:    10,
		Searchable:  true,
	}
	if slices.Index(field.Purpose, "do not show") != -1 {
		baseField.HideFacet = true
	}
	switch field.Type {
	case KEY:
		ws.Index.AddKeyField(baseField)
	case NUMBER:
		ws.Index.AddIntegerField(baseField)
	case DECIMAL:
		ws.Index.AddDecimalField(baseField)
	}
	err := ws.Db.SaveFacets(ws.Index.Facets)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
}

func (ws *WebServer) DeleteFacet(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, r, true, "0")
	facetIdString := r.PathValue("id")
	facetId, err := strconv.Atoi(facetIdString)
	if err != nil {
		http.Error(w, "Invalid facet ID", http.StatusBadRequest)
		return
	}
	_, ok := ws.Index.Facets[uint(facetId)]
	if !ok {
		http.Error(w, "Facet not found", http.StatusNotFound)
		return
	}
	delete(ws.Index.Facets, uint(facetId))
	if err = ws.Db.SaveFacets(ws.Index.Facets); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
}

func (ws *WebServer) UpdateFacet(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, r, true, "0")
	facetIdString := r.PathValue("id")
	facetId, err := strconv.Atoi(facetIdString)
	if err != nil {
		http.Error(w, "Invalid facet ID", http.StatusBadRequest)
		return
	}
	data := types.BaseField{}
	err = json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	facet, ok := ws.Index.Facets[uint(facetId)]
	if !ok {
		http.Error(w, "Facet not found", http.StatusNotFound)
		return
	}
	current := facet.GetBaseField()
	if current == nil {
		http.Error(w, "Facet not found", http.StatusNotFound)
		return
	}
	current.UpdateFrom(&data)

	if err = ws.Db.SaveFacets(ws.Index.Facets); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	changes := make([]types.FieldChange, 0)
	changes = append(changes, types.FieldChange{
		Action:    types.UPDATE_FIELD,
		BaseField: current,
		FieldType: facet.GetType(),
	})
	if ws.Index.ChangeHandler != nil {
		ws.Index.ChangeHandler.FieldsChanged(changes)
	}
	w.WriteHeader(http.StatusOK)
}

func (ws *WebServer) MissingFacets(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, r, true, "0")
	w.WriteHeader(http.StatusOK)
	missing := make([]*FieldData, 0)
	for _, field := range ws.FieldData {
		_, ok := ws.Index.Facets[field.Id]
		if !ok {
			missing = append(missing, field)
		}
	}

	err := json.NewEncoder(w).Encode(missing)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) GetSettings(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, r, true, "0")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(types.CurrentSettings)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) GetSearchIndexedFacets(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, r, true, "5")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(types.CurrentSettings.FieldsToIndex)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) SetSearchIndexedFacets(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, r, true, "0")
	if r.Method == "POST" {
		err := json.NewDecoder(r.Body).Decode(&types.CurrentSettings.FieldsToIndex)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ws.Db.SaveSettings()
	}
	w.WriteHeader(http.StatusOK)
}

func (ws *WebServer) HandleRelationGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		types.CurrentSettings.Lock()
		defer types.CurrentSettings.Unlock()
		err := json.NewDecoder(r.Body).Decode(&types.CurrentSettings.FacetRelations)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = ws.Db.SaveSettings()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	defaultHeaders(w, r, true, "1200")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(types.CurrentSettings.FacetRelations)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) HandleFacetGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		types.CurrentSettings.Lock()
		defer types.CurrentSettings.Unlock()
		err := json.NewDecoder(r.Body).Decode(&types.CurrentSettings.FacetGroups)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = ws.Db.SaveSettings()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	defaultHeaders(w, r, true, "1200")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(types.CurrentSettings.FacetGroups)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) GetFacetList(w http.ResponseWriter, r *http.Request) {
	publicHeaders(w, r, true, "10")

	w.WriteHeader(http.StatusOK)

	res := make([]types.BaseField, len(ws.Index.Facets))
	idx := 0
	for _, f := range ws.Index.Facets {
		res[idx] = *f.GetBaseField()
		idx++
	}
	enc := json.NewEncoder(w)

	enc.Encode(res)
}

type FacetGroupingData struct {
	GroupId   uint   `json:"group_id"`
	GroupName string `json:"group_name"`
	FacetIds  []uint `json:"facet_ids"`
}

func (ws *WebServer) FacetGroupUpdate(w http.ResponseWriter, r *http.Request) {
	data := FacetGroupingData{}
	json.NewDecoder(r.Body).Decode(&data)
	if data.GroupId == 0 {
		http.Error(w, "Group ID is required", http.StatusBadRequest)
		return
	}
	changes := make([]types.FieldChange, 0)

	types.CurrentSettings.Lock()
	defer types.CurrentSettings.Unlock()
	idx := slices.IndexFunc(types.CurrentSettings.FacetGroups, func(g types.FacetGroup) bool {
		return g.Id == data.GroupId
	})
	if idx == -1 {
		if len(data.GroupName) == 0 {
			http.Error(w, "Group name is required", http.StatusBadRequest)
			return
		}
		types.CurrentSettings.FacetGroups = append(types.CurrentSettings.FacetGroups, types.FacetGroup{
			Id:   data.GroupId,
			Name: data.GroupName,
		})
	}

	for _, id := range data.FacetIds {
		facet, ok := ws.Index.Facets[id]
		if !ok {
			continue
		}
		base := facet.GetBaseField()
		if base != nil && base.GroupId != data.GroupId {
			base.GroupId = data.GroupId

			changes = append(changes, types.FieldChange{
				Action:    types.UPDATE_FIELD,
				BaseField: base,
				FieldType: facet.GetType(),
			})
		}
	}

	if ws.Index.ChangeHandler != nil {
		ws.Index.ChangeHandler.FieldsChanged(changes)
	}
	err := ws.Db.SaveFacets(ws.Index.Facets)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type MatchedRule struct {
	Rule  interface{} `json:"rule"`
	Score float64     `json:"score"`
}

type PopularityResult struct {
	Popularity float64       `json:"popularity"`
	Matches    []MatchedRule `json:"matches"`
}

func (ws *WebServer) GetItemPopularity(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, r, true, "0")
	itemIdString := r.PathValue("id")
	itemId, err := strconv.Atoi(itemIdString)
	if err != nil {
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return
	}
	item, ok := ws.Index.Items[uint(itemId)]
	if !ok {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	ret := &PopularityResult{
		Popularity: 0,
		Matches:    make([]MatchedRule, 0),
	}
	rules := types.CurrentSettings.PopularityRules
	for _, rule := range *rules {
		if rule == nil {
			continue
		}
		score := rule.GetValue(item)
		if score != 0 {
			ret.Popularity += score
			ret.Matches = append(ret.Matches, MatchedRule{
				Rule:  rule,
				Score: score,
			})
		}
	}

	err = json.NewEncoder(w).Encode(ret)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) AdminHandler() *http.ServeMux {

	srv := http.NewServeMux()
	srv.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		defaultHeaders(w, r, false, "0")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("ok"))
		if err != nil {
			log.Println("Error writing health check response")
		}
	})
	tmp := map[string]*FieldData{}
	ws.Db.LoadJsonFile(&tmp, "fields.jz")
	for key, field := range tmp {
		if field.Name == "Supplier's EcoVadis Score" {
			continue
		}
		if field.Name == "Audio control" {
			continue
		}
		if field.Name == "Suitable for commercial use" {
			continue
		}
		if field.Name == "Suitable for children" {
			continue
		}
		ws.FieldData[key] = field
	}
	tmp = nil

	srv.HandleFunc("/login", ws.Login)
	srv.HandleFunc("/logout", ws.Logout)
	srv.HandleFunc("/user", ws.User)
	//srv.HandleFunc("GET /rag", ws.RagData)
	srv.HandleFunc("/auth_callback", ws.AuthCallback)
	srv.HandleFunc("/add", ws.AuthMiddleware(ws.AddItem))

	srv.HandleFunc("PUT /key-values", ws.AuthMiddleware(ws.UpdateCategories))
	srv.HandleFunc("/save", ws.AuthMiddleware(ws.Save))
	srv.HandleFunc("PUT /fields", ws.AuthMiddleware(ws.HandleUpdateFields))
	srv.HandleFunc("/clean-fields", ws.CleanFields)
	srv.HandleFunc("/update-fields", ws.UpdateFacetsFromFields)
	srv.HandleFunc("DELETE /facets/{id}", ws.AuthMiddleware(ws.DeleteFacet))
	srv.HandleFunc("GET /facets", ws.GetFacetList)
	srv.HandleFunc("PUT /facets/{id}", ws.AuthMiddleware(ws.UpdateFacet))
	srv.HandleFunc("GET /index/facets", ws.AuthMiddleware(ws.GetSearchIndexedFacets))
	srv.HandleFunc("POST /index/facets", ws.AuthMiddleware(ws.SetSearchIndexedFacets))
	srv.HandleFunc("GET /item/{id}/popularity", ws.AuthMiddleware(ws.GetItemPopularity))
	srv.HandleFunc("GET /fields/{id}/add", ws.AuthMiddleware(ws.CreateFacetFromField))
	srv.HandleFunc("GET /fields", ws.GetFields)
	srv.HandleFunc("GET /item/{id}", ws.AuthMiddleware(JsonHandler(ws.Tracking, ws.GetItem)))
	srv.HandleFunc("GET /settings", ws.GetSettings)
	srv.HandleFunc("PUT /facet-group", ws.AuthMiddleware(ws.FacetGroupUpdate))

	srv.HandleFunc("GET /missing-fields", ws.AuthMiddleware(ws.MissingFacets))
	srv.HandleFunc("GET /fields/{id}", ws.GetField)
	srv.HandleFunc("/rules/popular", ws.AuthMiddleware(ws.HandlePopularRules))
	srv.HandleFunc("/sort/popular", ws.AuthMiddleware(ws.HandlePopularOverride))
	srv.HandleFunc("/relation-groups", ws.HandleRelationGroups)
	srv.HandleFunc("/facet-groups", ws.HandleFacetGroups)
	//srv.HandleFunc("/sort/static", ws.AuthMiddleware(ws.HandleStaticPositions))
	//srv.HandleFunc("/sort/fields", ws.AuthMiddleware(ws.HandleFieldSort))
	return srv
}
