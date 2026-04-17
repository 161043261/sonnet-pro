package store

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"
)

// Define a simple Value type for testing
type testValue string

func (v testValue) Len() int {
	return len(v)
}

// Test basic cache operations
func TestCacheBasic(t *testing.T) {
	t.Run("Initialize cache", func(t *testing.T) {
		c := Create(10)
		if c == nil {
			t.Fatal("Failed to create cache")
		}
		if c.last != 0 {
			t.Fatalf("Initial last should be 0, actual is%d", c.last)
		}
		if len(c.m) != 10 {
			t.Fatalf("Cache capacity should be 10, actual is%d", len(c.m))
		}
		if len(c.dlnk) != 11 {
			t.Fatalf("List length should be cap+1(11), actual is%d", len(c.dlnk))
		}
	})

	t.Run("Add and Get", func(t *testing.T) {
		c := Create(5)
		var evictCount int
		onEvicted := func(key string, value Value) {
			evictCount++
		}

		// Add new item
		status := c.put("key1", testValue("value1"), 100, onEvicted)
		if status != 1 {
			t.Fatalf("Adding new item should return 1, actual returned%d", status)
		}
		if c.last != 1 {
			t.Fatalf("After adding one item, last should be 1, actual is%d", c.last)
		}

		// Get item
		node, status := c.get("key1")
		if status != 1 {
			t.Fatalf("Getting existing item should return 1, actual returned%d", status)
		}
		if node == nil {
			t.Fatal("Getting item returned nil")
		}
		if node.k != "key1" || node.v.(testValue) != "value1" || node.expireAt != 100 {
			t.Fatalf("Got item value mismatch: %+v", *node)
		}

		// Get non-existent item
		node, status = c.get("non-existent")
		if status != 0 {
			t.Fatalf("Getting non-existent item should return 0, actual returned%d", status)
		}
		if node != nil {
			t.Fatal("Getting non-existent item should not return a node")
		}

		// Update existing item
		status = c.put("key1", testValue("new value"), 200, onEvicted)
		if status != 0 {
			t.Fatalf("Updating item should return 0, actual returned%d", status)
		}

		// Verify updated value
		node, _ = c.get("key1")
		if node.v.(testValue) != "new value" || node.expireAt != 200 {
			t.Fatalf("Updated item value mismatch: %+v", *node)
		}
	})

	t.Run("Delete operation", func(t *testing.T) {
		c := Create(5)

		// Add item
		c.put("key1", testValue("value1"), 100, nil)

		// Delete existing item
		node, status, expireAt := c.del("key1")
		if status != 1 {
			t.Fatalf("Deleting existing item should return 1, actual returned%d", status)
		}
		if node == nil {
			t.Fatal("Deletion should return the deleted node")
		}
		if node.expireAt != 0 {
			t.Fatalf("After deletion expireAt should be 0, actual is%d", node.expireAt)
		}
		if expireAt != 100 {
			t.Fatalf("Deletion should return original expireAt(100), actual is%d", expireAt)
		}

		// Verify cannot get after deletion
		node, status = c.get("key1")
		if status != 1 {
			t.Fatal("Failed to get deleted item, but key should still exist in hash map")
		}
		if node.expireAt != 0 {
			t.Fatalf("Deleted item expireAt should be 0, actual is%d", node.expireAt)
		}

		// Delete non-existent item
		node, status, _ = c.del("non-existent")
		if status != 0 {
			t.Fatalf("Deleting non-existent item should return 0, actual returned%d", status)
		}
		if node != nil {
			t.Fatal("Deleting non-existent item should not return a node")
		}
	})

	t.Run("Capacity and eviction", func(t *testing.T) {
		c := Create(3) // Cache with capacity 3
		var evictedKeys []string

		onEvicted := func(key string, value Value) {
			evictedKeys = append(evictedKeys, key)
		}

		// Fill cache
		for i := 1; i <= 3; i++ {
			c.put("key"+string(rune('0'+i)), testValue("value"+string(rune('0'+i))), 100, onEvicted)
		}

		// Add one more item, should evict oldest key1
		c.put("key4", testValue("value4"), 100, onEvicted)

		if len(evictedKeys) != 1 {
			t.Fatalf("Should evict 1 item, actually evicted %d items", len(evictedKeys))
		}
		if evictedKeys[0] != "key1" {
			t.Fatalf("Should evict key1, actually evicted%s", evictedKeys[0])
		}

		// Verify cache state
		_, status := c.get("key1")
		if status != 0 {
			t.Fatal("key1 should be evicted")
		}

		for i := 2; i <= 4; i++ {
			node, status := c.get("key" + string(rune('0'+i)))
			if status != 1 || node == nil {
				t.Fatalf("key%dshould exist in cache", i)
			}
		}
	})

	t.Run("LRU order maintenance", func(t *testing.T) {
		c := Create(3)

		// Add 3 items in order
		for i := 1; i <= 3; i++ {
			c.put("key"+string(rune('0'+i)), testValue("value"+string(rune('0'+i))), 100, nil)
		}

		// Access order: key1 (latest), key2, key3 (oldest)
		c.get("key2")
		c.get("key1")

		// Add new item, should evict key3
		c.put("key4", testValue("value4"), 100, nil)

		// Verify key3 is evicted
		node, status := c.get("key3")
		if status != 0 || node != nil {
			t.Fatal("key3 should be evicted")
		}

		// Other keys should exist
		for i := 1; i <= 4; i++ {
			if i == 3 {
				continue
			}
			_, status := c.get("key" + string(rune('0'+i)))
			if status != 1 {
				t.Fatalf("key%dshould exist in cache", i)
			}
		}
	})

	t.Run("Walk cache", func(t *testing.T) {
		c := Create(5)

		// Add 3 items
		for i := 1; i <= 3; i++ {
			c.put("key"+string(rune('0'+i)), testValue("value"+string(rune('0'+i))), 100, nil)
		}

		// Walk and collect all keys
		var keys []string
		c.walk(func(key string, value Value, expireAt int64) bool {
			keys = append(keys, key)
			return true
		})

		// Should have 3 keys
		if len(keys) != 3 {
			t.Fatalf("Should have 3 keys, actually has%d", len(keys))
		}

		// Keys should be in reverse add order (new items at list head)
		expectedKeys := []string{"key3", "key2", "key1"}
		for i, key := range expectedKeys {
			if i >= len(keys) || keys[i] != key {
				t.Fatalf("Key %d should be %s, actual is %s", i, key, keys[i])
			}
		}

		// Test early termination of walk
		var earlyKeys []string
		c.walk(func(key string, value Value, expireAt int64) bool {
			earlyKeys = append(earlyKeys, key)
			return len(earlyKeys) < 2 // Collect only first 2 keys
		})

		// Should only have 2 keys
		if len(earlyKeys) != 2 {
			t.Fatalf("Should have 2 keys, actually has%d", len(earlyKeys))
		}
	})
}

