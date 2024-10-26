package server

import "github.com/matst80/slask-finder/pkg/index"

type SearchResponse struct {
	Facets    []index.JsonFacet `json:"facets,omitempty"`
	TotalHits int               `json:"totalHits"`
}
