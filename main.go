package main

import (
	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/index"
	"tornberg.me/facet-search/pkg/server"
)

func main() {

	srv := server.NewWebServer()
	srv.Index.AddField(1, facet.Field{Name: "first", Description: "first field"})
	srv.Index.AddField(2, facet.Field{Name: "other", Description: "other field"})
	srv.Index.AddNumberField(3, facet.Field{Name: "number", Description: "number field"})
	srv.Index.AddItem(index.Item{
		Id:    1,
		Title: "item1",
		Fields: map[int64]string{
			1: "test",
			2: "hej",
		},
		NumberFields: map[int64]float64{
			3: 1,
		},
		Props: map[string]string{
			"test": "test",
		},
	})
	srv.Index.AddItem(index.Item{
		Id:    2,
		Title: "item2",
		Fields: map[int64]string{
			1: "testar",
			2: "hej",
		},
		NumberFields: map[int64]float64{
			3: 3,
		},
		Props: map[string]string{
			"hej": "hej",
		},
	})
	srv.StartServer()
}
