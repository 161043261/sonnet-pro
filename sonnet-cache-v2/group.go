package lark_cache_v2

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hangtiancheng/lark_cache_v2/single_flight"
	"github.com/sirupsen/logrus"
)

var (
	groupsMu sync.RWMutex
	groups   = make(map[string]*Group)
)

// ErrKeyRequired key cannot be empty error
var ErrKeyRequired = errors.New("key is required")

// ErrValueRequired value cannot be empty error
var ErrValueRequired = errors.New("value is required")

// ErrGroupClosed group is closed error
var ErrGroupClosed = errors.New("cache group is closed")

// Getter callback function interface for loading key-value
type Getter interface {
	Get(ctx context.Context, key string) ([]byte, error)
}

// GetterFunc function type implements Getter interface
type GetterFunc func(ctx context.Context, key string) ([]byte, error)

// Get implements Getter interface
func (f GetterFunc) Get(ctx context.Context, key string) ([]byte, error) {
	return f(ctx, key)
}

// Group is a cache namespace
type Group struct {
	name       string
	getter     Getter
	mainCache  *Cache
	peers      PeerPicker
	loader     *single_flight.Group
	expiration time.Duration // Cache expiration time, 0 means never expire
	closed     int32         // Atomic variable, marks if group is closed
	stats      groupStats    // Statistics
}

// groupStats holds group statistics
type groupStats struct {
	loads        int64 // Load count
	localHits    int64 // Local cache hit count
	localMisses  int64 // Local cache miss count
	peerHits     int64 // Peer fetch success count
	peerMisses   int64 // Peer fetch failure count
	loaderHits   int64 // Loader fetch success count
	loaderErrors int64 // Loader fetch failure count
	loadDuration int64 // Total load duration (nanoseconds)
}

// GroupOption defines configuration options for Group
type GroupOption func(*Group)

// WithExpiration sets cache expiration time
func WithExpiration(d time.Duration) GroupOption {
	return func(g *Group) {
		g.expiration = d
	}
}

// WithPeers sets distributed peers
func WithPeers(peers PeerPicker) GroupOption {
	return func(g *Group) {
		g.peers = peers
	}
}

// WithCacheOptions sets cache options
func WithCacheOptions(opts CacheOptions) GroupOption {
	return func(g *Group) {
		g.mainCache = NewCache(opts)
	}
}

// NewGroup creates a new Group instance
func NewGroup(name string, cacheBytes int64, getter Getter, opts ...GroupOption) *Group {
	if getter == nil {
		panic("nil Getter")
	}

	// Create default cache options
	cacheOpts := DefaultCacheOptions()
	cacheOpts.MaxBytes = cacheBytes

	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: NewCache(cacheOpts),
		loader:    &single_flight.Group{},
	}

	// Apply options
	for _, opt := range opts {
		opt(g)
	}

	// Register to global group map
	groupsMu.Lock()
	defer groupsMu.Unlock()

	if _, exists := groups[name]; exists {
		logrus.Warnf("Group with name %s already exists, will be replaced", name)
	}

	groups[name] = g
	logrus.Infof("Created cache group [%s] with cacheBytes=%d, expiration=%v", name, cacheBytes, g.expiration)

	return g
}

// GetGroup gets a group by name
func GetGroup(name string) *Group {
	groupsMu.RLock()
	defer groupsMu.RUnlock()
	return groups[name]
}

// Get gets data from cache
func (g *Group) Get(ctx context.Context, key string) (ByteView, error) {
	// Check if group is closed
	if atomic.LoadInt32(&g.closed) == 1 {
		return ByteView{}, ErrGroupClosed
	}

	if key == "" {
		return ByteView{}, ErrKeyRequired
	}

	// Get from local cache
	view, ok := g.mainCache.Get(ctx, key)
	if ok {
		atomic.AddInt64(&g.stats.localHits, 1)
		return view, nil
	}

	atomic.AddInt64(&g.stats.localMisses, 1)

	// Try to fetch from peers or load
	return g.load(ctx, key)
}

// Set sets cache value
func (g *Group) Set(ctx context.Context, key string, value []byte) error {
	// Check if group is closed
	if atomic.LoadInt32(&g.closed) == 1 {
		return ErrGroupClosed
	}

	if key == "" {
		return ErrKeyRequired
	}
	if len(value) == 0 {
		return ErrValueRequired
	}

	// Check if request is synced from peer
	isPeerRequest := ctx.Value("from_peer") != nil

	// Create cache view
	view := ByteView{b: cloneBytes(value)}

	// Set to local cache
	if g.expiration > 0 {
		g.mainCache.AddWithExpiration(key, view, time.Now().Add(g.expiration))
	} else {
		g.mainCache.Add(key, view)
	}

	// Sync to peers if not from peer and distributed mode enabled
	if !isPeerRequest && g.peers != nil {
		go g.syncToPeers(ctx, "set", key, value)
	}

	return nil
}

// Delete deletes cache value
func (g *Group) Delete(ctx context.Context, key string) error {
	// Check if group is closed
	if atomic.LoadInt32(&g.closed) == 1 {
		return ErrGroupClosed
	}

	if key == "" {
		return ErrKeyRequired
	}

	// Delete from local cache
	g.mainCache.Delete(key)

	// Check if request is synced from peer
	isPeerRequest := ctx.Value("from_peer") != nil

	// Sync to peers if not from peer and distributed mode enabled
	if !isPeerRequest && g.peers != nil {
		go g.syncToPeers(ctx, "delete", key, nil)
	}

	return nil
}

