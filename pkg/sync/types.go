package sync

import "tornberg.me/facet-search/pkg/index"

type BaseClient struct {
	Server    *BaseMaster
	Index     *index.Index
	Transport *TransportClient
}

type RabbitConfig struct {
	ItemsUpsertedTopic string
	ItemDeletedTopic   string
	PriceLoweredTopic  string
	Url                string
}

func MakeBaseClient(index *index.Index, transport TransportClient) *BaseClient {
	return &BaseClient{
		Index:     index,
		Transport: &transport,
	}
}

type TransportMaster interface {
	Connect() error
	SendItemsAdded(item []*index.DataItem) error
	//SendItemChanged(item *index.DataItem) error
	SendItemDeleted(id uint) error
}

type TransportClient interface {
	Connect() error
	OnItemAdded(items []*index.DataItem)
	//OnItemChanged(item *index.DataItem)
	OnItemDeleted(id uint)
}

type BaseMaster struct {
	Clients   []*BaseClient
	Transport *TransportMaster
}
