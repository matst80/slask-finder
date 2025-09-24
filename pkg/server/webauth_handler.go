package server

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"runtime"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

func init() {
	gob.Register(webauthn.Credential{})
	gob.Register([]webauthn.Credential{})
}

type WebAuthHandler struct {
	*webauthn.WebAuthn
	userMutex    *sync.RWMutex
	sessionMutex *sync.RWMutex
	users        map[string]*User
	sessions     map[string]*webauthn.SessionData
}

func loadUsers() (map[string]*User, error) {
	// load users from persistent storage
	file, err := os.Open(path.Join("data", "users.gob.gz"))
	if err != nil {
		log.Println("Could not open users file, starting with empty user list:", err)
		return make(map[string]*User), nil
	}
	defer file.Close()
	defer runtime.GC()

	zipReader, err := gzip.NewReader(file)
	if err != nil {
		return make(map[string]*User), err
	}
	defer zipReader.Close()
	decoder := gob.NewDecoder(zipReader)

	var users map[string]*User
	err = decoder.Decode(&users)
	if err != nil {
		return make(map[string]*User), err
	}
	return users, nil
}

func (w *WebAuthHandler) saveUsers() error {
	// save users to persistent storage
	w.userMutex.RLock()
	defer w.userMutex.RUnlock()
	tmpFileName := path.Join("data", "users.gob.gz.tmp")

	file, err := os.Create(tmpFileName)
	if err != nil {
		return err
	}
	defer file.Close()

	zipWriter := gzip.NewWriter(file)
	defer zipWriter.Close()

	encoder := gob.NewEncoder(zipWriter)
	if err := encoder.Encode(w.users); err != nil {
		return err
	}

	if err := zipWriter.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpFileName, path.Join("data", "users.gob.gz")); err != nil {
		log.Println("Could not rename temporary users file:", err)
		return err
	}

	return nil
}

func NewWebAuthHandler(config *webauthn.Config) (*WebAuthHandler, error) {
	w, err := webauthn.New(config)
	if err != nil {
		return nil, err
	}

	r := &WebAuthHandler{
		WebAuthn:     w,
		userMutex:    &sync.RWMutex{},
		sessionMutex: &sync.RWMutex{},
		users:        make(map[string]*User),
		sessions:     make(map[string]*webauthn.SessionData),
	}
	users, err := loadUsers()
	if err == nil {
		r.users = users
		log.Println("Starting with loaded user list")
	} else {
		log.Println("Starting with empty user list")
	}

	return r, nil
}

func (w *WebAuthHandler) saveLoginSession(s *webauthn.SessionData) string {
	id := uuid.New().String()
	w.sessionMutex.Lock()
	defer w.sessionMutex.Unlock()
	w.sessions[id] = s
	return id
}

func (w *WebAuthHandler) loadSessionByID(id string) (*webauthn.SessionData, error) {
	w.sessionMutex.RLock()
	defer w.sessionMutex.RUnlock()
	s, ok := w.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}
	return s, nil
}

func (w *WebAuthHandler) createUserWithID(id, name, email string, isAdmin bool) (*User, error) {
	w.userMutex.Lock()
	defer w.userMutex.Unlock()
	if _, exists := w.users[id]; exists {
		return nil, fmt.Errorf("user with ID %s already exists", id)
	}
	user := &User{ID: []byte(id),
		Name: id, DisplayName: name, Email: email, IsAdmin: isAdmin, Credentials: make([]webauthn.Credential, 0)}
	w.users[id] = user
	go w.saveUsers()
	return user, nil
}

func (w *WebAuthHandler) loadUserByID(id string) (*User, error) {
	w.userMutex.RLock()
	defer w.userMutex.RUnlock()
	user, exists := w.users[id]
	if !exists {
		return nil, fmt.Errorf("user with ID %s does not exist", id)
	}
	return user, nil
}

func (w *WebAuthHandler) findUserByPasskey(rawID, userHandle []byte) (user webauthn.User, err error) {
	w.userMutex.RLock()
	defer w.userMutex.RUnlock()
	for _, user := range w.users {
		if string(user.ID) == string(userHandle) {
			return user, nil
		}
		for _, cred := range user.Credentials {
			if bytes.Equal(cred.ID, rawID) {
				return user, nil
			}
		}
	}
	return nil, fmt.Errorf("user not found")
}

