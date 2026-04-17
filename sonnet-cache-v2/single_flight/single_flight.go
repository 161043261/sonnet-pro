package single_flight

import (
	"sync"
)

// Represents an ongoing or finished request
type call struct {
	wg  sync.WaitGroup
	val any
	err error
}

// Group manages all kinds of calls
type Group struct {
	m sync.Map // Use sync.Map to optimize concurrent performance
}

// Do ensures fn is only called once for the same key across multiple calls
func (g *Group) Do(key string, fn func() (any, error)) (any, error) {
	// Check if there is already an ongoing call for this key
	if existing, ok := g.m.Load(key); ok {
		c := existing.(*call)
		c.wg.Wait()         // Wait for the existing request to finish
		return c.val, c.err // Return the result from the ongoing call
	}

	// If no ongoing request, create a new one
	c := &call{}
	c.wg.Add(1)
	g.m.Store(key, c) // Store the call in the map

	// Execute the function and set the result
	c.val, c.err = fn()
	c.wg.Done() // Mark the request as done

	// After the request is done, clean up the map
	g.m.Delete(key)

	return c.val, c.err
}
