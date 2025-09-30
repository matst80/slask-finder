package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/messaging"
	"github.com/matst80/slask-finder/pkg/types"
	amqp "github.com/rabbitmq/amqp091-go"
)

func (a *app) ConnectAmqp(amqpUrl string) {
	conn, err := amqp.DialConfig(amqpUrl, amqp.Config{
		Properties: amqp.NewConnectionProperties(),
	})
	a.conn = conn
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	// items listener
	err = messaging.ListenToTopic(ch, country, "item_added", func(d amqp.Delivery) error {
		var items []*index.DataItem
		if err := json.Unmarshal(d.Body, &items); err == nil {
			log.Printf("Got upserts %d", len(items))
			wg := &sync.WaitGroup{}
			for _, item := range items {
				a.itemIndex.HandleItem(item, wg)
				a.facetHandler.HandleItem(item, wg)
				a.sortingHandler.HandleItem(item, wg)
				a.searchIndex.HandleItem(item, wg)
			}
			wg.Wait()

		} else {
			log.Printf("Failed to unmarshal upset message %v", err)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to listen to item_added topic: %v", err)
	}

	log.Printf("Listening for item upserts")

	ticker := time.NewTicker(time.Minute * 1)
	go func() {
		for range ticker.C {
			if a.gotSaveTrigger {
				log.Println("Saving items due to trigger")
				err := a.storage.SaveItems(a.itemIndex.GetAllItems())
				if err != nil {
					log.Printf("Failed to save items: %v", err)
				}
				a.gotSaveTrigger = false
			}
		}
	}()
}

func (a *app) ConnectFacetChange() {
	ch, err := a.conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	err = messaging.ListenToTopic(ch, country, "facet_change", func(d amqp.Delivery) error {
		var items []types.FieldChange
		if err := json.Unmarshal(d.Body, &items); err == nil {
			log.Printf("Got fieldchanges %d", len(items))
			a.facetHandler.HandleFieldChanges(items)
		} else {
			log.Printf("Failed to unmarshal facet change message %v", err)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to listen to facet_change topic: %v", err)
	}
}

func (a *app) ConnectSettingsChange() {
	ch, err := a.conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	// items listener
	err = messaging.ListenToTopic(ch, country, "settings_change", func(d amqp.Delivery) error {
		var item types.SettingsChange
		if err := json.Unmarshal(d.Body, &item); err == nil {
			log.Printf("Got settings %v", item)
			err := a.storage.LoadSettings()
			if err != nil {
				log.Printf("Could not update settings from file: %v", err)
			}
		} else {
			log.Printf("Failed to unmarshal upset message %v", err)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to listen to settings_change topic: %v", err)
	}
}
