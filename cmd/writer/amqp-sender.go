package main

import (
	"log"

	"github.com/matst80/slask-finder/pkg/messaging"
	"github.com/matst80/slask-finder/pkg/types"
	amqp "github.com/rabbitmq/amqp091-go"
)

type AmqpSender struct {
	Country    string
	connection *amqp.Connection
}

func NewAmqpSender(country string, conn *amqp.Connection) *AmqpSender {
	r := &AmqpSender{
		Country:    country,
		connection: conn,
	}
	r.defineTopics()
	return r
}

// func (app *AmqpSender) SendItems(items []index.DataItem) error {
// 	return sync.SendChange(app.connection, app.Country, "item_added", items)
// }

func (app *AmqpSender) SendFacetChanges(items ...types.FieldChange) error {
	return messaging.SendChange(app.connection, app.Country, "facet_change", items)
}

func (app *AmqpSender) SendSettingsChange(item types.SettingsChange) error {
	return messaging.SendChange(app.connection, app.Country, "settings_change", item)
}

func (app *AmqpSender) defineTopics() {
	ch, err := app.connection.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()
	if err := messaging.DefineTopic(ch, app.Country, "item_added"); err != nil {
		log.Fatalf("Failed to declare topic item_added: %v", err)
	}
	if err := messaging.DefineTopic(ch, app.Country, "facet_change"); err != nil {
		log.Fatalf("Failed to declare topic facet_change: %v", err)
	}
	if err := messaging.DefineTopic(ch, app.Country, "settings_change"); err != nil {
		log.Fatalf("Failed to declare topic settings_change: %v", err)
	}
}
