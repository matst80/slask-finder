package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/storage"
	"github.com/matst80/slask-finder/pkg/sync"
	amqp "github.com/rabbitmq/amqp091-go"
)

var country = "se"

func init() {
	c, ok := os.LookupEnv("COUNTRY")
	if ok {
		country = c
	}
}

func main() {
	diskStorage := storage.NewDiskStorage(country, "data")
	watcher := NewPriceWatcher(diskStorage)
	app := &ItemWatcher{
		Items:   make(map[uint]int),
		watcher: *watcher,
	}

	// Load existing items from disk if available
	err := diskStorage.LoadJson(app.Items, "item_prices.json")
	if err != nil {
		log.Printf("Could not load item prices from file: %v", err)
	}

	amqpUrl, ok := os.LookupEnv("RABBIT_HOST")
	if !ok {
		log.Fatal("RABBIT_HOST environment variable is not set")
	}
	conn, err := amqp.DialConfig(amqpUrl, amqp.Config{
		Properties: amqp.NewConnectionProperties(),
	})
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	itemCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	sync.ListenToTopic(itemCh, country, "item_added", func(d amqp.Delivery) error {
		items := []index.DataItem{}
		app.mu.Lock()
		defer app.mu.Unlock()
		if err := json.Unmarshal(d.Body, &items); err == nil {
			log.Printf("Got upserts %d", len(items))
			app.HandleItems(items)
		}
		diskStorage.SaveJson(app.Items, "item_prices.json")
		return nil
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/push/watch/", watcher.WatchPriceChange)
	// mux.HandleFunc("/push/unwatch/", watcher.UnwatchPriceChange)
	// mux.HandleFunc("/push/list/", watcher.ListWatches)
}
