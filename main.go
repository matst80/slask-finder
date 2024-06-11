package main

import (
	"flag"
	"log"
	"os"
	"runtime/pprof"

	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/persistance"
	"tornberg.me/facet-search/pkg/server"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func Init() *server.WebServer {
	db := persistance.NewPersistance()
	srv := server.NewWebServer(&db)

	srv.Index.AddKeyField(&facet.BaseField{Id: 1, Name: "Article Type"})
	srv.Index.AddKeyField(&facet.BaseField{Id: 2, Name: "Brand", Description: "Brand name"})
	srv.Index.AddKeyField(&facet.BaseField{Id: 3, Name: "Stock level", Description: "Central stock level"})
	srv.Index.AddKeyField(&facet.BaseField{Id: 10, Name: "Category", Description: "Category"})
	srv.Index.AddKeyField(&facet.BaseField{Id: 11, Name: "Category parent", Description: ""})
	srv.Index.AddKeyField(&facet.BaseField{Id: 12, Name: "Master category", Description: ""})
	srv.Index.AddKeyField(&facet.BaseField{Id: 20, Name: "B grade", Description: "Outlet rating"})
	//srv.Index.AddBoolField(&facet.BaseField{Id: 21, Name: "Discounted", Description: ""})
	srv.Index.AddDecimalField(&facet.BaseField{Id: 4, Name: "Price", Description: "Current price"})
	srv.Index.AddDecimalField(&facet.BaseField{Id: 5, Name: "Regular price", Description: "Regular price"})
	srv.Index.AddDecimalField(&facet.BaseField{Id: 6, Name: "Average rating", Description: "Average rating"})
	srv.Index.AddIntegerField(&facet.BaseField{Id: 7, Name: "Review count", Description: "Total number of reviews"})

	addDbFields(srv.Index)
	srv.LoadDatabase()
	return &srv
}

func main() {
	flag.Parse()
	srv := Init()
	log.Printf("Db loaded, Starting server")
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Starting CPU profile")
		pprof.StartCPUProfile(f)

		defer pprof.StopCPUProfile()
		defer log.Printf("Stopping CPU profile")
	}
	srv.StartServer()

}
