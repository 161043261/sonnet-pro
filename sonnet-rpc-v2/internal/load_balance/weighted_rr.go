package loadbalance

import (
	"log"
	"lark_rpc_v2/internal/registry"
	"sync"
)

// Smooth weighted round robin algorithm
type WeightedRR struct {
	mu            sync.Mutex
	weights       []int // Fixed weights
	currentWeight []int // Current weight (dynamic)
	totalWeight   int   // Total weight
}

// Initialize
func NewWeightedRR(weights []int) *WeightedRR {
	w := &WeightedRR{
		weights:       make([]int, len(weights)),
		currentWeight: make([]int, len(weights)),
	}

	copy(w.weights, weights)

	total := 0
	for _, wt := range weights {
		if wt < 0 {
			wt = 0
		}
		total += wt
	}
	w.totalWeight = total

	return w
}

func (w *WeightedRR) Select(list []registry.Instance) registry.Instance {
	if len(list) == 0 {
		return registry.Instance{}
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// If instance count changes, reset is needed (return nil here)
	if len(list) != len(w.weights) {
		log.Println("Instance count and weight list size mismatch")
		return registry.Instance{}
	}

	maxIdx := -1
	// All nodes currentWeight += weight
	for i := 0; i < len(list); i++ {
		w.currentWeight[i] += w.weights[i]

		if maxIdx == -1 || w.currentWeight[i] > w.currentWeight[maxIdx] {
			maxIdx = i
		}
	}

	// Select node with max weight
	w.currentWeight[maxIdx] -= w.totalWeight
	return list[maxIdx]
}
