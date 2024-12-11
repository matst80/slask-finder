package main

import (
	"flag"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/matst80/slask-finder/pkg/embeddings"
	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/persistance"
	"github.com/matst80/slask-finder/pkg/promotions"
	"github.com/matst80/slask-finder/pkg/search"
	"github.com/matst80/slask-finder/pkg/server"
	"github.com/matst80/slask-finder/pkg/sync"
	"github.com/matst80/slask-finder/pkg/tracking"
	"github.com/matst80/slask-finder/pkg/types"
)

var enableProfiling = flag.Bool("profiling", true, "enable profiling endpoints")
var rabbitUrl = os.Getenv("RABBIT_URL")
var clientName = os.Getenv("NODE_NAME")
var redisUrl = os.Getenv("REDIS_URL")
var redisPassword = os.Getenv("REDIS_PASSWORD")
var listenAddress = ":8080"
var clientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")
var callbackUrl = os.Getenv("CALLBACK_URL")

var promotionStorage = promotions.DiskPromotionStorage{
	Path: "data/promotions.json",
}

// var cartStorage = cart.NewDiskCartStorage("data/carts", &promotionStorage)
var rabbitConfig = sync.RabbitConfig{
	//ItemChangedTopic: "item_changed",
	ItemsUpsertedTopic: "item_added",
	ItemDeletedTopic:   "item_deleted",
	PriceLoweredTopic:  "price_lowered",
	Url:                rabbitUrl,
}
var token = search.Tokenizer{MaxTokens: 128}
var freetext_search = search.NewFreeTextIndex(&token)
var idx = index.NewIndex(freetext_search)
var db = persistance.NewPersistance()

//var cartServer = cart.CartServer{
//
//	Storage:   cartStorage,
//	IdHandler: cartStorage,
//	Index:     idx,
//}

var promotionServer = promotions.PromotionServer{
	Storage: &promotionStorage,
}

var embeddingsIndex = embeddings.NewEmbeddingsIndex()
var contentIdx = index.NewContentIndex()

var srv = server.WebServer{
	Index:            idx,
	Db:               db,
	ContentIndex:     contentIdx,
	FacetLimit:       1024,
	SearchFacetLimit: 10280,
	Cache:            nil,
	Embeddings:       embeddingsIndex,
}

var done = false

