// Package cache provides caching for embeddings and selection results.
package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// EmbeddingCache caches embeddings for tools.
type EmbeddingCache struct {
	mu       sync.RWMutex
	cache    map[string][]float32 // text -> embedding
	filePath string               // Path to persistent cache
	dirty    bool                 // Whether cache has unsaved changes
}

// NewEmbeddingCache creates a new embedding cache.
func NewEmbeddingCache(cacheDir string) (*EmbeddingCache, error) {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, err
	}

	filePath := filepath.Join(cacheDir, "embeddings.json")

	cache := &EmbeddingCache{
		cache:    make(map[string][]float32),
		filePath: filePath,
	}

	// Load from disk if exists
	if err := cache.load(); err != nil {
		// Start with empty cache on error
		cache.cache = make(map[string][]float32)
	}

	return cache, nil
}

// Get retrieves an embedding from cache.
func (c *EmbeddingCache) Get(text string) ([]float32, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	embedding, ok := c.cache[text]
	if !ok {
		return nil, ErrCacheMiss
	}

	return embedding, nil
}

// Set stores an embedding in cache.
func (c *EmbeddingCache) Set(text string, embedding []float32) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[text] = embedding
	c.dirty = true

	return nil
}

// Save persists the cache to disk.
func (c *EmbeddingCache) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.dirty {
		return nil
	}

	data, err := json.MarshalIndent(c.cache, "", "  ")
	if err != nil {
		return err
	}

	// Write to temp file first
	tmpPath := c.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}

	// Atomic rename
	if err := os.Rename(tmpPath, c.filePath); err != nil {
		return err
	}

	c.dirty = false
	return nil
}

// load loads the cache from disk.
func (c *EmbeddingCache) load() error {
	data, err := os.ReadFile(c.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No cache file yet
		}
		return err
	}

	return json.Unmarshal(data, &c.cache)
}

// Clear clears the cache.
func (c *EmbeddingCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string][]float32)
	c.dirty = true
}

// Size returns the number of cached embeddings.
func (c *EmbeddingCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.cache)
}

// ErrCacheMiss is returned when an item is not in cache.
var ErrCacheMiss = errorString("cache miss")

// errorString implements error.
type errorString string

func (e errorString) Error() string {
	return string(e)
}

// SelectionCache caches selection results.
type SelectionCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	ttl     time.Duration
}

type cacheEntry struct {
	Result    *SelectionResultEntry
	ExpiresAt time.Time
}

// SelectionResultEntry is the serializable form of a selection result.
type SelectionResultEntry struct {
	ToolNames []string
	Scores    map[string]float64
	Reasoning string
	Strategy  string
	Timestamp time.Time
}

// NewSelectionCache creates a new selection cache.
func NewSelectionCache(ttl time.Duration) *SelectionCache {
	return &SelectionCache{
		entries: make(map[string]*cacheEntry),
		ttl:     ttl,
	}
}

// Get retrieves a selection result from cache.
func (c *SelectionCache) Get(taskHash string) (*SelectionResultEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[taskHash]
	if !ok {
		return nil, false
	}

	// Check expiration
	if time.Now().After(entry.ExpiresAt) {
		delete(c.entries, taskHash)
		return nil, false
	}

	return entry.Result, true
}

// Set stores a selection result in cache.
func (c *SelectionCache) Set(taskHash string, result *SelectionResultEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[taskHash] = &cacheEntry{
		Result:    result,
		ExpiresAt: time.Now().Add(c.ttl),
	}

	// Clean expired entries periodically
	if len(c.entries) > 1000 {
		c.cleanExpired()
	}
}

// cleanExpired removes expired entries from cache.
func (c *SelectionCache) cleanExpired() {
	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			delete(c.entries, key)
		}
	}
}

// Clear clears the cache.
func (c *SelectionCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*cacheEntry)
}
