package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/matst80/slask-finder/pkg/embeddings"
	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/messaging"
	"github.com/matst80/slask-finder/pkg/storage"
	"github.com/matst80/slask-finder/pkg/types"
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
	// Application entry point
	diskStorage := storage.NewDiskStorage(country, "data")
	// Entry point for the master command
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

	embeddingsEngine := embeddings.NewOllamaEmbeddingsEngine()
	embeddingsIndex := embeddings.NewItemEmbeddingsHandler(embeddings.DefaultEmbeddingsHandlerOptions(embeddingsEngine), func(data map[uint]types.Embeddings) error {
		log.Printf("Queue done, saving %d embeddings to disk", len(data))
		diskStorage.SaveEmbeddings(data)
		return nil
	})

	itemCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	messaging.ListenToTopic(itemCh, country, "item_added", func(d amqp.Delivery) error {
		items := []index.DataItem{}
		if err := json.Unmarshal(d.Body, &items); err == nil {
			log.Printf("Got upserts %d", len(items))
			for _, item := range items {
				embeddingsIndex.HandleItem(&item)
			}
		}
		return nil
	})

	err = diskStorage.LoadEmbeddings(embeddingsIndex.Embeddings)
	if err != nil {
		log.Printf("Could not load embeddings from file: %v", err)
	}

	a := &app{
		country:  country,
		storage:  diskStorage,
		index:    embeddingsIndex,
		proxyUrl: os.Getenv("PROXY_URL"),
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/ai/cosine-similar/{id}", a.CosineSimilar)
	mux.HandleFunc("/ai/natural", a.SearchEmbeddings)

	http.ListenAndServe(":8080", mux)
}
