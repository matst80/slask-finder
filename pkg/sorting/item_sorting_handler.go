package sorting

import (
	"iter"
	"sync"
	"time"

	"github.com/matst80/slask-finder/pkg/types"
)

type SortingProperty struct {
	Name   string
	values types.ByValue
	scores map[uint]float64
}

type Sorter interface {
	ProcessItem(item types.Item)
	GetSort() types.ByValue
}

type PopularitySorter struct {
	mu        sync.RWMutex
	overrides *SortOverride
	scores    map[uint]float64
}

func NewPopularitySorter() Sorter {
	return &PopularitySorter{
		mu:        sync.RWMutex{},
		overrides: &SortOverride{},
		scores:    make(map[uint]float64),
	}
}

func (s *PopularitySorter) ProcessItem(item types.Item) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := item.GetId()
	if item.IsDeleted() {
		delete(s.scores, id)
		return
	}
	s.scores[id] = types.CollectPopularity(item, *types.CurrentSettings.PopularityRules...)
}

func (s *PopularitySorter) GetSort() types.ByValue {
	s.mu.RLock()
	defer s.mu.RUnlock()
	l := len(s.scores)
	j := 0.0
	popularMap := make(types.ByValue, l)
	i := 0
	var id uint
	var popular float64
	for id, popular = range s.scores {
		j += 0.0000000000001
		popularMap[i] = types.Lookup{Id: id, Value: popular + (*s.overrides)[id] + j}
		i++
	}
	popularMap = popularMap[:i]
	SortByValues(popularMap)
	return popularMap
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

type PriceSorter struct {
	mu     sync.RWMutex
	scores map[uint]float64
}

func NewPriceSorter() Sorter {
	return &PriceSorter{
		mu:     sync.RWMutex{},
		scores: make(map[uint]float64),
	}
}

func (s *PriceSorter) ProcessItem(item types.Item) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := item.GetId()
	if item.IsDeleted() {
		delete(s.scores, id)
		return
	}
	price := item.GetPrice()
	if price > 0 && price <= 1000000000 {
		s.scores[id] = float64(price)
	}
}

func (s *PriceSorter) GetSort() types.ByValue {

	l := len(s.scores)
	j := 0.0

	popularMap := make(types.ByValue, l)
	i := 0
	var id uint
	var price float64
	s.mu.RLock()
	for id, price = range s.scores {
		j += 0.0000000000001
		popularMap[i] = types.Lookup{Id: id, Value: price + j}
		i++
	}
	s.mu.RUnlock()
	popularMap = popularMap[:i]
	SortByValues(popularMap)
	return popularMap
}

func NewLastUpdateSorter() Sorter {
	return &LastUpdateSorter{
		mu:     sync.RWMutex{},
		scores: make(map[uint]float64),
	}
}

func (s *LastUpdateSorter) GetSort() types.ByValue {
	s.mu.RLock()
	defer s.mu.RUnlock()
	l := len(s.scores)
	j := 0.0
	now := time.Now()
	ts := now.UnixMilli()
	updatedMap := make(types.ByValue, l)
	i := 0
	var id uint
	var lastUpdated float64
	for id, lastUpdated = range s.scores {
		j += 0.0000000000001
		updatedMap[i] = types.Lookup{Id: id, Value: float64(ts) - lastUpdated + j}
		i++
	}
	updatedMap = updatedMap[:i]
	SortByValues(updatedMap)
	return updatedMap
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

// Delegation methods for backward compatibility
func (h *SortingItemHandler) GetSort(id string) types.ByValue {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if r, ok := h.sortValues[id]; ok {
		return r
	}
	return nil
}
