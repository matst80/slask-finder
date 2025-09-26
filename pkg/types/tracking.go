package types

import (
	"net/http"
)

type Impression struct {
	Id       uint    `json:"id"`
	Position float32 `json:"position"`
}

type TrackingAction struct {
	Action string `json:"action"`
	Reason string `json:"reason"`
}

type Tracking interface {
	TrackSession(session_id int, r *http.Request)
	TrackSearch(session_id int, filters *Filters, resultLen int, query string, page int, r *http.Request)
	Close() error
}