func Init() {

	if redisUrl == "" {
		log.Fatalf("No redis url provided")

	}
	srv.Cache = server.NewCache(redisUrl, redisPassword, 0)

	srv.Sorting = index.NewSorting(redisUrl, redisPassword, 0)
	idx.Sorting = srv.Sorting
	log.Printf("Cache and sort distribution enabled, url: %s", redisUrl)

	idx.AddKeyField(&types.BaseField{Id: 1, Name: "Article Type", HideFacet: true, Priority: 0})
	idx.AddKeyField(&types.BaseField{Id: 2, Name: "Märke", Description: "Tillverkarens namn", Priority: 1199999999.0, Type: "brand"})
	idx.AddKeyField(&types.BaseField{Id: 3, Name: "Lager", Description: "Lagerstatus", Priority: 99999.0})
	idx.AddKeyField(&types.BaseField{Id: 9, Name: "Säljs av", Description: "", Priority: 199999.0})

	idx.AddKeyField(&types.BaseField{Id: 10, Name: "Huvudkategori", Description: "Category", Priority: 3999999999.0, IgnoreIfInSearch: true, CategoryLevel: 1})
	idx.AddKeyField(&types.BaseField{Id: 11, Name: "Underkaterori", Description: "Sub category", Priority: 2999999997.0, IgnoreIfInSearch: true, CategoryLevel: 2})
	idx.AddKeyField(&types.BaseField{Id: 12, Name: "Kategori", Description: "Tillhör kategori", Priority: 1999999996.0, IgnoreIfInSearch: true, CategoryLevel: 3})
	idx.AddKeyField(&types.BaseField{Id: 13, Name: "Kategori", Description: "Extra kategori", Priority: 1999999995.0, IgnoreIfInSearch: true, CategoryLevel: 4})

	idx.AddKeyField(&types.BaseField{Id: 20, Name: "Skick", Description: "Outlet rating", Priority: 111999.0, Type: "bgrade"})
	idx.AddKeyField(&types.BaseField{Id: 21, Name: "Promotion", Description: "", Priority: 999999999.0, Type: "virtual"})
	idx.AddKeyField(&types.BaseField{Id: 22, Name: "Virtual category", Description: "", Priority: 99.0, Type: "virtual"})

	idx.AddKeyField(&types.BaseField{Id: 23, Name: "Assigned taxonomy id", Description: "", Priority: 99.0})

	//idx.AddBoolField(&types.BaseField{Id: 21, Name: "Discounted", Description: "",Priority: 999999999.0})
	idx.AddIntegerField(&types.BaseField{Id: 4, Name: "Pris", Priority: 1099999995.5, Type: "currency"})
	idx.AddIntegerField(&types.BaseField{Id: 5, Name: "Tidigare pris", HideFacet: true, Priority: 1999, Type: "currency"})
	idx.AddIntegerField(&types.BaseField{Id: 6, Name: "Betyg", Description: "Average rating", Priority: 9999999.0, Type: "rating"})
	idx.AddIntegerField(&types.BaseField{Id: 7, Name: "Antal betyg", Description: "Total number of reviews", Priority: 9999998.0})
	idx.AddIntegerField(&types.BaseField{Id: 8, Name: "Rabatt", Description: "Discount value", Priority: 999.0, Type: "currency"})
	idx.AddIntegerField(&types.BaseField{Id: 14, Name: "Klubb pris", HideFacet: true, Priority: 1299999995.4, Type: "currency"})

	idx.AddKeyField(&types.BaseField{Id: 30, Name: "PT 1", HideFacet: true, Priority: 0, IgnoreIfInSearch: true})
	idx.AddKeyField(&types.BaseField{Id: 31, Name: "PT 2", HideFacet: true, Priority: 0, IgnoreIfInSearch: true})
	idx.AddKeyField(&types.BaseField{Id: 32, Name: "PT 3", HideFacet: true, Priority: 0, IgnoreIfInSearch: true})
	idx.AddKeyField(&types.BaseField{Id: 33, Name: "PT 4", HideFacet: true, Priority: 0, IgnoreIfInSearch: true})

	addDbFields(idx)
	//srv.Sorting.LoadAll()

	go populateContentFromCsv(contentIdx, "data/content.csv")

	go func() {

		err := db.LoadIndex(idx)

		if err != nil {
			log.Printf("Failed to load index %v", err)
		} else {
			log.Println("Index loaded")
			if rabbitUrl != "" {
				srv.Tracking = tracking.NewRabbitTracking(tracking.RabbitTrackingConfig{
					TrackingTopic: "tracking",
					Url:           rabbitUrl,
				})
				//cartServer.Tracking = srv.Tracking
				if clientName == "" {
					masterTransport := sync.RabbitTransportMaster{
						RabbitConfig: rabbitConfig,
					}
					log.Println("Starting as master")
					err := masterTransport.Connect()
					if err != nil {
						log.Printf("Failed to connect to RabbitMQ as master, %v", err)
					} else {
						idx.ChangeHandler = &sync.RabbitMasterChangeHandler{
							Master: masterTransport,
						}
					}
				} else {
					log.Printf("Starting as client: %s", clientName)
					clientTransport := sync.RabbitTransportClient{
						ClientName:   clientName,
						RabbitConfig: rabbitConfig,
					}
					err := clientTransport.Connect(idx)
					srv.Sorting.InitializeWithIndex(idx)
					srv.Sorting.StartListeningForChanges()

					go func() {

						for _, item := range idx.Items {
							embeddingsIndex.AddDocument(embeddings.MakeDocument(*item))
						}

						log.Printf("Embeddings index loaded")
					}()
					if err != nil {
						log.Printf("Failed to connect to RabbitMQ as clinet, %v", err)
					}
				}
			} else {
				log.Println("Starting as standalone")
			}

		}
		runtime.GC()
		done = true
		// db.SaveIndex(idx)
	}()

}

func main() {
	flag.Parse()

	Init()

	authConfig := &oauth2.Config{
		ClientID:     "1017700364201-hiv4l9c41osmqfkv17ju7gg08e570lfr.apps.googleusercontent.com",
		ClientSecret: clientSecret,
		RedirectURL:  callbackUrl,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if !done {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("not ready"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	if rabbitUrl != "" {
		if clientName == "" {
			mux.Handle("/admin/", http.StripPrefix("/admin", srv.AdminHandler()))
			srv.OAuthConfig = authConfig
		} else {
			mux.Handle("/api/", http.StripPrefix("/api", srv.ClientHandler()))
		}
	} else {
		mux.Handle("/admin/", http.StripPrefix("/admin", srv.AdminHandler()))
		srv.OAuthConfig = authConfig
		mux.Handle("/api/", http.StripPrefix("/api", srv.ClientHandler()))
	}

	//mux.Handle("/cart/", http.StripPrefix("/cart", cartServer.CartHandler()))
	mux.Handle("/promotion/", http.StripPrefix("/promotion", promotionServer.PromotionHandler()))
	mux.Handle("/metrics", promhttp.Handler())

	if enableProfiling != nil && *enableProfiling {
		log.Println("Profiling enabled")
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		//mux.Handle("/debug/pprof/", )
	}
	log.Printf("Starting server %v", listenAddress)
	log.Fatal(http.ListenAndServe(listenAddress, mux))
}
