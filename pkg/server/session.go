package server

import (
	"fmt"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

var sessions = map[string]*webauthn.SessionData{}

func saveLoginSession(s *webauthn.SessionData) string {
	id := uuid.New().String()
	sessions[id] = s
	return id
}

func loadSessionByID(id string) (*webauthn.SessionData, error) {
	s, ok := sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}
	return s, nil
}
