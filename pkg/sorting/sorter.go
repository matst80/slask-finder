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
	override    SortOverride
	scores      map[uint]float64
	name        string
	overrideKey string
	dirty       bool
	fn          func(item types.Item) float64
}

func NewBaseSorter(name string, fn func(item types.Item) float64) Sorter {
	return &BaseSorter{
		mu:          sync.RWMutex{},
		override:    SortOverride{},
		scores:      make(map[uint]float64),
		name:        name,
		dirty:       false,
		fn:          fn,
		overrideKey: name,
	}
}

func NewBaseSorterWithCustomOverrideKey(name string, fn func(item types.Item) float64, overrideKey string) Sorter {
	return &BaseSorter{
		mu:          sync.RWMutex{},
		override:    SortOverride{},
		scores:      make(map[uint]float64),
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
	s.scores[id] = s.fn(item)
	s.dirty = true
}

func (s *BaseSorter) GetSort() types.ByValue {
	s.mu.RLock()
	defer s.mu.RUnlock()
	l := len(s.scores)
	j := 0.0
	sortMap := make(types.ByValue, l)
	i := 0
	var id uint
	var score float64
	for id, score = range s.scores {
		j += 0.0000000000001
		sortMap[i] = types.Lookup{Id: id, Value: score + s.override[id] + j}
		i++
	}
	sortMap = sortMap[:i]
	SortByValues(sortMap)
	s.dirty = false
	return sortMap
}
