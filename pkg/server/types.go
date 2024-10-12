package server

import (
	"golang.org/x/oauth2"
	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/index"
	"tornberg.me/facet-search/pkg/persistance"
	"tornberg.me/facet-search/pkg/tracking"
)

type WebServer struct {
	OAuthConfig      *oauth2.Config
	Index            *index.Index
	Db               *persistance.Persistance
	Sorting          *index.Sorting
	Cache            *Cache
	Tracking         tracking.Tracking
	FacetLimit       int
	SearchFacetLimit int
}

type AddItemRequest []index.DataItem

type SuggestResult struct {
	Word  string   `json:"match"`
	Other []string `json:"other"`
	Hits  int      `json:"hits"`
}

type FieldValueAndItemId struct {
	Value int  `json:"value"`
	Id    uint `json:"id"`
}

type FacetItem struct {
	*facet.BaseField
	FieldType string `json:"fieldType"`
	Count     int    `json:"count"`
}
