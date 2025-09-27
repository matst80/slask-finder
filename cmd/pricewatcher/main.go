package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

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
		}
		err := diskStorage.SaveJson(app.Items, "item_prices.json")
		if err != nil {
			log.Printf("Could not save item prices to file: %v", err)
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
	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Println("pricewatcher server starting on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server listen error: %v", err)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("Shutting down pricewatcher server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Graceful shutdown failed: %v", err)
	}
	log.Println("Pricewatcher server stopped")
}
