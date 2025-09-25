package main

import (
	"log"

	"github.com/matst80/slask-finder/pkg/facet"
	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/search"
	"github.com/matst80/slask-finder/pkg/sorting"
	"github.com/matst80/slask-finder/pkg/storage"
)

var itemsFile = "data/index-v2.jz"

func main() {
	itemIndex := index.NewIndexWithStock()
	sortingHandler := sorting.NewSortingItemHandler()
	searchHandler := search.NewFreeTextItemHandler(search.DefaultFreeTextHandlerOptions())
	facetHandler := facet.NewFacetItemHandler(facet.FacetItemHandlerOptions{})
	storage.LoadFacets(facetHandler)
	storage.LoadItems(itemsFile, itemIndex, sortingHandler, facetHandler, searchHandler)
	log.Printf("loaded %d items", len(itemIndex.Items))
	//mux := http.NewServeMux()

	// mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
	// 	w.Write([]byte("ok"))
	// })
	// http.ListenAndServe(":8080", mux)
}
