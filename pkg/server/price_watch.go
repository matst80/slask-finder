package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/matst80/slask-finder/pkg/types"
	"google.golang.org/api/option"
)

var (
	priceWatchesMutex sync.RWMutex
	priceWatchesFile  = "data/price_watches_v2.json"
)

// PushSubscription represents a Web Push API subscription
type PushSubscription struct {
	Token string `json:"token"`
}

// PriceWatch represents a price watch entry
type PriceWatch struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId,omitempty"`
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"createdAt"`
}

// PriceWatchRequest represents the incoming request
type PriceWatchRequest struct {
	Token string `json:"token"`
}

// PriceWatchesData represents the structure of the watches file
type PriceWatchesData struct {
	Watches []PriceWatch `json:"watches"`
}

func NewPriceWatcher() *PriceWatchesData {
	r := &PriceWatchesData{
		Watches: []PriceWatch{},
	}
	err := loadPriceWatches(r)
	if err != nil {
		log.Printf("Error loading price watches: %v", err)
	}
	return r
}

// WatchPriceChange handles HTTP requests for adding price watches
func (p *PriceWatchesData) WatchPriceChange(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, r, false, "0")

	// Get the item ID from path
	itemID := r.PathValue("id")
	if itemID == "" {
		http.Error(w, "Item ID is required", http.StatusBadRequest)
		return
	}

	// Parse the request body
	var watchRequest PriceWatchRequest
	err := json.NewDecoder(r.Body).Decode(&watchRequest)
	if err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate the subscription
	if watchRequest.Token == "" {
		http.Error(w, "Subscription token is required", http.StatusBadRequest)
		return
	}

	// Create new watch entry
	newWatch := PriceWatch{
		ID:        itemID,
		Token:     watchRequest.Token,
		CreatedAt: time.Now(),
	}

	// Add to watches (remove existing watch for same item if exists)
	watchIndex := -1
	for i, watch := range p.Watches {
		if watch.ID == itemID && watch.Token == watchRequest.Token {
			watchIndex = i
			break
		}
	}

	if watchIndex >= 0 {
		p.Watches[watchIndex] = newWatch
	} else {
		p.Watches = append(p.Watches, newWatch)
	}

	// Save watches
	err = p.savePriceWatches()
	if err != nil {
		log.Printf("Error saving price watches: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Send test push notification
	subscription := PushSubscription{Token: watchRequest.Token}
	err = sendTestPushNotification(subscription, itemID)
	if err != nil {
		log.Printf("Error sending test push notification: %v", err)
		// Don't fail the request if push notification fails
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Price watch added successfully",
		"itemId":  itemID,
	})
}

// loadPriceWatches loads the price watches from file
func loadPriceWatches(instance *PriceWatchesData) error {
	priceWatchesMutex.RLock()
	defer priceWatchesMutex.RUnlock()

	// Check if file exists
	if _, err := os.Stat(priceWatchesFile); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(priceWatchesFile)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, instance)
	if err != nil {
		return err
	}

	if instance.Watches == nil {
		instance.Watches = []PriceWatch{}
	}

	return nil
}

// savePriceWatches saves the price watches to file
func (p *PriceWatchesData) savePriceWatches() error {
	priceWatchesMutex.Lock()
	defer priceWatchesMutex.Unlock()

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(priceWatchesFile, data, 0644)
}

// sendFirebaseNotification sends a notification using the Firebase Admin SDK.
func sendFirebaseNotification(registrationToken string, notification *messaging.Notification, data map[string]string) error {
	// GOOGLE_APPLICATION_CREDENTIALS should be set in the environment.
	// Or you can pass option.WithCredentialsFile("path/to/serviceAccountKey.json")
	var app *firebase.App
	var err error

	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
		opt := option.WithCredentialsFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
		app, err = firebase.NewApp(context.Background(), nil, opt)
	} else {
		app, err = firebase.NewApp(context.Background(), nil)
	}

	if err != nil {
		log.Printf("error initializing app: %v\n", err)
		return err
	}

	ctx := context.Background()
	client, err := app.Messaging(ctx)
	if err != nil {
		log.Printf("error getting Messaging client: %v\n", err)
		return err
	}

	message := &messaging.Message{
		Notification: notification,
		Data:         data,
		Token:        registrationToken,
	}

	response, err := client.Send(ctx, message)
	if err != nil {
		log.Printf("error sending message: %v\n", err)
		return err
	}
	log.Printf("Successfully sent message: %s\n", response)

	return nil
}

// sendTestPushNotification sends a test push notification to verify the subscription works
func sendTestPushNotification(subscription PushSubscription, itemID string) error {
	// Extract registration token from FCM endpoint
	// FCM endpoint format: https://fcm.googleapis.com/fcm/send/{token}

	// Create FCM message payload
	notification := &messaging.Notification{
		Title: "Price Watch Activated",
		Body:  "You will be notified when the price of item " + itemID + " changes.",
	}
	data := map[string]string{
		"itemId": itemID,
		"type":   "test",
		"icon":   "/icon-192x192.png",
		"tag":    "price-watch-test",
	}

	return sendFirebaseNotification(subscription.Token, notification, data)
}

// NotifyPriceWatchers sends notifications to all watchers for a specific item
func (p *PriceWatchesData) NotifyPriceWatchers(item types.Item) {

	for _, watch := range p.Watches {
		if watch.ID == item.GetSku() {
			notification := &messaging.Notification{
				Title: fmt.Sprintf("Price Update for %s", item.GetTitle()),
				Body:  fmt.Sprintf("The price is now %.2f", float64(item.GetPrice())/100),
			}
			data := map[string]string{
				"itemId":   item.GetSku(),
				"type":     "price-update",
				"newPrice": fmt.Sprintf("%.2f", float64(item.GetPrice())/100),
			}

			// We can probably make this concurrent if we have many watches
			err := sendFirebaseNotification(watch.Token, notification, data)
			if err != nil {
				log.Printf("Failed to send price watch notification for item %s to token %s: %v", watch.ID, watch.Token, err)
			}
		}
	}
}
