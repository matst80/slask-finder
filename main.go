package main

import (
	"flag"
	"log"
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

var rabbitConfig = sync.RabbitConfig{
	ItemChangedTopic: "item_changed",
	ItemAddedTopic:   "item_added",
	ItemDeletedTopic: "item_deleted",
	Url:              rabbitUrl,
}
var token = search.Tokenizer{MaxTokens: 128}
var freetext_search = search.NewFreeTextIndex(&token)
var idx = index.NewIndex(freetext_search)
var db = persistance.NewPersistance()
var masterTransport = sync.RabbitTransportMaster{
	RabbitConfig: rabbitConfig,
}

type RabbitMasterChangeHandler struct{}

func (r *RabbitMasterChangeHandler) ItemChanged(item *index.DataItem) {
	masterTransport.SendItemAdded(item)
}

func (r *RabbitMasterChangeHandler) ItemAdded(item *index.DataItem) {
	masterTransport.SendItemChanged(item)
}

func (r *RabbitMasterChangeHandler) ItemDeleted(id uint) {
	masterTransport.SendItemDeleted(id)
}

var srv = server.WebServer{
	Index:            idx,
	Db:               db,
	FacetLimit:       6400,
	SearchFacetLimit: 3500,
	ListenAddress:    ":8080",
	Cache:            nil,
}

func Init() {
	if clickhouseUrl != "" {
		trk, err := tracking.NewClickHouse(clickhouseUrl) //"10.10.3.19:9000"
		if err != nil {
			log.Fatalf("Failed to connect to ClickHouse %v", err)
		}

		srv.Tracking = trk
	}
	if redisUrl != "" {
		srv.Cache = server.NewCache(redisUrl, redisPassword, 0)
		log.Printf("Cache enabled, url: %s", redisUrl)
	}
	idx.AddKeyField(&facet.BaseField{Id: 1, Name: "Article Type", HideFacet: true})
	idx.AddKeyField(&facet.BaseField{Id: 2, Name: "Märke", Description: "Tillverkarens namn"})
	idx.AddKeyField(&facet.BaseField{Id: 3, Name: "Lager", Description: "Central stock level"})
	idx.AddKeyField(&facet.BaseField{Id: 10, Name: "Huvudkategori", Description: "Category"})
	idx.AddKeyField(&facet.BaseField{Id: 11, Name: "Kategori", Description: "Sub category"})
	idx.AddKeyField(&facet.BaseField{Id: 12, Name: "Kategori", Description: "Tillhör kategori"})
	idx.AddKeyField(&facet.BaseField{Id: 20, Name: "B grade", Description: "Outlet rating"})
	//idx.AddBoolField(&facet.BaseField{Id: 21, Name: "Discounted", Description: ""})
	idx.AddIntegerField(&facet.BaseField{Id: 4, Name: "Pris"})
	idx.AddIntegerField(&facet.BaseField{Id: 5, Name: "Tidigare pris"})
	idx.AddIntegerField(&facet.BaseField{Id: 6, Name: "Betyg", Description: "Average rating"})
	idx.AddIntegerField(&facet.BaseField{Id: 7, Name: "Antal betyg", Description: "Total number of reviews"})
	idx.AddIntegerField(&facet.BaseField{Id: 8, Name: "Rabatt", Description: "Discount value"})
	addDbFields(idx)

	go func() {
		err := db.LoadIndex(idx)
		if rabbitUrl != "" && err == nil {
			if clientName == "" {
				log.Println("Starting as master")
				masterTransport.Connect()
				idx.ChangeHandler = &RabbitMasterChangeHandler{}
			} else {
				log.Printf("Starting as client: %s", clientName)
				clientTransport := sync.RabbitTransportClient{
					ClientName:   clientName,
					RabbitConfig: rabbitConfig,
				}
				clientTransport.Connect(idx)
			}
		} else {
			log.Println("Starting as standalone")
		}
		if err != nil {
			log.Printf("Failed to load index %v", err)
		} else {
			fieldSort := MakeSortForFields()
			sortMap, priceSort := MakeSortFromNumberField(idx.Items, 4)
			srv.SortMethods = MakeSortMaps(idx.Items)
			idx.Search.BaseSortMap = ToMap(&sortMap)
			srv.DefaultSort = &priceSort
			srv.FieldSort = &fieldSort
			log.Println("Index loaded")
			idx.CreateDefaultFacets(&fieldSort)
			log.Println("Default facets created")
			runtime.GC()
		}

	}()

}

func main() {
	flag.Parse()
	Init()

	log.Printf("Starting server")

	srv.StartServer(*enableProfiling)

}
