package common

import (
	"iter"
	"slices"
	"sync"
	"time"
)

// QueueProcessor is a function that processes a batch of items from the queue.
type QueueProcessor[V any] func(items []V)

// QueueHandler is a generic queue handler that processes items in the background.
type QueueHandler[V any] struct {
	mu        sync.Mutex
	queue     []V
	processor QueueProcessor[V]
	chunkSize int
	done      chan struct{}
}

// NewQueueHandler creates a new QueueHandler.
func NewQueueHandler[V any](processor QueueProcessor[V], chunkSize int) *QueueHandler[V] {
	q := &QueueHandler[V]{
		queue:     make([]V, 0),
		processor: processor,
		chunkSize: chunkSize,
		done:      make(chan struct{}),
	}
	go q.processQueue()
	return q
}

// Add adds an item to the queue.
func (h *QueueHandler[V]) Add(item ...V) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.queue = append(h.queue, item...)
}

func (h *QueueHandler[V]) AddIter(item iter.Seq[V]) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.queue = append(h.queue, slices.Collect(item)...)
}

func (h *QueueHandler[V]) processQueue() {
	for {

		h.mu.Lock()
		if len(h.queue) == 0 {
			h.mu.Unlock()
			time.Sleep(time.Second)
			continue
		}

		items := h.queue[:min(h.chunkSize, len(h.queue))]

		h.queue = h.queue[len(items):]
		h.mu.Unlock()

		h.processor(items)

	}
}
