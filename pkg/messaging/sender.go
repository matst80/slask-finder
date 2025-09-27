package messaging

import (
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func DefineTopic(ch *amqp.Channel, prefix string, topic ChangeTopic) error {
	name := getName(prefix, topic)
	if err := ch.ExchangeDeclare(
		name,    // name
		"topic", // type
		true,    // durable
		false,   // auto-delete
		false,   // internal
		false,   // noWait
		nil,     // arguments
	); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(
		name,  // name of the queue
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // noWait
		nil,   // arguments
	); err != nil {
		return err
	}
	return nil
}

func getName(prefix string, topic ChangeTopic) string {
	return fmt.Sprintf("%s_%s", prefix, topic)
}

func SendChange[V any](c *amqp.Connection, prefix string, topic ChangeTopic, data V) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	ch, err := c.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()
	name := getName(prefix, topic)
	return ch.Publish(
		name,
		name,
		true,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        bytes,
		},
	)
}
