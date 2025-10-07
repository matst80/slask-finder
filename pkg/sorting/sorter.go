package sorting

import (
	"slices"
	"sync"

	"github.com/matst80/slask-finder/pkg/types"
)

type Sorter interface {
	ProcessItem(item types.Item)
	GetSort() types.ByValue
	IsDirty() bool
	Name() string
	HandleOverride(types.SortOverrideUpdate)
}

type BaseSorter struct {
	mu          sync.RWMutex
	override    types.SortOverride
	scores      map[types.ItemId]float64
	isReversed  bool
	name        string
	overrideKey string
	dirty       bool
	fn          func(item types.Item) float64
}

func NewBaseSorter(name string, fn func(item types.Item) float64, isReversed bool) Sorter {
	return &BaseSorter{
		mu:          sync.RWMutex{},
		override:    types.SortOverride{},
		scores:      make(map[types.ItemId]float64),
		isReversed:  isReversed,
		name:        name,
		dirty:       false,
		fn:          fn,
		overrideKey: name,
	}
}

func NewBaseSorterWithCustomOverrideKey(name string, fn func(item types.Item) float64, isReversed bool, overrideKey string) Sorter {
	return &BaseSorter{
		mu:          sync.RWMutex{},
		override:    types.SortOverride{},
		scores:      make(map[types.ItemId]float64),
		isReversed:  isReversed,
		name:        name,
		dirty:       false,
		fn:          fn,
		overrideKey: overrideKey,
	}
}

func (s *BaseSorter) HandleOverride(update types.SortOverrideUpdate) {
	if update.Key == s.overrideKey {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.override = update.Data
		s.dirty = true
	}
}

func (s *BaseSorter) IsDirty() bool {
	return s.dirty
}

func (s *BaseSorter) Name() string {
	return s.name
}

func (s *BaseSorter) ProcessItem(item types.Item) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := item.GetId()
	if item.IsDeleted() {
		delete(s.scores, id)
		return
	}
	newscore := s.fn(item)
	if current, ok := s.scores[id]; ok && current == newscore {
		return
	}
	s.scores[id] = newscore // + s.override[uint32(id)]
	s.dirty = true
}

func (s *BaseSorter) GetSort() types.ByValue {
	// Acquire read lock to snapshot scores & overrides quickly.
	s.mu.RLock()
	l := len(s.scores)
	sortMap := make(types.ByValue, 0, l)
	for id, score := range s.scores {
		ov := s.override[uint32(id)]
		sortMap = append(sortMap, types.Lookup{Id: uint32(id), Value: score + ov})
	}
	isReversed := s.isReversed
	s.mu.RUnlock() // release before the (potentially) more expensive sort

	if !isReversed {
		// Descending by Value, tie-break by Id (stable deterministic)
		slices.SortFunc(sortMap, func(a, b types.Lookup) int {
			if a.Value > b.Value {
				return -1
			}
			if a.Value < b.Value {
				return 1
			}
			if a.Id < b.Id {
				return -1
			}
			if a.Id > b.Id {
				return 1
			}
			return 0
		})
	} else {
		// Ascending by Value, tie-break by Id
		slices.SortFunc(sortMap, func(a, b types.Lookup) int {
			if a.Value < b.Value {
				return -1
			}
			if a.Value > b.Value {
				return 1
			}
			if a.Id < b.Id {
				return -1
			}
			if a.Id > b.Id {
				return 1
			}
			return 0
		})
	}

	// Mark clean under write lock.
	s.mu.Lock()
	s.dirty = false
	s.mu.Unlock()

	return sortMap
}