// Test cache capacity limits and LRU replacement policy
func TestCacheLRUEviction(t *testing.T) {
	var evictedKeys []string
	onEvicted := func(key string, value Value) {
		evictedKeys = append(evictedKeys, key)
	}

	// Create a cache with capacity 3
	c := Create(3)

	// Add 3 items, no eviction should occur
	c.put("key1", testValue("value1"), Now()+int64(time.Hour), onEvicted)
	c.put("key2", testValue("value2"), Now()+int64(time.Hour), onEvicted)
	c.put("key3", testValue("value3"), Now()+int64(time.Hour), onEvicted)

	if len(evictedKeys) != 0 {
		t.Errorf("Expected no evictions, got %v", evictedKeys)
	}

	// Access key1 to make it most recently used
	c.get("key1")

	// Add 4th item, should evict least recently used key2
	c.put("key4", testValue("value4"), Now()+int64(time.Hour), onEvicted)

	if len(evictedKeys) != 1 || evictedKeys[0] != "key2" {
		t.Errorf("Expected key2 to be evicted, got %v", evictedKeys)
	}

	// Verify key2 has been evicted
	node, status := c.get("key2")
	if status != 0 || node != nil {
		t.Errorf("Expected key2 to be evicted")
	}

	// Verify other keys still exist
	keys := []string{"key1", "key3", "key4"}
	for _, key := range keys {
		node, status := c.get(key)
		if status != 1 || node == nil {
			t.Errorf("Expected %s to exist", key)
		}
	}
}

