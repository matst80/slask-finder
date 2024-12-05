package server

import (
	"github.com/matst80/slask-finder/pkg/embeddings"
	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/persistance"
	"github.com/matst80/slask-finder/pkg/tracking"
	"github.com/matst80/slask-finder/pkg/types"
	"golang.org/x/oauth2"
)

type WebServer struct {
	OAuthConfig      *oauth2.Config
	Index            *index.Index
	ContentIndex     *index.ContentIndex
	Db               *persistance.Persistance
	Sorting          *index.Sorting
	Cache            *Cache
	Embeddings       embeddings.Embeddings
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
	*types.BaseField
	FieldType string `json:"fieldType"`
	Count     int    `json:"count"`
}
