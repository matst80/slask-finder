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
	FieldData        map[string]*FieldData
	FacetLimit       int
	SearchFacetLimit int
}

type DataType = int

const (
	KEY     = DataType(0)
	NUMBER  = DataType(1)
	DECIMAL = DataType(2)
)

type FieldData struct {
	Id          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	//Identifier  string   `json:"identifier"`
	Purpose   []string `json:"purpose"`
	Type      DataType `json:"type"`
	ItemCount int      `json:"itemCount"`
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
