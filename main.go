package main

import (
	"flag"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime"

	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/index"
	"tornberg.me/facet-search/pkg/persistance"
	"tornberg.me/facet-search/pkg/search"
	"tornberg.me/facet-search/pkg/server"
	"tornberg.me/facet-search/pkg/sync"
	"tornberg.me/facet-search/pkg/tracking"
)

var enableProfiling = flag.Bool("profiling", false, "enable profiling endpoints")
var rabbitUrl = os.Getenv("RABBIT_URL")
var clientName = os.Getenv("NODE_NAME")
var redisUrl = os.Getenv("REDIS_URL")
var clickhouseUrl = os.Getenv("CLICKHOUSE_URL")
var redisPassword = os.Getenv("REDIS_PASSWORD")
var listenAddress = ":8080"

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
var masterTransport = sync.RabbitTransportMaster{
	RabbitConfig: rabbitConfig,
}

type RabbitMasterChangeHandler struct{}

func (r *RabbitMasterChangeHandler) ItemsUpserted(items []*index.DataItem) {

	err := masterTransport.ItemsUpserted(items)
	if err != nil {
		log.Printf("Failed to send item changed %v", err)
	}
	log.Printf("Items changed %d", len(items))
}

func (r *RabbitMasterChangeHandler) ItemDeleted(id uint) {

	err := masterTransport.SendItemDeleted(id)
	if err != nil {
		log.Printf("Failed to send item deleted %v", err)
	}
	log.Printf("Item deleted %d", id)
}

var srv = server.WebServer{
	Index:            idx,
	Db:               db,
	FacetLimit:       10280,
	SearchFacetLimit: 10280,
	Cache:            nil,
}

func Init() {
	if clickhouseUrl != "" {
		trk, err := tracking.NewClickHouse(clickhouseUrl) //"10.10.3.19:9000"
		if err != nil {
			log.Fatalf("Failed to connect to ClickHouse %v", err)
		}

		srv.Tracking = trk
		log.Printf("Tracking enabled, url: %s", clickhouseUrl)
	}
	if redisUrl == "" {
		log.Fatalf("No redis url provided")

	}
	srv.Cache = server.NewCache(redisUrl, redisPassword, 0)

	srv.Sorting = index.NewSorting(redisUrl, redisPassword, 0)
	idx.Sorting = srv.Sorting
	log.Printf("Cache and sort distribution enabled, url: %s", redisUrl)

	idx.AddKeyField(&facet.BaseField{Id: 1, Name: "Article Type", HideFacet: true, Priority: 0})
	idx.AddKeyField(&facet.BaseField{Id: 2, Name: "Märke", Description: "Tillverkarens namn", Priority: 1199999999.0})
	idx.AddKeyField(&facet.BaseField{Id: 3, Name: "Lager", Description: "Central stock level", Priority: 99999.0})
	idx.AddKeyField(&facet.BaseField{Id: 10, Name: "Huvudkategori", Description: "Category", Priority: 3999999999.0, IgnoreIfInSearch: true})
	idx.AddKeyField(&facet.BaseField{Id: 11, Name: "Kategori", Description: "Sub category", Priority: 2999999999.0, IgnoreIfInSearch: true})
	idx.AddKeyField(&facet.BaseField{Id: 12, Name: "Kategori", Description: "Tillhör kategori", Priority: 1999999999.0, IgnoreIfInSearch: true})
	idx.AddKeyField(&facet.BaseField{Id: 20, Name: "B grade", Description: "Outlet rating", Priority: 19999.0})
	//idx.AddBoolField(&facet.BaseField{Id: 21, Name: "Discounted", Description: "",Priority: 999999999.0})
	idx.AddIntegerField(&facet.BaseField{Id: 4, Name: "Pris", Priority: 9999999999.0})
	idx.AddIntegerField(&facet.BaseField{Id: 5, Name: "Tidigare pris", Priority: 9999999999.0})
	idx.AddIntegerField(&facet.BaseField{Id: 6, Name: "Betyg", Description: "Average rating", Priority: 999999999.0})
	idx.AddIntegerField(&facet.BaseField{Id: 7, Name: "Antal betyg", Description: "Total number of reviews", Priority: 999999999.0})
	idx.AddIntegerField(&facet.BaseField{Id: 8, Name: "Rabatt", Description: "Discount value", Priority: 999999999.0})
	addDbFields(idx)
	//srv.Sorting.LoadAll()

	go func() {
		err := db.LoadIndex(idx)

		if rabbitUrl != "" && err == nil {
			if clientName == "" {
				log.Println("Starting as master")
				err := masterTransport.Connect()
				if err != nil {
					log.Printf("Failed to connect to RabbitMQ as master, %v", err)
				} else {
					idx.ChangeHandler = &RabbitMasterChangeHandler{}
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
		if err != nil {
			log.Printf("Failed to load index %v", err)
		} else {
			log.Println("Index loaded")
			srv.Sorting.InitializeWithIndex(idx)
			runtime.GC()
		}

	}()

}

func main() {
	flag.Parse()
	Init()

	log.Printf("Starting server %v", listenAddress)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.Handle("/api/", http.StripPrefix("/api", srv.ClientHandler()))
	mux.Handle("/admin/", http.StripPrefix("/admin", srv.AdminHandler()))

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
