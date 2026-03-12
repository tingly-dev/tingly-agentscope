package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bmatcuk/doublestar"
	"github.com/fsnotify/fsnotify"
)

// Event represents a file system event
type Event struct {
	Path     string
	Op       Op
	FileInfo os.FileInfo
}

// Op represents the type of file operation
type Op uint32

const (
	Create Op = 1 << iota
	Write
	Remove
	Rename
	Chmod
)

func (op Op) String() string {
	var ops []string
	if op&Create != 0 {
		ops = append(ops, "CREATE")
	}
	if op&Write != 0 {
		ops = append(ops, "WRITE")
	}
	if op&Remove != 0 {
		ops = append(ops, "REMOVE")
	}
	if op&Rename != 0 {
		ops = append(ops, "RENAME")
	}
	if op&Chmod != 0 {
		ops = append(ops, "CHMOD")
	}
	return strings.Join(ops, "|")
}

// Handler is called when a file event occurs
type Handler func(event Event) error

// Config holds watcher configuration
type Config struct {
	// Root directory to watch
	Root string
	// IgnorePatterns are glob patterns for files/directories to ignore
	IgnorePatterns []string
	// DebounceInterval is the time to wait before triggering handler
	DebounceInterval time.Duration
	// Recursive enables watching subdirectories
	Recursive bool
}

// DefaultConfig returns a default configuration
func DefaultConfig(root string) *Config {
	return &Config{
		Root:             root,
		IgnorePatterns:   DefaultIgnorePatterns(),
		DebounceInterval: 500 * time.Millisecond,
		Recursive:        true,
	}
}

// DefaultIgnorePatterns returns common ignore patterns
func DefaultIgnorePatterns() []string {
	return []string{
		".git/**",
		"node_modules/**",
		"vendor/**",
		".idea/**",
		".vscode/**",
		"*.tmp",
		"*.log",
		".DS_Store",
		"Thumbs.db",
		"__pycache__/**",
		"*.pyc",
		"*.pyo",
		"*.class",
		"target/**",
		"build/**",
		"dist/**",
		".env*",
	}
}

// Watcher watches files and directories for changes
type Watcher struct {
	config    *Config
	handler   Handler
	fsWatcher *fsnotify.Watcher
	debouncer *Debouncer
	mu        sync.RWMutex
	watching  map[string]bool
	ctx       context.Context
	cancel    context.CancelFunc
}

// New creates a new Watcher
func New(config *Config, handler Handler) (*Watcher, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if handler == nil {
		return nil, fmt.Errorf("handler is required")
	}

	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	w := &Watcher{
		config:    config,
		handler:   handler,
		fsWatcher: fsWatcher,
		watching:  make(map[string]bool),
		ctx:       ctx,
		cancel:    cancel,
	}

	w.debouncer = NewDebouncer(config.DebounceInterval, w.handleDebouncedEvent)

	return w, nil
}

// Start begins watching for file changes
func (w *Watcher) Start() error {
	// Add root directory and all subdirectories if recursive
	if err := w.addWatchPaths(w.config.Root); err != nil {
		return err
	}

	// Start event processing
	go w.processEvents()

	return nil
}

// Stop stops watching for file changes
func (w *Watcher) Stop() error {
	w.cancel()
	w.debouncer.Stop()
	return w.fsWatcher.Close()
}

// addWatchPaths adds paths to the watcher recursively
func (w *Watcher) addWatchPaths(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip ignored paths
		if w.shouldIgnore(path) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Watch directories
		if info.IsDir() {
			// Only watch subdirectories if recursive, or if it's the root
			if !w.config.Recursive && path != root {
				return filepath.SkipDir
			}

			if err := w.fsWatcher.Add(path); err != nil {
				return fmt.Errorf("failed to watch %s: %w", path, err)
			}

			w.mu.Lock()
			w.watching[path] = true
			w.mu.Unlock()
		}

		return nil
	})
}

// shouldIgnore checks if a path should be ignored
func (w *Watcher) shouldIgnore(path string) bool {
	rel, err := filepath.Rel(w.config.Root, path)
	if err != nil {
		return false
	}

	for _, pattern := range w.config.IgnorePatterns {
		matched, err := doublestar.Match(pattern, rel)
		if err != nil {
			continue
		}
		if matched {
			return true
		}

		// Also try matching against the filename
		matched, err = doublestar.Match(pattern, filepath.Base(path))
		if err != nil {
			continue
		}
		if matched {
			return true
		}
	}

	return false
}

// processEvents processes fsnotify events
func (w *Watcher) processEvents() {
	for {
		select {
		case <-w.ctx.Done():
			return

		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}
			w.handleFSEvent(event)

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)
		}
	}
}

// handleFSEvent converts fsnotify.Event to our Event and debounces it
func (w *Watcher) handleFSEvent(fsEvent fsnotify.Event) {
	// Skip ignored paths
	if w.shouldIgnore(fsEvent.Name) {
		return
	}

	// Convert fsnotify.Op to our Op
	var op Op
	if fsEvent.Op&fsnotify.Create != 0 {
		op |= Create
	}
	if fsEvent.Op&fsnotify.Write != 0 {
		op |= Write
	}
	if fsEvent.Op&fsnotify.Remove != 0 {
		op |= Remove
	}
	if fsEvent.Op&fsnotify.Rename != 0 {
		op |= Rename
	}
	if fsEvent.Op&fsnotify.Chmod != 0 {
		op |= Chmod
	}

	// Get file info
	fileInfo, _ := os.Stat(fsEvent.Name)

	// Handle new directories (add them to watcher if recursive)
	if op&Create != 0 && fileInfo != nil && fileInfo.IsDir() && w.config.Recursive {
		w.addWatchPaths(fsEvent.Name)
	}

	// Debounce the event
	w.debouncer.Add(Event{
		Path:     fsEvent.Name,
		Op:       op,
		FileInfo: fileInfo,
	})
}

// handleDebouncedEvent is called after debouncing
func (w *Watcher) handleDebouncedEvent(event Event) {
	if err := w.handler(event); err != nil {
		fmt.Fprintf(os.Stderr, "Handler error for %s: %v\n", event.Path, err)
	}
}

// IsWatching returns true if the path is being watched
func (w *Watcher) IsWatching(path string) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.watching[path]
}

// Watching returns a list of watched paths
func (w *Watcher) Watching() []string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	paths := make([]string, 0, len(w.watching))
	for path := range w.watching {
		paths = append(paths, path)
	}
	return paths
}
