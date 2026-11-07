package cache

import (
	"sync"
	"time"
)

// memoryItem is a single cached entry.
type memoryItem struct {
	value     []byte
	expiresAt time.Time // zero value means "never expires"
}

// MemoryCache is an in-process, concurrency-safe Cache with TTL support.
//
// Expiration is lazy: an entry is checked (and evicted) when it is accessed.
// This is sufficient for the bounded key space of an SDK (access tokens,
// tickets) and avoids a background sweeper and its lifecycle complexity.
type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]memoryItem
}

// NewMemoryCache creates an empty in-memory cache.
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{items: make(map[string]memoryItem)}
}

// Get returns the value for key and whether it was found and still valid.
// Expired entries are evicted lazily and reported as missing. The returned
// slice is a copy, so the caller may mutate it without affecting the cache.
func (c *MemoryCache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	it, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if !it.expiresAt.IsZero() && time.Now().After(it.expiresAt) {
		c.Delete(key) // lazy eviction
		return nil, false
	}
	out := make([]byte, len(it.value))
	copy(out, it.value)
	return out, true
}

// Set stores value under key with the given TTL.
// A negative ttl deletes the key immediately; a zero ttl stores it forever.
func (c *MemoryCache) Set(key string, value []byte, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ttl < 0 {
		delete(c.items, key)
		return
	}
	it := memoryItem{value: append([]byte(nil), value...)} // copy to own the data
	if ttl > 0 {
		it.expiresAt = time.Now().Add(ttl)
	}
	c.items[key] = it
}

// Delete removes key. Deleting a missing key is a no-op.
func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}

// Has reports whether key exists and is unexpired.
func (c *MemoryCache) Has(key string) bool {
	_, ok := c.Get(key)
	return ok
}

// SetNX implements SetNXer.
//
// The whole check-and-store runs under the write lock. Splitting it into a Has
// followed by a Set would reintroduce the race the interface exists to close,
// and would do so only under concurrent load -- the case that never shows up in
// a single-threaded test.
func (c *MemoryCache) SetNX(key string, value []byte, ttl time.Duration) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if it, ok := c.items[key]; ok {
		// An expired entry counts as absent, so a key whose TTL has elapsed can
		// be claimed again. Without this the entry would be stuck until the next
		// lazy eviction, and deduplication would silently stop working for that
		// key.
		if it.expiresAt.IsZero() || time.Now().Before(it.expiresAt) {
			return false, nil
		}
	}

	it := memoryItem{value: append([]byte(nil), value...)} // copy to own the data
	if ttl > 0 {
		it.expiresAt = time.Now().Add(ttl)
	}
	c.items[key] = it
	return true, nil
}
