package messaging

import (
	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/types"
)

type BaseClient struct {
	Server    *BaseMaster
	Index     *index.ItemIndex
	Transport *TransportClient
}

type RabbitConfig struct {
	ItemsUpsertedTopic string
	ItemDeletedTopic   string
	FieldChangeTopic   string
	PriceLoweredTopic  string
	Url                string
	VHost              string
}

func MakeBaseClient(index *index.ItemIndex, transport TransportClient) *BaseClient {
	return &BaseClient{
		Index:     index,
		Transport: &transport,
	}
}

type TransportMaster interface {
	Connect() error
	SendItemsAdded(item []*index.DataItem) error
	SendFieldChange([]types.FieldChange) error
	//SendItemChanged(item *index.DataItem) error
	SendItemDeleted(id uint) error
}

type TransportClient interface {
	Connect() error
	OnItemAdded(items []*index.DataItem)
	OnFieldChange(items []types.FieldChange)
	//OnItemChanged(item *index.DataItem)
	OnItemDeleted(id uint)
}

type BaseMaster struct {
	Clients   []*BaseClient
	Transport *TransportMaster
}