// Test walk method
func TestCacheWalk(t *testing.T) {
	c := Create(5)

	// Add several items
	c.put("key1", testValue("value1"), Now()+int64(time.Hour), nil)
	c.put("key2", testValue("value2"), Now()+int64(time.Hour), nil)
	c.put("key3", testValue("value3"), Now()+int64(time.Hour), nil)

	// Delete an item
	c.del("key2")

	// Collect all items using walk
	var keys []string
	c.walk(func(key string, value Value, expireAt int64) bool {
		keys = append(keys, key)
		return true
	})

	// Verify only undeleted items are traversed
	if len(keys) != 2 || !contains(keys, "key1") || !contains(keys, "key3") || contains(keys, "key2") {
		t.Errorf("Walk didn't return expected keys, got %v", keys)
	}

	// Test early termination of walk
	count := 0
	c.walk(func(key string, value Value, expireAt int64) bool {
		count++
		return false // Only process first item
	})

	if count != 1 {
		t.Errorf("Walk didn't stop early as expected")
	}
}

// Test adjust method
func TestCacheAdjust(t *testing.T) {
	c := Create(5)

	// Add several items to form list
	c.put("key1", testValue("value1"), Now()+int64(time.Hour), nil)
	c.put("key2", testValue("value2"), Now()+int64(time.Hour), nil)
	c.put("key3", testValue("value3"), Now()+int64(time.Hour), nil)

	// Get key1's index
	idx1 := c.hmap["key1"]

	// Move key1 to list head
	c.adjust(idx1, p, n)

	// Verify key1 is now most recently used
	if c.dlnk[0][n] != idx1 {
		t.Errorf("Expected key1 to be at the head of the list")
	}

	// Move key1 to list tail
	c.adjust(idx1, n, p)

	// Verify key1 is now least recently used
	if c.dlnk[0][p] != idx1 {
		t.Errorf("Expected key1 to be at the tail of the list")
	}
}

// Test lru2Store basic interface
func TestLRU2StoreBasicOperations(t *testing.T) {
	var evictedKeys []string
	onEvicted := func(key string, value Value) {
		evictedKeys = append(evictedKeys, fmt.Sprintf("%s:%v", key, value))
	}

	opts := Options{
		BucketCount:     4,
		CapPerBucket:    2,
		Level2Cap:       3,
		CleanupInterval: time.Minute,
		OnEvicted:       onEvicted,
	}

	store := newLRU2Cache(opts)
	defer store.Close()

	// Test Set and Get
	err := store.Set("key1", testValue("value1"))
	if err != nil {
		t.Errorf("Set failed: %v", err)
	}

	value, found := store.Get("key1")
	if !found || value != testValue("value1") {
		t.Errorf("Get failed, expected 'value1', got %v, found: %v", value, found)
	}

	// Test update
	err = store.Set("key1", testValue("value1-updated"))
	if err != nil {
		t.Errorf("Set update failed: %v", err)
	}

	value, found = store.Get("key1")
	if !found || value != testValue("value1-updated") {
		t.Errorf("Get after update failed, expected 'value1-updated', got %v", value)
	}

	// Test non-existent keys
	value, found = store.Get("nonexistent")
	if found {
		t.Errorf("Get nonexistent key should return false, got %v, %v", value, found)
	}

	// Test delete
	deleted := store.Delete("key1")
	if !deleted {
		t.Errorf("Delete should return true")
	}

	value, found = store.Get("key1")
	if found {
		t.Errorf("Get after delete should return false, got %v, %v", value, found)
	}

	// Test delete non-existent key
	deleted = store.Delete("nonexistent")
	if deleted {
		t.Errorf("Delete nonexistent key should return false")
	}
}

