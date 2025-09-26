package main

import (
	"encoding/json"
	"iter"
	"log"
	"net/http"
	"os"

	"github.com/matst80/slask-finder/pkg/common"
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

func asItems(items []index.DataItem) iter.Seq[types.Item] {
	return func(yield func(types.Item) bool) {
		for _, item := range items {
			if !yield(&item) {
				return
			}
		}
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

					handler.HandleItems(asItems(items))

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

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	var tracking types.Tracking = nil
	mux.HandleFunc("/update-sort", common.JsonHandler(tracking, app.UpdateSort))
	mux.HandleFunc("/stream", common.JsonHandler(tracking, app.SearchStreamed))
	mux.HandleFunc("/facets", common.JsonHandler(tracking, app.GetFacets))

	/*
		//mux.HandleFunc("/ai-search", common.JsonHandler(tracking, ws.SearchEmbeddings))
		mux.HandleFunc("/related/{id}", common.JsonHandler(tracking, app.Related))
		mux.HandleFunc("/compatible/{id}", common.JsonHandler(tracking, app.Compatible))
		mux.HandleFunc("/popular", common.JsonHandler(tracking, app.Popular))
		//mux.HandleFunc("/natural", common.JsonHandler(tracking, ws.SearchEmbeddings))
		mux.HandleFunc("/similar", common.JsonHandler(tracking, app.Similar))
		//mux.HandleFunc("/cosine-similar/{id}", common.JsonHandler(tracking, ws.CosineSimilar))
		//mux.HandleFunc("/trigger-words", common.JsonHandler(tracking, ws.TriggerWords))
		mux.HandleFunc("/facet-list", common.JsonHandler(tracking, app.Facets))
		mux.HandleFunc("/suggest", common.JsonHandler(tracking, app.Suggest))
		mux.HandleFunc("/find-related", common.JsonHandler(tracking, app.FindRelated))
		//mux.HandleFunc("/categories", common.JsonHandler(tracking, ws.Categories))
		//mux.HandleFunc("/search", ws.QueryIndex)
		//mux.HandleFunc("GET /settings", ws.GetSettings)

		mux.HandleFunc("/reload-settings", common.JsonHandler(tracking, app.ReloadSettings))
		mux.HandleFunc("GET /relation-groups", app.GetRelationGroups)

		mux.HandleFunc("/ids", common.JsonHandler(tracking, app.GetIds))
		mux.HandleFunc("GET /get/{id}", common.JsonHandler(tracking, app.GetItem))
		mux.HandleFunc("GET /by-sku/{sku}", common.JsonHandler(tracking, app.GetItemBySku))
		mux.HandleFunc("POST /get", common.JsonHandler(tracking, app.GetItems))
		mux.HandleFunc("/values/{id}", common.JsonHandler(tracking, app.GetValues))
		mux.HandleFunc("/predict-sequence", common.JsonHandler(tracking, app.PredictSequence))
		mux.HandleFunc("/predict-tree", common.JsonHandler(tracking, app.PredictTree))

	*/
	http.ListenAndServe(":8080", mux)
}
