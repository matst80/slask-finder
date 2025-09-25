package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/matst80/slask-finder/pkg/facet"
	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/search"
	"github.com/matst80/slask-finder/pkg/sorting"
	"github.com/matst80/slask-finder/pkg/storage"
	"github.com/matst80/slask-finder/pkg/sync"
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

type app struct {
	country        string
	storage        *storage.DiskStorage
	itemIndex      *index.ItemIndexWithStock
	searchIndex    *search.FreeTextItemHandler
	sortingHandler *sorting.SortingItemHandler
	facetHandler   *facet.FacetItemHandler
}

func (a *app) Connect(amqpUrl string, handlers ...types.ItemHandler) {
	conn, err := amqp.DialConfig(amqpUrl, amqp.Config{
		Properties: amqp.NewConnectionProperties(),
	})
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	toAdd, err := sync.DeclareBindAndConsume(ch, country, "item_added")
	if err != nil {
		log.Fatalf("Failed to declare and bind to topic: %v", err)
	}
	log.Printf("Connected to rabbit upsert topic")
	go func(msgs <-chan amqp.Delivery) {
		for d := range msgs {

			var items []index.DataItem
			if err := json.Unmarshal(d.Body, &items); err == nil {
				log.Printf("Got upserts %d", len(items))

				for _, handler := range handlers {
					for _, item := range items {
						handler.HandleItem(&item)
					}
				}
			} else {
				log.Printf("Failed to unmarshal upset message %v", err)
			}
		}
	}(toAdd)

}

func main() {
	diskStorage := storage.NewDiskStorage(country, "data")
	itemIndex := index.NewIndexWithStock()
	sortingHandler := sorting.NewSortingItemHandler()
	searchHandler := search.NewFreeTextItemHandler(search.DefaultFreeTextHandlerOptions())

	facets, err := facet.LoadFacetsFromStorage(diskStorage)
	if err != nil {
		log.Printf("Could not load facets from storage: %v", err)
	}
	facetHandler := facet.NewFacetItemHandler(facets)

	app := &app{
		country:        country,
		storage:        diskStorage,
		itemIndex:      itemIndex,
		searchIndex:    searchHandler,
		sortingHandler: sortingHandler,
		facetHandler:   facetHandler,
	}

	diskStorage.LoadItems(itemIndex, sortingHandler, facetHandler, searchHandler)

	amqpUrl, ok := os.LookupEnv("RABBIT_HOST")
	if ok {
		app.Connect(amqpUrl, itemIndex, sortingHandler, facetHandler, searchHandler)
	}
	mux := http.NewServeMux()

	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	http.ListenAndServe(":8080", mux)
}
