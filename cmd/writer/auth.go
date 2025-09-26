package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/matst80/slask-finder/pkg/index"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type AuthHandler interface {
	Login(w http.ResponseWriter, r *http.Request)
	Logout(w http.ResponseWriter, r *http.Request)
	AuthCallback(w http.ResponseWriter, r *http.Request)
	User(w http.ResponseWriter, r *http.Request)
	Middleware(next http.HandlerFunc) http.HandlerFunc
	//ParseJwt(tokenString string) (*jwt.Token, error)
}

type MockAuth struct{}

func (m *MockAuth) Login(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:  tokenCookieName,
		Value: "mock-token",
	})
	w.WriteHeader(http.StatusOK)
}

func (m *MockAuth) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   tokenCookieName,
		Value:  "",
		MaxAge: -1,
	})
	w.WriteHeader(http.StatusOK)
}

func (m *MockAuth) AuthCallback(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     tokenCookieName,
		Value:    "mock-token",
		HttpOnly: true,
	})
	http.Redirect(w, r, "/register", http.StatusTemporaryRedirect)
}

func (m *MockAuth) User(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte(`{"username":"mock-user","name":"Mock User","role":"admin"}`))
	if err != nil {
		log.Printf("error sending user response: %v", err)
	}
}

func (m *MockAuth) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

type GoogleAuth struct {
	serverKey    []byte
	serverApiKey string
	authConfig   *oauth2.Config
}

func NewGoogleAuth() (*GoogleAuth, error) {
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	callbackUrl := os.Getenv("CALLBACK_URL")
	clientId := os.Getenv("GOOGLE_CLIENT_ID")
	if clientId == "" || clientSecret == "" || callbackUrl == "" {
		return nil, fmt.Errorf("GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET or CALLBACK_URL environment variable not set")
	}
	hash := os.Getenv("SLASK_TOKEN_HASH")
	if hash == "" {
		return nil, fmt.Errorf("SLASK_TOKEN_HASH environment variable not set")
	}
	secretKey := []byte(hash)
	apiKey := os.Getenv("SLASK_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("SLASK_API_KEY environment variable not set")
	}
	authConfig := &oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		RedirectURL:  callbackUrl,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
	return &GoogleAuth{
		authConfig:   authConfig,
		serverKey:    secretKey,
		serverApiKey: apiKey,
	}, nil
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

func (a *GoogleAuth) createToken(username, name, role string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"username": username,
			"name":     name,
			"role":     role,
			"exp":      time.Now().Add(time.Hour * 24).Unix(),
		})

	tokenString, err := token.SignedString(a.serverKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (ws *GoogleAuth) Login(w http.ResponseWriter, r *http.Request) {
	oauthState := generateStateOauthCookie()
	url := ws.authConfig.AuthCodeURL(oauthState, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
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

func (ws *GoogleAuth) Logout(w http.ResponseWriter, _ *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   tokenCookieName,
		Value:  "",
		MaxAge: -1,
	})
	w.WriteHeader(http.StatusOK)
}

type ContextValue string

var ContextRole = ContextValue("role")

func (ws *GoogleAuth) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		role := "anonymous"
		if auth != ws.serverApiKey {
			cookie, err := r.Cookie(tokenCookieName)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			token, err := ws.ParseJwt(cookie.Value)
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
			role = claims["role"].(string)

			index.AllowConditionalData = role == "admin"

		} else {
			role = "api"
			index.AllowConditionalData = true
		}
		ctx := context.WithValue(r.Context(), ContextRole, role)
		next.ServeHTTP(w, r.WithContext(ctx))
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

func (ws *GoogleAuth) AuthCallback(w http.ResponseWriter, r *http.Request) {

	token, err := ws.authConfig.Exchange(context.Background(), r.FormValue("code"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userData, err := getUserData(token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ownToken, err := ws.createToken(userData.Email, userData.Name, "gmail")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     tokenCookieName,
		Value:    ownToken,
		Path:     "/",
		Expires:  time.Now().Add(time.Hour * 24),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	http.Redirect(w, r, "/register", http.StatusTemporaryRedirect)
}

func (ws *GoogleAuth) ParseJwt(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return ws.serverKey, nil
	})
}

func (ws *GoogleAuth) User(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(tokenCookieName)
	if err != nil || cookie.Value == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	token, err := ws.ParseJwt(cookie.Value)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !token.Valid {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		http.Error(w, "No claims found", http.StatusBadRequest)
		return
	}

	//defaultHeaders(w, r, true, "0")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(claims)
	if err != nil {
		log.Printf("error sending user response: %v", err)
	}

}
