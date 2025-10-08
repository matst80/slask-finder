package messaging

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func DeclareBindAndConsume(ch *amqp.Channel, prefix string, topic ChangeTopic) (<-chan amqp.Delivery, error) {
	name := getName(prefix, topic)
	q, err := ch.QueueDeclare(
		"",    // name
		false, // durable
		false, // delete when unused
		true,  // exclusive
		false, // no-wait

		nil, // arguments
	)
	if err != nil {
		return nil, err
	}
	err = ch.QueueBind(q.Name, name, name, false, nil)
	if err != nil {
		return nil, err
	}
	return ch.Consume(
		q.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
}

func ListenToTopic(ch *amqp.Channel, prefix string, topic ChangeTopic, filter func(amqp.Delivery) error) error {
	fc, err := DeclareBindAndConsume(ch, prefix, topic)
	if err != nil {
		return err
	}

	go func(msgs <-chan amqp.Delivery) {
		defer ch.Close()
		for d := range msgs {
			if err := filter(d); err != nil {
				log.Printf("Error processing message: %v", err)
				return // Exit the goroutine on error
			} else {
				d.Ack(false)
			}
		}

	}(fc)
	return nil
}
