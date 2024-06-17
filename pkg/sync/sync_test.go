package sync

import (
	"log"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"tornberg.me/facet-search/pkg/index"
	"tornberg.me/facet-search/pkg/search"
)

var rabbitConfig = RabbitConfig{
	ItemChangedTopic: "item_changed",
	ItemAddedTopic:   "item_added",
	ItemDeletedTopic: "item_deleted",
	Url:              "amqp://admin:12bananer@localhost:5672/",
}

func createTopic(ch *amqp.Channel, topic string) error {
	err := ch.ExchangeDeclare(
		topic,
		"topic", // type
		true,    // durable
		false,   // auto-deleted
		false,   // internal
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		return err
	}
	log.Printf("Declared queue %s", topic)
	return nil
}

func TestDeclareTopicsAndExchange(t *testing.T) {
	conn, err := amqp.Dial(rabbitConfig.Url)
	if err != nil {
		t.Error(err)
	}

	ch, err := conn.Channel()
	if err != nil {
		t.Error(err)
	}

	// err = ch.ExchangeDeclare(rabbitConfig.ExchangeName, "topic", true, false, false, false, nil)
	// if err != nil {
	// 	t.Error(err)
	// }

	if err = createTopic(ch, rabbitConfig.ItemAddedTopic); err != nil {
		t.Error(err)
	}
	if err = createTopic(ch, rabbitConfig.ItemChangedTopic); err != nil {
		t.Error(err)
	}
	if err = createTopic(ch, rabbitConfig.ItemDeletedTopic); err != nil {
		t.Error(err)
	}
}

func TestSendChanges(t *testing.T) {
	masterTransport := RabbitTransportMaster{
		RabbitConfig: rabbitConfig,
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
		Fields: []index.KeyFieldValue{
			{Value: "Test", Id: 1},
		},
	})

	if err != nil {
		t.Error(err)
	}
}

func TestSync(t *testing.T) {
	masterTransport := RabbitTransportMaster{
		RabbitConfig: rabbitConfig,
	}
	index1 := index.NewIndex(search.NewFreeTextIndex(&search.Tokenizer{MaxTokens: 128}))

	clientTransport1 := RabbitTransportClient{
		RabbitConfig: rabbitConfig,
	}

	err := masterTransport.Connect()
	if err != nil {
		t.Error(err)
	}
	err = clientTransport1.Connect(index1)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(time.Second)

	defer masterTransport.Close()
	defer clientTransport1.Close()

	item := &index.DataItem{
		BaseItem: index.BaseItem{
			Id:    1,
			Title: "Test",
		},
		Fields: []index.KeyFieldValue{
			{Value: "Test", Id: 1},
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

	item.Fields[0].Value = "Test2"

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

	if firstItem1.Fields[0].Value != "Test2" {
		t.Error("Item not updated on client 1")
	}

	masterTransport.SendItemDeleted(item.Id)
	time.Sleep(time.Second)

	if _, ok := index1.Items[1]; ok {
		t.Error("Item not deleted from client 1")
	}

}
