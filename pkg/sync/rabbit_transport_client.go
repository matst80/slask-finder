package sync

import (
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
	"tornberg.me/facet-search/pkg/index"
)

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
	toAdd, err := t.declareBindAndConsume(t.ItemAddedTopic)
	if err != nil {
		return err
	}
	go func(msgs <-chan amqp.Delivery) {
		for d := range msgs {
			var item index.DataItem
			if err := json.Unmarshal(d.Body, &item); err == nil {
				t.handler.UpsertItem(&item)
			}
		}
	}(toAdd)

	toUpdate, err := t.declareBindAndConsume(t.ItemChangedTopic)
	if err != nil {
		return err
	}
	go func(msgs <-chan amqp.Delivery) {
		for d := range msgs {
			var item index.DataItem
			if err := json.Unmarshal(d.Body, &item); err == nil {
				t.handler.UpsertItem(&item)
			}
		}
	}(toUpdate)

	toDelete, err := t.declareBindAndConsume(t.ItemDeletedTopic)
	if err != nil {
		return err
	}
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