// Test LRU2Store's LRU replacement policy
func TestLRU2StoreLRUEviction(t *testing.T) {
	var evictedKeys []string
	onEvicted := func(key string, value Value) {
		evictedKeys = append(evictedKeys, key)
	}

	opts := Options{
		BucketCount:     1, // Single bucket to simplify testing
		CapPerBucket:    2, // Level 1 cache capacity
		Level2Cap:       2, // Level 2 cache capacity
		CleanupInterval: time.Minute,
		OnEvicted:       onEvicted,
	}

	store := newLRU2Cache(opts)
	defer store.Close()

	// Add items exceeding L1 cache capacity
	store.Set("key1", testValue("value1"))
	store.Set("key2", testValue("value2"))
	store.Set("key3", testValue("value3")) // Should evict key1 to L2 cache

	// key1 should be in L2 cache
	value, found := store.Get("key1")
	if !found || value != testValue("value1") {
		t.Errorf("key1 should be in level2 cache, got %v, found: %v", value, found)
	}

	// Add more items, exceeding L2 capacity
	store.Set("key4", testValue("value4")) // Should evict key2 to L2 cache
	store.Set("key5", testValue("value5")) // Should evict key3, key1 should be evicted from L2 cache

	// key1 should be fully evicted
	value, found = store.Get("key1")
	if found {
		t.Errorf("key1 should be evicted, got %v, found: %v", value, found)
	}
}

// Test expiration time
func TestLRU2StoreExpiration(t *testing.T) {
	opts := Options{
		BucketCount:     1,
		CapPerBucket:    5,
		Level2Cap:       5,
		CleanupInterval: 100 * time.Millisecond, // Fast cleanup
		OnEvicted:       nil,
	}

	store := newLRU2Cache(opts)
	defer store.Close()

	// Add an item expiring quickly
	shortDuration := 200 * time.Millisecond
	store.SetWithExpiration("expires-soon", testValue("value"), shortDuration)

	// Add an item not expiring quickly
	store.SetWithExpiration("expires-later", testValue("value"), time.Hour)

	// Verify both can be retrieved
	_, found := store.Get("expires-soon")
	if !found {
		t.Errorf("expires-soon should be found initially")
	}

	_, found = store.Get("expires-later")
	if !found {
		t.Errorf("expires-later should be found")
	}

	// Wait for short-term item to expire
	time.Sleep(300 * time.Millisecond)

	// Verify short-term item expired, long-term item persists
	_, found = store.Get("expires-soon")
	if found {
		t.Errorf("expires-soon should have expired")
	}

	_, found = store.Get("expires-later")
	if !found {
		t.Errorf("expires-later should still be valid")
	}
}

// Test LRU2Store cleanup loop
func TestLRU2StoreCleanupLoop(t *testing.T) {
	opts := Options{
		BucketCount:     1,
		CapPerBucket:    5,
		Level2Cap:       5,
		CleanupInterval: 100 * time.Millisecond, // Fast cleanup
		OnEvicted:       nil,
	}

	store := newLRU2Cache(opts)
	defer store.Close()

	// Add several quickly expiring items
	shortDuration := 200 * time.Millisecond
	store.SetWithExpiration("expires1", testValue("value1"), shortDuration)
	store.SetWithExpiration("expires2", testValue("value2"), shortDuration)

	// Add an item not expiring quickly
	store.SetWithExpiration("keeps", testValue("value"), time.Hour)

	// Wait for items to expire and be processed by cleanup loop
	time.Sleep(500 * time.Millisecond)

	// Verify expired items are cleaned up
	_, found := store.Get("expires1")
	if found {
		t.Errorf("expires1 should have been cleaned up")
	}

	_, found = store.Get("expires2")
	if found {
		t.Errorf("expires2 should have been cleaned up")
	}

	// Verify unexpired items still exist
	_, found = store.Get("keeps")
	if !found {
		t.Errorf("keeps should still be valid")
	}
}

