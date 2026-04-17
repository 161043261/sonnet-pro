package store

import "time"

// Value cache value interface
type Value interface {
	Len() int // Returns data size
}

// Store cache interface
type Store interface {
	Get(key string) (Value, bool)
	Set(key string, value Value) error
	SetWithExpiration(key string, value Value, expiration time.Duration) error
	Delete(key string) bool
	Clear()
	Len() int
	Close()
}

// CacheType cache type
type CacheType string

const (
	LRU  CacheType = "lru"
	LRU2 CacheType = "lru2"
)

// Options general cache configuration options
type Options struct {
	MaxBytes        int64  // Maximum cache bytes (for lru)
	BucketCount     uint16 // Cache bucket count (for lru-2)
	CapPerBucket    uint16 // Capacity per bucket (for lru-2)
	Level2Cap       uint16 // L2 cache capacity in lru-2 (for lru-2)
	CleanupInterval time.Duration
	OnEvicted       func(key string, value Value)
}

func NewOptions() Options {
	return Options{
		MaxBytes:        8192,
		BucketCount:     16,
		CapPerBucket:    512,
		Level2Cap:       256,
		CleanupInterval: time.Minute,
		OnEvicted:       nil,
	}
}

// NewStore creates cache store instance
func NewStore(cacheType CacheType, opts Options) Store {
	switch cacheType {
	case LRU2:
		return newLRU2Cache(opts)
	case LRU:
		return newLRUCache(opts)
	default:
		return newLRUCache(opts)
	}
}
