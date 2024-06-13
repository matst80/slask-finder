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

var enableProfiling = flag.Bool("profiling", true, "enable profiling endpoints")

var idx = index.NewIndex()
var db = persistance.NewPersistance()
var srv = server.MakeWebServer(db, idx)
var freetext_search = search.NewFreeTextIndex(search.Tokenizer{MaxTokens: 128})

func Init() error {

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

	err := db.LoadIndex(idx)
	if err != nil {
		log.Printf("Failed to load index %v", err)
		return err
	}
	fieldSort := MakeSortForFields()
	priceSort := MakeSortFromNumberField(idx.Items, 4)
	srv.DefaultSort = &priceSort
	srv.FieldSort = &fieldSort

	idx.CreateDefaultFacets(&fieldSort)

	err = db.LoadFreeText(freetext_search)

	if err != nil {

		return err
	}
	return nil

}

func main() {
	flag.Parse()
	err := Init()
	if err == nil {
		log.Printf("Db loaded, Starting server")
	} else {
		log.Printf("Failed to load db: %v", err)
	}

	srv.StartServer(*enableProfiling)

}
