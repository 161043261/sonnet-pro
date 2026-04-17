package store

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type lru2Store struct {
	locks       []sync.Mutex
	caches      [][2]*cache
	onEvicted   func(key string, value Value)
	cleanupTick *time.Ticker
	mask        int32
}

func newLRU2Cache(opts Options) *lru2Store {
	if opts.BucketCount == 0 {
		opts.BucketCount = 16
	}
	if opts.CapPerBucket == 0 {
		opts.CapPerBucket = 1024
	}
	if opts.Level2Cap == 0 {
		opts.Level2Cap = 1024
	}
	if opts.CleanupInterval <= 0 {
		opts.CleanupInterval = time.Minute
	}

	mask := maskOfNextPowOf2(opts.BucketCount)
	s := &lru2Store{
		locks:       make([]sync.Mutex, mask+1),
		caches:      make([][2]*cache, mask+1),
		onEvicted:   opts.OnEvicted,
		cleanupTick: time.NewTicker(opts.CleanupInterval),
		mask:        int32(mask),
	}

	for i := range s.caches {
		s.caches[i][0] = Create(opts.CapPerBucket)
		s.caches[i][1] = Create(opts.Level2Cap)
	}

	if opts.CleanupInterval > 0 {
		go s.cleanupLoop()
	}

	return s
}

func (s *lru2Store) Get(key string) (Value, bool) {
	idx := hashBKRD(key) & s.mask
	s.locks[idx].Lock()
	defer s.locks[idx].Unlock()

	currentTime := Now()

	// First check L1 cache
	n1, status1, expireAt := s.caches[idx][0].del(key)
	if status1 > 0 {
		// Item found in L1 cache
		if expireAt > 0 && currentTime >= expireAt {
			// Item expired, delete it
			s.delete(key, idx)
			fmt.Println("Found item expired, deleting it")
			return nil, false
		}

		// Item valid, moving to L2 cache
		s.caches[idx][1].put(key, n1.v, expireAt, s.onEvicted)
		fmt.Println("Item valid, moving to L2 cache")
		return n1.v, true
	}

	// Not found in L1 cache, checking L2 cache
	n2, status2 := s._get(key, idx, 1)
	if status2 > 0 && n2 != nil {
		if n2.expireAt > 0 && currentTime >= n2.expireAt {
			// Item expired, delete it
			s.delete(key, idx)
			fmt.Println("Found item expired, deleting it")
			return nil, false
		}

		return n2.v, true
	}

	return nil, false
}

func (s *lru2Store) Set(key string, value Value) error {
	return s.SetWithExpiration(key, value, 9999999999999999)
}

func (s *lru2Store) SetWithExpiration(key string, value Value, expiration time.Duration) error {
	// Calculate expiration - ensure consistent units
	expireAt := int64(0)
	if expiration > 0 {
		// now() returns nanosecond timestamp, ensure expiration is also in nanoseconds
		expireAt = Now() + int64(expiration.Nanoseconds())
	}

	idx := hashBKRD(key) & s.mask
	s.locks[idx].Lock()
	defer s.locks[idx].Unlock()

	// Put into L1 cache
	s.caches[idx][0].put(key, value, expireAt, s.onEvicted)

	return nil
}

// Delete implements Store interface
func (s *lru2Store) Delete(key string) bool {
	idx := hashBKRD(key) & s.mask
	s.locks[idx].Lock()
	defer s.locks[idx].Unlock()

	return s.delete(key, idx)
}

// Clear implements Store interface
func (s *lru2Store) Clear() {
	var keys []string

	for i := range s.caches {
		s.locks[i].Lock()

		s.caches[i][0].walk(func(key string, value Value, expireAt int64) bool {
			keys = append(keys, key)
			return true
		})
		s.caches[i][1].walk(func(key string, value Value, expireAt int64) bool {
			// Check if key is already collected (avoid duplicates)
			for _, k := range keys {
				if key == k {
					return true
				}
			}
			keys = append(keys, key)
			return true
		})

		s.locks[i].Unlock()
	}

	for _, key := range keys {
		s.Delete(key)
	}

	//s.expirations = sync.Map{}
}

