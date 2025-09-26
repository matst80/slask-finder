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
	"github.com/matst80/slask-finder/pkg/tracking"
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
	tracker        types.Tracking
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
	err := diskStorage.LoadSettings()
	if err != nil {
		log.Printf("Could not load settings from file: %v", err)
	}
	itemIndex := index.NewIndexWithStock()
	sortingHandler := sorting.NewSortingItemHandler()
	searchHandler := search.NewFreeTextItemHandler(search.DefaultFreeTextHandlerOptions())
	facets := []facet.StorageFacet{}
	err = diskStorage.LoadFacets(&facets)
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
	var tracker types.Tracking = nil
	if amqpUrl != "" {
		tracker, err = tracking.NewRabbitTracking(amqpUrl, country)
		if err != nil {
			log.Printf("Failed to connect to rabbitmq for tracking: %v", err)
		} else {
			app.tracker = tracker
			defer tracker.Close()
		}
	}

	mux.HandleFunc("/api/stream", common.JsonHandler(tracker, app.SearchStreamed))
	mux.HandleFunc("/api/facets", common.JsonHandler(tracker, app.GetFacets))
	mux.HandleFunc("GET /api/facet-list", common.JsonHandler(tracker, app.Facets))
	mux.HandleFunc("GET /api/get/{id}", common.JsonHandler(tracker, app.GetItem))
	mux.HandleFunc("GET /api/by-sku/{sku}", common.JsonHandler(tracker, app.GetItemBySku))
	mux.HandleFunc("GET /api/related/{id}", common.JsonHandler(tracker, app.Related))
	mux.HandleFunc("GET /api/compatible/{id}", common.JsonHandler(tracker, app.Compatible))
	mux.HandleFunc("GET /api/values/{id}", common.JsonHandler(tracker, app.GetValues))
	mux.HandleFunc("GET /api/suggest", common.JsonHandler(tracker, app.Suggest))
	mux.HandleFunc("GET /api/popular", common.JsonHandler(tracker, app.Popular))
	mux.HandleFunc("GET /api/relation-groups", common.JsonHandler(tracker, app.GetRelationGroups))

	//mux.HandleFunc("/api/similar", common.JsonHandler(tracker, app.Similar))
	/*



		mux.HandleFunc("/natural", common.JsonHandler(tracking, ws.SearchEmbeddings))

		mux.HandleFunc("/cosine-similar/{id}", common.JsonHandler(tracking, ws.CosineSimilar))
		mux.HandleFunc("/trigger-words", common.JsonHandler(tracking, ws.TriggerWords))


		mux.HandleFunc("/find-related", common.JsonHandler(tracking, app.FindRelated))
		//mux.HandleFunc("/categories", common.JsonHandler(tracking, ws.Categories))
		//mux.HandleFunc("/search", ws.QueryIndex)
		//mux.HandleFunc("GET /settings", ws.GetSettings)

		mux.HandleFunc("/reload-settings", common.JsonHandler(tracking, app.ReloadSettings))


		mux.HandleFunc("/ids", common.JsonHandler(tracking, app.GetIds))

		mux.HandleFunc("POST /get", common.JsonHandler(tracking, app.GetItems))

		mux.HandleFunc("/predict-sequence", common.JsonHandler(tracking, app.PredictSequence))
		mux.HandleFunc("/predict-tree", common.JsonHandler(tracking, app.PredictTree))

	*/
	http.ListenAndServe(":8080", mux)
}
