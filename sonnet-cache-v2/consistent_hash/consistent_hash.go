package consistent_hash

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Map consistent hash implementation
type Map struct {
	mu sync.RWMutex
	// Configuration info
	config *Config
	// Hash ring
	keys []int
	// Hash ring to node mapping
	hashMap map[int]string
	// Mapping from node to virtual node count
	nodeReplicas map[string]int
	// Node load statistics
	nodeCounts map[string]int64
	// Total requests
	totalRequests int64
}

// New creates consistent hash instance
func New(opts ...Option) *Map {
	m := &Map{
		config:       DefaultConfig,
		hashMap:      make(map[int]string),
		nodeReplicas: make(map[string]int),
		nodeCounts:   make(map[string]int64),
	}

	for _, opt := range opts {
		opt(m)
	}

	m.startBalancer() // Start load balancer
	return m
}

// Option configuration options
type Option func(*Map)

// WithConfig sets configuration
func WithConfig(config *Config) Option {
	return func(m *Map) {
		m.config = config
	}
}

// Add adds nodes
func (m *Map) Add(nodes ...string) error {
	if len(nodes) == 0 {
		return errors.New("no nodes provided")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, node := range nodes {
		if node == "" {
			continue
		}

		// Add virtual nodes for a node
		m.addNode(node, m.config.DefaultReplicas)
	}

	// Re-sort
	sort.Ints(m.keys)
	return nil
}

// Remove removes a node
func (m *Map) Remove(node string) error {
	if node == "" {
		return errors.New("invalid node")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	replicas := m.nodeReplicas[node]
	if replicas == 0 {
		return fmt.Errorf("node %s not found", node)
	}

	// Remove all virtual nodes of the node
	for i := 0; i < replicas; i++ {
		hash := int(m.config.HashFunc([]byte(fmt.Sprintf("%s-%d", node, i))))
		delete(m.hashMap, hash)
		for j := 0; j < len(m.keys); j++ {
			if m.keys[j] == hash {
				m.keys = append(m.keys[:j], m.keys[j+1:]...)
				break
			}
		}
	}

	delete(m.nodeReplicas, node)
	delete(m.nodeCounts, node)
	return nil
}

// Get gets a node
func (m *Map) Get(key string) string {
	if key == "" {
		return ""
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.keys) == 0 {
		return ""
	}

	hash := int(m.config.HashFunc([]byte(key)))
	// Binary search
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	// Handle boundary cases
	if idx == len(m.keys) {
		idx = 0
	}

	node := m.hashMap[m.keys[idx]]
	count := m.nodeCounts[node]
	m.nodeCounts[node] = count + 1
	atomic.AddInt64(&m.totalRequests, 1)

	return node
}

// addNode adds virtual nodes for a node
func (m *Map) addNode(node string, replicas int) {
	for i := 0; i < replicas; i++ {
		hash := int(m.config.HashFunc([]byte(fmt.Sprintf("%s-%d", node, i))))
		m.keys = append(m.keys, hash)
		m.hashMap[hash] = node
	}
	m.nodeReplicas[node] = replicas
}

// checkAndRebalance checks and rebalances virtual nodes
func (m *Map) checkAndRebalance() {
	if atomic.LoadInt64(&m.totalRequests) < 1000 {
		return // Too few samples, skip adjustment
	}

	// Calculate load situation
	avgLoad := float64(m.totalRequests) / float64(len(m.nodeReplicas))
	var maxDiff float64

	for _, count := range m.nodeCounts {
		diff := math.Abs(float64(count) - avgLoad)
		if diff/avgLoad > maxDiff {
			maxDiff = diff / avgLoad
		}
	}

	// Adjust virtual nodes if load imbalance exceeds threshold
	if maxDiff > m.config.LoadBalanceThreshold {
		m.rebalanceNodes()
	}
}

// rebalanceNodes rebalances nodes
func (m *Map) rebalanceNodes() {
	m.mu.Lock()
	defer m.mu.Unlock()

	avgLoad := float64(m.totalRequests) / float64(len(m.nodeReplicas))

	// Adjust virtual node count for each node
	for node, count := range m.nodeCounts {
		currentReplicas := m.nodeReplicas[node]
		loadRatio := float64(count) / avgLoad

		var newReplicas int
		if loadRatio > 1 {
			// Load too high, reduce virtual nodes
			newReplicas = int(float64(currentReplicas) / loadRatio)
		} else {
			// Load too low, increase virtual nodes
			newReplicas = int(float64(currentReplicas) * (2 - loadRatio))
		}

		// Ensure within limits
		if newReplicas < m.config.MinReplicas {
			newReplicas = m.config.MinReplicas
		}
		if newReplicas > m.config.MaxReplicas {
			newReplicas = m.config.MaxReplicas
		}

		if newReplicas != currentReplicas {
			// Re-add virtual nodes for the node
			if err := m.Remove(node); err != nil {
				continue // Skip this node if removal fails
			}
			m.addNode(node, newReplicas)
		}
	}

	// Reset counters
	for node := range m.nodeCounts {
		m.nodeCounts[node] = 0
	}
	atomic.StoreInt64(&m.totalRequests, 0)

	// Re-sort
	sort.Ints(m.keys)
}

// GetStats gets load statistics
func (m *Map) GetStats() map[string]float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]float64)
	total := atomic.LoadInt64(&m.totalRequests)
	if total == 0 {
		return stats
	}

	for node, count := range m.nodeCounts {
		stats[node] = float64(count) / float64(total)
	}
	return stats
}

// Move checkAndRebalance to a separate goroutine
func (m *Map) startBalancer() {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for range ticker.C {
			m.checkAndRebalance()
		}
	}()
}
