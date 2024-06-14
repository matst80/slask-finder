package main

import (
	"flag"
	"log"

	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/index"
	"tornberg.me/facet-search/pkg/persistance"
	"tornberg.me/facet-search/pkg/search"
	"tornberg.me/facet-search/pkg/server"
)

var enableProfiling = flag.Bool("profiling", false, "enable profiling endpoints")
var token = search.Tokenizer{MaxTokens: 128}
var freetext_search = search.NewFreeTextIndex(&token)
var idx = index.NewIndex(freetext_search)
var db = persistance.NewPersistance()

var srv = server.MakeWebServer(db, idx)

func Init() {

	idx.AddKeyField(&facet.BaseField{Id: 1, Name: "Article Type", HideFacet: true})
	idx.AddKeyField(&facet.BaseField{Id: 2, Name: "Brand", Description: "Brand name"})
	idx.AddKeyField(&facet.BaseField{Id: 3, Name: "Stock level", Description: "Central stock level"})
	idx.AddKeyField(&facet.BaseField{Id: 10, Name: "Category", Description: "Category"})
	idx.AddKeyField(&facet.BaseField{Id: 11, Name: "Category parent", HideFacet: true})
	idx.AddKeyField(&facet.BaseField{Id: 12, Name: "Master category", HideFacet: true})
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
