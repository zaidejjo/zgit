// Package pcache provides a two-tier persistent cache: an in-memory LRU
// (L1) backed by SQLite (L2) via db.Store. Keys are scoped so entries
// never collide between repos or entry classes, and each class carries
// its own TTL (git data = short, GitHub data = longer).
//
// On a hit the value is served from L1 when present, falling through to
// L2 which repopulates L1. On a miss the caller recomputes and calls Set.
//
// Safe for concurrent use.
package pcache

import (
	"sync"

	"github.com/zaidejjo/zgit/pkg/core/cache"
	"github.com/zaidejjo/zgit/pkg/core/db"
)

// Class namespaces entries by kind so each can carry its own TTL.
type Class string

const (
	// ClassGit is for local git data (log, diff, status) — short TTL.
	ClassGit Class = "git"
	// ClassGitHub is for GitHub API data (PRs, issues, notifications) — longer TTL.
	ClassGitHub Class = "github"
)

// defaultTTL returns the default seconds for a class when none is given.
func defaultTTL(c Class) int64 {
	switch c {
	case ClassGit:
		return 30
	case ClassGitHub:
		return 900 // 15 minutes
	}
	return 30
}

// Cache is repo-scoped: all keys are namespaced under a single repo path.
type Cache struct {
	repo   string
	memory *cache.Cache
	store  *db.Store
	mu     sync.RWMutex
}

// New creates a pcache for repo. When store is nil it falls back to
// memory-only. When memory is nil a default LRU is created.
func New(repo string, store *db.Store, memory *cache.Cache) *Cache {
	if memory == nil {
		memory, _ = cache.New(500, 0) // no expiry in L1; L2 owns TTL
	}
	return &Cache{
		repo:   repo,
		memory: memory,
		store:  store,
	}
}

// key builds the scoped cache key: repo/class/key.
func (c *Cache) key(class Class, key string) string {
	return c.repo + "/" + string(class) + "/" + key
}

// Get returns the cached value if present and fresh. It checks L1 then L2,
// repopulating L1 on an L2 hit. ok is false on a miss or expiry.
func (c *Cache) Get(class Class, key string) ([]byte, bool) {
	k := c.key(class, key)

	c.mu.RLock()
	item, ok := c.memory.Get(k)
	c.mu.RUnlock()
	if ok {
		switch v := item.(type) {
		case []byte:
			return v, true
		case string:
			return []byte(v), true
		}
		return nil, false
	}

	if c.store != nil {
		ttl := c.ttl(class)
		data, ok := c.store.GetCacheEntry(c.repo, k, ttl)
		if ok {
			c.mu.Lock()
			c.memory.Set(k, data)
			c.mu.Unlock()
			return data, true
		}
	}
	return nil, false
}

// Set stores value under class/key with the class's default TTL.
func (c *Cache) Set(class Class, key string, value []byte) error {
	k := c.key(class, key)

	c.mu.Lock()
	c.memory.Set(k, value)
	c.mu.Unlock()

	if c.store != nil {
		return c.store.SetCacheEntry(c.repo, k, value)
	}
	return nil
}

// Invalidate drops one entry.
func (c *Cache) Invalidate(class Class, key string) {
	k := c.key(class, key)
	c.mu.Lock()
	c.memory.Remove(k)
	c.mu.Unlock()
	if c.store != nil {
		c.store.InvalidateCacheEntry(c.repo, k)
	}
}

// InvalidateRepo drops all entries for this repo (called on repo switch/close).
func (c *Cache) InvalidateRepo() {
	if c.store != nil {
		c.store.InvalidateCacheScope(c.repo)
	}
	c.memory.Clear()
}

// ttl returns the effective TTL seconds for a class.
func (c *Cache) ttl(class Class) int64 {
	switch class {
	case ClassGit:
		return defaultTTL(ClassGit)
	case ClassGitHub:
		return defaultTTL(ClassGitHub)
	}
	return defaultTTL(class)
}