// syncToPeers syncs operation to peers
func (g *Group) syncToPeers(ctx context.Context, op string, key string, value []byte) {
	if g.peers == nil {
		return
	}

	// Select peer
	peer, ok, isSelf := g.peers.PickPeer(key)
	if !ok || isSelf {
		return
	}

	// Create sync request context
	syncCtx := context.WithValue(context.Background(), "from_peer", true)

	var err error
	switch op {
	case "set":
		err = peer.Set(syncCtx, g.name, key, value)
	case "delete":
		_, err = peer.Delete(g.name, key)
	}

	if err != nil {
		logrus.Errorf("[larkCache] failed to sync %s to peer: %v", op, err)
	}
}

// Clear clears cache
func (g *Group) Clear() {
	// Check if group is closed
	if atomic.LoadInt32(&g.closed) == 1 {
		return
	}

	g.mainCache.Clear()
	logrus.Infof("[larkCache] cleared cache for group [%s]", g.name)
}

// Close closes group and releases resources
func (g *Group) Close() error {
	// Return directly if already closed
	if !atomic.CompareAndSwapInt32(&g.closed, 0, 1) {
		return nil
	}

	// Close local cache
	if g.mainCache != nil {
		g.mainCache.Close()
	}

	// Remove from global group map
	groupsMu.Lock()
	delete(groups, g.name)
	groupsMu.Unlock()

	logrus.Infof("[larkCache] closed cache group [%s]", g.name)
	return nil
}

// load loads data
func (g *Group) load(ctx context.Context, key string) (value ByteView, err error) {
	// Use single_flight to ensure concurrent requests load only once
	startTime := time.Now()
	viewi, err := g.loader.Do(key, func() (any, error) {
		return g.loadData(ctx, key)
	})

	// Record load time
	loadDuration := time.Since(startTime).Nanoseconds()
	atomic.AddInt64(&g.stats.loadDuration, loadDuration)
	atomic.AddInt64(&g.stats.loads, 1)

	if err != nil {
		atomic.AddInt64(&g.stats.loaderErrors, 1)
		return ByteView{}, err
	}

	view := viewi.(ByteView)

	// Set to local cache
	if g.expiration > 0 {
		g.mainCache.AddWithExpiration(key, view, time.Now().Add(g.expiration))
	} else {
		g.mainCache.Add(key, view)
	}

	return view, nil
}

// loadData is the actual method to load data
func (g *Group) loadData(ctx context.Context, key string) (value ByteView, err error) {
	// Try to fetch from remote peer
	if g.peers != nil {
		peer, ok, isSelf := g.peers.PickPeer(key)
		if ok && !isSelf {
			value, err := g.getFromPeer(ctx, peer, key)
			if err == nil {
				atomic.AddInt64(&g.stats.peerHits, 1)
				return value, nil
			}

			atomic.AddInt64(&g.stats.peerMisses, 1)
			logrus.Warnf("[larkCache] failed to get from peer: %v", err)
		}
	}

	// Load from data source
	bytes, err := g.getter.Get(ctx, key)
	if err != nil {
		return ByteView{}, fmt.Errorf("failed to get data: %w", err)
	}

	atomic.AddInt64(&g.stats.loaderHits, 1)
	return ByteView{b: cloneBytes(bytes)}, nil
}

// getFromPeer fetches data from peers
func (g *Group) getFromPeer(ctx context.Context, peer Peer, key string) (ByteView, error) {
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, fmt.Errorf("failed to get from peer: %w", err)
	}
	return ByteView{b: bytes}, nil
}

// RegisterPeers registers PeerPicker
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeers called more than once")
	}
	g.peers = peers
	logrus.Infof("[larkCache] registered peers for group [%s]", g.name)
}

// Stats returns cache statistics
func (g *Group) Stats() map[string]any {
	stats := map[string]any{
		"name":          g.name,
		"closed":        atomic.LoadInt32(&g.closed) == 1,
		"expiration":    g.expiration,
		"loads":         atomic.LoadInt64(&g.stats.loads),
		"local_hits":    atomic.LoadInt64(&g.stats.localHits),
		"local_misses":  atomic.LoadInt64(&g.stats.localMisses),
		"peer_hits":     atomic.LoadInt64(&g.stats.peerHits),
		"peer_misses":   atomic.LoadInt64(&g.stats.peerMisses),
		"loader_hits":   atomic.LoadInt64(&g.stats.loaderHits),
		"loader_errors": atomic.LoadInt64(&g.stats.loaderErrors),
	}

	// Calculate various hit rates
	totalGets := stats["local_hits"].(int64) + stats["local_misses"].(int64)
	if totalGets > 0 {
		stats["hit_rate"] = float64(stats["local_hits"].(int64)) / float64(totalGets)
	}

	totalLoads := stats["loads"].(int64)
	if totalLoads > 0 {
		stats["avg_load_time_ms"] = float64(atomic.LoadInt64(&g.stats.loadDuration)) / float64(totalLoads) / float64(time.Millisecond)
	}

	// Add cache size
	if g.mainCache != nil {
		cacheStats := g.mainCache.Stats()
		for k, v := range cacheStats {
			stats["cache_"+k] = v
		}
	}

	return stats
}

// ListGroups returns all cache group names
func ListGroups() []string {
	groupsMu.RLock()
	defer groupsMu.RUnlock()

	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}

	return names
}

// DestroyGroup destroys specified cache group
func DestroyGroup(name string) bool {
	groupsMu.Lock()
	defer groupsMu.Unlock()

	if g, exists := groups[name]; exists {
		g.Close()
		delete(groups, name)
		logrus.Infof("[larkCache] destroyed cache group [%s]", name)
		return true
	}

	return false
}

// DestroyAllGroups destroys all cache groups
func DestroyAllGroups() {
	groupsMu.Lock()
	defer groupsMu.Unlock()

	for name, g := range groups {
		g.Close()
		delete(groups, name)
		logrus.Infof("[larkCache] destroyed cache group [%s]", name)
	}
}
