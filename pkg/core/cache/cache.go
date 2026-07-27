// Package cache provides a TTL-aware in-memory cache for API responses.
package cache

import (
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

// Item holds a cached value with its expiration time.
type Item struct {
	Value     interface{}
	ExpiresAt time.Time
}

// IsExpired returns true if the item has passed its TTL.
func (i *Item) IsExpired() bool {
	return !i.ExpiresAt.IsZero() && time.Now().After(i.ExpiresAt)
}

// Cache is a TTL-aware LRU cache safe for concurrent use.
type Cache struct {
	store *lru.Cache[string, *Item]
	ttl   time.Duration
	mu    sync.RWMutex
}

// New creates a new cache with the given max size and default TTL.
// Items with zero TTL never expire.
func New(maxSize int, ttl time.Duration) (*Cache, error) {
	store, err := lru.New[string, *Item](maxSize)
	if err != nil {
		return nil, err
	}
	return &Cache{store: store, ttl: ttl}, nil
}

// Get retrieves a value by key. Returns nil and false if missing or expired.
func (c *Cache) Get(key string) (interface{}, bool) {
	item, ok := c.store.Get(key)
	if !ok {
		return nil, false
	}
	if item.IsExpired() {
		c.store.Remove(key)
		return nil, false
	}
	return item.Value, true
}

// Set stores a value with the default TTL.
func (c *Cache) Set(key string, value interface{}) {
	exp := time.Time{}
	if c.ttl > 0 {
		exp = time.Now().Add(c.ttl)
	}
	c.store.Add(key, &Item{Value: value, ExpiresAt: exp})
}

// SetWithTTL stores a value with a specific TTL.
func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	exp := time.Time{}
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	c.store.Add(key, &Item{Value: value, ExpiresAt: exp})
}

// Remove deletes a key from the cache.
func (c *Cache) Remove(key string) {
	c.store.Remove(key)
}

// Clear empties the cache.
func (c *Cache) Clear() {
	c.store.Purge()
}

// Len returns the number of items in the cache.
func (c *Cache) Len() int {
	return c.store.Len()
}

// Keys returns all cached keys (for debugging).
func (c *Cache) Keys() []string {
	return c.store.Keys()
}
