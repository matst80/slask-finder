package server

import (
	"bytes"
	"fmt"

	"github.com/go-webauthn/webauthn/webauthn"
)

type User struct {
	ID          []byte
	Name        string
	Email       string
	DisplayName string
	IsAdmin     bool
	Credentials []webauthn.Credential
	Session     *webauthn.SessionData
}

var users = map[string]*User{}

func (u *User) WebAuthnID() []byte {
	return u.ID
}

func (u *User) GetUserClaim() string {
	if u.IsAdmin {
		return "admin"
	}
	return "user"
}

func (u *User) WebAuthnName() string {
	return u.Name
}

func (u *User) WebAuthnDisplayName() string {
	return u.DisplayName
}

func (u *User) WebAuthnCredentials() []webauthn.Credential {
	return u.Credentials
}

func (u *User) AddCredential(cred webauthn.Credential) {
	u.Credentials = append(u.Credentials, cred)
}

func (u *User) UpdateCredential(updatedCred webauthn.Credential) error {
	for i, cred := range u.Credentials {
		if bytes.Equal(cred.ID, updatedCred.ID) {
			u.Credentials[i] = updatedCred
			return nil
		}
	}
	return fmt.Errorf("credential not found")
}

func (u *User) SetSessionData(s *webauthn.SessionData) {
	u.Session = s
}
