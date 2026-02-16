package cache

import (
	"sync"
	"time"
)

type entry[V any] struct {
	value     V
	expiresAt time.Time
}

// TTLCache is a simple thread-safe in-memory cache with per-entry expiration.
type TTLCache[V any] struct {
	entries map[string]entry[V]
	mu      sync.RWMutex
	ttl     time.Duration
}

// NewTTLCache creates a new TTLCache with the given TTL for each entry.
func NewTTLCache[V any](ttl time.Duration) *TTLCache[V] {
	return &TTLCache[V]{
		entries: make(map[string]entry[V]),
		ttl:     ttl,
	}
}

// Get returns the cached value for key if it exists and hasn't expired.
func (c *TTLCache[V]) Get(key string) (V, bool) {
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok || time.Now().After(e.expiresAt) {
		var zero V
		return zero, false
	}
	return e.value, true
}

// Set stores a value in the cache with the configured TTL.
func (c *TTLCache[V]) Set(key string, value V) {
	c.mu.Lock()
	c.entries[key] = entry[V]{value: value, expiresAt: time.Now().Add(c.ttl)}
	c.mu.Unlock()
}
