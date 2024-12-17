package tracking

import (
	"net/http"

	"github.com/matst80/slask-finder/pkg/types"
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
	TrackSearch(session_id int, filters *types.Filters, query string, page int)
}
