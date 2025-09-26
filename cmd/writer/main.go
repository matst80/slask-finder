package main

import (
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/matst80/slask-finder/pkg/facet"
	"github.com/matst80/slask-finder/pkg/storage"
	"github.com/matst80/slask-finder/pkg/types"
	amqp "github.com/rabbitmq/amqp091-go"
)

type MasterApp struct {
	mu            sync.RWMutex
	fieldData     map[string]*FieldData
	storageFacets []*facet.StorageFacet
	storage       *storage.DiskStorage
	// itemIndex       *index.ItemIndex
	// embeddingsIndex *embeddings.ItemEmbeddingsHandler
	amqpSender *AmqpSender
}

var country = "se"

func init() {
	c, ok := os.LookupEnv("COUNTRY")
	if ok {
		country = c
	}
}

func main() {

	diskStorage := storage.NewDiskStorage(country, "data")
	err := diskStorage.LoadJson(types.CurrentSettings, "settings.json")
	if err != nil {
		log.Printf("Could not load settings from file: %v", err)
	}

	// Entry point for the master command
	amqpUrl, ok := os.LookupEnv("RABBIT_HOST")
	if !ok {
		log.Fatal("RABBIT_HOST environment variable is not set")
	}
	conn, err := amqp.DialConfig(amqpUrl, amqp.Config{
		Properties: amqp.NewConnectionProperties(),
	})
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	// idx := index.NewItemIndex()
	// embeddingsEngine := embeddings.NewOllamaEmbeddingsEngine()
	// embeddingsIndex := embeddings.NewItemEmbeddingsHandler(embeddings.DefaultEmbeddingsHandlerOptions(embeddingsEngine), func() error {

	// 	log.Println("Embeddings queue processed")
	// 	//storage.SaveEmbeddings(embeddingsIndex.Embeddings, "data/embeddings-v2.jz")
	// 	return nil
	// })

	// err = diskStorage.LoadItems(idx, embeddingsIndex)
	// if err != nil {
	// 	log.Printf("Could not load items from file: %v", err)
	// }

	app := &MasterApp{
		mu:            sync.RWMutex{},
		fieldData:     map[string]*FieldData{},
		storageFacets: make([]*facet.StorageFacet, 0),
		amqpSender:    NewAmqpSender(country, conn),
		// itemIndex:       idx,
		// embeddingsIndex: embeddingsIndex,
		storage: diskStorage,
	}
	err = diskStorage.LoadGzippedJson(app.fieldData, "fields.jz")
	if err != nil {
		log.Printf("Could not load fields from file: %v", err)
	}
	err = diskStorage.LoadFacets(&app.storageFacets)
	if err != nil {
		log.Printf("Could not load facets from file: %v", err)
	}
	srv := http.NewServeMux()

	srv.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	var auth AuthHandler = nil
	auth, err = NewGoogleAuth()
	if err != nil {
		log.Printf("Failed to initialize auth: %v", err)
		auth = &MockAuth{}
	}
	srv.HandleFunc("GET /admin/login", auth.Login)
	srv.HandleFunc("GET /admin/logout", auth.Logout)
	srv.HandleFunc("GET /admin/user", auth.User)

	srv.HandleFunc("admin/auth_callback", auth.AuthCallback)

	srv.HandleFunc("POST /admin/add", auth.Middleware(app.handleItems))
	srv.HandleFunc("/admin/save", auth.Middleware(app.saveItems))
	//srv.HandleFunc("GET /admin/item/{id}", auth.Middleware(app.getAdminItemById))
	srv.HandleFunc("GET /admin/settings", auth.Middleware(app.GetSettings))
	srv.HandleFunc("PUT /admin/settings", auth.Middleware(app.UpdateSettings))
	srv.HandleFunc("GET /admin/fields", auth.Middleware(app.GetFields))
	srv.HandleFunc("POST /admin/update-fields", auth.Middleware(app.UpdateFacetsFromFields))
	srv.HandleFunc("PUT /fields", auth.Middleware(app.HandleUpdateFields))

	// srv.HandleFunc("GET /users", auth.Middleware(webAuth.ListUsers))
	//    srv.HandleFunc("DELETE /users/{id}", auth.Middleware(webAuth.DeleteUser))
	//    srv.HandleFunc("PUT /users/{id}", auth.Middleware(webAuth.UpdateUser))

	/*


	   //srv.HandleFunc("PUT /key-values", app.Middleware(app.UpdateCategories))

	   srv.HandleFunc("/store-embeddings", app.Middleware(app.SaveEmbeddings))
	   srv.HandleFunc("PUT /fields", app.Middleware(app.HandleUpdateFields))
	   srv.HandleFunc("/clean-fields", app.CleanFields)
	   srv.HandleFunc("/update-fields", app.UpdateFacetsFromFields)
	   srv.HandleFunc("DELETE /facets/{id}", app.Middleware(app.DeleteFacet))
	   srv.HandleFunc("GET /facets", app.GetFacetList)
	   srv.HandleFunc("PUT /facets/{id}", app.Middleware(app.UpdateFacet))
	   srv.HandleFunc("GET /index/facets", app.Middleware(app.GetSearchIndexedFacets))
	   srv.HandleFunc("POST /index/facets", app.Middleware(app.SetSearchIndexedFacets))
	   srv.HandleFunc("GET /item/{id}/popularity", app.Middleware(app.GetItemPopularity))
	   srv.HandleFunc("GET /fields/{id}/add", app.Middleware(app.CreateFacetFromField))
	   srv.HandleFunc("GET /fields", app.GetFields)

	   srv.HandleFunc("PUT /facet-group", app.Middleware(app.FacetGroupUpdate))
	   srv.HandleFunc("/words", app.Middleware(app.HandleWordReplacements))

	   srv.HandleFunc("POST /price-watch/{id}", priceWatcher.WatchPriceChange)

	   srv.HandleFunc("GET /missing-fields", app.Middleware(app.MissingFacets))
	   srv.HandleFunc("GET /fields/{id}", app.GetField)
	   srv.HandleFunc("/rules/popular", app.Middleware(app.HandlePopularRules))
	   srv.HandleFunc("/sort/popular", app.Middleware(app.HandlePopularOverride))
	   srv.HandleFunc("POST /relation-groups", app.SaveHandleRelationGroups)
	   srv.HandleFunc("/facet-groups", app.HandleFacetGroups)

	   srv.HandleFunc("GET /users", app.Middleware(auth.ListUsers))
	   srv.HandleFunc("DELETE /users/{id}", app.Middleware(auth.DeleteUser))
	   srv.HandleFunc("PUT /users/{id}", app.Middleware(auth.UpdateUser))

	   srv.HandleFunc("GET /webauthn/register/start", auth.CreateChallenge)
	   srv.HandleFunc("POST /webauthn/register/finish", auth.ValidateCreateChallengeResponse)
	   srv.HandleFunc("GET /webauthn/login/start", auth.LoginChallenge)
	   srv.HandleFunc("POST /webauthn/login/finish", auth.LoginChallengeResponse)
	*/
	http.ListenAndServe(":8080", srv)
}
