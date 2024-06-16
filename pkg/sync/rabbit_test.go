package sync

import (
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestRabbitConnection(t *testing.T) {
	conn, err := amqp.Dial("amqp://admin:12bananer@localhost:5672/")
	if err != nil {
		t.Errorf("Error connecting to RabbitMQ: %s", err)
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		t.Errorf("Error opening channel: %s", err)
	}
	defer ch.Close()
	q, err := ch.QueueDeclare(
		"test_queue",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		t.Errorf("Error declaring queue: %s", err)
	}
	err = ch.Publish(
		"",
		q.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte("Hello"),
		},
	)
	if err != nil {
		t.Errorf("Error publishing message: %s", err)
	}
	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		t.Errorf("Error consuming message: %s", err)
	}
	for d := range msgs {
		if string(d.Body) != "Hello" {
			t.Errorf("Expected: Hello, got: %s", string(d.Body))
		}
		break
	}
}
