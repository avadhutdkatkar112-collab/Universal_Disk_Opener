// Package cache implements multi-level intelligent caching.
// Sectors, metadata, directories, and previews are cached separately.
package cache

import (
	"container/list"
	"sync"
	"time"
)

// Cache is a generic LRU cache.
type Cache struct {
	mu       sync.RWMutex
	items    map[string]*list.Element
	order    *list.List
	capacity int
}

type cacheEntry struct {
	key       string
	value     interface{}
	expiresAt time.Time
}

// New creates a new cache with the given capacity.
func New(capacity int) *Cache {
	return &Cache{
		items:    make(map[string]*list.Element),
		order:    list.New(),
		capacity: capacity,
	}
}

// Get retrieves a value from the cache.
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		entry := elem.Value.(*cacheEntry)
		if entry.expiresAt.Before(time.Now()) {
			c.remove(elem)
			return nil, false
		}
		c.order.MoveToFront(elem)
		return entry.value, true
	}
	return nil, false
}

// Set stores a value in the cache with an optional TTL.
func (c *Cache) Set(key string, value interface{}, ttl ...time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.order.MoveToFront(elem)
		entry := elem.Value.(*cacheEntry)
		entry.value = value
		if len(ttl) > 0 {
			entry.expiresAt = time.Now().Add(ttl[0])
		}
		return
	}

	entry := &cacheEntry{
		key:   key,
		value: value,
	}
	if len(ttl) > 0 {
		entry.expiresAt = time.Now().Add(ttl[0])
	}

	elem := c.order.PushBack(entry)
	c.items[key] = elem

	for c.order.Len() > c.capacity {
		c.remove(c.order.Front())
	}
}

// Remove removes a value from the cache.
func (c *Cache) Remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.remove(elem)
	}
}

func (c *Cache) remove(elem *list.Element) {
	entry := elem.Value.(*cacheEntry)
	delete(c.items, entry.key)
	c.order.Remove(elem)
}

// Len returns the number of items in the cache.
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.order.Len()
}

// Clear removes all items from the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*list.Element)
	c.order.Init()
}

// MultiLevelCache manages multiple cache levels.
type MultiLevelCache struct {
	SectorCache     *Cache
	MetadataCache   *Cache
	DirectoryCache  *Cache
	PreviewCache    *Cache
	ThumbnailCache  *Cache
	SearchCache     *Cache
}

// NewMultiLevel creates a multi-level cache with sensible defaults.
func NewMultiLevel() *MultiLevelCache {
	return &MultiLevelCache{
		SectorCache:    New(10000),
		MetadataCache:  New(5000),
		DirectoryCache: New(2000),
		PreviewCache:   New(1000),
		ThumbnailCache: New(5000),
		SearchCache:    New(500),
	}
}

// ClearAll clears all cache levels.
func (m *MultiLevelCache) ClearAll() {
	m.SectorCache.Clear()
	m.MetadataCache.Clear()
	m.DirectoryCache.Clear()
	m.PreviewCache.Clear()
	m.ThumbnailCache.Clear()
	m.SearchCache.Clear()
}

// Stats returns cache statistics.
func (m *MultiLevelCache) Stats() map[string]int {
	return map[string]int{
		"sectors":   m.SectorCache.Len(),
		"metadata":  m.MetadataCache.Len(),
		"directory": m.DirectoryCache.Len(),
		"preview":   m.PreviewCache.Len(),
		"thumbnail": m.ThumbnailCache.Len(),
		"search":    m.SearchCache.Len(),
	}
}
