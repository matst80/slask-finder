package sync

import (
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"tornberg.me/facet-search/pkg/index"
)

type RabbitTopics struct {
	ItemChangedTopic string
	ItemAddedTopic   string
	ItemDeletedTopic string
}

type RabbitTransportMaster struct {
	RabbitTopics
	Url        string
	connection *amqp.Connection
	channel    *amqp.Channel
}

type RabbitTransportClient struct {
	RabbitTopics
	Url        string
	handler    index.UpdateHandler
	connection *amqp.Connection
	channel    *amqp.Channel
	quit       chan bool
}

func createQueue(ch *amqp.Channel, topic string) error {
	q, err := ch.QueueDeclare(
		topic,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}
	log.Printf("Declared queue %s: %v", topic, q)
	return nil
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
	if err = createQueue(ch, t.ItemAddedTopic); err != nil {
		return err
	}
	if err = createQueue(ch, t.ItemChangedTopic); err != nil {
		return err
	}
	if err = createQueue(ch, t.ItemDeletedTopic); err != nil {
		return err
	}

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
		"",
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
	toAdd, err := ch.Consume(
		t.ItemAddedTopic,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}
	go func(msgs <-chan amqp.Delivery) {

		for d := range msgs {
			var item index.DataItem
			err := json.Unmarshal(d.Body, &item)
			if err != nil {
				return
			}
			t.handler.UpsertItem(&item)
		}

	}(toAdd)
	toChange, err := ch.Consume(
		t.ItemChangedTopic,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	go func(msgs <-chan amqp.Delivery) {

		for d := range msgs {
			var item index.DataItem
			err := json.Unmarshal(d.Body, &item)
			if err != nil {
				return
			}
			t.handler.UpsertItem(&item)
		}

	}(toChange)

	toDelete, err := ch.Consume(
		t.ItemDeletedTopic,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	go func(msgs <-chan amqp.Delivery) {

		for d := range msgs {
			var itemId uint
			err := json.Unmarshal(d.Body, &itemId)
			if err != nil {
				return
			}
			t.handler.DeleteItem(itemId)
		}

	}(toDelete)
	// go func(c *amqp.Channel) {
	// 	for {
	// 		if c.IsClosed() {
	// 			return
	// 		}
	// 		msgs, err := c.Consume(
	// 			t.ItemDeletedTopic,
	// 			"",
	// 			true,
	// 			false,
	// 			false,
	// 			false,
	// 			nil,
	// 		)
	// 		if err != nil {
	// 			fmt.Println(err)
	// 		}
	// 		for d := range msgs {
	// 			var id uint
	// 			err := json.Unmarshal(d.Body, &id)
	// 			if err != nil {
	// 				return
	// 			}
	// 			t.handler.DeleteItem(id)
	// 		}
	// 		time.Sleep(time.Millisecond * 150)
	// 	}
	// }(t.channel)

	// go func(q chan bool, ch *amqp.Channel, c *amqp.Connection) {
	// 	for {
	// 		if ch.IsClosed() {
	// 			q <- true
	// 			return
	// 		}
	// 		select {
	// 		case <-q:
	// 			ch.Close()
	// 			c.Close()
	// 			return
	// 		default:
	// 			{

	// 				time.Sleep(time.Millisecond * 150)
	// 			}
	// 		}
	// 	}
	// }(t.quit, ch, conn)

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
