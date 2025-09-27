package main

import (
	"context"
	"iter"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/matst80/slask-finder/pkg/common"
	"github.com/matst80/slask-finder/pkg/facet"
	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/search"
	"github.com/matst80/slask-finder/pkg/sorting"
	"github.com/matst80/slask-finder/pkg/storage"
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
		for i := range items {
			if !yield(&items[i]) { // use index to avoid pointer to loop variable copy
				return
			}
		}
	}
}

type app struct {
	gotSaveTrigger bool
	country        string
	tracker        types.Tracking
	conn           *amqp.Connection
	storage        *storage.DiskStorage
	itemIndex      *index.ItemIndexWithStock
	searchIndex    *search.FreeTextItemHandler
	sortingHandler *sorting.SortingItemHandler
	facetHandler   *facet.FacetItemHandler
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

	err = diskStorage.LoadItems(itemIndex, sortingHandler, facetHandler, searchHandler)
	if err != nil {
		log.Printf("Could not load items from file: %v", err)
	}

	amqpUrl, ok := os.LookupEnv("RABBIT_HOST")
	if ok {
		app.ConnectAmqp(amqpUrl)
		app.ConnectFacetChange()
		app.ConnectSettingsChange()
	}

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
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("ok")); err != nil {
			log.Printf("Failed to write health response: %v", err)
		}
	})

	mux.HandleFunc("/api/stream", common.JsonHandler(tracker, app.SearchStreamed))
	mux.HandleFunc("/api/facets", common.JsonHandler(tracker, app.GetFacets))
	mux.HandleFunc("GET /api/facet-list", common.JsonHandler(tracker, app.Facets))
	mux.HandleFunc("GET /api/get/{id}", common.JsonHandler(tracker, app.GetItem))
	mux.HandleFunc("GET /api/by-sku/{sku}", common.JsonHandler(tracker, app.GetItemBySku))
	mux.HandleFunc("GET /api/related/{id}", common.JsonHandler(tracker, app.Related))
	mux.HandleFunc("/api/compatible/{id}", common.JsonHandler(tracker, app.Compatible))
	mux.HandleFunc("GET /api/values/{id}", common.JsonHandler(tracker, app.GetValues))
	mux.HandleFunc("GET /api/suggest", common.JsonHandler(tracker, app.Suggest))
	mux.HandleFunc("GET /api/popular", common.JsonHandler(tracker, app.Popular))
	mux.HandleFunc("GET /api/save-trigger", common.JsonHandler(tracker, app.SaveTrigger))
	mux.HandleFunc("GET /api/relation-groups", common.JsonHandler(tracker, app.GetRelationGroups))
	mux.HandleFunc("GET /api/facet-groups", common.JsonHandler(tracker, app.GetFacetGroups))
	mux.HandleFunc("POST /api/stream-items", app.StreamItemsFromIds)

	//mux.HandleFunc("/api/similar", common.JsonHandler(tracker, app.Similar))
	/*

		mux.HandleFunc("/trigger-words", common.JsonHandler(tracking, ws.TriggerWords))
		mux.HandleFunc("/find-related", common.JsonHandler(tracking, app.FindRelated))


		mux.HandleFunc("/reload-settings", common.JsonHandler(tracking, app.ReloadSettings))


		mux.HandleFunc("/ids", common.JsonHandler(tracking, app.GetIds))

		mux.HandleFunc("POST /get", common.JsonHandler(tracking, app.GetItems))

		mux.HandleFunc("/predict-sequence", common.JsonHandler(tracking, app.PredictSequence))
		mux.HandleFunc("/predict-tree", common.JsonHandler(tracking, app.PredictTree))

	*/
	// Load timeout configuration from env with defaults
	cfg := common.LoadTimeoutConfig(common.TimeoutConfig{
		ReadHeader: 5 * time.Second,
		Read:       15 * time.Second,
		Write:      30 * time.Second,
		Idle:       60 * time.Second,
		Shutdown:   15 * time.Second,
		Hook:       5 * time.Second,
	})
	server := common.NewServerWithTimeouts(&http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: cfg.ReadHeader}, cfg)

	saveHook := func(ctx context.Context) error {
		if app.gotSaveTrigger {
			log.Println("Saving index before shutdown (triggered)")
			return app.storage.SaveItems(app.itemIndex.GetAllItems())
		}
		return nil
	}
	common.RunServerWithShutdown(server, "reader server", cfg.Shutdown, cfg.Hook, saveHook)
}
