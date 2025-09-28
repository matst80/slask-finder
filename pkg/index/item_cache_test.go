package index

import (
	"iter"
	"sync"
	"testing"
	"time"

	"github.com/matst80/slask-finder/pkg/types"
	"github.com/stretchr/testify/assert"
)

// MockStorageProvider is a mock implementation of the types.StorageProvider interface for testing.
type MockStorageProvider struct {
	mu    sync.RWMutex
	store map[string]types.Item
}

// LoadGzippedGob implements types.StorageProvider.
func (m *MockStorageProvider) LoadGzippedGob(output interface{}, filename string) error {
	panic("unimplemented")
}

// LoadGzippedJson implements types.StorageProvider.
func (m *MockStorageProvider) LoadGzippedJson(data interface{}, filename string) error {
	panic("unimplemented")
}

// LoadItems implements types.StorageProvider.
func (m *MockStorageProvider) LoadItems(handlers ...types.ItemHandler) error {
	panic("unimplemented")
}

// SaveGzippedGob implements types.StorageProvider.
func (m *MockStorageProvider) SaveGzippedGob(embeddings any, filename string) error {
	panic("unimplemented")
}

// SaveGzippedJson implements types.StorageProvider.
func (m *MockStorageProvider) SaveGzippedJson(data any, filename string) error {
	panic("unimplemented")
}

// SaveItems implements types.StorageProvider.
func (m *MockStorageProvider) SaveItems(items iter.Seq[types.Item]) error {
	panic("unimplemented")
}

func NewMockStorageProvider() *MockStorageProvider {
	return &MockStorageProvider{
		store: make(map[string]types.Item),
	}
}

func (m *MockStorageProvider) SaveJson(data interface{}, filename string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[filename] = data.(types.Item)
	return nil
}

func (m *MockStorageProvider) LoadJson(data interface{}, filename string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	item, ok := m.store[filename]
	if !ok {
		return assert.AnError
	}
	// This is a bit of a hack to get the data back into the pointer
	*(data.(*DataItem)) = *(item.(*DataItem))
	return nil
}

func (m *MockStorageProvider) ItemCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.store)
}

func TestItemCache_EvictionAndReload(t *testing.T) {
	ttl := 50 * time.Millisecond
	cleanupInterval := 10 * time.Millisecond
	storage := NewMockStorageProvider()
	cachePath := t.TempDir()

	cache, err := NewItemCache(ttl, cleanupInterval, storage, cachePath)
	assert.NoError(t, err)
	defer cache.StopCleanup()

	// Create a test item
	testItem := &DataItem{
		BaseItem: &BaseItem{
			Id:    1,
			Sku:   "test-sku",
			Title: "Test Item",
		},
		Fields: make(types.ItemFields),
	}

	// 1. Set the item in the cache
	cache.Set(testItem.GetId(), testItem)

	// 2. Immediately get the item, should be in memory
	item, ok := cache.Get(testItem.GetId())
	assert.True(t, ok, "Item should be found in cache immediately after set")
	assert.Equal(t, testItem, item, "Retrieved item should be the same as the one set")

	// 3. Wait for TTL to expire and cleanup to run
	time.Sleep(ttl + cleanupInterval*2)

	// 4. Check that the item is no longer in the in-memory cache
	cache.mu.RLock()
	_, inMemory := cache.cache[testItem.GetId()]
	cache.mu.RUnlock()
	assert.False(t, inMemory, "Item should be evicted from in-memory cache after TTL")

	// 5. Check that the item was saved to the mock storage
	assert.Equal(t, 1, storage.ItemCount(), "Item should be saved to the mock storage")

	// 6. Get the item again, it should be reloaded from disk
	reloadedItem, ok := cache.Get(testItem.GetId())
	assert.True(t, ok, "Item should be found in cache after being reloaded from disk")
	assert.Equal(t, testItem.GetId(), reloadedItem.GetId(), "Reloaded item should have the same ID")
	assert.Equal(t, testItem.GetTitle(), reloadedItem.GetTitle(), "Reloaded item should have the same title")

	// 7. Check that the item is now back in the in-memory cache
	cache.mu.RLock()
	_, inMemory = cache.cache[testItem.GetId()]
	cache.mu.RUnlock()
	assert.True(t, inMemory, "Item should be back in memory after being reloaded")
}
