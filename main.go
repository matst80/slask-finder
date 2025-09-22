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

	"github.com/matst80/slask-finder/pkg/embeddings"
	"github.com/matst80/slask-finder/pkg/storage"
	"github.com/matst80/slask-finder/pkg/tracking"
	"github.com/matst80/slask-finder/pkg/types"

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

// Ollama embeddings engine for semantic search capability
var embeddingsEngine types.EmbeddingsEngine = embeddings.NewOllamaEmbeddingsEngineWithMultipleEndpoints("elkjop-ecom", "http://10.10.11.135:11434/api/embeddings")

// Initialize simple index without handlers
var db = storage.NewPersistance()
var idx = index.NewIndex()

// var embeddingsIndex = embeddings.NewEmbeddingsIndex()
var contentIdx = index.NewContentIndex()

// Server will be set based on configuration (admin for master/standalone, client for clients)
var adminSrv *server.AdminWebServer
var clientSrv *server.ClientWebServer

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
	adminSrv.OAuthConfig = authConfig
	clientSrv.OAuthConfig = authConfig
}

func LoadIndex(wg *sync.WaitGroup) {
	wg.Add(1)
	log.Printf("amqp url: %s", rabbitUrl)
	log.Printf("clientName: %s", clientName)

	// itemHandlers := []types.ItemHandler{
	// 	idx,
	// }

	// Determine server type based on configuration
	if rabbitUrl != "" && clientName == "" {
		// Master/standalone mode - use AdminWebServer
		idx.IsMaster = true
		adminSrv = server.NewAdminWebServer(idx, db, contentIdx)
		log.Println("Starting with reduced memory consumption (admin mode)")
	} else {
		// Client mode - use ClientWebServer
		clientSrv = server.NewClientWebServer(idx, db, contentIdx)
		clientSrv.Cache = server.NewCache(redisUrl, redisPassword, 0)
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

		if adminSrv != nil {
			adminSrv.Tracking = trk
		} else {
			clientSrv.Tracking = trk
		}
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

				// Initialize appropriate server handlers after index is loaded
				if adminSrv != nil {
					// Admin mode - initialize all handlers
					handlerOpts := server.DefaultHandlerOptions(embeddingsEngine, func() error {
						return db.SaveIndex(idx)
					})
					handlerOpts.RedisAddr = redisUrl
					handlerOpts.RedisPassword = redisPassword
					handlerOpts.RedisDB = 0

					err = adminSrv.InitializeHandlers(handlerOpts)
					if err != nil {
						log.Fatalf("Failed to initialize admin handlers: %v", err)
					}

					// if adminSrv.SortingHandler != nil && adminSrv.SortingHandler.Sorting != nil {
					// 	adminSrv.SortingHandler.Sorting.InitializeWithIndex(idx)
					// 	adminSrv.SortingHandler.Sorting.StartListeningForChanges()
					// }
				} else {
					// Client mode - initialize minimal handlers
					clientOpts := server.DefaultClientHandlerOptions()
					clientOpts.RedisAddr = redisUrl
					clientOpts.RedisPassword = redisPassword
					clientOpts.RedisDB = 0

					err = clientSrv.InitializeClientHandlers(clientOpts)
					if err != nil {
						log.Fatalf("Failed to initialize client handlers: %v", err)
					}

					// if clientSrv.SortingHandler != nil && clientSrv.SortingHandler.Sorting != nil {
					// 	clientSrv.SortingHandler.Sorting.InitializeWithIndex(idx)
					// 	clientSrv.SortingHandler.Sorting.StartListeningForChanges()
					// }
				}

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

		if adminSrv != nil {
			// Admin server handles both admin and client endpoints
			mux.Handle("/admin/", http.StripPrefix("/admin", adminSrv.Handle()))
			//mux.Handle("/api/", http.StripPrefix("/api", adminSrv.Handle()))
		}
		if clientSrv != nil {
			// Client server only handles client endpoints
			mux.Handle("/api/", http.StripPrefix("/api", clientSrv.Handle()))
		}

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
