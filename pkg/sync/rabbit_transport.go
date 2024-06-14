package sync

import (
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
}

type RabbitTransportClient struct {
	RabbitTopics
	Url     string
	channel *amqp.Channel
}

func (t *RabbitTransportMaster) Connect() error {
	conn, err := amqp.Dial(t.Url)
	if err != nil {
		return err
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	t.channel = ch
	return nil
}

func (t *RabbitTransportMaster) SendItemAdded(item *index.DataItem) error {
	return nil
}

func (t *RabbitTransportMaster) SendItemChanged(item *index.DataItem) error {
	return nil
}

func (t *RabbitTransportMaster) SendItemDeleted(id uint) error {
	return nil
}

func (t *RabbitTransportClient) OnItemAdded(item *index.DataItem) {
}

func (t *RabbitTransportClient) OnItemChanged(item *index.DataItem) {
}

func (t *RabbitTransportClient) OnItemDeleted(id uint) {

}

func (t *RabbitTransportClient) Connect() error {
	conn, err := amqp.Dial(t.Url)
	if err != nil {
		return err
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	t.channel = ch
	return nil
}
