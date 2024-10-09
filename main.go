package main

import (
	"flag"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime"

	"tornberg.me/facet-search/pkg/cart"
	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/index"
	"tornberg.me/facet-search/pkg/persistance"
	"tornberg.me/facet-search/pkg/promotions"
	"tornberg.me/facet-search/pkg/search"
	"tornberg.me/facet-search/pkg/server"
	"tornberg.me/facet-search/pkg/sync"
	"tornberg.me/facet-search/pkg/tracking"
)

var enableProfiling = flag.Bool("profiling", false, "enable profiling endpoints")
var rabbitUrl = os.Getenv("RABBIT_URL")
var clientName = os.Getenv("NODE_NAME")
var redisUrl = os.Getenv("REDIS_URL")
var redisPassword = os.Getenv("REDIS_PASSWORD")
var listenAddress = ":8080"

var promotionStorage = promotions.DiskPromotionStorage{
	Path: "data/promotions.json",
}
var cartStorage = cart.NewDiskCartStorage("data/carts", &promotionStorage)
var rabbitConfig = sync.RabbitConfig{
	//ItemChangedTopic: "item_changed",
	ItemsUpsertedTopic: "item_added",
	ItemDeletedTopic:   "item_deleted",
	Url:                rabbitUrl,
}
var token = search.Tokenizer{MaxTokens: 128}
var freetext_search = search.NewFreeTextIndex(&token)
var idx = index.NewIndex(freetext_search)
var db = persistance.NewPersistance()

var cartServer = cart.CartServer{

	Storage:   cartStorage,
	IdHandler: cartStorage,
	Index:     idx,
}

var promotionServer = promotions.PromotionServer{
	Storage: &promotionStorage,
}

var srv = server.WebServer{
	Index:            idx,
	Db:               db,
	FacetLimit:       10280,
	SearchFacetLimit: 10280,
	Cache:            nil,
}

var done = false

func Init() {

	if redisUrl == "" {
		log.Fatalf("No redis url provided")

	}
	srv.Cache = server.NewCache(redisUrl, redisPassword, 0)

	srv.Sorting = index.NewSorting(redisUrl, redisPassword, 0)
	idx.Sorting = srv.Sorting
	log.Printf("Cache and sort distribution enabled, url: %s", redisUrl)

	idx.AddKeyField(&facet.BaseField{Id: 1, Name: "Article Type", HideFacet: true, Priority: 0})
	idx.AddKeyField(&facet.BaseField{Id: 2, Name: "Märke", Description: "Tillverkarens namn", Priority: 1199999999.0, Type: "brand"})
	idx.AddKeyField(&facet.BaseField{Id: 3, Name: "Lager", Description: "Lagerstatus", Priority: 99999.0})
	idx.AddKeyField(&facet.BaseField{Id: 9, Name: "Säljs av", Description: "", Priority: 199999.0})
	idx.AddKeyField(&facet.BaseField{Id: 10, Name: "Huvudkategori", Description: "Category", Priority: 3999999999.0, IgnoreIfInSearch: true, CategoryLevel: 1})
	idx.AddKeyField(&facet.BaseField{Id: 11, Name: "Underkaterori", Description: "Sub category", Priority: 2999999997.0, IgnoreIfInSearch: true, CategoryLevel: 2})
	idx.AddKeyField(&facet.BaseField{Id: 12, Name: "Kategori", Description: "Tillhör kategori", Priority: 1999999996.0, IgnoreIfInSearch: true, CategoryLevel: 3})
	idx.AddKeyField(&facet.BaseField{Id: 12, Name: "Kategori", Description: "Extra kategori", Priority: 1999999995.0, IgnoreIfInSearch: true, CategoryLevel: 4})
	idx.AddKeyField(&facet.BaseField{Id: 20, Name: "Skick", Description: "Outlet rating", Priority: 111999.0, Type: "bgrade"})
	idx.AddKeyField(&facet.BaseField{Id: 21, Name: "Promotion", Description: "", Priority: 999999999.0, Type: "virtual"})
	idx.AddKeyField(&facet.BaseField{Id: 22, Name: "Virtual category", Description: "", Priority: 99.0, Type: "virtual"})
	//idx.AddBoolField(&facet.BaseField{Id: 21, Name: "Discounted", Description: "",Priority: 999999999.0})
	idx.AddIntegerField(&facet.BaseField{Id: 4, Name: "Pris", Priority: 1999999995.5, Type: "currency"})
	idx.AddIntegerField(&facet.BaseField{Id: 5, Name: "Tidigare pris", Priority: 1999999995.4, Type: "currency"})
	idx.AddIntegerField(&facet.BaseField{Id: 6, Name: "Betyg", Description: "Average rating", Priority: 9999999.0, Type: "rating"})
	idx.AddIntegerField(&facet.BaseField{Id: 7, Name: "Antal betyg", Description: "Total number of reviews", Priority: 9999998.0})
	idx.AddIntegerField(&facet.BaseField{Id: 8, Name: "Rabatt", Description: "Discount value", Priority: 999.0, Type: "currency"})
	addDbFields(idx)
	//srv.Sorting.LoadAll()

	go func() {

		if rabbitUrl != "" {
			srv.Tracking = tracking.NewRabbitTracking(tracking.RabbitTrackingConfig{
				TrackingTopic: "tracking",
				Url:           rabbitUrl,
			})
			cartServer.Tracking = srv.Tracking
			if clientName == "" {
				masterTransport := sync.RabbitTransportMaster{
					RabbitConfig: rabbitConfig,
				}
				log.Println("Starting as master")
				err := masterTransport.Connect()
				if err != nil {
					log.Printf("Failed to connect to RabbitMQ as master, %v", err)
				} else {
					idx.ChangeHandler = &sync.RabbitMasterChangeHandler{
						Master: masterTransport,
					}
				}
			} else {
				log.Printf("Starting as client: %s", clientName)
				clientTransport := sync.RabbitTransportClient{
					ClientName:   clientName,
					RabbitConfig: rabbitConfig,
				}
				err := clientTransport.Connect(idx)
				if err != nil {
					log.Printf("Failed to connect to RabbitMQ as clinet, %v", err)
				}
			}
		} else {
			log.Println("Starting as standalone")
		}
		err := db.LoadIndex(idx)
		if err != nil {
			log.Printf("Failed to load index %v", err)
		} else {
			log.Println("Index loaded")
			srv.Sorting.InitializeWithIndex(idx)
			runtime.GC()
		}
		done = true
	}()

}

func main() {
	flag.Parse()

	Init()

	log.Printf("Starting server %v", listenAddress)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if !done {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("not ready"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	if rabbitUrl != "" {
		if clientName == "" {
			mux.Handle("/admin/", http.StripPrefix("/admin", srv.AdminHandler()))
		} else {
			mux.Handle("/api/", http.StripPrefix("/api", srv.ClientHandler()))
		}
	} else {
		mux.Handle("/admin/", http.StripPrefix("/admin", srv.AdminHandler()))
		mux.Handle("/api/", http.StripPrefix("/api", srv.ClientHandler()))
	}

	mux.Handle("/cart/", http.StripPrefix("/cart", cartServer.CartHandler()))
	mux.Handle("/promotion/", http.StripPrefix("/promotion", promotionServer.PromotionHandler()))

	if enableProfiling != nil && *enableProfiling {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		//mux.Handle("/debug/pprof/", )
	}
	log.Fatal(http.ListenAndServe(listenAddress, mux))
}
