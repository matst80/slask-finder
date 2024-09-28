package sync

import "tornberg.me/facet-search/pkg/index"

type SyncMaster struct {
	Clients map[string]*SyncClient
}

type SyncClient struct {
}

type SyncItems []*index.DataItem
