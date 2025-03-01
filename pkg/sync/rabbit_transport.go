package sync

import (
	"encoding/json"
	"log"

	"github.com/matst80/slask-finder/pkg/types"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitTransportMaster struct {
	RabbitConfig
	connection *amqp.Connection
	channel    *amqp.Channel
}

func (t *RabbitTransportMaster) Connect() error {

	conn, err := amqp.Dial(t.Url)
	//conn.Config.Vhost = t.VHost
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
	log.Println("Closing master channel")
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

func (t *RabbitTransportMaster) ItemsUpserted(items []types.Item) error {
	return t.send(t.ItemsUpsertedTopic, items)
}

func (t *RabbitTransportMaster) SendItemDeleted(id uint) error {
	return t.send(t.ItemDeletedTopic, id)
}

func (t *RabbitTransportMaster) SendPriceLowered(items []types.Item) error {
	return t.send(t.PriceLoweredTopic, items)
}
