package sorting

import (
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
	name        string
	overrideKey string
	dirty       bool
	fn          func(item types.Item) float64
}

func NewBaseSorter(name string, fn func(item types.Item) float64) Sorter {
	return &BaseSorter{
		mu:          sync.RWMutex{},
		override:    types.SortOverride{},
		scores:      make(map[types.ItemId]float64),
		name:        name,
		dirty:       false,
		fn:          fn,
		overrideKey: name,
	}
}

func NewBaseSorterWithCustomOverrideKey(name string, fn func(item types.Item) float64, overrideKey string) Sorter {
	return &BaseSorter{
		mu:          sync.RWMutex{},
		override:    types.SortOverride{},
		scores:      make(map[types.ItemId]float64),
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
	s.scores[id] = newscore + s.override[uint32(id)]
	s.dirty = true
}

func (s *BaseSorter) GetSort() types.ByValue {
	s.mu.RLock()
	defer s.mu.RUnlock()
	l := len(s.scores)
	j := 0.0
	sortMap := make(types.ByValue, l)
	i := 0
	var id types.ItemId
	var score float64
	var o float64
	for id, score = range s.scores {
		j += 0.0000000000001
		o = s.override[uint32(id)]
		sortMap[i] = types.Lookup{Id: uint32(id), Value: score + o + j}
		i++
	}
	sortMap = sortMap[:i]
	SortByValues(sortMap)
	s.dirty = false
	return sortMap
}
