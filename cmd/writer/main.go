package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/matst80/slask-finder/pkg/common"
	"github.com/matst80/slask-finder/pkg/facet"
	"github.com/matst80/slask-finder/pkg/storage"
	"github.com/matst80/slask-finder/pkg/types"
	amqp "github.com/rabbitmq/amqp091-go"
)

type app struct {
	mu            sync.RWMutex
	fieldData     map[string]FieldData
	storageFacets []facet.StorageFacet
	storage       *storage.DiskStorage
	amqpSender    *AmqpSender
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

	app := &app{
		mu:            sync.RWMutex{},
		fieldData:     make(map[string]FieldData),
		storageFacets: make([]facet.StorageFacet, 3000),
		amqpSender:    NewAmqpSender(country, conn),
		// itemIndex:       idx,
		// embeddingsIndex: embeddingsIndex,
		storage: diskStorage,
	}
	// Load stored field metadata (map must be passed by pointer for decoder)
	if err = diskStorage.LoadGzippedJson(&app.fieldData, "fields.jz"); err != nil {
		if os.IsNotExist(err) {
			log.Printf("fields.jz not found, starting with empty field map")
		} else {
			log.Printf("Could not load fields from file: %v", err)
		}
	}
	err = diskStorage.LoadFacets(&app.storageFacets)
	if err != nil {
		log.Printf("Could not load facets from file: %v", err)
	}
	srv := http.NewServeMux()

	srv.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("ok")); err != nil {
			log.Printf("Failed to write health check response: %v", err)
		}
	})
	var auth AuthHandler
	auth, err = NewGoogleAuth()
	if err != nil {
		log.Printf("Failed to initialize auth: %v", err)
		auth = &MockAuth{}
	}
	srv.HandleFunc("GET /admin/login", auth.Login)
	srv.HandleFunc("GET /admin/logout", auth.Logout)
	srv.HandleFunc("GET /admin/user", auth.User)

	srv.HandleFunc("admin/auth_callback", auth.AuthCallback)

	srv.HandleFunc("POST /admin/add", auth.Middleware(app.dummyResponse))
	srv.HandleFunc("/admin/save", auth.Middleware(app.dummyResponse))
	//srv.HandleFunc("GET /admin/item/{id}", auth.Middleware(app.getAdminItemById))
	srv.HandleFunc("GET /admin/facets", app.GetFacetList)
	srv.HandleFunc("DELETE /admin/facets/{id}", auth.Middleware(app.DeleteFacet))
	srv.HandleFunc("PUT /admin/facets/{id}", auth.Middleware(app.UpdateFacet))
	srv.HandleFunc("GET /admin/settings", auth.Middleware(app.GetSettings))
	srv.HandleFunc("PUT /admin/settings", auth.Middleware(app.UpdateSettings))
	srv.HandleFunc("GET /admin/fields", auth.Middleware(app.GetFields))
	srv.HandleFunc("PUT /admin/fields", auth.Middleware(app.HandleUpdateFields))

	srv.HandleFunc("GET /admin/fields/{id}/add", auth.Middleware(app.CreateFacetFromField))
	srv.HandleFunc("GET /admin/missing-fields", auth.Middleware(app.MissingFacets))
	srv.HandleFunc("POST /admin/update-fields", auth.Middleware(app.UpdateFacetsFromFields))

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
	server := &http.Server{Addr: ":8080", Handler: srv, ReadHeaderTimeout: 5 * time.Second}

	saveHook := func(ctx context.Context) error {
		log.Println("saving facets before shutdown")
		return app.storage.SaveFacets(app.storageFacets)
	}

	common.RunServerWithShutdown(server, "writer server", 15*time.Second, 5*time.Second, saveHook)
}
