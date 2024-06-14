package sync

type SyncMaster struct {
	Clients map[string]*SyncClient
}

type SyncClient struct {
}
