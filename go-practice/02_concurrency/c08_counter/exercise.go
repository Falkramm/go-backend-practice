package c08

import (
	"maps"
	"sync"
)

// Counter has a usable zero value.
// TODO: Add state and synchronization for concurrent operations.
type Counter struct {
	mu sync.RWMutex
	mp map[string]int
}

func NewCounter() *Counter {
	return &Counter{
		mp: make(map[string]int),
	}
}

func (c *Counter) Add(key string, delta int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.mp[key] += delta
}

func (c *Counter) Get(key string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	v, ok := c.mp[key]
	if !ok {
		return 0
	}

	return v
}

func (c *Counter) Snapshot() map[string]int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return maps.Clone(c.mp)
}