// Test LRU2Store Clear method
func TestLRU2StoreClear(t *testing.T) {
	opts := Options{
		BucketCount:     2,
		CapPerBucket:    5,
		Level2Cap:       5,
		CleanupInterval: time.Minute,
		OnEvicted:       nil,
	}

	store := newLRU2Cache(opts)
	defer store.Close()

	// Add some items
	for i := 0; i < 10; i++ {
		store.Set(fmt.Sprintf("key%d", i), testValue(fmt.Sprintf("value%d", i)))
	}

	// Verify length
	if length := store.Len(); length != 10 {
		t.Errorf("Expected length 10, got %d", length)
	}

	// Clear cache
	store.Clear()

	// Verify length is 0
	if length := store.Len(); length != 0 {
		t.Errorf("Expected length 0 after Clear, got %d", length)
	}

	// Verify items are deleted
	for i := 0; i < 10; i++ {
		_, found := store.Get(fmt.Sprintf("key%d", i))
		if found {
			t.Errorf("key%d should not be found after Clear", i)
		}
	}
}

// Test internal _get method
func TestLRU2Store_Get(t *testing.T) {
	opts := Options{
		BucketCount:     1,
		CapPerBucket:    5,
		Level2Cap:       5,
		CleanupInterval: time.Minute,
		OnEvicted:       nil,
	}

	store := newLRU2Cache(opts)
	defer store.Close()

	// Add item to L1 cache
	idx := hashBKRD("test-key") & store.mask
	store.caches[idx][0].put("test-key", testValue("test-value"), Now()+int64(time.Hour), nil)

	// Use _get to fetch directly from L1 cache
	node, status := store._get("test-key", idx, 0)
	if status != 1 || node == nil || node.v != testValue("test-value") {
		t.Errorf("_get failed to retrieve from level 0")
	}

	// Add item to L2 cache
	store.caches[idx][1].put("test-key2", testValue("test-value2"), Now()+int64(time.Hour), nil)

	// Use _get to fetch directly from L2 cache
	node, status = store._get("test-key2", idx, 1)
	if status != 1 || node == nil || node.v != testValue("test-value2") {
		t.Errorf("_get failed to retrieve from level 1")
	}

	// Test getting non-existent key
	node, status = store._get("nonexistent", idx, 0)
	if status != 0 || node != nil {
		t.Errorf("_get should return status 0 for nonexistent key")
	}

	// Test expired items
	store.caches[idx][0].put("expired", testValue("value"), Now()-1000, nil) // Expired
	node, status = store._get("expired", idx, 0)
	if status != 0 || node != nil {
		t.Errorf("_get should return status 0 for expired key")
	}
}

// Test internal delete method
func TestLRU2StoreDelete(t *testing.T) {
	var evictedKeys []string
	onEvicted := func(key string, value Value) {
		evictedKeys = append(evictedKeys, key)
	}

	opts := Options{
		BucketCount:     1,
		CapPerBucket:    5,
		Level2Cap:       5,
		CleanupInterval: time.Minute,
		OnEvicted:       onEvicted,
	}

	store := newLRU2Cache(opts)
	defer store.Close()

	// Add item to L1 cache
	idx := hashBKRD("test-key") & store.mask
	store.caches[idx][0].put("test-key", testValue("test-value"), Now()+int64(time.Hour), nil)

	// Add item to L2 cache
	store.caches[idx][1].put("test-key2", testValue("test-value2"), Now()+int64(time.Hour), nil)

	// Delete item from L1 cache
	deleted := store.delete("test-key", idx)
	if !deleted {
		t.Errorf("delete should return true for existing key")
	}

	// Verify items are deleted and callback invoked
	if len(evictedKeys) != 1 || evictedKeys[0] != "test-key" {
		t.Errorf("OnEvicted callback not called correctly, got %v", evictedKeys)
	}

	// Reset callback record
	evictedKeys = nil

	// Delete item from L2 cache
	deleted = store.delete("test-key2", idx)
	if !deleted {
		t.Errorf("delete should return true for existing key in level 1")
	}

	// Verify items are deleted and callback invoked
	if len(evictedKeys) != 1 || evictedKeys[0] != "test-key2" {
		t.Errorf("OnEvicted callback not called correctly, got %v", evictedKeys)
	}

	// Test delete non-existent key
	deleted = store.delete("nonexistent", idx)
	if deleted {
		t.Errorf("delete should return false for nonexistent key")
	}
}

