package sync

import (
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
	"tornberg.me/facet-search/pkg/index"
)

type RabbitConfig struct {
	ItemChangedTopic string
	ItemAddedTopic   string
	ItemDeletedTopic string
	Url              string
}

type RabbitTransportMaster struct {
	RabbitConfig
	connection *amqp.Connection
	channel    *amqp.Channel
}

type RabbitTransportClient struct {
	RabbitConfig

	ClientName string
	handler    index.UpdateHandler
	connection *amqp.Connection
	channel    *amqp.Channel
	quit       chan bool
}

func (t *RabbitTransportMaster) Connect() error {

	conn, err := amqp.Dial(t.Url)
	if err != nil {
		return err
	}
	t.connection = conn
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	t.channel = ch

	return nil
}

func (t *RabbitTransportMaster) Close() error {
	defer t.connection.Close()
	return t.channel.Close()
}

func (t *RabbitTransportMaster) send(topic string, data any) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return t.channel.Publish(
		topic,
		topic,
		true,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        bytes,
		},
	)
}

func (t *RabbitTransportMaster) SendItemAdded(item *index.DataItem) error {
	return t.send(t.ItemAddedTopic, item)
}

func (t *RabbitTransportMaster) SendItemChanged(item *index.DataItem) error {
	return t.send(t.ItemChangedTopic, item)
}

func (t *RabbitTransportMaster) SendItemDeleted(id uint) error {
	return t.send(t.ItemDeletedTopic, id)
}
