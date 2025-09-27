package messaging

import "github.com/matst80/slask-finder/pkg/index"

type SyncMaster struct {
	Clients map[string]*SyncClient
}

type SyncClient struct {
}

type SyncItems []*index.DataItem

type ChangeTopic string

const (
	ItemsChanged  ChangeTopic = "item_changed"
	FacetsChanged ChangeTopic = "facets_changed"
)
