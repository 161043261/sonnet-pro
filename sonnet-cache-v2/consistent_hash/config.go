package consistent_hash

import "hash/crc32"

// Config consistent hash configuration
type Config struct {
	// Virtual nodes per real node
	DefaultReplicas int
	// Minimum virtual nodes
	MinReplicas int
	// Maximum virtual nodes
	MaxReplicas int
	// Hash function
	HashFunc func(data []byte) uint32
	// Load balance threshold, exceeding this triggers virtual node adjustment
	LoadBalanceThreshold float64
}

// DefaultConfig default configuration
var DefaultConfig = &Config{
	DefaultReplicas:      50,
	MinReplicas:          10,
	MaxReplicas:          200,
	HashFunc:             crc32.ChecksumIEEE,
	LoadBalanceThreshold: 0.25, // 25% load imbalance triggers adjustment
}
