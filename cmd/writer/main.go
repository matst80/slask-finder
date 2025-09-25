package main

import (
	"log"
	"net/http"
	"os"

	"github.com/matst80/slask-finder/pkg/embeddings"
	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/storage"
	amqp "github.com/rabbitmq/amqp091-go"
)

type MasterApp struct {
	storage         *storage.DiskStorage
	itemIndex       *index.ItemIndex
	embeddingsIndex *embeddings.ItemEmbeddingsHandler
	amqpSender      *AmqpSender
}

var country = "se"

func init() {
	c, ok := os.LookupEnv("COUNTRY")
	if ok {
		country = c
	}
}

func main() {
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

	idx := index.NewItemIndex()
	embeddingsEngine := embeddings.NewOllamaEmbeddingsEngine()
	embeddingsIndex := embeddings.NewItemEmbeddingsHandler(embeddings.DefaultEmbeddingsHandlerOptions(embeddingsEngine), func() error {

		log.Println("Embeddings queue processed")
		//storage.SaveEmbeddings(embeddingsIndex.Embeddings, "data/embeddings-v2.jz")
		return nil
	})

	err = diskStorage.LoadItems(idx, embeddingsIndex)
	if err != nil {
		log.Printf("Could not load items from file: %v", err)
	}

	app := &MasterApp{
		amqpSender:      NewAmqpSender(country, conn),
		itemIndex:       idx,
		embeddingsIndex: embeddingsIndex,
		storage:         diskStorage,
	}

	srv := http.NewServeMux()
	srv.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	srv.HandleFunc("POST /admin/add", app.handleItems)
	srv.HandleFunc("/admin/save", app.saveItems)
	srv.HandleFunc("GET /admin/item/{id}", app.getAdminItemById)
}
