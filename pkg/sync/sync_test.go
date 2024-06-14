package sync

import (
	"testing"

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

func TestSync(t *testing.T) {
	masterTransport := RabbitTransportMaster{
		RabbitTopics: RabbitTopics{
			ItemChangedTopic: "item_changed",
			ItemAddedTopic:   "item_added",
			ItemDeletedTopic: "item_deleted",
		},
		Url: "amqp://admin:12bananer@localhost:5672/",
	}
	clientTransport1 := RabbitTransportClient{
		RabbitTopics: RabbitTopics{
			ItemChangedTopic: "item_changed",
			ItemAddedTopic:   "item_added",
			ItemDeletedTopic: "item_deleted",
		},
		Url: "amqp://admin:12bananer@localhost:5672/",
	}
	clientTransport2 := RabbitTransportClient{
		RabbitTopics: RabbitTopics{
			ItemChangedTopic: "item_changed",
			ItemAddedTopic:   "item_added",
			ItemDeletedTopic: "item_deleted",
		},
		Url: "amqp://admin:12bananer@localhost:5672/",
	}

	server := NewBaseServer(&masterTransport)
	index1 := index.NewIndex(search.NewFreeTextIndex(&search.Tokenizer{MaxTokens: 128}))
	index2 := index.NewIndex(search.NewFreeTextIndex(&search.Tokenizer{MaxTokens: 128}))
	client1 := MakeBaseClient(index1, &clientTransport1)
	client2 := MakeBaseClient(index2, &clientTransport2)

	server.RegisterClient(client1)
	server.RegisterClient(client2)

	item := &index.DataItem{
		BaseItem: index.BaseItem{
			Id:    1,
			Title: "Test",
		},
		Fields: map[uint]string{
			1: "Test",
		},
	}

	server.ItemAdded(item)

	if _, ok := client1.Index.Items[1]; !ok {
		t.Error("Item not added to client 1")
	}

	if _, ok := client2.Index.Items[1]; !ok {
		t.Error("Item not added to client 2")
	}

	item.Fields[1] = "Test2"

	server.ItemChanged(item)

	if *client1.Index.Items[1].Fields[1].Value != "Test2" {
		t.Error("Item not updated on client 1")
	}

	if *client2.Index.Items[1].Fields[1].Value != "Test2" {
		t.Error("Item not updated on client 2")
	}

	server.ItemDeleted(item)

	if _, ok := client1.Index.Items[1]; ok {
		t.Error("Item not deleted from client 1")
	}

	if _, ok := client2.Index.Items[1]; ok {
		t.Error("Item not deleted from client 2")
	}
}
