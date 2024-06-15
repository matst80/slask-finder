package sync

import (
	"testing"
	"time"

	"tornberg.me/facet-search/pkg/index"
	"tornberg.me/facet-search/pkg/search"
)

func NewBaseServer(transport TransportMaster) *BaseMaster {
	return &BaseMaster{
		Clients:   []*BaseClient{},
		Transport: &transport,
	}
}

func (s *BaseMaster) RegisterClient(client *BaseClient) {
	s.Clients = append(s.Clients, client)
}

func (s *BaseMaster) ItemChanged(item *index.DataItem) {

	for _, client := range s.Clients {
		client.UpsertItem(item)
	}
}

func (s *BaseMaster) ItemDeleted(item *index.DataItem) {
	for _, client := range s.Clients {
		client.DeleteItem(item.Id)
	}
}

func (s *BaseMaster) ItemAdded(item *index.DataItem) {
	for _, client := range s.Clients {
		client.UpsertItem(item)
	}
}

func (c *BaseClient) UpsertItem(item *index.DataItem) {
	c.Index.UpsertItem(item)
}

func (c *BaseClient) DeleteItem(id uint) {
	c.Index.DeleteItem(id)
}

func TestSendChanges(t *testing.T) {
	masterTransport := RabbitTransportMaster{
		RabbitTopics: RabbitTopics{
			ItemChangedTopic: "item_changed",
			ItemAddedTopic:   "item_added",
			ItemDeletedTopic: "item_deleted",
		},
		Url: "amqp://admin:12bananer@localhost:5672/",
	}
	err := masterTransport.Connect()
	if err != nil {
		t.Error(err)
	}
	err = masterTransport.SendItemAdded(&index.DataItem{
		BaseItem: index.BaseItem{
			Id:    3,
			Title: "Test",
		},
		Fields: map[uint]string{
			1: "Test",
		},
	})

	if err != nil {
		t.Error(err)
	}
}

func TestSync(t *testing.T) {
	masterTransport := RabbitTransportMaster{
		RabbitTopics: RabbitTopics{
			ItemChangedTopic: "item_changed",
			ItemAddedTopic:   "item_added",
			ItemDeletedTopic: "item_deleted",
		},
		Url: "amqp://admin:12bananer@localhost:5672/",
	}
	index1 := index.NewIndex(search.NewFreeTextIndex(&search.Tokenizer{MaxTokens: 128}))

	clientTransport1 := RabbitTransportClient{
		RabbitTopics: RabbitTopics{
			ItemChangedTopic: "item_changed",
			ItemAddedTopic:   "item_added",
			ItemDeletedTopic: "item_deleted",
		},

		Url: "amqp://admin:12bananer@localhost:5672/",
	}

	err := masterTransport.Connect()
	if err != nil {
		t.Error(err)
	}
	err = clientTransport1.Connect(index1)
	if err != nil {
		t.Error(err)
	}

	defer masterTransport.Close()
	defer clientTransport1.Close()

	item := &index.DataItem{
		BaseItem: index.BaseItem{
			Id:    1,
			Title: "Test",
		},
		Fields: map[uint]string{
			1: "Test",
		},
	}

	err = masterTransport.SendItemAdded(item)

	if err != nil {
		t.Error(err)
	}
	time.Sleep(time.Second)

	if _, ok := index1.Items[1]; !ok {
		t.Error("Item not added to client 1")
	}

	item.Fields[1] = "Test2"

	err = masterTransport.SendItemChanged(item)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(time.Second)

	firstItem1, ok1 := index1.Items[1]

	if !ok1 {
		t.Error("Item not updated on client")
		return
	}

	if *firstItem1.Fields[1].Value != "Test2" {
		t.Error("Item not updated on client 1")
	}

	masterTransport.SendItemDeleted(item.Id)
	time.Sleep(time.Second)

	if _, ok := index1.Items[1]; ok {
		t.Error("Item not deleted from client 1")
	}

}
