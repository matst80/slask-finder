package server

import (
	"github.com/matst80/slask-finder/pkg/embeddings"
	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/storage"
	"github.com/matst80/slask-finder/pkg/tracking"
	"github.com/matst80/slask-finder/pkg/types"
	"golang.org/x/oauth2"
)

// WebServerInterface defines the common interface for both admin and client web servers
type WebServerInterface interface {
	// Core operations
	UpsertItems(items []types.Item)
	GetFacet(id uint) (types.Facet, bool)
	GetAllFacets() map[uint]types.Facet
	SearchItems(query string) *types.ItemList
	FilterItems(query string, res *types.ItemList)
	GetEmbeddings(itemId uint) (types.Embeddings, bool)
	HasEmbeddings(itemId uint) bool
}

// Deprecated: Use AdminWebServer or ClientWebServer instead
type WebServer struct {
	OAuthConfig      *oauth2.Config
	Index            *index.ItemIndex
	ContentIndex     *index.ContentIndex
	Db               *storage.DataRepository
	Sorting          *index.Sorting
	Cache            *Cache
	Embeddings       embeddings.Embeddings
	Tracking         tracking.Tracking
	FieldData        map[string]*FieldData
	PriceWatches     *PriceWatchesData
	FacetLimit       int
	SearchFacetLimit int
	// New handlers moved from Index
	FacetHandler      *index.FacetItemHandler
	SearchHandler     *index.FreeTextItemHandler
	SortingHandler    *index.SortingItemHandler
	EmbeddingsHandler *index.ItemEmbeddingsHandler
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
	LastSeen  int64    `json:"lastSeen"`
	Created   int64    `json:"created"`
}

type AddItemRequest []index.DataItem

type SuggestResult struct {
	Prefix string   `json:"prefix"`
	Word   string   `json:"match"`
	Other  []string `json:"other"`
	Hits   int      `json:"hits"`
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
