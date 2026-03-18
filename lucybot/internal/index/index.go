package index

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/tingly-dev/lucybot/internal/watcher"
)

// Index manages code indexing with file watching and SQLite storage
type Index struct {
	root     string
	dbPath   string
	db       *DB
	watcher  *watcher.Watcher
	registry *ParserRegistry
	touched  map[string]time.Time
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
}

// Config holds index configuration
type Config struct {
	Root           string
	DBPath         string
	Watch          bool
	IgnorePatterns []string
	Languages      []Language // Languages to index (empty = all)
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
		root:     cfg.Root,
		dbPath:   cfg.DBPath,
		touched:  make(map[string]time.Time),
		ctx:      ctx,
		cancel:   cancel,
		registry: NewParserRegistry(),
	}

	// Register default parsers
	idx.registerDefaultParsers()

	// Open database
	db, err := Open(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	idx.db = db

	// Create watcher if enabled
	if cfg.Watch {
		watchConfig := watcher.DefaultConfig(cfg.Root)
		if len(cfg.IgnorePatterns) > 0 {
			watchConfig.IgnorePatterns = cfg.IgnorePatterns
		}

		w, err := watcher.New(watchConfig, idx.handleFileChange)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to create watcher: %w", err)
		}
		idx.watcher = w
	}

	return idx, nil
}

// registerDefaultParsers registers the built-in language parsers
func (idx *Index) registerDefaultParsers() {
	// Import and register parsers
	// The init() functions in the languages package auto-register to DefaultRegistry
	// We copy them to our local registry
	for _, lang := range DefaultRegistry.GetSupportedLanguages() {
		if parser := DefaultRegistry.GetParser(lang); parser != nil {
			idx.registry.Register(parser)
		}
	}
}