// Test concurrent operations
func TestLRU2StoreConcurrent(t *testing.T) {
	opts := Options{
		BucketCount:     8,
		CapPerBucket:    100,
		Level2Cap:       200,
		CleanupInterval: time.Minute,
		OnEvicted:       nil,
	}

	store := newLRU2Cache(opts)
	defer store.Close()

	const goroutines = 10
	const operationsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()

			// Each goroutine operates on its own set of keys
			prefix := fmt.Sprintf("g%d-", id)

			// Add operations
			for i := 0; i < operationsPerGoroutine; i++ {
				key := prefix + strconv.Itoa(i)
				value := testValue(fmt.Sprintf("value-%s", key))

				err := store.Set(key, value)
				if err != nil {
					t.Errorf("Set failed: %v", err)
				}
			}

			// Get operations
			for i := 0; i < operationsPerGoroutine; i++ {
				key := prefix + strconv.Itoa(i)
				expectedValue := testValue(fmt.Sprintf("value-%s", key))

				value, found := store.Get(key)
				if !found {
					t.Errorf("Get failed for key %s", key)
				} else if value != expectedValue {
					t.Errorf("Get returned wrong value for %s: expected %s, got %v", key, expectedValue, value)
				}
			}

			// Delete operation
			for i := 0; i < operationsPerGoroutine/2; i++ { // Delete half of the keys
				key := prefix + strconv.Itoa(i)
				deleted := store.Delete(key)
				if !deleted {
					t.Errorf("Delete failed for key %s", key)
				}
			}
		}(g)
	}

	wg.Wait()

	// Verify approximate length
	// Each goroutine added operationsPerGoroutine items, deleted half
	expectedItems := goroutines * operationsPerGoroutine / 2
	actualItems := store.Len()

	// Allow some error margin due to possible key collisions or pending operations
	tolerance := expectedItems / 10
	if actualItems < expectedItems-tolerance || actualItems > expectedItems+tolerance {
		t.Errorf("Expected approximately %d items, got %d", expectedItems, actualItems)
	}
}

// Test cache hit rate statistics
func TestLRU2StoreHitRatio(t *testing.T) {
	opts := Options{
		BucketCount:     4,
		CapPerBucket:    10,
		Level2Cap:       20,
		CleanupInterval: time.Minute,
		OnEvicted:       nil,
	}

	store := newLRU2Cache(opts)
	defer store.Close()

	// Add 50 items
	for i := 0; i < 50; i++ {
		store.Set(fmt.Sprintf("key%d", i), testValue(fmt.Sprintf("value%d", i)))
	}

	// Count hit times
	hits := 0
	attempts := 0

	// Try getting 100 keys, half exist, half non-existent
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key%d", i)
		_, found := store.Get(key)
		attempts++
		if found {
			hits++
		}
	}

	// Calculate hit rate
	hitRatio := float64(hits) / float64(attempts)

	// Verify hit rate is approx 0.25-0.35 (due to buckets and LRU eviction)
	if hitRatio < 0.25 || hitRatio > 0.35 {
		t.Errorf("Hit ratio out of expected range: got %.2f", hitRatio)
	}
}

// Test cache capacity growth and performance
func BenchmarkLRU2StoreOperations(b *testing.B) {
	opts := Options{
		BucketCount:     16,
		CapPerBucket:    1000,
		Level2Cap:       2000,
		CleanupInterval: time.Minute,
		OnEvicted:       nil,
	}

	store := newLRU2Cache(opts)
	defer store.Close()

	// Pre-fill some data
	for i := 0; i < 5000; i++ {
		store.Set(fmt.Sprintf("init-key%d", i), testValue(fmt.Sprintf("value%d", i)))
	}

	b.ResetTimer()

	// Mixed operations benchmark
	b.Run("MixedOperations", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("bench-key%d", i%10000)

			// 75% chance for Get, 25% chance for Set
			if i%4 != 0 {
				store.Get(key)
			} else {
				store.Set(key, testValue(fmt.Sprintf("value%d", i)))
			}
		}
	})

	// Get operations benchmark
	b.Run("GetOnly", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("init-key%d", i%5000)
			store.Get(key)
		}
	})

	// Set operations benchmark
	b.Run("SetOnly", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("new-key%d", i)
			store.Set(key, testValue(fmt.Sprintf("value%d", i)))
		}
	})
}

// Helper function: check if slice contains string
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
