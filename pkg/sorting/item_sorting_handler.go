package sorting

import (
	"encoding/json"
	"iter"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/matst80/slask-finder/pkg/messaging"
	"github.com/matst80/slask-finder/pkg/types"

	amqp "github.com/rabbitmq/amqp091-go"
)

func NewPopularitySorter() Sorter {
	return NewBaseSorter("popular", func(item types.Item) float64 {
		return types.CollectPopularity(item, *types.CurrentSettings.PopularityRules...)
	})
}

func NewPriceSorter() Sorter {
	return NewBaseSorter("price", func(item types.Item) float64 {
		price := item.GetPrice()
		if price > 0 && price <= 1000000000 {
			return float64(price)
		}
		return 0
	})
}

func NewLastUpdateSorter() Sorter {
	return NewBaseSorter("updated", func(item types.Item) float64 {
		lastUpdated := item.GetLastUpdated()
		if lastUpdated > 0 {
			return float64(lastUpdated)
		}
		return 0
	})
}

type SortingItemHandler struct {
	mu         sync.RWMutex
	country    string
	overrides  map[string]SortOverride
	Sorters    []Sorter
	sortValues map[string]types.ByValue
}

func NewSortingItemHandler() *SortingItemHandler {
	handler := &SortingItemHandler{
		mu:         sync.RWMutex{},
		overrides:  make(map[string]SortOverride),
		sortValues: make(map[string]types.ByValue, 3),
		Sorters: []Sorter{
			NewPopularitySorter(),
			NewLastUpdateSorter(),
			NewPriceSorter(),
		},
	}
	ticker := time.NewTicker(time.Second * 10)
	go func() {
		for range ticker.C {
			handler.UpdateSorts()
		}
	}()
	return handler
}

func (h *SortingItemHandler) Connect(conn *amqp.Connection, country string) {
	h.country = country
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	err = messaging.ListenToTopic(ch, "global", "sort_override", func(d amqp.Delivery) error {
		var item types.SortOverrideUpdate
		if err := json.Unmarshal(d.Body, &item); err == nil {
			log.Printf("Got sort override")
			h.HandleSortOverrideUpdate(item)
		} else {
			log.Printf("Failed to unmarshal facet change message %v", err)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to listen to facet_change topic: %v", err)
	}
}

func (h *SortingItemHandler) HandleSortOverrideUpdate(item types.SortOverrideUpdate) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if strings.Contains(item.Key, "session-") {
		// Session specific overrides are ignored for now
		return
	}
	h.overrides[item.Key] = item.Data
	log.Printf("Applied sort override: %s", item.Key)
	for _, s := range h.Sorters {
		s.HandleOverride(item)
	}
}

func (h *SortingItemHandler) HandleItems(it iter.Seq[types.Item]) {
	for item := range it {
		h.handleItemUnsafe(item)
	}
}

func (h *SortingItemHandler) HandleItem(item types.Item, wg *sync.WaitGroup) {
	wg.Go(func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.handleItemUnsafe(item)
	})
}

func (h *SortingItemHandler) handleItemUnsafe(item types.Item) {
	for _, s := range h.Sorters {
		go s.ProcessItem(item)
	}
}

func (h *SortingItemHandler) updateSorter(s Sorter) {

	if !s.IsDirty() {
		return
	}

	name := s.Name()
	sort := s.GetSort()

	if len(sort) > 0 {
		h.mu.Lock()
		h.sortValues[name] = sort
		h.mu.Unlock()

		log.Printf("Updated sort: %s, items: %d", name, len(sort))
	}
}

func (h *SortingItemHandler) UpdateSorts() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, s := range h.Sorters {
		go h.updateSorter(s)
	}
}

// Delegation methods for backward compatibility
func (h *SortingItemHandler) GetSort(id string) types.ByValue {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if r, ok := h.sortValues[id]; ok {
		return r
	}
	return nil
}

func (s *SortingItemHandler) GetSortedItemsIterator(sessionId int, sort string, items types.ItemList, start int) iter.Seq[uint] {
	precalculated := s.GetSort(sort)
	c := 0
	return func(yield func(uint) bool) {
		for _, v := range precalculated {
			if _, ok := items[v.Id]; !ok {
				continue
			}
			if c < start {
				c++
				continue
			}

			if !yield(v.Id) {
				break
			}
		}

	}

}
