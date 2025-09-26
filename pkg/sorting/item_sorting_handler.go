package sorting

import (
	"iter"
	"log"
	"sync"
	"time"

	"github.com/matst80/slask-finder/pkg/types"
)

func NewPopularitySorter() Sorter {
	return NewBaseSorter("popular", func(item types.Item) float64 {
		return types.CollectPopularity(item, *types.CurrentSettings.PopularityRules...)
	})
}

type LastUpdateSorter struct {
	mu     sync.RWMutex
	scores map[uint]float64
}

func (s *LastUpdateSorter) ProcessItem(item types.Item) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := item.GetId()
	if item.IsDeleted() {
		delete(s.scores, id)
		return
	}
	lastUpdated := item.GetLastUpdated()
	if lastUpdated > 0 {
		s.scores[id] = float64(lastUpdated)
	}
}

func NewPriceSorter() Sorter {
	return NewBaseSorter("price_asc", func(item types.Item) float64 {
		price := item.GetPrice()
		if price > 0 && price <= 1000000000 {
			return float64(price)
		}
		return 0
	})
}

func NewLastUpdateSorter() Sorter {
	return NewBaseSorter("last_updated", func(item types.Item) float64 {
		lastUpdated := item.GetLastUpdated()
		if lastUpdated > 0 {
			return float64(lastUpdated)
		}
		return 0
	})
}

type SortingItemHandler struct {
	mu         sync.RWMutex
	Sorters    []Sorter
	sortValues map[string]types.ByValue
}

func NewSortingItemHandler() *SortingItemHandler {
	handler := &SortingItemHandler{
		mu:         sync.RWMutex{},
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

// ItemHandler interface implementation
func (h *SortingItemHandler) HandleItem(item types.Item) {
	h.HandleItemUnsafe(item)
}

func (h *SortingItemHandler) HandleItems(it iter.Seq[types.Item]) {
	for item := range it {
		h.HandleItemUnsafe(item)
	}
}

func (h *SortingItemHandler) HandleItemUnsafe(item types.Item) {
	for _, s := range h.Sorters {
		go s.ProcessItem(item)
	}
}

func (h *SortingItemHandler) Lock() {

}

func (h *SortingItemHandler) Unlock() {

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
