package index

import (
	"iter"
	"maps"
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

	i.handleItem(item)
}

func (i *ItemIndex) GetAllItems() iter.Seq[types.Item] {
	i.mu.RLock()
	defer i.mu.Unlock()

	return maps.Values(i.Items)
	// for _, item := range i.Items {
	// 	ret = append(ret, item)
	// }
	// return ret
}

func (i *ItemIndex) handleItem(item types.Item) {
	i.mu.Lock()
	defer i.mu.Unlock()
	id := item.GetId()

	if item.IsDeleted() {
		delete(i.Items, id)

		go noDeletes.Inc()
		return
	}

	i.Items[id] = item
	go noUpdates.Inc()
}

func (i *ItemIndex) GetItem(id uint) (types.Item, bool) {
	item, ok := i.Items[id]
	return item, ok
}

func (i *ItemIndex) HasItem(id uint) bool {
	_, ok := i.Items[id]
	return ok
}
