package server

import (
	"encoding/json"
	"net/http"

	"tornberg.me/facet-search/pkg/index"
)

type SearchRequest struct {
	Search index.Filters `json:"filter"`

	// Sort     []index.Sort `json:"sort"`
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}

func QueryFromRequest(r *http.Request) (SearchRequest, error) {
	sr := SearchRequest{
		Page:     0,
		PageSize: 25,
	}
	err := json.NewDecoder(r.Body).Decode(&sr)
	if err != nil {
		return sr, err
	}
	return sr, nil
}
