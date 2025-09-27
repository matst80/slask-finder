package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/matst80/slask-finder/pkg/common"

	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/messaging"
	"github.com/matst80/slask-finder/pkg/storage"
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
		mu:      sync.RWMutex{},
		watcher: *watcher,
	}

	// Load existing items from disk if available
	// Must pass pointer so JSON decoder can populate map
	err := diskStorage.LoadJson(&app.Items, "item_prices.json")
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("item_prices.json not found, starting empty")
		} else {
			log.Printf("Could not load item prices from file: %v", err)
		}
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
	err = messaging.ListenToTopic(itemCh, country, "item_added", func(d amqp.Delivery) error {
		items := []index.DataItem{}
		app.mu.Lock()
		defer app.mu.Unlock()
		if err := json.Unmarshal(d.Body, &items); err == nil {
			log.Printf("Got upserts %d", len(items))
			app.HandleItems(items)

			err := diskStorage.SaveJson(app.Items, "item_prices.json")
			if err != nil {
				log.Printf("Could not save item prices to file: %v", err)
			}
		} else {
			log.Printf("Failed to unmarshal upsert message %v", err)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to start listening to topic: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/push/watch/", watcher.WatchPriceChange)
	// mux.HandleFunc("/push/unwatch/", watcher.UnwatchPriceChange)
	// mux.HandleFunc("/push/list/", watcher.ListWatches)
	cfg := common.LoadTimeoutConfig(common.TimeoutConfig{
		ReadHeader: 5 * time.Second,
		Read:       15 * time.Second,
		Write:      30 * time.Second,
		Idle:       60 * time.Second,
		Shutdown:   20 * time.Second,
		Hook:       5 * time.Second,
	})
	server := common.NewServerWithTimeouts(&http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: cfg.ReadHeader}, cfg)
	common.RunServerWithShutdown(server, "pricewatcher server", cfg.Shutdown, cfg.Hook)
}
