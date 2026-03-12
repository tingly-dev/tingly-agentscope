package index

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/tingly-dev/lucybot/internal/watcher"
)

// Index manages code indexing with file watching
type Index struct {
	root      string
	dbPath    string
	watcher   *watcher.Watcher
	touched   map[string]time.Time
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// Config holds index configuration
type Config struct {
	Root           string
	DBPath         string
	Watch          bool
	IgnorePatterns []string
}

// New creates a new Index
func New(cfg *Config) (*Index, error) {
	if cfg.Root == "" {
		cfg.Root = "."
	}
	if cfg.DBPath == "" {
		cfg.DBPath = filepath.Join(cfg.Root, ".lucybot", "index.db")
	}

	ctx, cancel := context.WithCancel(context.Background())

	idx := &Index{
		root:    cfg.Root,
		dbPath:  cfg.DBPath,
		touched: make(map[string]time.Time),
		ctx:     ctx,
		cancel:  cancel,
	}

	// Create watcher if enabled
	if cfg.Watch {
		watchConfig := watcher.DefaultConfig(cfg.Root)
		if len(cfg.IgnorePatterns) > 0 {
			watchConfig.IgnorePatterns = cfg.IgnorePatterns
		}

		w, err := watcher.New(watchConfig, idx.handleFileChange)
		if err != nil {
			return nil, fmt.Errorf("failed to create watcher: %w", err)
		}
		idx.watcher = w
	}

	return idx, nil
}

// Build builds the initial index
func (idx *Index) Build() error {
	fmt.Printf("🔍 Building index for: %s\n", idx.root)

	// Ensure index directory exists
	dir := filepath.Dir(idx.dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create index directory: %w", err)
	}

	// Walk directory and collect file info
	count := 0
	err := filepath.Walk(idx.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip directories and hidden files
		if info.IsDir() {
			if shouldSkipDir(path) {
				return filepath.SkipDir
			}
			return nil
		}

		if shouldSkipFile(path) {
			return nil
		}

		idx.mu.Lock()
		idx.touched[path] = info.ModTime()
		idx.mu.Unlock()

		count++
		if count%100 == 0 {
			fmt.Printf("  Indexed %d files...\r", count)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	fmt.Printf("✅ Indexed %d files\n", count)

	// Start watching if configured
	if idx.watcher != nil {
		if err := idx.watcher.Start(); err != nil {
			return fmt.Errorf("failed to start watcher: %w", err)
		}
		fmt.Println("👁️  Watching for changes...")
	}

	return nil
}

// Stop stops the index and watcher
func (idx *Index) Stop() error {
	idx.cancel()
	if idx.watcher != nil {
		return idx.watcher.Stop()
	}
	return nil
}

// handleFileChange handles file changes from the watcher
func (idx *Index) handleFileChange(event watcher.Event) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	switch {
	case event.Op&watcher.Remove != 0:
		delete(idx.touched, event.Path)
		fmt.Printf("🗑️  Removed: %s\n", event.Path)

	case event.Op&(watcher.Create|watcher.Write) != 0:
		idx.touched[event.Path] = time.Now()
		fmt.Printf("📝 Updated: %s\n", event.Path)
	}

	return nil
}

// GetModifiedFiles returns files modified since the given time
func (idx *Index) GetModifiedFiles(since time.Time) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	var files []string
	for path, mtime := range idx.touched {
		if mtime.After(since) {
			files = append(files, path)
		}
	}
	return files
}

// Stats returns index statistics
func (idx *Index) Stats() map[string]interface{} {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return map[string]interface{}{
		"root":        idx.root,
		"db_path":     idx.dbPath,
		"file_count":  len(idx.touched),
		"watching":    idx.watcher != nil,
	}
}

// shouldSkipDir returns true if the directory should be skipped
func shouldSkipDir(path string) bool {
	skipped := []string{
		".git",
		"node_modules",
		"vendor",
		".idea",
		".vscode",
		"__pycache__",
		"target",
		"build",
		"dist",
	}

	base := filepath.Base(path)
	for _, s := range skipped {
		if base == s {
			return true
		}
	}
	return false
}

// shouldSkipFile returns true if the file should be skipped
func shouldSkipFile(path string) bool {
	ext := filepath.Ext(path)
	skipped := map[string]bool{
		".tmp":  true,
		".log":  true,
		".exe":  true,
		".dll":  true,
		".so":   true,
		".dylib": true,
		".bin":  true,
		".obj":  true,
		".o":    true,
		".class": true,
		".pyc":  true,
		".pyo":  true,
	}

	if skipped[ext] {
		return true
	}

	// Skip hidden files
	if len(filepath.Base(path)) > 0 && filepath.Base(path)[0] == '.' {
		return true
	}

	return false
}
