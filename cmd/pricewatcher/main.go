package main

import (
	"encoding/json"
	"log"
	"maps"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/matst80/slask-finder/pkg/common"
	"github.com/matst80/slask-finder/pkg/types"

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
		Items:   make(map[types.ItemId]int),
		mu:      sync.RWMutex{},
		watcher: *watcher,
	}

	var tmp map[types.ItemId]int
	err := diskStorage.LoadJson(&tmp, "item_prices.json")
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("item_prices.json not found, starting empty")
		} else {
			log.Printf("Could not load item prices from file: %v", err)
		}
	} else {
		maps.Copy(app.Items, tmp)
		log.Printf("Loaded %d item prices from disk", len(app.Items))
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

		if err := json.Unmarshal(d.Body, &items); err == nil {
			log.Printf("Got upserts %d", len(items))
			app.HandleItems(items)
			log.Printf("Updated, saving %d item prices to disk", len(app.Items))
			app.mu.RLock()
			err := diskStorage.SaveJson(&app.Items, "item_prices.json")
			app.mu.RUnlock()
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
	mux.HandleFunc("POST /push/watch/{id}", watcher.WatchPriceChange)
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
