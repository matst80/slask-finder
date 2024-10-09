package tracking

import (
	"net/http"

	"tornberg.me/facet-search/pkg/index"
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
	TrackClick(session_id uint32, item_id uint, position float32) error
	TrackAddToCart(session_id uint32, item_id uint, quantity uint) error
	TrackPurchase(session_id uint32, item_id uint, quantity uint) error
	TrackImpressions(session_id uint32, viewedItems []Impression) error
	TrackAction(session_id uint32, value TrackingAction) error
}
