package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeywordExtractor_Extract(t *testing.T) {
	extractor := NewKeywordExtractor()

	tests := []struct {
		input    string
		expected []string
	}{
		{
			input:    "Please help me with the database",
			expected: []string{"database"},
		},
		{
			input:    "Search for files using grep pattern",
			expected: []string{"search", "files", "grep", "pattern"},
		},
		{
			input:    "The quick brown fox",
			expected: []string{"quick", "brown", "fox"},
		},
		{
			input:    "I need to fetch data from the web API",
			expected: []string{"fetch", "data", "web", "api"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractor.Extract(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestKeywordExtractor_MatchScore(t *testing.T) {
	extractor := NewKeywordExtractor()

	tests := []struct {
		input          string
		serverKeywords []string
		expectedScore  float64
	}{
		{
			input:          "search for files",
			serverKeywords: []string{"search", "files", "grep"},
			expectedScore:  1.0, // All input keywords match
		},
		{
			input:          "database query",
			serverKeywords: []string{"search", "files"},
			expectedScore:  0.0, // No match
		},
		{
			input:          "search and grep pattern",
			serverKeywords: []string{"search", "files"},
			expectedScore:  0.33, // 1 out of 3 matches
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			score := extractor.MatchScore(tt.input, tt.serverKeywords)
			assert.InDelta(t, tt.expectedScore, score, 0.01)
		})
	}
}

func TestKeywordExtractor_FindMatches(t *testing.T) {
	extractor := NewKeywordExtractor()

	servers := map[string][]string{
		"filesystem": {"search", "files", "read", "write"},
		"database":   {"query", "sql", "table", "database"},
		"web":        {"fetch", "http", "url", "web"},
	}

	tests := []struct {
		name      string
		input     string
		threshold float64
		expected  []string
	}{
		{
			name:      "filesystem match",
			input:     "search for files",
			threshold: 0.3,
			expected:  []string{"filesystem"},
		},
		{
			name:      "database match",
			input:     "query the database",
			threshold: 0.3,
			expected:  []string{"database"},
		},
		{
			name:      "web match",
			input:     "fetch from web url",
			threshold: 0.3,
			expected:  []string{"web"},
		},
		{
			name:      "no match below threshold",
			input:     "something completely different",
			threshold: 0.3,
			expected:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := extractor.FindMatches(tt.input, servers, tt.threshold)
			var matchNames []string
			for _, m := range matches {
				matchNames = append(matchNames, m.ServerName)
			}
			assert.Equal(t, tt.expected, matchNames)
		})
	}
}

func TestLazyLoader_IsLoaded(t *testing.T) {
	registry := NewRegistry()
	loader := NewLazyLoader(registry, DefaultLazyLoadingConfig())

	// Initially not loaded
	assert.False(t, loader.IsLoaded("test-server"))

	// Mark as loaded (simulating load)
	loader.mu.Lock()
	loader.loadedServers["test-server"] = true
	loader.mu.Unlock()

	assert.True(t, loader.IsLoaded("test-server"))
}

func TestLazyLoader_SetServerKeywords(t *testing.T) {
	registry := NewRegistry()
	loader := NewLazyLoader(registry, DefaultLazyLoadingConfig())

	loader.SetServerKeywords("filesystem", []string{"search", "files"})
	loader.SetServerKeywords("database", []string{"query", "sql"})

	loader.mu.RLock()
	defer loader.mu.RUnlock()

	assert.Equal(t, []string{"search", "files"}, loader.serverKeywords["filesystem"])
	assert.Equal(t, []string{"query", "sql"}, loader.serverKeywords["database"])
}

func TestLazyLoader_SetPreloadChain(t *testing.T) {
	registry := NewRegistry()
	loader := NewLazyLoader(registry, DefaultLazyLoadingConfig())

	loader.SetPreloadChain("main", []string{"base", "common"})

	loader.mu.RLock()
	defer loader.mu.RUnlock()

	assert.Equal(t, []string{"base", "common"}, loader.preloadChains["main"])
}

func TestLazyLoader_AnalyzeInput_Disabled(t *testing.T) {
	registry := NewRegistry()
	config := LazyLoadingConfig{Enabled: false}
	loader := NewLazyLoader(registry, config)

	loader.SetServerKeywords("test", []string{"keyword"})

	decision, err := loader.AnalyzeInput(nil, "keyword search")
	assert.NoError(t, err)
	assert.False(t, decision.ShouldLoad)
}

func TestLazyLoader_AnalyzeInput_EmptyInput(t *testing.T) {
	registry := NewRegistry()
	loader := NewLazyLoader(registry, DefaultLazyLoadingConfig())

	decision, err := loader.AnalyzeInput(nil, "")
	assert.NoError(t, err)
	assert.False(t, decision.ShouldLoad)
}

func TestLazyLoader_AnalyzeInput_NoMatch(t *testing.T) {
	registry := NewRegistry()
	loader := NewLazyLoader(registry, DefaultLazyLoadingConfig())

	loader.SetServerKeywords("filesystem", []string{"search", "files"})

	decision, err := loader.AnalyzeInput(nil, "completely unrelated query")
	assert.NoError(t, err)
	assert.False(t, decision.ShouldLoad)
}

func TestLazyLoader_AnalyzeInput_WithMatch(t *testing.T) {
	registry := NewRegistry()
	loader := NewLazyLoader(registry, DefaultLazyLoadingConfig())

	loader.SetServerKeywords("filesystem", []string{"search", "files", "grep"})

	decision, err := loader.AnalyzeInput(nil, "search for files")
	assert.NoError(t, err)
	assert.True(t, decision.ShouldLoad)
	assert.Contains(t, decision.ServersToLoad, "filesystem")
}

func TestLazyLoader_AnalyzeInput_WithPreload(t *testing.T) {
	registry := NewRegistry()
	loader := NewLazyLoader(registry, DefaultLazyLoadingConfig())

	loader.SetServerKeywords("main", []string{"main", "primary"})
	loader.SetPreloadChain("main", []string{"base", "common"})

	decision, err := loader.AnalyzeInput(nil, "use main server")
	assert.NoError(t, err)
	assert.True(t, decision.ShouldLoad)
	assert.Contains(t, decision.ServersToLoad, "main")
	assert.Contains(t, decision.Preloads, "base")
	assert.Contains(t, decision.Preloads, "common")
}

func TestLazyLoader_GetLoadedServers(t *testing.T) {
	registry := NewRegistry()
	loader := NewLazyLoader(registry, DefaultLazyLoadingConfig())

	// Initially empty
	assert.Empty(t, loader.GetLoadedServers())

	// Add some loaded servers
	loader.mu.Lock()
	loader.loadedServers["server1"] = true
	loader.loadedServers["server2"] = true
	loader.mu.Unlock()

	servers := loader.GetLoadedServers()
	assert.Len(t, servers, 2)
	assert.Contains(t, servers, "server1")
	assert.Contains(t, servers, "server2")
}

func TestLazyLoader_GetLoadHistory(t *testing.T) {
	registry := NewRegistry()
	loader := NewLazyLoader(registry, DefaultLazyLoadingConfig())

	// Initially empty
	assert.Empty(t, loader.GetLoadHistory())

	// Add history entries
	loader.mu.Lock()
	loader.loadHistory = append(loader.loadHistory, LoadResult{
		ServerName: "test",
		Success:    true,
	})
	loader.mu.Unlock()

	history := loader.GetLoadHistory()
	assert.Len(t, history, 1)
	assert.Equal(t, "test", history[0].ServerName)
}

func TestLazyLoader_Reset(t *testing.T) {
	registry := NewRegistry()
	loader := NewLazyLoader(registry, DefaultLazyLoadingConfig())

	// Add some state
	loader.mu.Lock()
	loader.loadedServers["server1"] = true
	loader.loadHistory = append(loader.loadHistory, LoadResult{ServerName: "test"})
	loader.mu.Unlock()

	// Reset
	loader.Reset()

	// Verify cleared
	assert.Empty(t, loader.GetLoadedServers())
	assert.Empty(t, loader.GetLoadHistory())
}

func TestDefaultLazyLoadingConfig(t *testing.T) {
	config := DefaultLazyLoadingConfig()

	assert.True(t, config.Enabled)
	assert.Equal(t, 0.3, config.AutoLoadThreshold)
	assert.Equal(t, 3, config.MaxConcurrentLoads)
}
