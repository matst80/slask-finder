package sync

import (
	"encoding/json"
	"log"

	"github.com/matst80/slask-finder/pkg/index"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitTransportClient struct {
	RabbitConfig

	ClientName string
	handler    index.UpdateHandler
	connection *amqp.Connection
	channel    *amqp.Channel
	quit       chan bool
}

func (t *RabbitTransportClient) declareBindAndConsume(topic string) (<-chan amqp.Delivery, error) {
	q, err := t.channel.QueueDeclare(
		"",    // name
		false, // durable
		false, // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return nil, err
	}
	err = t.channel.QueueBind(q.Name, topic, topic, false, nil)
	if err != nil {
		return nil, err
	}
	return t.channel.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
}

func (t *RabbitTransportClient) Connect(handler index.UpdateHandler) error {
	conn, err := amqp.Dial(t.Url)

	t.quit = make(chan bool)
	if err != nil {
		return err
	}
	t.connection = conn
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	t.handler = handler
	t.channel = ch
	toAdd, err := t.declareBindAndConsume(t.ItemsUpsertedTopic)
	if err != nil {
		return err
	}
	log.Printf("Connected to rabbit upsert topic: %s", t.ItemsUpsertedTopic)
	go func(msgs <-chan amqp.Delivery) {
		for d := range msgs {

			var items []index.DataItem
			if err := json.Unmarshal(d.Body, &items); err == nil {
				log.Printf("Got upserts %d", len(items))
				t.handler.UpsertItems(index.ToItemArray(items))
			} else {
				log.Printf("Failed to unmarshal upset message %v", err)
			}
		}
	}(toAdd)

	toDelete, err := t.declareBindAndConsume(t.ItemDeletedTopic)
	if err != nil {
		return err
	}
	log.Printf("Connected to rabbit delete topic: %s", t.ItemDeletedTopic)
	go func(msgs <-chan amqp.Delivery) {
		for d := range msgs {
			var item uint
			if err := json.Unmarshal(d.Body, &item); err == nil {
				t.handler.DeleteItem(item)
			}
		}
	}(toDelete)
	return nil
}

func (t *RabbitTransportClient) Close() {
	if (t.channel != nil) && (!t.channel.IsClosed()) {
		t.channel.Close()
	}
	if (t.connection != nil) && (!t.connection.IsClosed()) {
		t.connection.Close()
	}
	//t.quit <- true

}
