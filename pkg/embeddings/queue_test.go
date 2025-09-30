package embeddings

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/types"
	"github.com/stretchr/testify/assert"
)

// MockEmbeddingsEngine is a mock implementation of the EmbeddingsEngine interface for testing.
type MockEmbeddingsEngine struct {
	GenerateEmbeddingsFunc func(text string) (types.Embeddings, error)
}

func (m *MockEmbeddingsEngine) GenerateEmbeddings(text string) (types.Embeddings, error) {
	if m.GenerateEmbeddingsFunc != nil {
		// Simulate work
		time.Sleep(5 * time.Millisecond)
		return m.GenerateEmbeddingsFunc(text)
	}
	// Default behavior: return a dummy embedding
	return types.Embeddings{}, nil
}

func TestEmbeddingsQueue_IdleHandling(t *testing.T) {
	mockEngine := &MockEmbeddingsEngine{}

	storedEmbeddings := make(map[uint]types.Embeddings)
	var storeMutex sync.Mutex
	storeFunc := func(id uint, embeddings types.Embeddings) {
		storeMutex.Lock()
		storedEmbeddings[id] = embeddings
		storeMutex.Unlock()
	}

	var doneFuncCalled int
	var doneMutex sync.Mutex
	var wg sync.WaitGroup

	doneFunc := func() error {
		doneMutex.Lock()
		doneFuncCalled++
		doneMutex.Unlock()
		wg.Done()
		return nil
	}

	queue := NewEmbeddingsQueue(mockEngine, storeFunc, doneFunc, 2, 100)
	queue.Start()
	defer queue.Stop()

	// First batch
	wg.Add(1)
	for i := range 5 {
		queue.QueueItem(&index.StorageDataItem{BaseItem: &index.BaseItem{Id: uint(i)}})
	}
	wg.Wait()

	doneMutex.Lock()
	assert.Equal(t, 1, doneFuncCalled, "idle callback once after first batch")
	doneMutex.Unlock()

	storeMutex.Lock()
	assert.Len(t, storedEmbeddings, 5)
	storeMutex.Unlock()

	// Second batch
	wg.Add(1)
	for i := 5; i < 10; i++ {
		queue.QueueItem(&index.StorageDataItem{BaseItem: &index.BaseItem{Id: uint(i)}})
	}
	wg.Wait()

	doneMutex.Lock()
	assert.Equal(t, 2, doneFuncCalled, "idle callback twice after two batches")
	doneMutex.Unlock()

	storeMutex.Lock()
	assert.Len(t, storedEmbeddings, 10)
	storeMutex.Unlock()
}

func TestEmbeddingsQueue_StopWithRemainingItems(t *testing.T) {
	mockEngine := &MockEmbeddingsEngine{}
	storedEmbeddings := make(map[uint]types.Embeddings)
	var storeMutex sync.Mutex
	storeFunc := func(id uint, embeddings types.Embeddings) {
		storeMutex.Lock()
		storedEmbeddings[id] = embeddings
		storeMutex.Unlock()
	}

	var doneCalls int
	var doneMu sync.Mutex
	var wg sync.WaitGroup
	doneFunc := func() error {
		doneMu.Lock()
		doneCalls++
		doneMu.Unlock()
		wg.Done()
		return nil
	}

	queue := NewEmbeddingsQueue(mockEngine, storeFunc, doneFunc, 1, 100)
	queue.Start()

	wg.Add(1)
	for i := range 10 {
		queue.QueueItem(&index.StorageDataItem{BaseItem: &index.BaseItem{Id: uint(i)}})
	}
	wg.Wait() // wait for idle first time

	assert.Eventually(t, func() bool {
		storeMutex.Lock()
		l := len(storedEmbeddings)
		storeMutex.Unlock()
		return l == 10
	}, time.Second, 20*time.Millisecond, "all items processed")

	doneMu.Lock()
	firstDone := doneCalls
	doneMu.Unlock()

	queue.Stop() // should NOT invoke doneFunc again

	doneMu.Lock()
	assert.Equal(t, firstDone, doneCalls, "stop should not invoke idle doneFunc again")
	doneMu.Unlock()
}

func TestEmbeddingsQueue_DoneFuncError(t *testing.T) {
	mockEngine := &MockEmbeddingsEngine{}
	var wg sync.WaitGroup
	var called int32
	erroringDoneFunc := func() error {
		atomic.AddInt32(&called, 1)
		wg.Done()
		return errors.New("done func error")
	}

	queue := NewEmbeddingsQueue(mockEngine, func(u uint, e types.Embeddings) {}, erroringDoneFunc, 1, 5)
	queue.Start()

	wg.Add(1)
	queue.QueueItem(&index.StorageDataItem{BaseItem: &index.BaseItem{Id: 1}})
	wg.Wait()

	assert.Equal(t, int32(1), atomic.LoadInt32(&called), "done should be called exactly once even on error")
	queue.Stop()
}
