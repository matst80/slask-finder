package tracking

import (
	"net/http"

	"github.com/matst80/slask-finder/pkg/index"
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
	TrackSession(session_id uint32, r *http.Request) error
	TrackSearch(session_id uint32, filters *index.Filters, query string, page int) error
}
