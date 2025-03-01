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
	//channel    *amqp.Channel
}

func (t *RabbitTransportMaster) Connect() error {

	conn, err := amqp.DialConfig(t.Url, amqp.Config{
		Vhost:      t.VHost,
		Properties: amqp.NewConnectionProperties(),
	})

	if err != nil {
		return err
	}
	t.connection = conn
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()
	if err := ch.ExchangeDeclare(
		t.RabbitConfig.ItemsUpsertedTopic, // name
		"topic",                           // type
		true,                              // durable
		false,                             // auto-delete
		false,                             // internal
		false,                             // noWait
		nil,                               // arguments
	); err != nil {
		return err
	}
	if err := ch.ExchangeDeclare(
		t.RabbitConfig.ItemDeletedTopic, // name
		"topic",                         // type
		true,                            // durable
		false,                           // auto-delete
		false,                           // internal
		false,                           // noWait
		nil,                             // arguments
	); err != nil {
		return err
	}

	if _, err = ch.QueueDeclare(
		t.RabbitConfig.ItemsUpsertedTopic, // name of the queue
		true,                              // durable
		false,                             // delete when unused
		false,                             // exclusive
		false,                             // noWait
		nil,                               // arguments
	); err != nil {
		return err
	}
	if _, err = ch.QueueDeclare(
		t.RabbitConfig.ItemDeletedTopic, // name of the queue
		true,                            // durable
		false,                           // delete when unused
		false,                           // exclusive
		false,                           // noWait
		nil,                             // arguments
	); err != nil {
		return err
	}
	//	t.channel = ch

	return nil
}

func (t *RabbitTransportMaster) Close() error {
	log.Println("Closing master channel")
	return t.connection.Close()
	//return t.channel.Close()
}

func (t *RabbitTransportMaster) send(topic string, data any) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	ch, err := t.connection.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()
	return ch.Publish(
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
