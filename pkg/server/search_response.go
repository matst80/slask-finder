package server

import "tornberg.me/facet-search/pkg/index"

type SearchResponse struct {
	Items     []index.ResultItem `json:"items"`
	Facets    index.Facets       `json:"facets"`
	Page      int                `json:"page"`
	PageSize  int                `json:"pageSize"`
	TotalHits int                `json:"totalHits"`
}
