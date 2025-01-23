package app

import (
	"container/list"
	"sync"
	"time"
)

var IDCache *Cache

// CacheItem represents an item in the cache, holding the MD5, file path, and expiration time.
type CacheItem struct {
	md5        string
	filePath   string
	timestamp  int64
	expiration time.Time
}

// Cache represents the memory cache.
type Cache struct {
	mu       sync.Mutex
	list     *list.List
	items    map[string]*list.Element
	duration time.Duration
}

// NewCache creates a new cache with a given duration for item expiration.
func NewCache(duration time.Duration) *Cache {
	return &Cache{
		list:     list.New(),
		items:    make(map[string]*list.Element),
		duration: duration,
	}
}

// Add adds an item to the cache.
func (c *Cache) Add(md5, filePath string, timestamp int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	expiration := time.Now().Add(c.duration)
	item := &CacheItem{md5: md5, filePath: filePath, timestamp: timestamp, expiration: expiration}
	element := c.list.PushBack(item)
	c.items[md5] = element
}

// Get retrieves an item from the cache, returns nil if not found or expired.
func (c *Cache) Get(md5 string) *CacheItem {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, ok := c.items[md5]; ok {
		item := element.Value.(*CacheItem)
		if time.Now().Before(item.expiration) {
			return item
		}
		// Item has expired, remove it
		c.list.Remove(element)
		delete(c.items, md5)
	}
	return nil
}

// Delete removes an item from the cache.
func (c *Cache) Delete(md5 string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, ok := c.items[md5]; ok {
		c.list.Remove(element)
		delete(c.items, md5)
	}
}

// Start a goroutine to periodically clean up expired items
func (c *Cache) startCleaner() {
	go func() {
		for {
			time.Sleep(1 * time.Hour) // Check every hour
			c.mu.Lock()
			for e := c.list.Front(); e != nil; {
				item := e.Value.(*CacheItem)
				if time.Now().After(item.expiration) {
					next := e.Next()
					c.list.Remove(e)
					delete(c.items, item.md5)
					e = next
				} else {
					e = e.Next()
				}
			}
			c.mu.Unlock()
		}
	}()
}
