package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	cache_v2 "github.com/hangtiancheng/lark_cache_v2"
)

func main() {
	// Add command line args to distinguish nodes
	port := flag.Int("port", 8001, "Node port")
	nodeID := flag.String("node", "A", "Node identifier")
	flag.Parse()

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("[Node %s] Started, address: %s", *nodeID, addr)

	// Create node
	node, err := cache_v2.NewServer(addr, "lark-cache",
		cache_v2.WithEtcdEndpoints([]string{"localhost:2379"}),
		cache_v2.WithDialTimeout(5*time.Second),
	)
	if err != nil {
		log.Fatal("Failed to create node:", err)
	}

	// Create node selector
	picker, err := cache_v2.NewClientPicker(addr)
	if err != nil {
		log.Fatal("Failed to create node selector:", err)
	}

	// Create cache group
	group := cache_v2.NewGroup("test", 2<<20, cache_v2.GetterFunc(
		func(ctx context.Context, key string) ([]byte, error) {
			log.Printf("[Node %s] Triggered data source load: key=%s", *nodeID, key)
			return []byte(fmt.Sprintf("Node %s's data source value", *nodeID)), nil
		}),
	)

	// Register node selector
	group.RegisterPeers(picker)

	// Start node
	go func() {
		log.Printf("[Node %s] Starting service...", *nodeID)
		if err := node.Start(); err != nil {
			log.Fatal("Failed to start node:", err)
		}
	}()

	// Wait for node registration to complete
	log.Printf("[Node %s] Waiting for node registration...", *nodeID)
	time.Sleep(5 * time.Second)

	ctx := context.Background()

	// Set specific key-value pair for this node
	localKey := fmt.Sprintf("key_%s", *nodeID)
	localValue := []byte(fmt.Sprintf("This is node%s's data", *nodeID))

	fmt.Printf("\n=== Node %s: Set local data ===\n", *nodeID)
	err = group.Set(ctx, localKey, localValue)
	if err != nil {
		log.Fatal("Set local datafailed:", err)
	}
	fmt.Printf("Node %s: Set key %s successfully\n", *nodeID, localKey)

	// Wait for other nodes to finish setting up
	log.Printf("[Node %s] Waiting for other nodes to be ready...", *nodeID)
	time.Sleep(30 * time.Second)

	// Print currently discovered nodes
	picker.PrintPeers()

	// Test getting local data
	fmt.Printf("\n=== Node %s: Get local data ===\n", *nodeID)
	fmt.Printf("Querying local cache directly...\n")

	// Print cache statistics
	stats := group.Stats()
	fmt.Printf("Cache statistics: %+v\n", stats)

	if val, err := group.Get(ctx, localKey); err == nil {
		fmt.Printf("Node %s: Get local key %s successfully: %s\n", *nodeID, localKey, val.String())
	} else {
		fmt.Printf("Node %s: Get local keyfailed: %v\n", *nodeID, err)
	}

	// Test getting other node's data
	otherKeys := []string{"key_A", "key_B", "key_C"}
	for _, key := range otherKeys {
		if key == localKey {
			continue // Skip key of current node
		}
		fmt.Printf("\n=== Node %s: Trying to get remote data %s ===\n", *nodeID, key)
		log.Printf("[Node %s] Start looking for key %s 's remote node", *nodeID, key)
		if val, err := group.Get(ctx, key); err == nil {
			fmt.Printf("Node %s: Get remote key %s successfully: %s\n", *nodeID, key, val.String())
		} else {
			fmt.Printf("Node %s: Get remote keyfailed: %v\n", *nodeID, err)
		}
	}

	// Keep program running
	select {}
}
