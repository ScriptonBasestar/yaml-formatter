package testdata

import (
	"sync"
	"time"
)

// CacheEntry represents a cached test data entry
type CacheEntry struct {
	Data      []byte
	Timestamp time.Time
	Category  string
}

// TestDataCache provides caching functionality for test data
type TestDataCache struct {
	cache map[string]*CacheEntry
	mutex sync.RWMutex
	ttl   time.Duration
}

// NewTestDataCache creates a new test data cache with specified TTL
func NewTestDataCache(ttl time.Duration) *TestDataCache {
	return &TestDataCache{
		cache: make(map[string]*CacheEntry),
		ttl:   ttl,
	}
}

// Get retrieves data from cache if available and not expired
func (c *TestDataCache) Get(path string) ([]byte, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	entry, exists := c.cache[path]
	if !exists {
		return nil, false
	}
	
	// Check if entry is expired
	if time.Since(entry.Timestamp) > c.ttl {
		// Don't delete here to avoid write lock, let Set clean up
		return nil, false
	}
	
	return entry.Data, true
}

// Set stores data in cache with current timestamp
func (c *TestDataCache) Set(path string, data []byte, category string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	// Clean expired entries while we have write lock
	c.cleanExpiredLocked()
	
	c.cache[path] = &CacheEntry{
		Data:      data,
		Timestamp: time.Now(),
		Category:  category,
	}
}

// cleanExpiredLocked removes expired entries (must be called with write lock)
func (c *TestDataCache) cleanExpiredLocked() {
	now := time.Now()
	for path, entry := range c.cache {
		if now.Sub(entry.Timestamp) > c.ttl {
			delete(c.cache, path)
		}
	}
}

// Clear removes all cached entries
func (c *TestDataCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.cache = make(map[string]*CacheEntry)
}

// Size returns the number of cached entries
func (c *TestDataCache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	return len(c.cache)
}

// GetByCategory returns all cached entries for a specific category
func (c *TestDataCache) GetByCategory(category string) map[string][]byte {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	result := make(map[string][]byte)
	now := time.Now()
	
	for path, entry := range c.cache {
		if entry.Category == category && now.Sub(entry.Timestamp) <= c.ttl {
			result[path] = entry.Data
		}
	}
	
	return result
}

// Global cache instance with 5-minute TTL
var globalCache = NewTestDataCache(5 * time.Minute)

// GetCachedTestData retrieves test data from cache or loads and caches it
func GetCachedTestData(path string, category string) ([]byte, error) {
	// Try cache first
	if data, hit := globalCache.Get(path); hit {
		return data, nil
	}
	
	// Load from embedded filesystem
	data, err := TestFiles.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	// Cache for future use
	globalCache.Set(path, data, category)
	
	return data, nil
}

// ClearTestDataCache clears the global test data cache
func ClearTestDataCache() {
	globalCache.Clear()
}

// GetCacheStats returns cache statistics
func GetCacheStats() (size int, categories map[string]int) {
	globalCache.mutex.RLock()
	defer globalCache.mutex.RUnlock()
	
	size = len(globalCache.cache)
	categories = make(map[string]int)
	
	for _, entry := range globalCache.cache {
		categories[entry.Category]++
	}
	
	return size, categories
}