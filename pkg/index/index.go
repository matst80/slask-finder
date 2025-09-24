package index

import (
	"log"
	"sync"

	"github.com/matst80/slask-finder/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	noUpdates = promauto.NewCounter(prometheus.CounterOpts{
		Name: "slaskfinder_index_updates_total",
		Help: "The total number of item updates",
	})
	noDeletes = promauto.NewCounter(prometheus.CounterOpts{
		Name: "slaskfinder_index_deletes_total",
		Help: "The total number of item deletions",
	})
)

type ItemIndex struct {
	mu    sync.RWMutex
	Items map[uint]types.Item
}

func NewItemIndex() *ItemIndex {
	idx := &ItemIndex{
		mu:    sync.RWMutex{},
		Items: make(map[uint]types.Item, 350_000),
		//categories:   make(map[uint]*Category),
	}
	return idx
}

func (i *ItemIndex) HandleItem(item types.Item) {
	log.Printf("Handling item %d", item.GetId())
	i.mu.Lock()
	defer i.mu.Unlock()
	i.HandleItemUnsafe(item)
}

func (i *ItemIndex) HandleItems(items []types.Item) {
	l := len(items)
	if l == 0 {
		return
	}
	log.Printf("Handling items %d", l)
	i.mu.Lock()
	defer i.mu.Unlock()

	for _, it := range items {
		i.HandleItemUnsafe(it)

	}

	go noUpdates.Add(float64(l))
}

func (i *ItemIndex) Lock() {
	i.mu.RLock()
}

func (i *ItemIndex) Unlock() {
	i.mu.RUnlock()
}

func (i *ItemIndex) StartUnsafe() {
	i.mu.Lock()
}

func (i *ItemIndex) EndUnsafe() {
	i.mu.Unlock()
}

func (i *ItemIndex) HandleItemUnsafe(item types.Item) {

	id := item.GetId()

	if item.IsDeleted() {
		delete(i.Items, id)

		go noDeletes.Inc()
		return
	}

	i.Items[id] = item
	go noUpdates.Inc()
}

func (i *ItemIndex) HasItem(id uint) bool {
	_, ok := i.Items[id]
	return ok
}
