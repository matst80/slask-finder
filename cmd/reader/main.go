package main

import (
	"context"
	"encoding/json"
	"iter"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/matst80/slask-finder/pkg/common"
	"github.com/matst80/slask-finder/pkg/facet"
	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/messaging"
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

func (a *app) ConnectAmqp(amqpUrl string) {
	conn, err := amqp.DialConfig(amqpUrl, amqp.Config{
		Properties: amqp.NewConnectionProperties(),
	})
	a.conn = conn
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	// items listener
	messaging.ListenToTopic(ch, country, "item_added", func(d amqp.Delivery) error {
		var items []index.DataItem
		if err := json.Unmarshal(d.Body, &items); err == nil {
			log.Printf("Got upserts %d", len(items))

			go a.itemIndex.HandleItems(asItems(items))
			go a.facetHandler.HandleItems(asItems(items))
			go a.sortingHandler.HandleItems(asItems(items))
			go a.searchIndex.HandleItems(asItems(items))

		} else {
			log.Printf("Failed to unmarshal upset message %v", err)
		}
		return nil
	})

	log.Printf("Listening for item upserts")

	ticker := time.NewTicker(time.Minute * 1)
	go func() {
		for range ticker.C {
			if a.gotSaveTrigger {
				log.Println("Saving items due to trigger")
				err := a.storage.SaveItems(a.itemIndex.GetAllItems())
				if err != nil {
					log.Printf("Failed to save items: %v", err)
				}
				a.gotSaveTrigger = false
			}
		}
	}()
}

func (a *app) ConnectFacetChange() {
	ch, err := a.conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	messaging.ListenToTopic(ch, country, "facet_change", func(d amqp.Delivery) error {
		var items []types.FieldChange
		if err := json.Unmarshal(d.Body, &items); err == nil {
			log.Printf("Got fieldchanges %d", len(items))
			a.facetHandler.HandleFieldChanges(items)
		} else {
			log.Printf("Failed to unmarshal facet change message %v", err)
		}
		return nil
	})
}

func (a *app) ConnectSettingsChange() {
	ch, err := a.conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	// items listerner
	messaging.ListenToTopic(ch, country, "settings_change", func(d amqp.Delivery) error {
		var item types.SettingsChange
		if err := json.Unmarshal(d.Body, &item); err == nil {
			log.Printf("Got settings %v", item)
			a.storage.LoadSettings()
		} else {
			log.Printf("Failed to unmarshal upset message %v", err)
		}
		return nil
	})
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
		app.ConnectAmqp(amqpUrl)
		app.ConnectFacetChange()
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
		w.Write([]byte("ok"))
	})

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
	mux.HandleFunc("GET /api/save-trigger", common.JsonHandler(tracker, app.SaveTrigger))
	mux.HandleFunc("GET /api/relation-groups", common.JsonHandler(tracker, app.GetRelationGroups))
	mux.HandleFunc("POST /api/stream-items", app.StreamItemsFromIds)

	//mux.HandleFunc("/api/similar", common.JsonHandler(tracker, app.Similar))
	/*
		  	mux.HandleFunc("/natural", common.JsonHandler(tracking, ws.SearchEmbeddings))
				mux.HandleFunc("/cosine-similar/{id}", common.JsonHandler(tracking, ws.CosineSimilar))
				mux.HandleFunc("/trigger-words", common.JsonHandler(tracking, ws.TriggerWords))
				mux.HandleFunc("/find-related", common.JsonHandler(tracking, app.FindRelated))


				mux.HandleFunc("/reload-settings", common.JsonHandler(tracking, app.ReloadSettings))


				mux.HandleFunc("/ids", common.JsonHandler(tracking, app.GetIds))

				mux.HandleFunc("POST /get", common.JsonHandler(tracking, app.GetItems))

				mux.HandleFunc("/predict-sequence", common.JsonHandler(tracking, app.PredictSequence))
				mux.HandleFunc("/predict-tree", common.JsonHandler(tracking, app.PredictTree))

	*/
	server := &http.Server{Addr: ":8080", Handler: mux}

	go func() {
		log.Println("starting server on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on %s: %v\n", server.Addr, err)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down server...")
	if app.gotSaveTrigger {
		// Save the index
		log.Println("Saving index...")
		if err := app.storage.SaveItems(app.itemIndex.GetAllItems()); err != nil {
			log.Printf("Failed to save items: %v", err)
		} else {
			log.Println("Index saved successfully.")
		}
	}

	// Shutdown the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}

	log.Println("Server gracefully stopped")
}
