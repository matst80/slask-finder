package sync

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
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
	Url     string
	channel *amqp.Channel
	enc     gob.Encoder
}

type RabbitTransportClient struct {
	RabbitTopics
	Index   *index.Index
	Url     string
	channel *amqp.Channel
	decoder gob.Decoder
}

func (t *RabbitTransportMaster) Connect() error {

	conn, err := amqp.Dial(t.Url)
	if err != nil {
		return err
	}
	//defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	t.channel = ch
	q, err := ch.QueueDeclare(
		t.ItemAddedTopic,
		true,
		false,
		false,
		false,
		nil,
	)
	q, err = ch.QueueDeclare(
		t.ItemChangedTopic,
		true,
		false,
		false,
		false,
		nil,
	)
	q, err = ch.QueueDeclare(
		t.ItemDeletedTopic,
		true,
		false,
		false,
		false,
		nil,
	)
	log.Printf("Declared queues: %v", q)
	return nil
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

func (t *RabbitTransportClient) OnItemAdded(item *index.DataItem) {
	t.Index.UpsertItem(item)
}

func (t *RabbitTransportClient) OnItemChanged(item *index.DataItem) {
	t.Index.UpsertItem(item)
}

func (t *RabbitTransportClient) OnItemDeleted(id uint) {
	t.Index.DeleteItem(id)
}

func (t *RabbitTransportClient) Connect() error {
	conn, err := amqp.Dial(t.Url)
	if err != nil {
		return err
	}
	//defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	t.channel = ch
	go func(c *amqp.Channel) {
		for {
			msgs, err := c.Consume(
				t.ItemAddedTopic,
				"",
				true,
				false,
				false,
				false,
				nil,
			)
			if err != nil {
				fmt.Println(err)
			} else {
				for d := range msgs {
					var item index.DataItem
					err := json.Unmarshal(d.Body, &item)
					if err != nil {
						return
					}
					t.OnItemAdded(&item)
				}
			}
		}
	}(t.channel)
	go func(c *amqp.Channel) {
		for {
			msgs, err := c.Consume(
				t.ItemChangedTopic,
				"",
				true,
				false,
				false,
				false,
				nil,
			)
			if err != nil {
				fmt.Println(err)
			}
			for d := range msgs {
				var item index.DataItem
				err := json.Unmarshal(d.Body, &item)
				if err != nil {
					return
				}
				t.OnItemChanged(&item)
			}
		}
	}(t.channel)
	go func(c *amqp.Channel) {
		for {
			msgs, err := c.Consume(
				t.ItemDeletedTopic,
				"",
				true,
				false,
				false,
				false,
				nil,
			)
			if err != nil {
				fmt.Println(err)
			}
			for d := range msgs {
				var id uint
				err := json.Unmarshal(d.Body, &id)
				if err != nil {
					return
				}
				t.OnItemDeleted(id)
			}
		}
	}(t.channel)
	return nil
}