func getClaimsFromToken(r *http.Request) (jwt.MapClaims, error) {
	cookie, err := r.Cookie(tokenCookieName)
	if err != nil {
		return nil, err
	}

	token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

func (w *WebAuthHandler) CreateChallenge(rw http.ResponseWriter, r *http.Request) {

	// Crude / Abstract example of retrieving the user this registration will belong to. The user must be logged in
	// for this step unless you plan to register the user and the credential at the same time i.e. usernameless.
	// The user should have a unique and stable value returned from WebAuthnID that can be used to retrieve the
	// account details for the user.
	claims, err := getClaimsFromToken(r)
	if err != nil {
		http.Error(rw, "Unauthorized", http.StatusUnauthorized)
		return
	}
	id := claims["exp"].(float64)
	user, err := w.createUserWithID(fmt.Sprintf("%d", int64(id)), claims["name"].(string), claims["username"].(string), false)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)

		return
	}

	creation, s, err := w.BeginMediatedRegistration(
		user,
		protocol.MediationDefault,
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
		webauthn.WithExclusions(webauthn.Credentials(user.WebAuthnCredentials()).CredentialDescriptors()),
		webauthn.WithExtensions(map[string]any{"credProps": true, "payment": map[string]any{
			"isPayment": true,
		}}),
	)

	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)

		return
	}

	user.SetSessionData(s)

	encoder := json.NewEncoder(rw)

	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	rw.WriteHeader(http.StatusOK)

	if err = encoder.Encode(creation); err != nil {
		rw.WriteHeader(http.StatusInternalServerError)

		return
	}

}

func (w *WebAuthHandler) ValidateCreateChallengeResponse(rw http.ResponseWriter, r *http.Request) {

	// Crude / Abstract example of retrieving the user this registration will belong to. The user must be logged in
	// for this step unless you plan to register the user and the credential at the same time i.e. usernameless.
	// The user should have a unique and stable value returned from WebAuthnID that can be used to retrieve the
	// account details for the user.
	claims, err := getClaimsFromToken(r)
	if err != nil {
		http.Error(rw, "Unauthorized", http.StatusUnauthorized)
		return
	}
	id := claims["exp"].(float64)

	user, err := w.loadUserByID(fmt.Sprintf("%d", int64(id)))

	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)

		return
	}

	// Crude example loading the session data securely from the start step for the register action. This should be
	// loaded from a place the user and user agent has no access to it. For example using an opaque session cookie.
	s := user.Session

	credential, err := w.FinishRegistration(user, *s, r)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)

		return
	}

	// Crude / Abstract example of adding the credential to the list of credentials for the user. This is critical
	// for performing future logins.
	user.AddCredential(*credential)
	go w.saveUsers()

	// save users
	rw.WriteHeader(http.StatusOK)
	encoder := json.NewEncoder(rw)

	if err = encoder.Encode(user.Credentials); err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
	}
}

func (w *WebAuthHandler) LoginChallenge(rw http.ResponseWriter, r *http.Request) {

	assertion, s, err := w.BeginDiscoverableMediatedLogin(protocol.MediationDefault)

	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)

		return
	}

	// Crude example saving the session data securely to be loaded in the finish step of the login action. This
	// should be stored in such a way that the user and user agent has no access to it. For example using an opaque
	// session cookie.
	id := w.saveLoginSession(s)
	http.SetCookie(rw, &http.Cookie{
		Name:     "session_id",
		Value:    id,
		HttpOnly: true,
	})
	encoder := json.NewEncoder(rw)

	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	rw.WriteHeader(http.StatusOK)

	if err = encoder.Encode(assertion); err != nil {
		rw.WriteHeader(http.StatusInternalServerError)

		return
	}
}

