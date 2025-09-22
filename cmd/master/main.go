package master

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/sync"
	amqp "github.com/rabbitmq/amqp091-go"
)

type MasterApp struct {
	itemIndex       *index.ItemIndex
	embeddingsIndex *index.ItemEmbeddingsHandler
	connection      *amqp.Connection
}

var country = "se"
var itemsFile = "data/index-v2.jz"

func init() {
	c, ok := os.LookupEnv("COUNTRY")
	if ok {
		country = c
	}
}

func (app *MasterApp) defineTopics() {
	ch, err := app.connection.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()
	if err := sync.DefineTopic(ch, country, "item_added"); err != nil {
		log.Fatalf("Failed to declare topic item_added: %v", err)
	}
}

func (app *MasterApp) handleItems(w http.ResponseWriter, r *http.Request) {
	items := make([]index.DataItem, 0)
	err := json.NewDecoder(r.Body).Decode(&items)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	go func() {
		app.itemIndex.StartUnsafe()
		for _, item := range items {
			app.itemIndex.HandleItemUnsafe(&item)
		}
		app.itemIndex.EndUnsafe()
	}()

	go sync.SendChange(app.connection, country, "item_added", items)

	// totalItems.Set(float64(len(ws.Index.Items)))
	// toUpdate = nil
	w.WriteHeader(http.StatusOK)
}

func (app *MasterApp) saveItems(w http.ResponseWriter, r *http.Request) {
	app.itemIndex.Lock()
	defer app.itemIndex.Unlock()
	err := SaveItems(app.itemIndex.Items, itemsFile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func main() {
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

	index := index.NewItemIndex()
	embeddingsIndex := index.NewItemEmbeddingsHandler()
	
	err = LoadItems(&index.Items, itemsFile, embeddingsIndex)
	if err != nil {
		log.Printf("Could not load items from file: %v", err)
	}

	app := &MasterApp{
		connection: conn,
		itemIndex:  index,
	}
	app.defineTopics()

	srv := http.NewServeMux()
	srv.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	srv.HandleFunc("/admin/add", app.handleItems)
	srv.HandleFunc("/admin/save", app.saveItems)
	srv.HandleFunc("GET /admin/item/{id}", ws.AuthMiddleware(JsonHandler(ws.Tracking, ws.GetItem)))
}
