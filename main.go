package main

import (
	"flag"
	"log"
	"net/http"
	httpprof "net/http/pprof"
	"os"
	"runtime/pprof"
	"sync"

	"github.com/matst80/slask-finder/pkg/storage"
	"github.com/matst80/slask-finder/pkg/tracking"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/search"
	"github.com/matst80/slask-finder/pkg/server"
	ffSync "github.com/matst80/slask-finder/pkg/sync"
)

var enableProfiling = flag.Bool("profiling", true, "enable profiling endpoints")
var profileLoad = flag.String("profile-startup", "", "write cpu profile to file")

var rabbitUrl = os.Getenv("RABBIT_URL")
var clientName = os.Getenv("NODE_NAME")
var redisUrl = os.Getenv("REDIS_URL")
var redisPassword = os.Getenv("REDIS_PASSWORD")
var listenAddress = ":8080"
var debugAddress = ":8081"
var clientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")
var callbackUrl = os.Getenv("CALLBACK_URL")

var rabbitConfig = ffSync.RabbitConfig{
	//ItemChangedTopic: "item_changed",
	FieldChangeTopic:   "field_change",
	ItemsUpsertedTopic: "items_added",
	ItemDeletedTopic:   "item_deleted",
	PriceLoweredTopic:  "price_lowered",
	VHost:              os.Getenv("RABBIT_HOST"),
	Url:                rabbitUrl,
}
var token = search.Tokenizer{MaxTokens: 128}

var idx = index.NewIndex()
var db = storage.NewPersistance()

// var embeddingsIndex = embeddings.NewEmbeddingsIndex()
var contentIdx = index.NewContentIndex()

var srv = server.WebServer{
	Index:            idx,
	Db:               db,
	ContentIndex:     contentIdx,
	FacetLimit:       1024,
	SearchFacetLimit: 10280,
	FieldData:        map[string]*server.FieldData{},
	Cache:            nil,
	//Embeddings:       embeddingsIndex,
}

var done = false

func init() {
	flag.Parse()

	if redisUrl == "" {
		log.Fatalf("No redis url provided")
	}

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
	srv.OAuthConfig = authConfig
}