// Len implements Store interface
func (s *lru2Store) Len() int {
	count := 0

	for i := range s.caches {
		s.locks[i].Lock()

		s.caches[i][0].walk(func(key string, value Value, expireAt int64) bool {
			count++
			return true
		})
		s.caches[i][1].walk(func(key string, value Value, expireAt int64) bool {
			count++
			return true
		})

		s.locks[i].Unlock()
	}

	return count
}

// Close closes cache related resources
func (s *lru2Store) Close() {
	if s.cleanupTick != nil {
		s.cleanupTick.Stop()
	}
}

// Internal clock to reduce GC pressure from time.Now() calls
var clock, p, n = time.Now().UnixNano(), uint16(0), uint16(1)

// Returns current value of clock variable. atomic.LoadInt64 is atomic for safe concurrent reads
func Now() int64 { return atomic.LoadInt64(&clock) }

func init() {
	go func() {
		for {
			atomic.StoreInt64(&clock, time.Now().UnixNano()) // Calibrate once per second
			for i := 0; i < 9; i++ {
				time.Sleep(100 * time.Millisecond)
				atomic.AddInt64(&clock, int64(100*time.Millisecond)) // Keep clock within an accurate range, avoiding frequent syscalls
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()
}

// Implements BKDR hash algorithm to calculate key hash
func hashBKRD(s string) (hash int32) {
	for i := 0; i < len(s); i++ {
		hash = hash*131 + int32(s[i])
	}

	return hash
}

// maskOfNextPowOf2 calculates nearest power of 2 minus one >= input as mask
func maskOfNextPowOf2(cap uint16) uint16 {
	if cap > 0 && cap&(cap-1) == 0 {
		return cap - 1
	}

	// Fills all bits right of highest 1 with 1s via shifts and ORs
	cap |= cap >> 1
	cap |= cap >> 2
	cap |= cap >> 4

	return cap | (cap >> 8)
}

type node struct {
	k        string
	v        Value
	expireAt int64 // Expiration timestamp, expireAt = 0 means deleted
}

// Internal core cache implementation with doubly linked list and node storage
type cache struct {
	// dlnk[0] is sentinel node tracking head/tail, [p] is tail index, [n] is head index
	dlnk [][2]uint16       // Doubly linked list, 0 is predecessor, 1 is successor
	m    []node            // Pre-allocated memory for nodes
	hmap map[string]uint16 // Key to node index mapping
	last uint16            // Index of the last node element
}

func Create(cap uint16) *cache {
	return &cache{
		dlnk: make([][2]uint16, cap+1),
		m:    make([]node, cap),
		hmap: make(map[string]uint16, cap),
		last: 0,
	}
}

// Add item to cache, returns 1 if new, 0 if updated
func (c *cache) put(key string, val Value, expireAt int64, onEvicted func(string, Value)) int {
	if idx, ok := c.hmap[key]; ok {
		c.m[idx-1].v, c.m[idx-1].expireAt = val, expireAt
		c.adjust(idx, p, n) // Refresh to list head
		return 0
	}

	if c.last == uint16(cap(c.m)) {
		tail := &c.m[c.dlnk[0][p]-1]
		if onEvicted != nil && (*tail).expireAt > 0 {
			onEvicted((*tail).k, (*tail).v)
		}

		delete(c.hmap, (*tail).k)
		c.hmap[key], (*tail).k, (*tail).v, (*tail).expireAt = c.dlnk[0][p], key, val, expireAt
		c.adjust(c.dlnk[0][p], p, n)

		return 1
	}

	c.last++
	if len(c.hmap) <= 0 {
		c.dlnk[0][p] = c.last
	} else {
		c.dlnk[c.dlnk[0][n]][p] = c.last
	}

	// Initialize new node and update list pointers
	c.m[c.last-1].k = key
	c.m[c.last-1].v = val
	c.m[c.last-1].expireAt = expireAt
	c.dlnk[c.last] = [2]uint16{0, c.dlnk[0][n]}
	c.hmap[key] = c.last
	c.dlnk[0][n] = c.last

	return 1
}

// Get node and status for key from cache
func (c *cache) get(key string) (*node, int) {
	if idx, ok := c.hmap[key]; ok {
		c.adjust(idx, p, n)
		return &c.m[idx-1], 1
	}
	return nil, 0
}

// Delete item for key from cache
func (c *cache) del(key string) (*node, int, int64) {
	if idx, ok := c.hmap[key]; ok && c.m[idx-1].expireAt > 0 {
		e := c.m[idx-1].expireAt
		c.m[idx-1].expireAt = 0 // Mark as deleted
		c.adjust(idx, n, p)     // Move to list tail
		return &c.m[idx-1], 1, e
	}

	return nil, 0, 0
}

// Walk all valid items in cache
func (c *cache) walk(walker func(key string, value Value, expireAt int64) bool) {
	for idx := c.dlnk[0][n]; idx != 0; idx = c.dlnk[idx][n] {
		if c.m[idx-1].expireAt > 0 && !walker(c.m[idx-1].k, c.m[idx-1].v, c.m[idx-1].expireAt) {
			return
		}
	}
}

// Adjust node position in list
// When f=0, t=1 move to head; otherwise move to tail
func (c *cache) adjust(idx, f, t uint16) {
	if c.dlnk[idx][f] != 0 {
		c.dlnk[c.dlnk[idx][t]][f] = c.dlnk[idx][f]
		c.dlnk[c.dlnk[idx][f]][t] = c.dlnk[idx][t]
		c.dlnk[idx][f] = 0
		c.dlnk[idx][t] = c.dlnk[0][t]
		c.dlnk[c.dlnk[0][t]][f] = idx
		c.dlnk[0][t] = idx
	}
}

func (s *lru2Store) _get(key string, idx, level int32) (*node, int) {
	if n, st := s.caches[idx][level].get(key); st > 0 && n != nil {
		currentTime := Now()
		if n.expireAt <= 0 || currentTime >= n.expireAt {
			// Expired or deleted
			return nil, 0
		}
		return n, st
	}

	return nil, 0
}

func (s *lru2Store) delete(key string, idx int32) bool {
	n1, s1, _ := s.caches[idx][0].del(key)
	n2, s2, _ := s.caches[idx][1].del(key)
	deleted := s1 > 0 || s2 > 0

	if deleted && s.onEvicted != nil {
		if n1 != nil && n1.v != nil {
			s.onEvicted(key, n1.v)
		} else if n2 != nil && n2.v != nil {
			s.onEvicted(key, n2.v)
		}
	}

	if deleted {
		//s.expirations.Delete(key)
	}

	return deleted
}

func (s *lru2Store) cleanupLoop() {
	for range s.cleanupTick.C {
		currentTime := Now()

		for i := range s.caches {
			s.locks[i].Lock()

			// Check and cleanup expired items
			var expiredKeys []string

			s.caches[i][0].walk(func(key string, value Value, expireAt int64) bool {
				if expireAt > 0 && currentTime >= expireAt {
					expiredKeys = append(expiredKeys, key)
				}
				return true
			})

			s.caches[i][1].walk(func(key string, value Value, expireAt int64) bool {
				if expireAt > 0 && currentTime >= expireAt {
					for _, k := range expiredKeys {
						if key == k {
							// Avoid duplicates
							return true
						}
					}
					expiredKeys = append(expiredKeys, key)
				}
				return true
			})

			for _, key := range expiredKeys {
				s.delete(key, int32(i))
			}

			s.locks[i].Unlock()
		}
	}
}
