package index

import (
	"sync"

	"github.com/matst80/slask-finder/pkg/types"
)

type SortingItemHandler struct {
	mu               sync.RWMutex
	sessionMu        sync.RWMutex
	popularOverrides *SortOverride
	popularMap       *SortOverride
	sessionOverrides map[uint]*SortOverride
	sortMethods      map[string]*types.ByValue
	ItemScores       map[uint]float64
	ItemPrices map[uint]float64
	rules            []types.ItemPopularityRule
}

func NewSortingItemHandler() *SortingItemHandler {
	handler := &SortingItemHandler{
		mu:               sync.RWMutex{},
		sessionMu:        sync.RWMutex{},
		popularOverrides: &SortOverride{},
		popularMap:       &SortOverride{},
		sessionOverrides: make(map[uint]*SortOverride),
		sortMethods:      make(map[string]*types.ByValue),
		ItemScores:       make(map[uint]float64),
		rules:            types.CurrentSettings.PopularityRules,
	}

	return handler
}

// ItemHandler interface implementation
func (h *SortingItemHandler) HandleItem(item types.Item) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.HandleItemUnsafe(item)
}

func (h *SortingItemHandler) HandleItems(items []types.Item) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, item := range items {
		h.HandleItemUnsafe(item)
	}
}

func (h *SortingItemHandler) HandleItemUnsafe(item types.Item) {
	// Sorting handler doesn't need to process individual items during upsert
	// Sorting happens on demand or periodically
	id := item.GetId()
	if item.IsDeleted() {
		delete(h.ItemScores, id)
		delete(h.ItemPrices, id)
		delete(h.ItemUpdated, id)
		delete(h.ItemCreated, id)
		return
	}
	price := item.GetPrice()
	lastUpdate := item.GetLastUpdated()
	created := item.GetCreated()
	h.ItemScores[id] = types.CollectPopularity(item, h.rules...)
	if price > 0 && <= 1000000000 {
		h.ItemPrices[id] = price
	}
	if lastUpdate >0 {
		h.ItemUpdated[id] = lastUpdate
	}
	if created >0 {
		h.ItemCreated[id] = created
	}
}

func (h *SortingItemHandler) Lock() {
	h.mu.Lock()
}

func (h *SortingItemHandler) Unlock() {
	h.mu.Unlock()
}

func (h *SortingItemHandler) UpdateSorting() {
	
	overrides := *s.popularOverrides

	l := len(s.ItemScores)
	j := 0.0
	now := time.Now()
	ts := now.UnixMilli()
	popularMap := make(types.ByValue, l)
	priceMap := make(types.ByValue, l)
	updatedMap := make(types.ByValue, l)
	createdMap := make(types.ByValue, l)
	popularSearchMap := make(SortOverride)
	i := 0
	var itemScore float64

	var id uint
	var popular float64
	var partPopular float64

	for id, itemScore = range s.ItemScores {

		
		j += 0.0000000000001

		popular = itemScore + (overrides[id] * 30)

		partPopular = popular / 10000.0
		if item.GetLastUpdated() == 0 {
			updatedMap[i] = types.Lookup{Id: id, Value: j}
		} else {
			updatedMap[i] = types.Lookup{Id: id, Value: float64(ts-item.GetLastUpdated()/1000) + j}
		}
		if item.GetCreated() == 0 {
			createdMap[i] = types.Lookup{Id: id, Value: partPopular + j}
		} else {
			createdMap[i] = types.Lookup{Id: id, Value: partPopular + float64(ts-item.GetCreated()/1000) + j}
		}

		priceMap[i] = types.Lookup{Id: id, Value: float64(item.GetPrice()) + j}
		popularMap[i] = types.Lookup{Id: id, Value: popular + j}
		popularSearchMap[id] = popular / 100.0
		i++
	}

	// if s.idx != nil {
	// 	s.idx.SetBaseSortMap(popularSearchMap)
	// }
	go func() {
		popularMap = popularMap[:i]
		priceMap = priceMap[:i]
		updatedMap = updatedMap[:i]
		createdMap = createdMap[:i]
		s.muOverride.Lock()
		defer s.muOverride.Unlock()
		s.mu.Lock()
		defer s.mu.Unlock()
		s.popularMap = &popularSearchMap
		SortByValues(popularMap)
		s.sortMethods[POPULAR_SORT] = &popularMap
		SortByValues(priceMap)
		s.sortMethods[PRICE_DESC_SORT] = &priceMap
		s.sortMethods[PRICE_SORT] = cloneReversed(&priceMap)
		SortByValues(updatedMap)
		slices.Reverse(updatedMap)
		//s.sortMethods[UPDATED_DESC_SORT] = &updatedMap
		s.sortMethods[UPDATED_SORT] = &updatedMap
		SortByValues(createdMap)
		s.sortMethods[CREATED_SORT] = &createdMap
		//s.sortMethods[CREATED_DESC_SORT] = cloneReversed(&createdMap)
	}()

}
}


// Delegation methods for backward compatibility
func (h *SortingItemHandler) GetSort(id string) *types.ByValue {
	if h.Sorting != nil {
		return h.Sorting.GetSort(id)
	}
	return nil
}

func (h *SortingItemHandler) GetSortedFields(items []*JsonFacet) []*JsonFacet {
	if h.Sorting != nil {
		return h.Sorting.GetSortedFields(items)
	}
	return items
}
