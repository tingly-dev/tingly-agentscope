package mcp

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// LoadingDecision represents a decision about which servers to load
type LoadingDecision struct {
	ShouldLoad      bool
	ServersToLoad   []string
	Preloads        []string
	EstimatedTokens int
	Reason          string
}

// LoadResult represents the result of loading a server
type LoadResult struct {
	ServerName     string
	Success        bool
	ToolsLoaded    []string
	TokensConsumed int
	Error          string
	LoadTimeMs     int64
}

// LazyLoadingConfig configures lazy loading behavior
type LazyLoadingConfig struct {
	Enabled            bool
	AutoLoadThreshold  float64
	MaxConcurrentLoads int
}

// DefaultLazyLoadingConfig returns a default configuration
func DefaultLazyLoadingConfig() LazyLoadingConfig {
	return LazyLoadingConfig{
		Enabled:            true,
		AutoLoadThreshold:  0.3,
		MaxConcurrentLoads: 3,
	}
}

// LazyLoader manages on-demand loading of MCP servers
type LazyLoader struct {
	registry         *Registry
	keywordExtractor *KeywordExtractor
	config           LazyLoadingConfig
	loadHistory      []LoadResult
	loadedServers    map[string]bool
	serverKeywords   map[string][]string
	preloadChains    map[string][]string
	mu               sync.RWMutex
}

// NewLazyLoader creates a new lazy loader
func NewLazyLoader(registry *Registry, config LazyLoadingConfig) *LazyLoader {
	return &LazyLoader{
		registry:         registry,
		keywordExtractor: NewKeywordExtractor(),
		config:           config,
		loadHistory:      make([]LoadResult, 0),
		loadedServers:    make(map[string]bool),
		serverKeywords:   make(map[string][]string),
		preloadChains:    make(map[string][]string),
	}
}

// SetServerKeywords sets keywords for a server to use in matching
func (l *LazyLoader) SetServerKeywords(serverName string, keywords []string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.serverKeywords[serverName] = keywords
}

// SetPreloadChain sets servers that should be loaded when a server is loaded
func (l *LazyLoader) SetPreloadChain(serverName string, preloads []string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.preloadChains[serverName] = preloads
}

// AnalyzeInput analyzes user input and decides which servers to load
func (l *LazyLoader) AnalyzeInput(ctx context.Context, userInput string) (*LoadingDecision, error) {
	if !l.config.Enabled || userInput == "" {
		return &LoadingDecision{}, nil
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	// Find matching servers based on keywords
	matches := l.keywordExtractor.FindMatches(userInput, l.serverKeywords, l.config.AutoLoadThreshold)

	// Filter out already loaded servers
	var newMatches []ServerMatch
	for _, m := range matches {
		if !l.loadedServers[m.ServerName] {
			newMatches = append(newMatches, m)
		}
	}

	if len(newMatches) == 0 {
		return &LoadingDecision{}, nil
	}

	// Build decision
	decision := &LoadingDecision{
		ShouldLoad:    true,
		ServersToLoad: make([]string, 0, len(newMatches)),
		Reason:        fmt.Sprintf("Keyword match: %s (score: %.2f)", newMatches[0].ServerName, newMatches[0].Score),
	}

	for _, m := range newMatches {
		decision.ServersToLoad = append(decision.ServersToLoad, m.ServerName)
	}

	// Add preloads
	preloadSet := make(map[string]bool)
	for _, serverName := range decision.ServersToLoad {
		if preloads, ok := l.preloadChains[serverName]; ok {
			for _, p := range preloads {
				if !l.loadedServers[p] && !preloadSet[p] {
					preloadSet[p] = true
					decision.Preloads = append(decision.Preloads, p)
				}
			}
		}
	}

	// Estimate tokens (rough estimation)
	decision.EstimatedTokens = len(userInput) * 2

	return decision, nil
}

// LoadServer loads a specific server
func (l *LazyLoader) LoadServer(ctx context.Context, serverName string, isPreload bool) (*LoadResult, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if already loaded
	if l.loadedServers[serverName] {
		return &LoadResult{
			ServerName: serverName,
			Success:    true,
		}, nil
	}

	startTime := time.Now()

	// Connect via registry
	if err := l.registry.Connect(ctx, serverName); err != nil {
		result := &LoadResult{
			ServerName: serverName,
			Success:    false,
			Error:      err.Error(),
			LoadTimeMs: time.Since(startTime).Milliseconds(),
		}
		l.loadHistory = append(l.loadHistory, *result)
		return result, err
	}

	// Get client to list tools
	client, err := l.registry.GetClient(serverName)
	if err != nil {
		result := &LoadResult{
			ServerName: serverName,
			Success:    false,
			Error:      err.Error(),
			LoadTimeMs: time.Since(startTime).Milliseconds(),
		}
		l.loadHistory = append(l.loadHistory, *result)
		return result, err
	}

	// Get tools list
	tools, err := client.ListTools(ctx)
	if err != nil {
		result := &LoadResult{
			ServerName: serverName,
			Success:    false,
			Error:      err.Error(),
			LoadTimeMs: time.Since(startTime).Milliseconds(),
		}
		l.loadHistory = append(l.loadHistory, *result)
		return result, err
	}

	toolNames := make([]string, len(tools))
	for i, t := range tools {
		toolNames[i] = t.Name
	}

	// Mark as loaded
	l.loadedServers[serverName] = true

	result := &LoadResult{
		ServerName:     serverName,
		Success:        true,
		ToolsLoaded:    toolNames,
		TokensConsumed: len(toolNames) * 100, // Rough estimate
		LoadTimeMs:     time.Since(startTime).Milliseconds(),
	}

	l.loadHistory = append(l.loadHistory, *result)
	return result, nil
}

// IsLoaded checks if a server is loaded
func (l *LazyLoader) IsLoaded(serverName string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.loadedServers[serverName]
}

// GetLoadHistory returns the load history
func (l *LazyLoader) GetLoadHistory() []LoadResult {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Return a copy
	history := make([]LoadResult, len(l.loadHistory))
	copy(history, l.loadHistory)
	return history
}

// GetLoadedServers returns list of loaded servers
func (l *LazyLoader) GetLoadedServers() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	servers := make([]string, 0, len(l.loadedServers))
	for name, loaded := range l.loadedServers {
		if loaded {
			servers = append(servers, name)
		}
	}
	return servers
}

// UnloadServer unloads a specific server
func (l *LazyLoader) UnloadServer(serverName string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.registry.Disconnect(serverName); err != nil {
		return err
	}

	delete(l.loadedServers, serverName)
	return nil
}

// Reset clears all loaded servers and history
func (l *LazyLoader) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.loadedServers = make(map[string]bool)
	l.loadHistory = make([]LoadResult, 0)
}
