package server

import (
	"tornberg.me/facet-search/pkg/index"
	"tornberg.me/facet-search/pkg/persistance"
	"tornberg.me/facet-search/pkg/tracking"
)

type WebServer struct {
	Index            *index.Index
	Db               *persistance.Persistance
	Sorting          *index.Sorting
	Cache            *Cache
	Tracking         *tracking.ClickHouse
	FacetLimit       int
	SearchFacetLimit int
}

type AddItemRequest []index.DataItem

type SuggestResult struct {
	Word string `json:"match"`
	Hits int    `json:"hits"`
}

type FieldValueAndItemId struct {
	Value int  `json:"value"`
	Id    uint `json:"id"`
}

type FacetItem struct {
	Id    uint   `json:"id"`
	Name  string `json:"value"`
	Count int    `json:"count"`
}
