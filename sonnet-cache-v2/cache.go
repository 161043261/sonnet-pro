package lark_cache_v2

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hangtiancheng/lark_cache_v2/store"

	"github.com/sirupsen/logrus"
)

// Cache is a wrapper for underlying cache storage
type Cache struct {
	mu          sync.RWMutex
	store       store.Store  // Underlying storage implementation
	opts        CacheOptions // Cache configuration options
	hits        int64        // Cache hit count
	misses      int64        // Cache miss count
	initialized int32        // Atomic variable, marks if cache is initialized
	closed      int32        // Atomic variable, marks if cache is closed
}

// CacheOptions cache configuration options
type CacheOptions struct {
	CacheType    store.CacheType                     // Cache type: LRU, LRU2, etc.
	MaxBytes     int64                               // Max memory usage
	BucketCount  uint16                              // Cache bucket count (for LRU2)
	CapPerBucket uint16                              // Capacity per cache bucket (for LRU2)
	Level2Cap    uint16                              // L2 cache bucket capacity (for LRU2)
	CleanupTime  time.Duration                       // Cleanup interval
	OnEvicted    func(key string, value store.Value) // Eviction callback
}

// DefaultCacheOptions returns default cache options
func DefaultCacheOptions() CacheOptions {
	return CacheOptions{
		CacheType:    store.LRU2,
		MaxBytes:     8 * 1024 * 1024, // 8MB
		BucketCount:  16,
		CapPerBucket: 512,
		Level2Cap:    256,
		CleanupTime:  time.Minute,
		OnEvicted:    nil,
	}
}

// NewCache creates a new cache instance
func NewCache(opts CacheOptions) *Cache {
	return &Cache{
		opts: opts,
	}
}

// ensureInitialized ensures cache is initialized
func (c *Cache) ensureInitialized() {
	// Quick check if cache is initialized, avoiding unnecessary lock contention
	if atomic.LoadInt32(&c.initialized) == 1 {
		return
	}

	// Double-checked locking pattern
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.initialized == 0 {
		// Create storage options
		storeOpts := store.Options{
			MaxBytes:        c.opts.MaxBytes,
			BucketCount:     c.opts.BucketCount,
			CapPerBucket:    c.opts.CapPerBucket,
			Level2Cap:       c.opts.Level2Cap,
			CleanupInterval: c.opts.CleanupTime,
			OnEvicted:       c.opts.OnEvicted,
		}

		// Create storage instance
		c.store = store.NewStore(c.opts.CacheType, storeOpts)

		// Mark as initialized
		atomic.StoreInt32(&c.initialized, 1)

		logrus.Infof("Cache initialized with type %s, max bytes: %d", c.opts.CacheType, c.opts.MaxBytes)
	}
}

// Add adds a key-value pair to cache
func (c *Cache) Add(key string, value ByteView) {
	if atomic.LoadInt32(&c.closed) == 1 {
		logrus.Warnf("Attempted to add to a closed cache: %s", key)
		return
	}

	c.ensureInitialized()

	if err := c.store.Set(key, value); err != nil {
		logrus.Warnf("Failed to add key %s to cache: %v", key, err)
	}
}

// Get gets value from cache
func (c *Cache) Get(ctx context.Context, key string) (value ByteView, ok bool) {
	if atomic.LoadInt32(&c.closed) == 1 {
		return ByteView{}, false
	}

	// If cache not initialized, return miss directly
	if atomic.LoadInt32(&c.initialized) == 0 {
		atomic.AddInt64(&c.misses, 1)
		return ByteView{}, false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	// Get from underlying storage
	val, found := c.store.Get(key)
	if !found {
		atomic.AddInt64(&c.misses, 1)
		return ByteView{}, false
	}

	// Update hit count
	atomic.AddInt64(&c.hits, 1)

	// Convert and return
	if bv, ok := val.(ByteView); ok {
		return bv, true
	}

	// Type assertion failed
	logrus.Warnf("Type assertion failed for key %s, expected ByteView", key)
	atomic.AddInt64(&c.misses, 1)
	return ByteView{}, false
}

// AddWithExpiration adds a key-value pair with expiration to cache
func (c *Cache) AddWithExpiration(key string, value ByteView, expirationTime time.Time) {
	if atomic.LoadInt32(&c.closed) == 1 {
		logrus.Warnf("Attempted to add to a closed cache: %s", key)
		return
	}

	c.ensureInitialized()

	// Calculate expiration time
	expiration := time.Until(expirationTime)
	if expiration <= 0 {
		logrus.Debugf("Key %s already expired, not adding to cache", key)
		return
	}

	// Set to underlying storage
	if err := c.store.SetWithExpiration(key, value, expiration); err != nil {
		logrus.Warnf("Failed to add key %s to cache with expiration: %v", key, err)
	}
}

// Delete deletes a key from cache
func (c *Cache) Delete(key string) bool {
	if atomic.LoadInt32(&c.closed) == 1 || atomic.LoadInt32(&c.initialized) == 0 {
		return false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.store.Delete(key)
}

// Clear clears cache
func (c *Cache) Clear() {
	if atomic.LoadInt32(&c.closed) == 1 || atomic.LoadInt32(&c.initialized) == 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.store.Clear()

	// Reset statistics
	atomic.StoreInt64(&c.hits, 0)
	atomic.StoreInt64(&c.misses, 0)
}

// Len returns current item count in cache
func (c *Cache) Len() int {
	if atomic.LoadInt32(&c.closed) == 1 || atomic.LoadInt32(&c.initialized) == 0 {
		return 0
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.store.Len()
}

// Close closes cache and releases resources
func (c *Cache) Close() {
	// Return directly if already closed
	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Close underlying storage
	if c.store != nil {
		if closer, ok := c.store.(interface{ Close() }); ok {
			closer.Close()
		}
		c.store = nil
	}

	// Reset cache state
	atomic.StoreInt32(&c.initialized, 0)

	logrus.Debugf("Cache closed, hits: %d, misses: %d", atomic.LoadInt64(&c.hits), atomic.LoadInt64(&c.misses))
}

// Stats returns cache statistics
func (c *Cache) Stats() map[string]any {
	stats := map[string]any{
		"initialized": atomic.LoadInt32(&c.initialized) == 1,
		"closed":      atomic.LoadInt32(&c.closed) == 1,
		"hits":        atomic.LoadInt64(&c.hits),
		"misses":      atomic.LoadInt64(&c.misses),
	}

	if atomic.LoadInt32(&c.initialized) == 1 {
		stats["size"] = c.Len()

		// Calculate hit rate
		totalRequests := stats["hits"].(int64) + stats["misses"].(int64)
		if totalRequests > 0 {
			stats["hit_rate"] = float64(stats["hits"].(int64)) / float64(totalRequests)
		} else {
			stats["hit_rate"] = 0.0
		}
	}

	return stats
}
