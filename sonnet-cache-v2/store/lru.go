package store

import (
	"container/list"
	"sync"
	"time"
)

// lruCache is an LRU cache implementation based on stdlib list
type lruCache struct {
	mu              sync.RWMutex
	list            *list.List               // Doubly linked list for maintaining LRU order
	items           map[string]*list.Element // Key to list node mapping
	expires         map[string]time.Time     // Expiration time mapping
	maxBytes        int64                    // Maximum allowed bytes
	usedBytes       int64                    // Currently used bytes
	onEvicted       func(key string, value Value)
	cleanupInterval time.Duration
	cleanupTicker   *time.Ticker
	closeCh         chan struct{} // For graceful shutdown of cleanup goroutine
}

// lruEntry represents an entry in cache
type lruEntry struct {
	key   string
	value Value
}

// newLRUCache creates a new LRU cache instance
func newLRUCache(opts Options) *lruCache {
	// Set default cleanup interval
	cleanupInterval := opts.CleanupInterval
	if cleanupInterval <= 0 {
		cleanupInterval = time.Minute
	}

	c := &lruCache{
		list:            list.New(),
		items:           make(map[string]*list.Element),
		expires:         make(map[string]time.Time),
		maxBytes:        opts.MaxBytes,
		onEvicted:       opts.OnEvicted,
		cleanupInterval: cleanupInterval,
		closeCh:         make(chan struct{}),
	}

	// Start periodic cleanup goroutine
	c.cleanupTicker = time.NewTicker(c.cleanupInterval)
	go c.cleanupLoop()

	return c
}

// Get fetches cache item, returns if exists and not expired
func (c *lruCache) Get(key string) (Value, bool) {
	c.mu.RLock()
	elem, ok := c.items[key]
	if !ok {
		c.mu.RUnlock()
		return nil, false
	}

	// Check if expired
	if expTime, hasExp := c.expires[key]; hasExp && time.Now().After(expTime) {
		c.mu.RUnlock()

		// Asynchronously delete expired items to avoid doing it within read lock
		go c.Delete(key)

		return nil, false
	}

	// Get value and release read lock
	entry := elem.Value.(*lruEntry)
	value := entry.value
	c.mu.RUnlock()

	// Updating LRU position requires write lock
	c.mu.Lock()
	// Check again if element still exists (might be deleted by another goroutine during lock acquisition)
	if _, ok := c.items[key]; ok {
		c.list.MoveToBack(elem)
	}
	c.mu.Unlock()

	return value, true
}

// Set adds or updates cache item
func (c *lruCache) Set(key string, value Value) error {
	return c.SetWithExpiration(key, value, 0)
}

// SetWithExpiration adds or updates cache item with expiration
func (c *lruCache) SetWithExpiration(key string, value Value, expiration time.Duration) error {
	if value == nil {
		c.Delete(key)
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Calculate expiration time
	var expTime time.Time
	if expiration > 0 {
		expTime = time.Now().Add(expiration)
		c.expires[key] = expTime
	} else {
		delete(c.expires, key)
	}

	// If key exists, update new value
	if elem, ok := c.items[key]; ok {
		oldEntry := elem.Value.(*lruEntry)
		c.usedBytes += int64(value.Len() - oldEntry.value.Len())
		oldEntry.value = value
		c.list.MoveToBack(elem)
		return nil
	}

	// Add new item
	entry := &lruEntry{key: key, value: value}
	elem := c.list.PushBack(entry)
	c.items[key] = elem
	c.usedBytes += int64(len(key) + value.Len())

	// Check if old items need eviction
	c.evict()

	return nil
}

// Delete removes item for specified key from cache
func (c *lruCache) Delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.removeElement(elem)
		return true
	}
	return false
}

// Clear clears cache
func (c *lruCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If callback set, walk through all items and call it
	if c.onEvicted != nil {
		for _, elem := range c.items {
			entry := elem.Value.(*lruEntry)
			c.onEvicted(entry.key, entry.value)
		}
	}

	c.list.Init()
	c.items = make(map[string]*list.Element)
	c.expires = make(map[string]time.Time)
	c.usedBytes = 0
}

// Len returns number of items in cache
func (c *lruCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.list.Len()
}

// removeElement deletes element from cache, must hold lock before calling
func (c *lruCache) removeElement(elem *list.Element) {
	entry := elem.Value.(*lruEntry)
	c.list.Remove(elem)
	delete(c.items, entry.key)
	delete(c.expires, entry.key)
	c.usedBytes -= int64(len(entry.key) + entry.value.Len())

	if c.onEvicted != nil {
		c.onEvicted(entry.key, entry.value)
	}
}

// evict cleans up expired and memory-exceeding items, must hold lock before calling
func (c *lruCache) evict() {
	// First cleanup expired items
	now := time.Now()
	for key, expTime := range c.expires {
		if now.After(expTime) {
			if elem, ok := c.items[key]; ok {
				c.removeElement(elem)
			}
		}
	}

	// Then cleanup least recently used items based on memory limits
	for c.maxBytes > 0 && c.usedBytes > c.maxBytes && c.list.Len() > 0 {
		elem := c.list.Front() // Get least recently used item (list head)
		if elem != nil {
			c.removeElement(elem)
		}
	}
}

// cleanupLoop periodic cleanup goroutine for expired cache
func (c *lruCache) cleanupLoop() {
	for {
		select {
		case <-c.cleanupTicker.C:
			c.mu.Lock()
			c.evict()
			c.mu.Unlock()
		case <-c.closeCh:
			return
		}
	}
}

// Close closes cache and stops cleanup goroutine
func (c *lruCache) Close() {
	if c.cleanupTicker != nil {
		c.cleanupTicker.Stop()
		close(c.closeCh)
	}
}

// GetWithExpiration gets cache item and remaining expiration time
func (c *lruCache) GetWithExpiration(key string) (Value, time.Duration, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	elem, ok := c.items[key]
	if !ok {
		return nil, 0, false
	}

	// Check if expired
	now := time.Now()
	if expTime, hasExp := c.expires[key]; hasExp {
		if now.After(expTime) {
			// Expired
			return nil, 0, false
		}

		// Calculate remaining expiration time
		ttl := expTime.Sub(now)
		c.list.MoveToBack(elem)
		return elem.Value.(*lruEntry).value, ttl, true
	}

	// No expiration time
	c.list.MoveToBack(elem)
	return elem.Value.(*lruEntry).value, 0, true
}

// GetExpiration gets expiration time of a key
func (c *lruCache) GetExpiration(key string) (time.Time, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	expTime, ok := c.expires[key]
	return expTime, ok
}

// UpdateExpiration updates expiration time
func (c *lruCache) UpdateExpiration(key string, expiration time.Duration) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.items[key]; !ok {
		return false
	}

	if expiration > 0 {
		c.expires[key] = time.Now().Add(expiration)
	} else {
		delete(c.expires, key)
	}

	return true
}

// UsedBytes returns currently used bytes
func (c *lruCache) UsedBytes() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.usedBytes
}

// MaxBytes returns maximum allowed bytes
func (c *lruCache) MaxBytes() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.maxBytes
}

// SetMaxBytes sets maximum allowed bytes and triggers eviction
func (c *lruCache) SetMaxBytes(maxBytes int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.maxBytes = maxBytes
	if maxBytes > 0 {
		c.evict()
	}
}
