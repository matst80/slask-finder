package main

import (
	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/persistance"
	"tornberg.me/facet-search/pkg/server"
)

func main() {
	db := persistance.NewPersistance()
	srv := server.NewWebServer(&db)

	// srv.Index.AddItem(index.Item{
	// 	Id:    1,
	// 	Title: "item1",
	// 	Fields: map[int64]string{
	// 		1: "test",
	// 		2: "hej",
	// 	},
	// 	NumberFields: map[int64]float64{
	// 		3: 1,
	// 	},
	// 	Props: map[string]string{
	// 		"test": "test",
	// 	},
	// })
	// srv.Index.AddItem(index.Item{
	// 	Id:    2,
	// 	Title: "item2",
	// 	Fields: map[int64]string{
	// 		1: "testar",
	// 		2: "hej",
	// 	},
	// 	NumberFields: map[int64]float64{
	// 		3: 3,
	// 	},
	// 	Props: map[string]string{
	// 		"hej": "hej",
	// 	},
	// })
	srv.Index.AddField(1, facet.Field{Name: "Article Type"})
	srv.Index.AddField(2, facet.Field{Name: "Brand", Description: "Brand name"})
	srv.Index.AddField(3, facet.Field{Name: "Market", Description: "Country of sale"})
	srv.Index.AddNumberField(4, facet.Field{Name: "Price", Description: "Current price"})

	srv.StartServer()

}
