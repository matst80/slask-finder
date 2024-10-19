package server

import "tornberg.me/facet-search/pkg/index"

type SearchResponse struct {
	Facets    []index.JsonFacet `json:"facets,omitempty"`
	TotalHits int               `json:"totalHits"`
}
