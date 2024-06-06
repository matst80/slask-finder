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
		Fields: []facet.StringFieldReference{
			{Value: "test", Id: 1},
			{Value: "hej", Id: 2},
		},
		NumberFields: []facet.NumberFieldReference{
			{Value: 1, Id: 3},
		},
		Props: map[string]string{
			"test": "test",
		},
	})
	srv.Index.AddItem(index.Item{
		Id:    2,
		Title: "item2",
		Fields: []facet.StringFieldReference{
			{Value: "testar", Id: 1},
			{Value: "hej", Id: 2},
		},
		NumberFields: []facet.NumberFieldReference{
			{Value: 3, Id: 3},
		},
		Props: map[string]string{
			"hej": "hej",
		},
	})
	srv.StartServer()
}
