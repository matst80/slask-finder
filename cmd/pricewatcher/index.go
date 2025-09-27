package main

import (
	"sync"

	"github.com/matst80/slask-finder/pkg/index"
)

type ItemWatcher struct {
	mu      sync.RWMutex
	Items   map[uint]int `json:"items"`
	watcher PriceWatchesData
}

func (app *ItemWatcher) HandleItems(items []index.DataItem) {
	app.mu.Lock()
	defer app.mu.Unlock()
	for _, item := range items {
		id := item.GetId()
		itemPrice := item.GetPrice()
		if itemPrice <= 0 {
			continue
		}

		existingPrice, ok := app.Items[id]
		if ok {
			if existingPrice == itemPrice {
				continue
			}
			if existingPrice > itemPrice {
				app.watcher.NotifyPriceWatchers(&item)
			}
		}

		app.Items[id] = itemPrice

	}
}