func LoadIndex(wg *sync.WaitGroup) {
	wg.Add(1)
	log.Printf("amqp url: %s", rabbitUrl)
	log.Printf("clientName: %s", clientName)

	if rabbitUrl != "" && clientName == "" {
		idx.IsMaster = true
		index.AllowConditionalData = true
		log.Println("Starting with reduced memory consumption")
	} else {
		srv.Cache = server.NewCache(redisUrl, redisPassword, 0)
		srv.Sorting = index.NewSorting(redisUrl, redisPassword, 0)
		idx.Sorting = srv.Sorting
		//idx.AutoSuggest = index.NewAutoSuggest(&token)
		idx.Search = search.NewFreeTextIndex(&token)
		log.Printf("Cache and sort distribution enabled, url: %s", redisUrl)
	}

	// idx.AddKeyField(&types.BaseField{Id: 1, Name: "Article Type", HideFacet: true, Priority: 0}) // 949259
	// idx.AddKeyField(&types.BaseField{Id: 2, Name: "Märke", Description: "Tillverkarens namn", Priority: 119999.0, Type: "brand", ValueSorting: 1})
	// idx.AddKeyField(&types.BaseField{Id: 3, Name: "Lager", Description: "Lagerstatus", Priority: 99999.0})
	// idx.AddKeyField(&types.BaseField{Id: 9, Name: "Säljs av", Description: "", Priority: 199999.0})

	// idx.AddKeyField(&types.BaseField{Id: 10, Name: "Huvudkategori", Description: "Category", Priority: 399999.0, IgnoreIfInSearch: true, CategoryLevel: 1})
	// idx.AddKeyField(&types.BaseField{Id: 11, Name: "Underkaterori", Description: "Sub category", Priority: 299999.0, IgnoreIfInSearch: true, CategoryLevel: 2})
	// idx.AddKeyField(&types.BaseField{Id: 12, Name: "Kategori", Description: "Tillhör kategori", Priority: 199999.0, IgnoreIfInSearch: true, CategoryLevel: 3})
	// idx.AddKeyField(&types.BaseField{Id: 13, Name: "Kategori", Description: "Extra kategori", Priority: 199999.0, IgnoreIfInSearch: true, CategoryLevel: 4})

	// idx.AddKeyField(&types.BaseField{Id: 20, Name: "Skick", Description: "Outlet rating", Priority: 111999.0, Type: "bgrade"})
	// idx.AddKeyField(&types.BaseField{Id: 21, Name: "Promotion", Description: "", Priority: 9999.0, Type: "virtual"})
	// idx.AddKeyField(&types.BaseField{Id: 22, Name: "Virtual category", Description: "", Priority: 99.0, Type: "virtual"})

	// idx.AddKeyField(&types.BaseField{Id: 23, Name: "Assigned taxonomy id", HideFacet: true, Description: "", Priority: 99.0})
	// idx.AddKeyField(&types.BaseField{Id: 24, Name: "Seller id", HideFacet: true, Description: "", Priority: 99.0})

	// //idx.AddBoolField(&types.BaseField{Id: 21, Name: "Discounted", Description: "",Priority: 999999999.0})
	// idx.AddIntegerField(&types.BaseField{Id: 4, Name: "Pris", Priority: 1099999995.5, Type: "currency"})
	// idx.AddIntegerField(&types.BaseField{Id: 5, Name: "Tidigare pris", HideFacet: true, Priority: 1999, Type: "currency"})
	// idx.AddIntegerField(&types.BaseField{Id: 6, Name: "Betyg", Description: "Average rating", Priority: 999999.0, Type: "rating"})
	// idx.AddIntegerField(&types.BaseField{Id: 7, Name: "Antal betyg", Description: "Total number of reviews", Priority: 999998.0})
	// idx.AddIntegerField(&types.BaseField{Id: 8, Name: "Rabatt", Description: "Discount value", Priority: 999.0, Type: "currency"})
	// idx.AddIntegerField(&types.BaseField{Id: 14, Name: "Klubb pris", HideFacet: true, Priority: 99.4, Type: "currency"})
	// //idx.AddKeyField(&types.BaseField{Id: 15, Name: "Article type", HideFacet: true, Priority: 99.3})

	// idx.AddKeyField(&types.BaseField{Id: 30, Name: "PT 1", HideFacet: true, Priority: 0, IgnoreIfInSearch: true, IgnoreCategoryIfSearched: true})
	// idx.AddKeyField(&types.BaseField{Id: 31, Name: "PT 2", HideFacet: true, Priority: 0, IgnoreIfInSearch: true, IgnoreCategoryIfSearched: true})
	// idx.AddKeyField(&types.BaseField{Id: 32, Name: "PT 3", HideFacet: true, Priority: 0, IgnoreIfInSearch: true, IgnoreCategoryIfSearched: true})
	// idx.AddKeyField(&types.BaseField{Id: 33, Name: "PT 4", HideFacet: true, Priority: 0, IgnoreIfInSearch: true, IgnoreCategoryIfSearched: true})

	// idx.AddKeyField(&types.BaseField{Id: 35, Name: "CGM", HideFacet: true, Priority: 0, IgnoreIfInSearch: true, IgnoreCategoryIfSearched: true})
	// idx.AddKeyField(&types.BaseField{Id: 36, Name: "Category group", HideFacet: true, Priority: 0, IgnoreIfInSearch: true, IgnoreCategoryIfSearched: true})

	// addDbFields(idx)

	if rabbitUrl != "" {
		trk, err := tracking.NewRabbitTracking(tracking.RabbitTrackingConfig{
			TrackingTopic: "tracking",
			Url:           rabbitUrl,
		})
		if err != nil {
			log.Fatalf("Failed to create rabbit tracking")
		}
		srv.Tracking = trk
	}

	go func() {
		if *profileLoad != "" {
			f, err := os.Create(*profileLoad)
			if err != nil {
				log.Fatal(err)
			}
			pprof.StartCPUProfile(f)
			defer pprof.StopCPUProfile()
		}
		defer wg.Done()
		err := db.LoadIndex(idx)

		if err != nil {
			log.Printf("Failed to load index %v", err)
		} else {
			log.Println("Index loaded")

			//cartServer.Tracking = srv.Tracking
			if rabbitUrl != "" && clientName == "" {
				log.Println("Starting as master")
				masterTransport := ffSync.RabbitTransportMaster{
					RabbitConfig: rabbitConfig,
				}
				err := masterTransport.Connect()
				if err != nil {
					log.Printf("Failed to connect to RabbitMQ as master, %v", err)
				} else {
					log.Print("Connected to RabbitMQ as master")
					idx.ChangeHandler = &ffSync.RabbitMasterChangeHandler{
						Master: masterTransport,
					}
				}
			} else {
				if clientName == "" {
					log.Printf("Starting as standalone")
				} else {
					log.Printf("Starting as client: %s", clientName)
				}
				if rabbitUrl != "" {
					clientTransport := ffSync.RabbitTransportClient{
						ClientName:   clientName,
						RabbitConfig: rabbitConfig,
					}
					err := clientTransport.Connect(idx)
					if err != nil {
						log.Fatalf("Failed to connect to RabbitMQ as client, %v", err)
					}
				}
				srv.Sorting.InitializeWithIndex(idx)
				srv.Sorting.StartListeningForChanges()

				wg.Add(1)
				go populateContentFromCsv(contentIdx, "data/content.csv", wg)

				// go func() {
				// 	idx.Lock()
				// 	for _, item := range idx.Items {
				// 		embeddingsIndex.AddDocument(embeddings.MakeDocument(*item))
				// 	}
				// 	idx.Unlock()
				// 	log.Printf("Embeddings index loaded")
				// }()
			}
		}

		done = true
	}()

}

func main() {

	wg := sync.WaitGroup{}
	LoadIndex(&wg)

	debugMux := http.NewServeMux()
	debugMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if !done {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("not ready"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	go func() {
		mux := http.NewServeMux()
		log.Println("Waiting for index to load...")
		wg.Wait()
		log.Println("Starting api")

		mux.Handle("/admin/", http.StripPrefix("/admin", srv.AdminHandler()))

		mux.Handle("/api/", http.StripPrefix("/api", srv.ClientHandler()))

		log.Printf("Starting server %v", listenAddress)
		log.Fatal(http.ListenAndServe(listenAddress, mux))

	}()

	debugMux.Handle("/metrics", promhttp.Handler())

	if enableProfiling != nil && *enableProfiling {
		log.Println("Profiling enabled")
		debugMux.HandleFunc("/debug/pprof/", httpprof.Index)
		debugMux.HandleFunc("/debug/pprof/cmdline", httpprof.Cmdline)
		debugMux.HandleFunc("/debug/pprof/profile", httpprof.Profile)
		debugMux.HandleFunc("/debug/pprof/symbol", httpprof.Symbol)
		debugMux.HandleFunc("/debug/pprof/trace", httpprof.Trace)
		//mux.Handle("/debug/pprof/", )
	}
	log.Printf("Starting debug server %v", debugAddress)
	log.Fatal(http.ListenAndServe(debugAddress, debugMux))
}
