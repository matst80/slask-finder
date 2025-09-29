package index

import (
	"iter"
	"sync"
	"time"

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
	Items *ItemCache
}

func NewItemIndex(storage types.StorageProvider, cachePath string) *ItemIndex {
	cache, err := NewItemCache(25*time.Hour, time.Minute, storage, cachePath)
	if err != nil {
		panic(err)
	}
	idx := &ItemIndex{
		mu:    sync.RWMutex{},
		Items: cache,
	}
	return idx
}

func (i *ItemIndex) HandleItem(item types.Item) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.handleItemUnsafe(item)
}

func (i *ItemIndex) HandleItems(it iter.Seq[types.Item]) {
	i.mu.Lock()
	defer i.mu.Unlock()
	for item := range it {
		i.handleItemUnsafe(item)
	}
}

func (i *ItemIndex) GetAllItems() iter.Seq[types.Item] {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return i.Items.AllItems()
}

func (i *ItemIndex) handleItemUnsafe(item types.Item) {
	id := item.GetId()

	if item.IsDeleted() {
		i.Items.Delete(id)
		go noDeletes.Inc()
		return
	}

	i.Items.Set(id, item)
	go noUpdates.Inc()
}

func (i *ItemIndex) GetItem(id uint) (types.Item, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.Items.Get(id)
}

func (i *ItemIndex) HasItem(id uint) bool {
	i.mu.RLock()
	defer i.mu.RUnlock()
	_, ok := i.Items.Get(id)
	return ok
}
