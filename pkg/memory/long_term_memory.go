package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// LongTermMemoryConfig holds configuration for long-term memory
type LongTermMemoryConfig struct {
	StoragePath string        // Path to store memory files
	MaxEntries  int           // Maximum entries per memory type (0 = unlimited)
	TTL         time.Duration // Time-to-live for memory entries (0 = no expiration)
}

// DefaultLongTermMemoryConfig returns default configuration
func DefaultLongTermMemoryConfig(storagePath string) *LongTermMemoryConfig {
	return &LongTermMemoryConfig{
		StoragePath: storagePath,
		MaxEntries:  1000,
		TTL:         0, // No expiration by default
	}
}

// MemoryEntry represents a single memory entry
type MemoryEntry struct {
	ID        string         `json:"id"`
	Content   string         `json:"content"`
	Timestamp string         `json:"timestamp"`
	ExpiresAt string         `json:"expires_at,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// LongTermMemory provides persistent memory storage with file backing
type LongTermMemory struct {
	config    *LongTermMemoryConfig
	entries   map[string][]*MemoryEntry // memory_type -> entries
	mu        sync.RWMutex
	storage   *FileStorage
	typeIndex map[string]int // memory_type -> last used index
}

// FileStorage handles persistent file operations
type FileStorage struct {
	basePath string
	mu       sync.Mutex
}

// NewFileStorage creates a new file storage
func NewFileStorage(basePath string) *FileStorage {
	return &FileStorage{
		basePath: basePath,
	}
}

// Save saves data to a file
func (fs *FileStorage) Save(key string, data any) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Ensure directory exists
	if err := os.MkdirAll(fs.basePath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	filePath := filepath.Join(fs.basePath, key+".json")
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Load loads data from a file
func (fs *FileStorage) Load(key string, target any) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	filePath := filepath.Join(fs.basePath, key+".json")
	jsonData, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist is not an error
		}
		return fmt.Errorf("failed to read file: %w", err)
	}

	if err := json.Unmarshal(jsonData, target); err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return nil
}

// Delete removes a file
func (fs *FileStorage) Delete(key string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	filePath := filepath.Join(fs.basePath, key+".json")
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// List lists all files with a given prefix
func (fs *FileStorage) List(prefix string) ([]string, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	entries, err := os.ReadDir(fs.basePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var result []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if prefix == "" || len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			// Remove .json extension
			result = append(result, name[:len(name)-5])
		}
	}

	return result, nil
}

// NewLongTermMemory creates a new long-term memory instance
func NewLongTermMemory(config *LongTermMemoryConfig) (*LongTermMemory, error) {
	if config == nil {
		config = DefaultLongTermMemoryConfig("./memory")
	}

	storage := NewFileStorage(config.StoragePath)

	mem := &LongTermMemory{
		config:    config,
		entries:   make(map[string][]*MemoryEntry),
		storage:   storage,
		typeIndex: make(map[string]int),
	}

	// Load existing memories from disk
	if err := mem.loadFromDisk(); err != nil {
		return nil, fmt.Errorf("failed to load memories: %w", err)
	}

	return mem, nil
}

// Add adds a new memory entry
func (ltm *LongTermMemory) Add(ctx context.Context, memoryType, content string, metadata map[string]any) (string, error) {
	ltm.mu.Lock()
	defer ltm.mu.Unlock()

	entry := &MemoryEntry{
		ID:        types.GenerateID(),
		Content:   content,
		Timestamp: types.Timestamp(),
		Metadata:  metadata,
	}

	// Set expiration if TTL is configured
	if ltm.config.TTL > 0 {
		expiresAt := time.Now().Add(ltm.config.TTL)
		entry.ExpiresAt = expiresAt.Format(time.RFC3339)
	}

	// Initialize type slice if needed
	if ltm.entries[memoryType] == nil {
		ltm.entries[memoryType] = make([]*MemoryEntry, 0)
	}

	// Add entry
	ltm.entries[memoryType] = append(ltm.entries[memoryType], entry)

	// Enforce max entries limit
	if ltm.config.MaxEntries > 0 && len(ltm.entries[memoryType]) > ltm.config.MaxEntries {
		// Remove oldest entries
		removeCount := len(ltm.entries[memoryType]) - ltm.config.MaxEntries
		ltm.entries[memoryType] = ltm.entries[memoryType][removeCount:]
	}

	// Persist to disk
	if err := ltm.persistType(memoryType); err != nil {
		return "", fmt.Errorf("failed to persist memory: %w", err)
	}

	return entry.ID, nil
}

// Get retrieves a specific memory entry by ID and type
func (ltm *LongTermMemory) Get(ctx context.Context, memoryType, id string) (*MemoryEntry, error) {
	ltm.mu.RLock()
	defer ltm.mu.RUnlock()

	entries := ltm.entries[memoryType]
	if entries == nil {
		return nil, fmt.Errorf("memory type '%s' not found", memoryType)
	}

	for _, entry := range entries {
		if entry.ID == id {
			// Check expiration
			if entry.ExpiresAt != "" {
				expiresAt, err := time.Parse(time.RFC3339, entry.ExpiresAt)
				if err == nil && time.Now().After(expiresAt) {
					return nil, fmt.Errorf("memory entry has expired")
				}
			}
			return entry, nil
		}
	}

	return nil, fmt.Errorf("memory entry with ID '%s' not found", id)
}

// Search searches for memory entries by content
func (ltm *LongTermMemory) Search(ctx context.Context, memoryType, query string, limit int) ([]*MemoryEntry, error) {
	ltm.mu.RLock()
	defer ltm.mu.RUnlock()

	entries := ltm.entries[memoryType]
	if entries == nil {
		return []*MemoryEntry{}, nil
	}

	var results []*MemoryEntry
	now := time.Now()

	for _, entry := range entries {
		// Check expiration
		if entry.ExpiresAt != "" {
			expiresAt, err := time.Parse(time.RFC3339, entry.ExpiresAt)
			if err == nil && now.After(expiresAt) {
				continue
			}
		}

		// Simple substring search (can be enhanced with fuzzy matching)
		if query == "" || contains(entry.Content, query) {
			results = append(results, entry)
			if limit > 0 && len(results) >= limit {
				break
			}
		}
	}

	return results, nil
}

// GetRecent returns recent memories of a specific type
func (ltm *LongTermMemory) GetRecent(ctx context.Context, memoryType string, limit int) ([]*MemoryEntry, error) {
	ltm.mu.RLock()
	defer ltm.mu.RUnlock()

	entries := ltm.entries[memoryType]
	if entries == nil {
		return []*MemoryEntry{}, nil
	}

	now := time.Now()
	var results []*MemoryEntry

	// Get most recent entries (in reverse order)
	for i := len(entries) - 1; i >= 0 && len(results) < limit; i-- {
		entry := entries[i]

		// Check expiration
		if entry.ExpiresAt != "" {
			expiresAt, err := time.Parse(time.RFC3339, entry.ExpiresAt)
			if err == nil && now.After(expiresAt) {
				continue
			}
		}

		results = append(results, entry)
	}

	return results, nil
}

// GetAllTypes returns all memory types
func (ltm *LongTermMemory) GetAllTypes(ctx context.Context) ([]string, error) {
	ltm.mu.RLock()
	defer ltm.mu.RUnlock()

	types := make([]string, 0, len(ltm.entries))
	for t := range ltm.entries {
		types = append(types, t)
	}

	return types, nil
}

// Delete removes a memory entry
func (ltm *LongTermMemory) Delete(ctx context.Context, memoryType, id string) error {
	ltm.mu.Lock()
	defer ltm.mu.Unlock()

	entries := ltm.entries[memoryType]
	if entries == nil {
		return fmt.Errorf("memory type '%s' not found", memoryType)
	}

	// Find and remove the entry
	newEntries := make([]*MemoryEntry, 0, len(entries))
	found := false
	for _, entry := range entries {
		if entry.ID == id {
			found = true
			continue
		}
		newEntries = append(newEntries, entry)
	}

	if !found {
		return fmt.Errorf("memory entry with ID '%s' not found", id)
	}

	ltm.entries[memoryType] = newEntries

	// Persist to disk
	if err := ltm.persistType(memoryType); err != nil {
		return fmt.Errorf("failed to persist after delete: %w", err)
	}

	return nil
}

// Clear removes all entries of a specific type
func (ltm *LongTermMemory) Clear(ctx context.Context, memoryType string) error {
	ltm.mu.Lock()
	defer ltm.mu.Unlock()

	delete(ltm.entries, memoryType)

	// Delete from disk
	if err := ltm.storage.Delete(memoryType); err != nil {
		return fmt.Errorf("failed to delete from disk: %w", err)
	}

	return nil
}

// ClearAll removes all memory entries
func (ltm *LongTermMemory) ClearAll(ctx context.Context) error {
	ltm.mu.Lock()
	defer ltm.mu.Unlock()

	// Clear in-memory
	ltm.entries = make(map[string][]*MemoryEntry)

	// Clear all files from disk
	types, err := ltm.storage.List("")
	if err != nil {
		return fmt.Errorf("failed to list memory types: %w", err)
	}

	for _, t := range types {
		if err := ltm.storage.Delete(t); err != nil {
			return fmt.Errorf("failed to delete %s: %w", t, err)
		}
	}

	return nil
}

// loadFromDisk loads all memories from disk
func (ltm *LongTermMemory) loadFromDisk() error {
	types, err := ltm.storage.List("")
	if err != nil {
		return err
	}

	for _, memoryType := range types {
		var entries []*MemoryEntry
		if err := ltm.storage.Load(memoryType, &entries); err != nil {
			return fmt.Errorf("failed to load %s: %w", memoryType, err)
		}

		// Filter expired entries
		now := time.Now()
		validEntries := make([]*MemoryEntry, 0, len(entries))
		for _, entry := range entries {
			if entry.ExpiresAt != "" {
				expiresAt, err := time.Parse(time.RFC3339, entry.ExpiresAt)
				if err == nil && now.After(expiresAt) {
					continue // Skip expired entries
				}
			}
			validEntries = append(validEntries, entry)
		}

		ltm.entries[memoryType] = validEntries
	}

	return nil
}

// persistType saves entries of a specific type to disk
func (ltm *LongTermMemory) persistType(memoryType string) error {
	entries := ltm.entries[memoryType]
	if entries == nil {
		entries = []*MemoryEntry{}
	}

	return ltm.storage.Save(memoryType, entries)
}

// StateDict returns the state for serialization
func (ltm *LongTermMemory) StateDict() map[string]any {
	ltm.mu.RLock()
	defer ltm.mu.RUnlock()

	typeCounts := make(map[string]int)
	for t, entries := range ltm.entries {
		typeCounts[t] = len(entries)
	}

	return map[string]any{
		"storage_path": ltm.config.StoragePath,
		"max_entries":  ltm.config.MaxEntries,
		"ttl":          ltm.config.TTL.String(),
		"type_counts":  typeCounts,
	}
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		len(s) > len(substr) && containsIgnoreCase(s, substr))
}

func containsIgnoreCase(s, substr string) bool {
	// Simple case-insensitive contains
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			c1 := s[i+j]
			c2 := substr[j]
			if c1 >= 'A' && c1 <= 'Z' {
				c1 += 32
			}
			if c2 >= 'A' && c2 <= 'Z' {
				c2 += 32
			}
			if c1 != c2 {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
