package tracking

import (
	"net/http"

	"tornberg.me/facet-search/pkg/index"
)

type Tracking interface {
	TrackSession(session_id uint32, r *http.Request) error
	TrackSearch(session_id uint32, filters *index.Filters, query string) error
	TrackClick(session_id uint32, item_id uint, position float32) error
	TrackAddToCart(session_id uint32, item_id uint, quantity uint) error
	TrackPurchase(session_id uint32, item_id uint, quantity uint) error
	TrackImpression(session_id uint32, item_id uint, position float32) error
}
