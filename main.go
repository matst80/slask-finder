package main

import (
	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/persistance"
	"tornberg.me/facet-search/pkg/server"
)

func main() {
	db := persistance.NewPersistance()
	srv := server.NewWebServer(&db)

	srv.Index.AddField(facet.Field{Id: 1, Name: "Article Type"})
	srv.Index.AddField(facet.Field{Id: 2, Name: "Brand", Description: "Brand name"})
	srv.Index.AddField(facet.Field{Id: 3, Name: "Stock level", Description: "Central stock level"})
	srv.Index.AddField(facet.Field{Id: 10, Name: "Category", Description: "Category"})
	srv.Index.AddField(facet.Field{Id: 11, Name: "Category parent", Description: ""})
	srv.Index.AddField(facet.Field{Id: 12, Name: "Master category", Description: ""})
	srv.Index.AddField(facet.Field{Id: 20, Name: "B grade", Description: "Outlet rating"})
	srv.Index.AddBoolField(facet.Field{Id: 21, Name: "Discounted", Description: ""})
	srv.Index.AddNumberField(facet.Field{Id: 4, Name: "Price", Description: "Current price"})
	srv.Index.AddNumberField(facet.Field{Id: 5, Name: "Regular price", Description: "Regular price"})
	srv.Index.AddNumberField(facet.Field{Id: 6, Name: "Average rating", Description: "Average rating"})
	srv.Index.AddNumberField(facet.Field{Id: 7, Name: "Review count", Description: "Total number of reviews"})

	addDbFields(srv.Index)
	srv.StartServer()

}