func (w *WebAuthHandler) LoginChallengeResponse(rw http.ResponseWriter, r *http.Request) {

	// Crude example loading the session data securely from the start step for the login action. This should be
	// loaded from a place the user and user agent has no access to it. For example using an opaque session cookie.
	sessionID, err := r.Cookie("session_id")
	if err != nil {
		rw.WriteHeader(http.StatusUnauthorized)

		return
	}

	s, err := w.loadSessionByID(sessionID.Value)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)

		return
	}

	validatedUser, validatedCredential, err := w.FinishPasskeyLogin(w.findUserByPasskey, *s, r)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)

		return
	}

	// This type assertion is necessary to perform the necessary updates.
	user, ok := validatedUser.(*User)
	if !ok {
		rw.WriteHeader(http.StatusInternalServerError)

		return
	}

	// Modify the matching credential in the user struct which is critical for proper future validations as the
	// metadata for this credential has been updated. No type assertion is required here since the LoadUser function
	// returns the concrete implementation, you may have to adjust this if you return the abstract implementation
	// instead.

	err = user.UpdateCredential(*validatedCredential)
	go w.saveUsers()
	if err != nil {

		rw.WriteHeader(http.StatusInternalServerError)

		return
	}
	ownToken, err := createToken(user.Email, user.Name, user.GetUserClaim())
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(rw, &http.Cookie{
		Name:     tokenCookieName,
		Value:    ownToken,
		Path:     "/",
		Expires:  time.Now().Add(time.Hour * 24),
		HttpOnly: true,
		//Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	rw.WriteHeader(http.StatusOK)
}

// UserSummary represents a user without sensitive credential information
type UserSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	IsAdmin     bool   `json:"isAdmin"`
}

// ListUsers returns a list of all users without sensitive data
func (w *WebAuthHandler) ListUsers(rw http.ResponseWriter, r *http.Request) {
	w.userMutex.RLock()
	defer w.userMutex.RUnlock()

	users := make([]UserSummary, 0, len(w.users))
	for _, user := range w.users {
		users = append(users, UserSummary{
			ID:          string(user.ID),
			Name:        user.Name,
			Email:       user.Email,
			DisplayName: user.DisplayName,
			IsAdmin:     user.IsAdmin,
		})
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(rw).Encode(users); err != nil {
		log.Printf("Error encoding users response: %v", err)
		http.Error(rw, "Internal server error", http.StatusInternalServerError)
	}
}

// DeleteUser removes a user by ID
func (w *WebAuthHandler) DeleteUser(rw http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	if userID == "" {
		http.Error(rw, "User ID is required", http.StatusBadRequest)
		return
	}

	w.userMutex.Lock()
	defer w.userMutex.Unlock()

	if _, exists := w.users[userID]; !exists {
		http.Error(rw, "User not found", http.StatusNotFound)
		return
	}

	delete(w.users, userID)
	go w.saveUsers()

	rw.WriteHeader(http.StatusNoContent)
}

// UserUpdateRequest represents the request body for updating a user
type UserUpdateRequest struct {
	Name        *string `json:"name,omitempty"`
	Email       *string `json:"email,omitempty"`
	DisplayName *string `json:"displayName,omitempty"`
	IsAdmin     *bool   `json:"isAdmin,omitempty"`
}

// UpdateUser updates user properties
func (w *WebAuthHandler) UpdateUser(rw http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	if userID == "" {
		http.Error(rw, "User ID is required", http.StatusBadRequest)
		return
	}

	var updateRequest UserUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&updateRequest); err != nil {
		http.Error(rw, "Invalid request body", http.StatusBadRequest)
		return
	}

	w.userMutex.Lock()
	defer w.userMutex.Unlock()

	user, exists := w.users[userID]
	if !exists {
		http.Error(rw, "User not found", http.StatusNotFound)
		return
	}

	// Update only the fields that were provided
	if updateRequest.Name != nil {
		user.Name = *updateRequest.Name
	}
	if updateRequest.Email != nil {
		user.Email = *updateRequest.Email
	}
	if updateRequest.DisplayName != nil {
		user.DisplayName = *updateRequest.DisplayName
	}
	if updateRequest.IsAdmin != nil {
		user.IsAdmin = *updateRequest.IsAdmin
	}

	go w.saveUsers()

	// Return the updated user summary
	updatedUser := UserSummary{
		ID:          string(user.ID),
		Name:        user.Name,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		IsAdmin:     user.IsAdmin,
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(rw).Encode(updatedUser); err != nil {
		log.Printf("Error encoding updated user response: %v", err)
		http.Error(rw, "Internal server error", http.StatusInternalServerError)
	}
}
