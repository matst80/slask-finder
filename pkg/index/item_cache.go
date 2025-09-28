package index

import (
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/matst80/slask-finder/pkg/types"
)

// CacheEntry is used to hold a value in the cache
type CacheEntry struct {
	key        uint
	value      types.Item
	lastAccess time.Time
}

// ItemCache is a TTL-based cache that offloads to disk
type ItemCache struct {
	ttl       time.Duration
	storage   types.StorageProvider
	cache     map[uint]*CacheEntry
	mu        sync.RWMutex
	cachePath string
	stopChan  chan struct{}
}

// NewItemCache creates a new ItemCache
func NewItemCache(ttl, cleanupInterval time.Duration, storage types.StorageProvider, cachePath string) (*ItemCache, error) {
	if err := os.MkdirAll(cachePath, 0755); err != nil {
		return nil, err
	}
	c := &ItemCache{
		ttl:       ttl,
		storage:   storage,
		cache:     make(map[uint]*CacheEntry),
		cachePath: cachePath,
		stopChan:  make(chan struct{}),
	}

	go c.cleanupLoop(cleanupInterval)

	return c, nil
}

// StopCleanup stops the background cleanup goroutine
func (c *ItemCache) StopCleanup() {
	close(c.stopChan)
}

// Get retrieves an item from the cache
func (c *ItemCache) Get(key uint) (types.Item, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, hit := c.cache[key]; hit {
		entry.lastAccess = time.Now()
		return entry.value, true
	}

	// Try to load from disk
	if item, err := c.loadFromDisk(key); err == nil {
		c.addToCache(key, item)
		return item, true
	}

	return nil, false
}

// Set adds an item to the cache
func (c *ItemCache) Set(key uint, value types.Item) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, hit := c.cache[key]; hit {
		entry.value = value
		entry.lastAccess = time.Now().Add(-c.ttl)
		return
	}

	c.addToCache(key, value)
}

// Delete removes an item from the cache and disk
func (c *ItemCache) Delete(key uint) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, key)

	// Also delete from disk
	_ = os.Remove(c.filePath(key))
}

// AllItems returns all items currently in the in-memory cache and on disk
func (c *ItemCache) AllItems() iter.Seq[types.Item] {
	return func(yield func(types.Item) bool) {
		c.mu.RLock()
		// Yield all items from memory first
		for _, entry := range c.cache {
			if !yield(entry.value) {
				c.mu.RUnlock()
				return
			}
		}
		c.mu.RUnlock()

		// Now yield items from disk that are not in memory
		files, err := os.ReadDir(c.cachePath)
		if err != nil {
			return
		}

		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
				continue
			}

			keyStr := strings.TrimSuffix(file.Name(), ".json")
			key64, err := strconv.ParseUint(keyStr, 10, 64)
			if err != nil {
				continue
			}
			key := uint(key64)

			c.mu.RLock()
			_, inMemory := c.cache[key]
			c.mu.RUnlock()

			if !inMemory {
				if item, err := c.loadFromDisk(key); err == nil {
					if !yield(item) {
						return
					}
				}
			}
		}
	}
}

func (c *ItemCache) addToCache(key uint, value types.Item) {
	entry := &CacheEntry{
		key:        key,
		value:      value,
		lastAccess: time.Now(),
	}
	c.cache[key] = entry
}

func (c *ItemCache) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.evictExpired()
		case <-c.stopChan:
			return
		}
	}
}

func (c *ItemCache) evictExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.cache {
		if now.Sub(entry.lastAccess) > c.ttl {
			// Evict
			delete(c.cache, key)
			// Save to disk
			_ = c.saveToDisk(entry.value)
		}
	}
}

func (c *ItemCache) filePath(key uint) string {
	return filepath.Join(c.cachePath, fmt.Sprintf("%d.json", key))
}

func (c *ItemCache) saveToDisk(item types.Item) error {
	fileName := c.filePath(item.GetId())
	return c.storage.SaveJson(item, fileName)
}

func (c *ItemCache) loadFromDisk(key uint) (types.Item, error) {
	var item DataItem
	fileName := c.filePath(key)
	err := c.storage.LoadJson(&item, fileName)
	if err != nil {
		return nil, err
	}
	return &item, nil
}
