package main

import (
	"flag"
	"fmt"
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
var contentFile = os.Getenv("CONTENT_FILE")
var topicPrefix = os.Getenv("TOPIC_PREFIX")

var rabbitConfig = ffSync.RabbitConfig{
	//ItemChangedTopic: "item_changed",
	FieldChangeTopic:   fmt.Sprintf("%sfield_change", topicPrefix),
	ItemsUpsertedTopic: fmt.Sprintf("%sitems_added", topicPrefix),
	ItemDeletedTopic:   fmt.Sprintf("%sitem_deleted", topicPrefix),
	PriceLoweredTopic:  fmt.Sprintf("%sprice_lowered", topicPrefix),
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

	clientId := os.Getenv("GOOGLE_CLIENT_ID")
	if clientId == "" {
		log.Printf("GOOGLE_CLIENT_ID not provided")
		clientId = "1017700364201-hiv4l9c41osmqfkv17ju7gg08e570lfr.apps.googleusercontent.com"
	}

	authConfig := &oauth2.Config{
		ClientID:     clientId,
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
			err = pprof.StartCPUProfile(f)
			if err != nil {
				log.Fatal(err)
			}
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
				if contentFile != "" {
					wg.Add(1)
					go populateContentFromCsv(contentIdx, "data/content.csv", wg)
				}

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
	err := db.LoadSettings()
	if err != nil {
		log.Fatalf("Failed to load settings %v", err)
	}
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