// Build builds the initial index
func (idx *Index) Build() error {
	fmt.Printf("🔍 Building code index for: %s\n", idx.root)

	// Ensure index directory exists
	dir := filepath.Dir(idx.dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create index directory: %w", err)
	}

	// Walk directory and index files
	count := 0
	symbolCount := 0

	err := filepath.Walk(idx.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip directories
		if info.IsDir() {
			if shouldSkipDir(path) {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip non-source files
		if shouldSkipFile(path) {
			return nil
		}

		// Check if we have a parser for this file
		if !idx.registry.CanParse(path) {
			return nil
		}

		// Index the file
		symbols, err := idx.indexFile(path, info)
		if err != nil {
			fmt.Printf("  ⚠️  Failed to index %s: %v\n", path, err)
			return nil // Continue with other files
		}

		idx.mu.Lock()
		idx.touched[path] = info.ModTime()
		idx.mu.Unlock()

		count++
		symbolCount += symbols
		if count%100 == 0 {
			fmt.Printf("  Indexed %d files (%d symbols)...\r", count, symbolCount)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	fmt.Printf("✅ Indexed %d files (%d symbols)\n", count, symbolCount)

	// Print statistics
	stats, err := idx.db.GetStats(idx.ctx)
	if err == nil {
		fmt.Printf("📊 Database stats: %d symbols, %d references, %d files\n",
			stats["symbols"], stats["symbol_references"], stats["file_info"])
	}

	// Start watching if configured
	if idx.watcher != nil {
		if err := idx.watcher.Start(); err != nil {
			return fmt.Errorf("failed to start watcher: %w", err)
		}
		fmt.Println("👁️  Watching for changes...")
	}

	return nil
}

// indexFile indexes a single file and returns the number of symbols indexed
func (idx *Index) indexFile(path string, info os.FileInfo) (int, error) {
	// Check if file needs reindexing
	existing, err := idx.db.GetFileInfo(idx.ctx, path)
	if err == nil && existing != nil {
		// Check if file has changed
		if !existing.ModTime.Before(info.ModTime()) {
			return existing.SymbolCount, nil // Skip unchanged files
		}
		// Delete old data
		idx.db.DeleteFile(idx.ctx, path)
	}

	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	// Get parser
	parser := idx.registry.GetParserForFile(path)
	if parser == nil {
		return 0, nil
	}

	// Parse file
	result, err := parser.Parse(idx.ctx, content, path)
	if err != nil {
		return 0, err
	}

	// Save symbols
	for _, symbol := range result.Symbols {
		if err := idx.db.SaveSymbol(idx.ctx, symbol); err != nil {
			return 0, err
		}
	}

	// Save references
	for _, ref := range result.References {
		if err := idx.db.SaveReference(idx.ctx, ref); err != nil {
			return 0, err
		}
	}

	// Save scopes
	for _, scope := range result.Scopes {
		if err := idx.db.SaveScope(idx.ctx, scope); err != nil {
			return 0, err
		}
	}

	// Save file info
	hash := sha256.Sum256(content)
	fileInfo := &FileInfo{
		Path:        path,
		Language:    result.FileInfo.Language,
		Size:        info.Size(),
		ModTime:     info.ModTime(),
		Hash:        fmt.Sprintf("%x", hash[:8]),
		SymbolCount: len(result.Symbols),
		IndexedAt:   time.Now(),
	}
	if err := idx.db.SaveFileInfo(idx.ctx, fileInfo); err != nil {
		return 0, err
	}

	return len(result.Symbols), nil
}

// Stop stops the index and watcher
func (idx *Index) Stop() error {
	idx.cancel()
	if idx.watcher != nil {
		if err := idx.watcher.Stop(); err != nil {
			return err
		}
	}
	if idx.db != nil {
		return idx.db.Close()
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
		idx.db.DeleteFile(idx.ctx, event.Path)
		fmt.Printf("🗑️  Removed: %s\n", event.Path)

	case event.Op&(watcher.Create|watcher.Write) != 0:
		info, err := os.Stat(event.Path)
		if err != nil {
			return err
		}

		symbols, err := idx.indexFile(event.Path, info)
		if err != nil {
			fmt.Printf("⚠️  Failed to index %s: %v\n", event.Path, err)
			return nil
		}

		idx.touched[event.Path] = time.Now()
		action := "Updated"
		if event.Op&watcher.Create != 0 {
			action = "Created"
		}
		fmt.Printf("📝 %s: %s (%d symbols)\n", action, event.Path, symbols)
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

	stats := map[string]interface{}{
		"root":       idx.root,
		"db_path":    idx.dbPath,
		"file_count": len(idx.touched),
		"watching":   idx.watcher != nil,
	}

	// Add database stats if available
	if idx.db != nil {
		dbStats, err := idx.db.GetStats(idx.ctx)
		if err == nil {
			for k, v := range dbStats {
				stats["db_"+k] = v
			}
		}
	}

	return stats
}

// DB returns the underlying database (for advanced queries)
func (idx *Index) DB() *DB {
	return idx.db
}

// FindSymbol finds a symbol by name
func (idx *Index) FindSymbol(name string) ([]*Symbol, error) {
	return idx.db.FindSymbolByName(idx.ctx, name)
}

// FindSymbolByQualifiedName finds a symbol by qualified name
func (idx *Index) FindSymbolByQualifiedName(qname string) ([]*Symbol, error) {
	return idx.db.FindSymbolByQualifiedName(idx.ctx, qname)
}

// SearchSymbols searches symbols by query
func (idx *Index) SearchSymbols(query string, limit int) ([]*Symbol, error) {
	return idx.db.SearchSymbols(idx.ctx, query, limit)
}

// ctx returns the index context
func (idx *Index) Context() context.Context {
	return idx.ctx
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
		".lucybot", // Don't index our own directory
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
		".tmp":   true,
		".log":   true,
		".exe":   true,
		".dll":   true,
		".so":    true,
		".dylib": true,
		".bin":   true,
		".obj":   true,
		".o":     true,
		".class": true,
		".pyc":   true,
		".pyo":   true,
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
