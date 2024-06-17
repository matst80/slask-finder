package main

import (
	"flag"
	"log"
	"os"

	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/index"
	"tornberg.me/facet-search/pkg/persistance"
	"tornberg.me/facet-search/pkg/search"
	"tornberg.me/facet-search/pkg/server"
	"tornberg.me/facet-search/pkg/sync"
)

var enableProfiling = flag.Bool("profiling", false, "enable profiling endpoints")
var rabbitUrl = os.Getenv("RABBIT_URL")
var clientName = os.Getenv("NODE_NAME")

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
	FacetLimit:       5000,
	SearchFacetLimit: 1500,
	ListenAddress:    ":8080",
}

func Init() {

	idx.AddKeyField(&facet.BaseField{Id: 1, Name: "Article Type", HideFacet: true})
	idx.AddKeyField(&facet.BaseField{Id: 2, Name: "Brand", Description: "Brand name"})
	idx.AddKeyField(&facet.BaseField{Id: 3, Name: "Stock level", Description: "Central stock level"})
	idx.AddKeyField(&facet.BaseField{Id: 10, Name: "Category", Description: "Category"})
	idx.AddKeyField(&facet.BaseField{Id: 11, Name: "Category parent"})
	idx.AddKeyField(&facet.BaseField{Id: 12, Name: "Master category"})
	idx.AddKeyField(&facet.BaseField{Id: 20, Name: "B grade", Description: "Outlet rating"})
	//idx.AddBoolField(&facet.BaseField{Id: 21, Name: "Discounted", Description: ""})
	idx.AddIntegerField(&facet.BaseField{Id: 4, Name: "Price", Description: "Current price"})
	idx.AddIntegerField(&facet.BaseField{Id: 5, Name: "Regular price", Description: "Regular price"})
	idx.AddIntegerField(&facet.BaseField{Id: 6, Name: "Average rating", Description: "Average rating"})
	idx.AddIntegerField(&facet.BaseField{Id: 7, Name: "Review count", Description: "Total number of reviews"})
	idx.AddIntegerField(&facet.BaseField{Id: 8, Name: "Discount", Description: "Discount value"})
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
			priceSort := MakeSortFromNumberField(idx.Items, 4)
			srv.DefaultSort = &priceSort
			srv.FieldSort = &fieldSort
			log.Println("Index loaded")
			idx.CreateDefaultFacets(&fieldSort)
			log.Println("Default facets created")

		}

	}()

}

func main() {
	flag.Parse()
	Init()

	log.Printf("Starting server")

	srv.StartServer(*enableProfiling)

}
