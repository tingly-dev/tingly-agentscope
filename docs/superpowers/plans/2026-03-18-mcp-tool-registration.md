# MCP Tool Registration and Lazy Loading Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement full MCP (Model Context Protocol) tool registration and lazy loading mechanism, allowing LucyBot to dynamically load MCP servers based on user input keywords.

**Architecture:** A lightweight metadata registry maintains MCP server information without active connections. A keyword-based matching system analyzes user input to determine which servers to load on-demand. The lazy loader orchestrates server connections and tool registration, while callbacks notify the UI of loading events.

**Tech Stack:** Go, TOML config, existing MCP client infrastructure in `lucybot/internal/mcp/`

---

## File Structure

| File | Purpose |
|------|---------|
| `lucybot/internal/config/mcp.go` | MCP configuration structs (MCPServerConfig, MCPConfig, LazyLoadingConfig) |
| `lucybot/internal/config/config.go` | Modify to include MCPConfig in main Config struct |
| `lucybot/internal/mcp/tool_registry.go` | Lightweight metadata registry with keyword indexing |
| `lucybot/internal/mcp/keyword_extractor.go` | Extract keywords from tool schemas, calculate relevance scores |
| `lucybot/internal/mcp/lazy_loader.go` | Orchestrate on-demand loading with callbacks |
| `lucybot/internal/mcp/client_manager.go` | Manage active MCP connections (extends existing Registry) |
| `lucybot/internal/mcp/tool_wrapper.go` | Wrap MCP tools for AgentScope toolkit integration |
| `lucybot/internal/tools/init.go` | Add `load_mcp_server` tool for explicit loading |
| `lucybot/internal/agent/agent.go` | Integrate MCP initialization, input analysis, system prompt updates |

---

## Task 1: Add MCP Configuration

**Files:**
- Create: `lucybot/internal/config/mcp.go`
- Modify: `lucybot/internal/config/config.go:54-58` (add MCP field to Config struct)
- Test: `lucybot/internal/config/mcp_test.go`

- [ ] **Step 1.1: Write the failing test**

Create `lucybot/internal/config/mcp_test.go`:
```go
package config

import (
	"testing"
)

func TestMCPServerConfigValidation(t *testing.T) {
	// Test valid config
	validConfig := MCPServerConfig{
		Name:    "test-server",
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
		Enabled: true,
	}
	if err := validConfig.Validate(); err != nil {
		t.Errorf("Expected valid config, got error: %v", err)
	}

	// Test missing name
	invalidConfig := MCPServerConfig{
		Command: "npx",
	}
	if err := invalidConfig.Validate(); err == nil {
		t.Error("Expected error for missing name, got nil")
	}

	// Test missing command
	invalidConfig2 := MCPServerConfig{
		Name: "test-server",
	}
	if err := invalidConfig2.Validate(); err == nil {
		t.Error("Expected error for missing command, got nil")
	}
}

func TestMCPConfigDefaults(t *testing.T) {
	config := MCPConfig{}
	config.ApplyDefaults()

	if !config.LazyLoading.Enabled {
		t.Error("Expected lazy loading enabled by default")
	}
	if config.LazyLoading.AutoLoadThreshold != 0.3 {
		t.Errorf("Expected default threshold 0.3, got %f", config.LazyLoading.AutoLoadThreshold)
	}
}
```

- [ ] **Step 1.2: Run test to verify it fails**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes/lucybot
go test ./internal/config -run TestMCPServerConfigValidation -v
```

Expected: FAIL with "undefined: MCPServerConfig"

- [ ] **Step 1.3: Write minimal implementation**

Create `lucybot/internal/config/mcp.go`:
```go
package config

import (
	"fmt"
)

// MCPServerConfig represents an MCP server configuration with lazy loading support
type MCPServerConfig struct {
	Name        string            `toml:"name" json:"name"`
	Command     string            `toml:"command" json:"command"`
	Args        []string          `toml:"args" json:"args"`
	Env         map[string]string `toml:"env,omitempty" json:"env,omitempty"`
	Enabled     bool              `toml:"enabled" json:"enabled"`
	LazyLoad    *bool             `toml:"lazy_load,omitempty" json:"lazy_load,omitempty"` // nil = use global default
	Triggers    []string          `toml:"triggers" json:"triggers"`                         // Keywords that trigger auto-loading
	PreloadWith []string          `toml:"preload_with" json:"preload_with"`                 // Servers to preload when this loads
}

// Validate validates the server configuration
func (c *MCPServerConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("server name is required")
	}
	if c.Command == "" {
		return fmt.Errorf("command is required for server '%s'", c.Name)
	}
	return nil
}

// ShouldLazyLoad returns whether this server should use lazy loading
func (c *MCPServerConfig) ShouldLazyLoad(globalDefault bool) bool {
	if c.LazyLoad != nil {
		return *c.LazyLoad
	}
	return globalDefault
}

// MCPToolSummary represents lightweight tool metadata for lazy loading
type MCPToolSummary struct {
	Name        string `toml:"name" json:"name"`
	Description string `toml:"description" json:"description"`
}

// MCPServerRegistryEntry stores lightweight metadata for lazy-loaded servers
type MCPServerRegistryEntry struct {
	Tools           []MCPToolSummary `toml:"tools" json:"tools"`
	EstimatedTokens int              `toml:"estimated_tokens" json:"estimated_tokens"`
	Triggers        []string         `toml:"triggers" json:"triggers"`
	PreloadWith     []string         `toml:"preload_with" json:"preload_with"`
}

// LazyLoadingConfig holds global lazy loading settings
type LazyLoadingConfig struct {
	Enabled          bool    `toml:"enabled" json:"enabled"`                       // Master toggle
	AutoLoadThreshold float64 `toml:"auto_load_threshold" json:"auto_load_threshold"` // Confidence threshold (0.0-1.0)
	MaxInitialTokens  int     `toml:"max_initial_tokens" json:"max_initial_tokens"`   // Token budget for registry at startup
}

// MCPConfig holds all MCP-related configuration
type MCPConfig struct {
	Servers     map[string]MCPServerConfig `toml:"servers" json:"servers"`
	LazyLoading LazyLoadingConfig          `toml:"lazy_loading" json:"lazy_loading"`
}

// ApplyDefaults sets default values for MCP configuration
func (c *MCPConfig) ApplyDefaults() {
	if c.Servers == nil {
		c.Servers = make(map[string]MCPServerConfig)
	}
	if c.LazyLoading.AutoLoadThreshold == 0 {
		c.LazyLoading.AutoLoadThreshold = 0.3
	}
	// Default lazy loading enabled
	if !c.LazyLoading.Enabled && c.LazyLoading.MaxInitialTokens == 0 {
		c.LazyLoading.Enabled = true
		c.LazyLoading.MaxInitialTokens = 1000
	}
}

// GetServer returns a server configuration by name
func (c *MCPConfig) GetServer(name string) (MCPServerConfig, bool) {
	server, ok := c.Servers[name]
	return server, ok
}

// GetEnabledServers returns all enabled server names
func (c *MCPConfig) GetEnabledServers() []string {
	var names []string
	for name, server := range c.Servers {
		if server.Enabled {
			names = append(names, name)
		}
	}
	return names
}
```

- [ ] **Step 1.4: Modify main Config struct**

Modify `lucybot/internal/config/config.go` line 54-58:

Find:
```go
// Config holds the complete configuration for LucyBot
type Config struct {
	Agent   AgentConfig   `toml:"agent"`
	Index   IndexConfig   `toml:"index"`
	Session SessionConfig `toml:"session"`
}
```

Replace with:
```go
// Config holds the complete configuration for LucyBot
type Config struct {
	Agent   AgentConfig   `toml:"agent"`
	Index   IndexConfig   `toml:"index"`
	Session SessionConfig `toml:"session"`
	MCP     MCPConfig     `toml:"mcp"`
}
```

- [ ] **Step 1.5: Run test to verify it passes**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes/lucybot
go test ./internal/config -run TestMCPServerConfigValidation -v
go test ./internal/config -run TestMCPConfigDefaults -v
```

Expected: PASS

- [ ] **Step 1.6: Commit**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes
git add lucybot/internal/config/mcp.go lucybot/internal/config/mcp_test.go lucybot/internal/config/config.go
git commit -m "feat(config): add MCP configuration with lazy loading support"
```

---

## Task 2: Create Keyword Extractor

**Files:**
- Create: `lucybot/internal/mcp/keyword_extractor.go`
- Test: `lucybot/internal/mcp/keyword_extractor_test.go`

- [ ] **Step 2.1: Write the failing test**

Create `lucybot/internal/mcp/keyword_extractor_test.go`:
```go
package mcp

import (
	"strings"
	"testing"
)

func TestKeywordExtractorExtractFromTool(t *testing.T) {
	extractor := NewKeywordExtractor()

	tool := Tool{
		Name:        "read_file",
		Description: "Read the contents of a file at the given path. Returns the file content as a string.",
		InputSchema: []byte(`{"type":"object","properties":{"path":{"type":"string","description":"The file path to read"}}}`),
	}

	keywords := extractor.ExtractFromTool(tool)

	// Should extract from name, description, and parameter descriptions
	foundPath := false
	foundFile := false
	for _, kw := range keywords {
		if kw == "path" {
			foundPath = true
		}
		if kw == "file" {
			foundFile = true
		}
	}

	if !foundPath {
		t.Error("Expected 'path' in keywords")
	}
	if !foundFile {
		t.Error("Expected 'file' in keywords")
	}
}

func TestKeywordExtractorCalculateRelevance(t *testing.T) {
	extractor := NewKeywordExtractor()

	keywords := []string{"file", "read", "path", "content"}

	// High relevance - multiple keywords match
	score := extractor.CalculateRelevance("how do I read a file", keywords)
	if score < 0.5 {
		t.Errorf("Expected high relevance for 'read a file', got %f", score)
	}

	// Low relevance - no keywords match
	score = extractor.CalculateRelevance("what's the weather today", keywords)
	if score > 0.2 {
		t.Errorf("Expected low relevance for weather query, got %f", score)
	}
}

func TestKeywordExtractorTokenize(t *testing.T) {
	extractor := NewKeywordExtractor()

	tokens := extractor.Tokenize("Read the FILE path and return content!")

	expected := []string{"read", "file", "path", "return", "content"}
	for _, exp := range expected {
		found := false
		for _, tok := range tokens {
			if tok == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected token '%s' not found in %v", exp, tokens)
		}
	}
}
```

- [ ] **Step 2.2: Run test to verify it fails**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes/lucybot
go test ./internal/mcp -run TestKeywordExtractor -v
```

Expected: FAIL with "undefined: NewKeywordExtractor"

- [ ] **Step 2.3: Write implementation**

Create `lucybot/internal/mcp/keyword_extractor.go`:
```go
package mcp

import (
	"encoding/json"
	"regexp"
	"strings"
)

// KeywordExtractor extracts keywords from tool schemas for intelligent matching
type KeywordExtractor struct {
	stopWords map[string]bool
}

// NewKeywordExtractor creates a new keyword extractor with default stop words
func NewKeywordExtractor() *KeywordExtractor {
	return &KeywordExtractor{
		stopWords: map[string]bool{
			"the": true, "a": true, "an": true, "and": true, "or": true,
			"to": true, "of": true, "in": true, "on": true, "at": true,
			"for": true, "with": true, "by": true, "from": true, "as": true,
			"is": true, "are": true, "was": true, "were": true, "be": true,
			"been": true, "being": true, "have": true, "has": true, "had": true,
			"do": true, "does": true, "did": true, "will": true, "would": true,
			"could": true, "should": true, "may": true, "might": true, "must": true,
			"can": true, "shall": true, "this": true, "that": true, "these": true,
			"those": true, "i": true, "you": true, "he": true, "she": true,
			"it": true, "we": true, "they": true, "me": true, "him": true,
			"her": true, "us": true, "them": true, "my": true, "your": true,
			"his": true, "its": true, "our": true, "their": true,
		},
	}
}

// ExtractFromTool extracts keywords from a tool's name, description, and parameters
func (ke *KeywordExtractor) ExtractFromTool(tool Tool) []string {
	keywordSet := make(map[string]bool)

	// Extract from name
	for _, word := range ke.Tokenize(tool.Name) {
		keywordSet[word] = true
	}

	// Extract from description
	for _, word := range ke.Tokenize(tool.Description) {
		keywordSet[word] = true
	}

	// Extract from input schema properties
	if len(tool.InputSchema) > 0 {
		var schema map[string]interface{}
		if err := json.Unmarshal(tool.InputSchema, &schema); err == nil {
			ke.extractFromSchema(schema, keywordSet)
		}
	}

	// Convert set to slice
	keywords := make([]string, 0, len(keywordSet))
	for kw := range keywordSet {
		keywords = append(keywords, kw)
	}

	return keywords
}

// extractFromSchema recursively extracts keywords from JSON schema
func (ke *KeywordExtractor) extractFromSchema(schema map[string]interface{}, keywords map[string]bool) {
	// Extract from property names and descriptions
	if props, ok := schema["properties"].(map[string]interface{}); ok {
		for name, prop := range props {
			// Add property name
			for _, word := range ke.Tokenize(name) {
				keywords[word] = true
			}

			// Extract from property schema
			if propMap, ok := prop.(map[string]interface{}); ok {
				if desc, ok := propMap["description"].(string); ok {
					for _, word := range ke.Tokenize(desc) {
						keywords[word] = true
					}
				}
			}
		}
	}

	// Recurse into nested objects
	if props, ok := schema["properties"].(map[string]interface{}); ok {
		for _, prop := range props {
			if propMap, ok := prop.(map[string]interface{}); ok {
				ke.extractFromSchema(propMap, keywords)
			}
		}
	}
}

// CalculateRelevance calculates a relevance score (0.0-1.0) between input text and keywords
// Score is based on: coverage (how many keywords matched) and density (how many input tokens matched)
// Formula: (coverage * 0.7) + (density * 0.3)
func (ke *KeywordExtractor) CalculateRelevance(inputText string, keywords []string) float64 {
	if len(keywords) == 0 {
		return 0.0
	}

	inputTokens := ke.Tokenize(inputText)
	if len(inputTokens) == 0 {
		return 0.0
	}

	// Count matches
	matchedKeywords := 0
	matchedTokens := 0

	inputTokenSet := make(map[string]bool)
	for _, token := range inputTokens {
		inputTokenSet[token] = true
	}

	keywordSet := make(map[string]bool)
	for _, kw := range keywords {
		keywordSet[kw] = true
		if inputTokenSet[kw] {
			matchedKeywords++
		}
	}

	for _, token := range inputTokens {
		if keywordSet[token] {
			matchedTokens++
		}
	}

	// Calculate coverage: what fraction of keywords were matched
	coverage := float64(matchedKeywords) / float64(len(keywords))

	// Calculate density: what fraction of input tokens matched keywords
	density := float64(matchedTokens) / float64(len(inputTokens))

	// Weighted combination
	score := (coverage * 0.7) + (density * 0.3)

	return score
}

// Tokenize breaks text into lowercase tokens, removing punctuation and stop words
func (ke *KeywordExtractor) Tokenize(text string) []string {
	// Convert to lowercase
	text = strings.ToLower(text)

	// Remove punctuation and special characters
	re := regexp.MustCompile(`[^a-z0-9\s]`)
	text = re.ReplaceAllString(text, " ")

	// Split into words
	words := strings.Fields(text)

	// Filter out stop words and short words
	var tokens []string
	for _, word := range words {
		if len(word) > 2 && !ke.stopWords[word] {
			tokens = append(tokens, word)
		}
	}

	return tokens
}
```

- [ ] **Step 2.4: Run test to verify it passes**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes/lucybot
go test ./internal/mcp -run TestKeywordExtractor -v
```

Expected: PASS

- [ ] **Step 2.5: Commit**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes
git add lucybot/internal/mcp/keyword_extractor.go lucybot/internal/mcp/keyword_extractor_test.go
git commit -m "feat(mcp): add keyword extractor for intelligent tool matching"
```

---

## Task 3: Create Tool Registry

**Files:**
- Create: `lucybot/internal/mcp/tool_registry.go`
- Test: `lucybot/internal/mcp/tool_registry_test.go`

- [ ] **Step 3.1: Write the failing test**

Create `lucybot/internal/mcp/tool_registry_test.go`:
```go
package mcp

import (
	"testing"

	"github.com/tingly-dev/lucybot/internal/config"
)

func TestToolRegistryLoadFromConfig(t *testing.T) {
	cfg := &config.MCPConfig{
		Servers: map[string]config.MCPServerConfig{
			"filesystem": {
				Name:     "filesystem",
				Command:  "npx",
				Args:     []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
				Enabled:  true,
				LazyLoad: boolPtr(true),
				Triggers: []string{"file", "read", "write"},
			},
		},
		LazyLoading: config.LazyLoadingConfig{
			Enabled:           true,
			AutoLoadThreshold: 0.3,
		},
	}

	registry := NewToolRegistry()
	registry.LoadFromConfig(cfg)

	// Check server was registered
	metadata, ok := registry.GetServerMetadata("filesystem")
	if !ok {
		t.Fatal("Expected filesystem server to be registered")
	}

	if metadata.ServerName != "filesystem" {
		t.Errorf("Expected server name 'filesystem', got '%s'", metadata.ServerName)
	}

	if !metadata.LazyLoad {
		t.Error("Expected lazy load to be true")
	}
}

func TestToolRegistryFindServersByInput(t *testing.T) {
	cfg := &config.MCPConfig{
		Servers: map[string]config.MCPServerConfig{
			"filesystem": {
				Name:     "filesystem",
				Command:  "npx",
				Enabled:  true,
				LazyLoad: boolPtr(true),
				Triggers: []string{"file", "read", "directory"},
			},
			"crypto": {
				Name:     "crypto",
				Command:  "node",
				Args:     []string{"crypto-server.js"},
				Enabled:  true,
				LazyLoad: boolPtr(true),
				Triggers: []string{"bitcoin", "ethereum", "price", "crypto"},
			},
		},
		LazyLoading: config.LazyLoadingConfig{
			Enabled:           true,
			AutoLoadThreshold: 0.3,
		},
	}

	registry := NewToolRegistry()
	registry.LoadFromConfig(cfg)

	// Should find filesystem for file-related query
	matches := registry.FindServersByInput("how do I read a file", 0.3, 10)
	foundFilesystem := false
	for _, match := range matches {
		if match.ServerName == "filesystem" && match.Confidence > 0.3 {
			foundFilesystem = true
		}
	}
	if !foundFilesystem {
		t.Error("Expected to find filesystem server for 'read a file' query")
	}

	// Should find crypto for crypto-related query
	matches = registry.FindServersByInput("what's the bitcoin price", 0.3, 10)
	foundCrypto := false
	for _, match := range matches {
		if match.ServerName == "crypto" && match.Confidence > 0.3 {
			foundCrypto = true
		}
	}
	if !foundCrypto {
		t.Error("Expected to find crypto server for 'bitcoin price' query")
	}
}

func TestToolRegistryGetPreloadChain(t *testing.T) {
	cfg := &config.MCPConfig{
		Servers: map[string]config.MCPServerConfig{
			"database": {
				Name:        "database",
				Command:     "npx",
				Enabled:     true,
				LazyLoad:    boolPtr(true),
				PreloadWith: []string{"orm"},
			},
			"orm": {
				Name:        "orm",
				Command:     "npx",
				Enabled:     true,
				LazyLoad:    boolPtr(true),
				PreloadWith: []string{"types"},
			},
			"types": {
				Name:     "types",
				Command:  "npx",
				Enabled:  true,
				LazyLoad: boolPtr(true),
			},
		},
	}

	registry := NewToolRegistry()
	registry.LoadFromConfig(cfg)

	chain := registry.GetPreloadChain("database")

	// Should include all transitive preloads
	if len(chain) != 2 {
		t.Errorf("Expected preload chain length 2, got %d: %v", len(chain), chain)
	}

	// Check for cycles - should not hang
	cfgWithCycle := &config.MCPConfig{
		Servers: map[string]config.MCPServerConfig{
			"a": {
				Name:        "a",
				Command:     "cmd",
				Enabled:     true,
				PreloadWith: []string{"b"},
			},
			"b": {
				Name:        "b",
				Command:     "cmd",
				Enabled:     true,
				PreloadWith: []string{"a"}, // Cycle!
			},
		},
	}

	registry2 := NewToolRegistry()
	registry2.LoadFromConfig(cfgWithCycle)
	chain2 := registry2.GetPreloadChain("a")
	// Should detect cycle and return partial chain
	if len(chain2) > 2 {
		t.Error("Cycle detection failed - chain too long")
	}
}

func boolPtr(b bool) *bool {
	return &b
}
```

- [ ] **Step 3.2: Run test to verify it fails**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes/lucybot
go test ./internal/mcp -run TestToolRegistry -v
```

Expected: FAIL with "undefined: NewToolRegistry"

- [ ] **Step 3.3: Write implementation**

Create `lucybot/internal/mcp/tool_registry.go`:
```go
package mcp

import (
	"sort"
	"strings"
	"sync"

	"github.com/tingly-dev/lucybot/internal/config"
)

// ServerMetadata holds lightweight metadata for a registered MCP server
type ServerMetadata struct {
	ServerName     string
	LazyLoad       bool
	Triggers       []string
	PreloadWith    []string
	EstimatedTokens int
	ToolCount      int
	Keywords       []string // Aggregated from all tools
	IsLoaded       bool
}

// MatchResult represents a server match with confidence score
type MatchResult struct {
	ServerName string
	Confidence float64
	Metadata   *ServerMetadata
}

// ToolRegistry maintains lightweight metadata for MCP servers without active connections
type ToolRegistry struct {
	mu       sync.RWMutex
	servers  map[string]*ServerMetadata
	keywords map[string][]string // keyword -> []serverName (inverted index)
	extractor *KeywordExtractor
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		servers:   make(map[string]*ServerMetadata),
		keywords:  make(map[string][]string),
		extractor: NewKeywordExtractor(),
	}
}

// LoadFromConfig loads server metadata from configuration
func (tr *ToolRegistry) LoadFromConfig(cfg *config.MCPConfig) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	globalLazyLoad := cfg.LazyLoading.Enabled

	for name, serverCfg := range cfg.Servers {
		if !serverCfg.Enabled {
			continue
		}

		metadata := &ServerMetadata{
			ServerName:  name,
			LazyLoad:    serverCfg.ShouldLazyLoad(globalLazyLoad),
			Triggers:    serverCfg.Triggers,
			PreloadWith: serverCfg.PreloadWith,
			IsLoaded:    false,
		}

		// Aggregate keywords from triggers
		keywordSet := make(map[string]bool)
		for _, trigger := range serverCfg.Triggers {
			for _, word := range tr.extractor.Tokenize(trigger) {
				keywordSet[word] = true
			}
		}

		// Convert to slice
		for kw := range keywordSet {
			metadata.Keywords = append(metadata.Keywords, kw)
		}

		tr.servers[name] = metadata

		// Build inverted index
		for kw := range keywordSet {
			tr.keywords[kw] = append(tr.keywords[kw], name)
		}
	}
}

// FindServersByInput matches user input to servers via keyword index
func (tr *ToolRegistry) FindServersByInput(userInput string, threshold float64, maxResults int) []MatchResult {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	// Get input tokens
	inputTokens := tr.extractor.Tokenize(userInput)

	// Find candidate servers by direct keyword match
	candidateScores := make(map[string]float64)
	for _, token := range inputTokens {
		if servers, ok := tr.keywords[token]; ok {
			for _, serverName := range servers {
				candidateScores[serverName] += 1.0
			}
		}
	}

	// Calculate relevance for each candidate
	var matches []MatchResult
	for serverName, baseScore := range candidateScores {
		metadata := tr.servers[serverName]
		if metadata == nil {
			continue
		}

		// Calculate proper relevance score
		confidence := tr.extractor.CalculateRelevance(userInput, metadata.Keywords)

		// Boost by direct trigger matches
		for _, trigger := range metadata.Triggers {
			triggerTokens := tr.extractor.Tokenize(trigger)
			for _, tt := range triggerTokens {
				for _, it := range inputTokens {
					if strings.Contains(it, tt) || strings.Contains(tt, it) {
						confidence += 0.1 // Small boost for partial matches
					}
				}
			}
		}

		// Cap at 1.0
		if confidence > 1.0 {
			confidence = 1.0
		}

		// Only include if above threshold
		if confidence >= threshold {
			matches = append(matches, MatchResult{
				ServerName: serverName,
				Confidence: confidence,
				Metadata:   metadata,
			})
		}

		_ = baseScore // Use baseScore to avoid unused variable warning
	}

	// Sort by confidence (descending)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Confidence > matches[j].Confidence
	})

	// Limit results
	if maxResults > 0 && len(matches) > maxResults {
		matches = matches[:maxResults]
	}

	return matches
}

// GetPreloadChain resolves transitive preloads, avoiding cycles
func (tr *ToolRegistry) GetPreloadChain(serverName string) []string {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	visited := make(map[string]bool)
	var chain []string

	var visit func(string)
	visit = func(name string) {
		if visited[name] {
			return
		}
		visited[name] = true

		metadata, ok := tr.servers[name]
		if !ok {
			return
		}

		for _, preload := range metadata.PreloadWith {
			if !visited[preload] {
				chain = append(chain, preload)
				visit(preload)
			}
		}
	}

	visit(serverName)
	return chain
}

// GetServerMetadata returns metadata for a server
func (tr *ToolRegistry) GetServerMetadata(name string) (*ServerMetadata, bool) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	metadata, ok := tr.servers[name]
	return metadata, ok
}

// MarkServerLoaded marks a server as loaded
func (tr *ToolRegistry) MarkServerLoaded(name string) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	if metadata, ok := tr.servers[name]; ok {
		metadata.IsLoaded = true
	}
}

// IsServerLoaded checks if a server is loaded
func (tr *ToolRegistry) IsServerLoaded(name string) bool {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	if metadata, ok := tr.servers[name]; ok {
		return metadata.IsLoaded
	}
	return false
}

// GetLazyServers returns all servers configured for lazy loading
func (tr *ToolRegistry) GetLazyServers() []string {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	var names []string
	for name, metadata := range tr.servers {
		if metadata.LazyLoad && !metadata.IsLoaded {
			names = append(names, name)
		}
	}
	return names
}

// GetEagerServers returns all servers that should load immediately
func (tr *ToolRegistry) GetEagerServers() []string {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	var names []string
	for name, metadata := range tr.servers {
		if !metadata.LazyLoad && !metadata.IsLoaded {
			names = append(names, name)
		}
	}
	return names
}

// GetAllServers returns all registered server names
func (tr *ToolRegistry) GetAllServers() []string {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	names := make([]string, 0, len(tr.servers))
	for name := range tr.servers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// UpdateServerKeywords updates keywords for a server (after tool discovery)
func (tr *ToolRegistry) UpdateServerKeywords(serverName string, tools []Tool) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	metadata, ok := tr.servers[serverName]
	if !ok {
		return
	}

	// Extract keywords from discovered tools
	keywordSet := make(map[string]bool)
	for _, tool := range tools {
		toolKeywords := tr.extractor.ExtractFromTool(tool)
		for _, kw := range toolKeywords {
			keywordSet[kw] = true
		}
	}

	// Merge with existing keywords
	for _, kw := range metadata.Keywords {
		keywordSet[kw] = true
	}

	// Update metadata
	metadata.Keywords = make([]string, 0, len(keywordSet))
	for kw := range keywordSet {
		metadata.Keywords = append(metadata.Keywords, kw)
	}
	metadata.ToolCount = len(tools)

	// Rebuild inverted index for this server's keywords
	for kw, servers := range tr.keywords {
		// Remove old entry
		var newServers []string
		for _, s := range servers {
			if s != serverName {
				newServers = append(newServers, s)
			}
		}
		if len(newServers) > 0 {
			tr.keywords[kw] = newServers
		} else {
			delete(tr.keywords, kw)
		}
	}

	// Add new entries
	for kw := range keywordSet {
		tr.keywords[kw] = append(tr.keywords[kw], serverName)
	}
}
```

- [ ] **Step 3.4: Run test to verify it passes**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes/lucybot
go test ./internal/mcp -run TestToolRegistry -v
```

Expected: PASS

- [ ] **Step 3.5: Commit**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes
git add lucybot/internal/mcp/tool_registry.go lucybot/internal/mcp/tool_registry_test.go
git commit -m "feat(mcp): add tool registry with keyword indexing and lazy loading metadata"
```

---

## Task 4: Create Client Manager

**Files:**
- Create: `lucybot/internal/mcp/client_manager.go`
- Test: `lucybot/internal/mcp/client_manager_test.go`

- [ ] **Step 4.1: Write the failing test**

Create `lucybot/internal/mcp/client_manager_test.go`:
```go
package mcp

import (
	"context"
	"testing"

	"github.com/tingly-dev/lucybot/internal/config"
)

func TestClientManagerSingleton(t *testing.T) {
	cm1 := GetClientManager()
	cm2 := GetClientManager()

	if cm1 != cm2 {
		t.Error("Expected GetClientManager to return the same instance")
	}
}

func TestClientManagerInitialize(t *testing.T) {
	cm := NewClientManager()

	cfg := &config.MCPConfig{
		Servers: map[string]config.MCPServerConfig{
			"test-server": {
				Name:    "test-server",
				Command: "echo",
				Args:    []string{"test"},
				Enabled: true,
			},
		},
		LazyLoading: config.LazyLoadingConfig{
			Enabled: true,
		},
	}

	toolRegistry := NewToolRegistry()
	toolRegistry.LoadFromConfig(cfg)

	err := cm.Initialize(cfg, toolRegistry)
	if err != nil {
		t.Errorf("Expected successful initialization, got error: %v", err)
	}

	// Check that config was stored
	if cm.config == nil {
		t.Error("Expected config to be stored")
	}
}
```

- [ ] **Step 4.2: Run test to verify it fails**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes/lucybot
go test ./internal/mcp -run TestClientManager -v
```

Expected: FAIL with "undefined: GetClientManager"

- [ ] **Step 4.3: Write implementation**

Create `lucybot/internal/mcp/client_manager.go`:
```go
package mcp

import (
	"context"
	"fmt"
	"sync"

	"github.com/tingly-dev/lucybot/internal/config"
)

// ClientManager manages active MCP connections (singleton pattern)
type ClientManager struct {
	mu       sync.RWMutex
	registry *Registry // Existing MCP registry for connections
	config   *config.MCPConfig
	toolReg  *ToolRegistry // Lazy loading metadata registry
}

var (
	instance *ClientManager
	once     sync.Once
)

// GetClientManager returns the singleton ClientManager instance
func GetClientManager() *ClientManager {
	once.Do(func() {
		instance = NewClientManager()
	})
	return instance
}

// NewClientManager creates a new ClientManager (use GetClientManager for singleton)
func NewClientManager() *ClientManager {
	return &ClientManager{
		registry: NewRegistry(),
	}
}

// Initialize sets up the client manager with configuration
func (cm *ClientManager) Initialize(cfg *config.MCPConfig, toolReg *ToolRegistry) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.config = cfg
	cm.toolReg = toolReg

	// Register all configured servers (but don't connect yet for lazy ones)
	for name, serverCfg := range cfg.Servers {
		if !serverCfg.Enabled {
			continue
		}

		// Convert to old ServerConfig format for the registry
		oldConfig := &ServerConfig{
			Name:    serverCfg.Name,
			Command: serverCfg.Command,
			Args:    serverCfg.Args,
			Env:     serverCfg.Env,
			Enabled: serverCfg.Enabled,
		}

		if err := cm.registry.Register(oldConfig); err != nil {
			return fmt.Errorf("failed to register server '%s': %w", name, err)
		}
	}

	return nil
}

// ConnectServer connects to a specific MCP server and discovers its tools
func (cm *ClientManager) ConnectServer(ctx context.Context, name string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Connect via existing registry
	if err := cm.registry.Connect(ctx, name); err != nil {
		return fmt.Errorf("failed to connect to server '%s': %w", name, err)
	}

	// Get client to discover tools
	client, err := cm.registry.GetClient(name)
	if err != nil {
		return fmt.Errorf("failed to get client for '%s': %w", name, err)
	}

	// List tools to update registry with discovered keywords
	tools, err := client.ListTools(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tools for '%s': %w", name, err)
	}

	// Update tool registry with discovered tools
	if cm.toolReg != nil {
		cm.toolReg.UpdateServerKeywords(name, tools)
		cm.toolReg.MarkServerLoaded(name)
	}

	return nil
}

// DisconnectServer disconnects from a server
func (cm *ClientManager) DisconnectServer(name string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	return cm.registry.Disconnect(name)
}

// GetClient returns the MCP client for a connected server
func (cm *ClientManager) GetClient(name string) (*Client, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.registry.GetClient(name)
}

// IsConnected checks if a server is connected
func (cm *ClientManager) IsConnected(name string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	_, err := cm.registry.GetClient(name)
	return err == nil
}

// GetConnectedServers returns all connected server names
func (cm *ClientManager) GetConnectedServers() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.registry.ListConnected()
}

// GetAllTools returns all tools from all connected servers
func (cm *ClientManager) GetAllTools() []ToolInfo {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	adapter := NewToolAdapter(cm.registry)
	return adapter.GetAllTools()
}

// CallTool executes a tool call on a connected server
func (cm *ClientManager) CallTool(ctx context.Context, fullName string, arguments map[string]interface{}) (*ToolCallResult, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	adapter := NewToolAdapter(cm.registry)
	resp, err := adapter.Call(ctx, fullName, arguments)
	if err != nil {
		return nil, err
	}

	// Convert ToolResponse back to ToolCallResult
	// This is a simplified conversion - in practice you might need more handling
	return &ToolCallResult{
		Content: []Content{
			{Type: "text", Text: resp.Content[0].Text},
		},
	}, nil
}

// Reset clears all connections (useful for testing)
func (cm *ClientManager) Reset() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.registry.DisconnectAll()
	cm.registry = NewRegistry()
}
```

- [ ] **Step 4.4: Run test to verify it passes**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes/lucybot
go test ./internal/mcp -run TestClientManager -v
```

Expected: PASS

- [ ] **Step 4.5: Commit**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes
git add lucybot/internal/mcp/client_manager.go lucybot/internal/mcp/client_manager_test.go
git commit -m "feat(mcp): add client manager for managing MCP connections"
```

---

## Task 5: Create Lazy Loader

**Files:**
- Create: `lucybot/internal/mcp/lazy_loader.go`
- Test: `lucybot/internal/mcp/lazy_loader_test.go`

- [ ] **Step 5.1: Write the failing test**

Create `lucybot/internal/mcp/lazy_loader_test.go`:
```go
package mcp

import (
	"context"
	"testing"

	"github.com/tingly-dev/lucybot/internal/config"
)

func TestLazyLoaderAnalyzeInput(t *testing.T) {
	cfg := &config.MCPConfig{
		Servers: map[string]config.MCPServerConfig{
			"filesystem": {
				Name:     "filesystem",
				Command:  "npx",
				Enabled:  true,
				LazyLoad: boolPtr(true),
				Triggers: []string{"file", "read", "write", "directory"},
			},
		},
		LazyLoading: config.LazyLoadingConfig{
			Enabled:           true,
			AutoLoadThreshold: 0.3,
		},
	}

	toolReg := NewToolRegistry()
	toolReg.LoadFromConfig(cfg)

	cm := NewClientManager()
	cm.Initialize(cfg, toolReg)

	loader := NewLazyLoader(toolReg, cm, cfg.LazyLoading)

	// Analyze input that should trigger filesystem server
	decision := loader.AnalyzeInput("how do I read a file")

	if len(decision.ServersToLoad) == 0 {
		t.Error("Expected servers to be suggested for file-related query")
	}

	foundFilesystem := false
	for _, server := range decision.ServersToLoad {
		if server == "filesystem" {
			foundFilesystem = true
			break
		}
	}
	if !foundFilesystem {
		t.Error("Expected filesystem server to be suggested")
	}
}

func TestLazyLoaderPreloadChain(t *testing.T) {
	cfg := &config.MCPConfig{
		Servers: map[string]config.MCPServerConfig{
			"database": {
				Name:        "database",
				Command:     "npx",
				Enabled:     true,
				LazyLoad:    boolPtr(true),
				PreloadWith: []string{"orm"},
			},
			"orm": {
				Name:     "orm",
				Command:  "npx",
				Enabled:  true,
				LazyLoad: boolPtr(true),
			},
		},
		LazyLoading: config.LazyLoadingConfig{
			Enabled:           true,
			AutoLoadThreshold: 0.3,
		},
	}

	toolReg := NewToolRegistry()
	toolReg.LoadFromConfig(cfg)

	cm := NewClientManager()
	cm.Initialize(cfg, toolReg)

	loader := NewLazyLoader(toolReg, cm, cfg.LazyLoading)

	// The preload chain should be resolved
	chain := toolReg.GetPreloadChain("database")
	if len(chain) != 1 || chain[0] != "orm" {
		t.Errorf("Expected preload chain [orm], got %v", chain)
	}

	_ = loader
}

func boolPtr(b bool) *bool {
	return &b
}
```

- [ ] **Step 5.2: Run test to verify it fails**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes/lucybot
go test ./internal/mcp -run TestLazyLoader -v
```

Expected: FAIL with "undefined: NewLazyLoader"

- [ ] **Step 5.3: Write implementation**

Create `lucybot/internal/mcp/lazy_loader.go`:
```go
package mcp

import (
	"context"
	"fmt"
	"sync"

	"github.com/tingly-dev/lucybot/internal/config"
)

// LoadingDecision represents the result of analyzing input for lazy loading
type LoadingDecision struct {
	ServersToLoad []string
	Preloads      []string
	TotalTokens   int
	Reason        string
}

// LoadResult represents the result of loading a server
type LoadResult struct {
	ServerName string
	Success    bool
	Error      error
	ToolsCount int
}

// LazyLoaderCallback is called when server loading events occur
type LazyLoaderCallback func(event string, serverName string, data interface{})

// LazyLoader orchestrates on-demand loading of MCP servers
type LazyLoader struct {
	toolReg   *ToolRegistry
	clientMgr *ClientManager
	config    config.LazyLoadingConfig
	callbacks []LazyLoaderCallback
	mu        sync.RWMutex
}

// NewLazyLoader creates a new lazy loader
func NewLazyLoader(toolReg *ToolRegistry, clientMgr *ClientManager, cfg config.LazyLoadingConfig) *LazyLoader {
	return &LazyLoader{
		toolReg:   toolReg,
		clientMgr: clientMgr,
		config:    cfg,
		callbacks: make([]LazyLoaderCallback, 0),
	}
}

// RegisterCallback registers a callback for loading events
func (ll *LazyLoader) RegisterCallback(callback LazyLoaderCallback) {
	ll.mu.Lock()
	defer ll.mu.Unlock()
	ll.callbacks = append(ll.callbacks, callback)
}

// fireCallback fires all registered callbacks for an event
func (ll *LazyLoader) fireCallback(event string, serverName string, data interface{}) {
	ll.mu.RLock()
	defer ll.mu.RUnlock()
	for _, cb := range ll.callbacks {
		go cb(event, serverName, data) // Fire async to avoid blocking
	}
}

// AnalyzeInput analyzes user input to determine which servers to load
func (ll *LazyLoader) AnalyzeInput(userInput string) LoadingDecision {
	if !ll.config.Enabled {
		return LoadingDecision{
			Reason: "Lazy loading is disabled",
		}
	}

	// Find matching servers
	matches := ll.toolReg.FindServersByInput(userInput, ll.config.AutoLoadThreshold, 10)

	var serversToLoad []string
	for _, match := range matches {
		if !ll.toolReg.IsServerLoaded(match.ServerName) {
			serversToLoad = append(serversToLoad, match.ServerName)
		}
	}

	// Calculate preloads
	preloadSet := make(map[string]bool)
	for _, server := range serversToLoad {
		chain := ll.toolReg.GetPreloadChain(server)
		for _, preload := range chain {
			if !ll.toolReg.IsServerLoaded(preload) {
				preloadSet[preload] = true
			}
		}
	}

	var preloads []string
	for p := range preloadSet {
		preloads = append(preloads, p)
	}

	// Estimate tokens (simplified)
	totalTokens := len(serversToLoad) * 100 // Rough estimate

	return LoadingDecision{
		ServersToLoad: serversToLoad,
		Preloads:      preloads,
		TotalTokens:   totalTokens,
		Reason:        fmt.Sprintf("Matched %d servers based on input analysis", len(serversToLoad)),
	}
}

// LoadServer loads a specific MCP server
func (ll *LazyLoader) LoadServer(ctx context.Context, serverName string, isPreload bool) LoadResult {
	result := LoadResult{
		ServerName: serverName,
	}

	// Check if already loaded
	if ll.toolReg.IsServerLoaded(serverName) {
		result.Success = true
		result.ToolsCount = 0
		return result
	}

	// Fire preload started callback
	if isPreload {
		ll.fireCallback("preload_started", serverName, nil)
	}

	// Connect to server
	if err := ll.clientMgr.ConnectServer(ctx, serverName); err != nil {
		result.Error = err
		ll.fireCallback("load_failed", serverName, err)
		return result
	}

	// Get tools count
	tools := ll.clientMgr.GetAllTools()
	toolCount := 0
	for _, tool := range tools {
		if tool.ServerName == serverName {
			toolCount++
		}
	}

	result.Success = true
	result.ToolsCount = toolCount

	// Fire loaded callback
	ll.fireCallback("server_loaded", serverName, toolCount)

	return result
}

// LoadServers loads multiple servers
func (ll *LazyLoader) LoadServers(ctx context.Context, serverNames []string) []LoadResult {
	var results []LoadResult
	for _, name := range serverNames {
		result := ll.LoadServer(ctx, name, false)
		results = append(results, result)
	}
	return results
}

// LoadDecision executes a loading decision
func (ll *LazyLoader) LoadDecision(ctx context.Context, decision LoadingDecision) []LoadResult {
	// Load preloads first
	for _, preload := range decision.Preloads {
		ll.LoadServer(ctx, preload, true)
	}

	// Then load main servers
	return ll.LoadServers(ctx, decision.ServersToLoad)
}

// LoadEagerServers loads all servers configured with lazy_load=false
func (ll *LazyLoader) LoadEagerServers(ctx context.Context) []LoadResult {
	eagerServers := ll.toolReg.GetEagerServers()
	return ll.LoadServers(ctx, eagerServers)
}

// IsServerAvailable checks if a server is configured and available for loading
func (ll *LazyLoader) IsServerAvailable(serverName string) bool {
	_, ok := ll.toolReg.GetServerMetadata(serverName)
	return ok
}

// GetAvailableServers returns all servers that can be loaded
func (ll *LazyLoader) GetAvailableServers() []string {
	return ll.toolReg.GetLazyServers()
}

// GetServerInfo returns information about a server for display
func (ll *LazyLoader) GetServerInfo(serverName string) (map[string]interface{}, bool) {
	metadata, ok := ll.toolReg.GetServerMetadata(serverName)
	if !ok {
		return nil, false
	}

	return map[string]interface{}{
		"name":             metadata.ServerName,
		"lazy_load":        metadata.LazyLoad,
		"triggers":         metadata.Triggers,
		"tool_count":       metadata.ToolCount,
		"estimated_tokens": metadata.EstimatedTokens,
		"is_loaded":        metadata.IsLoaded,
	}, true
}
```

- [ ] **Step 5.4: Run test to verify it passes**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes/lucybot
go test ./internal/mcp -run TestLazyLoader -v
```

Expected: PASS

- [ ] **Step 5.5: Commit**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes
git add lucybot/internal/mcp/lazy_loader.go lucybot/internal/mcp/lazy_loader_test.go
git commit -m "feat(mcp): add lazy loader for on-demand MCP server loading"
```

---

## Task 6: Create Tool Wrapper for AgentScope Integration

**Files:**
- Create: `lucybot/internal/mcp/tool_wrapper.go`
- Modify: `lucybot/internal/tools/init.go` (add load_mcp_server tool)

- [ ] **Step 6.1: Write tool wrapper implementation**

Create `lucybot/internal/mcp/tool_wrapper.go`:
```go
package mcp

import (
	"context"
	"fmt"

	"github.com/tingly-dev/lucybot/internal/config"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
)

// ToolWrapper wraps MCP tools for integration with LucyBot's toolkit
type ToolWrapper struct {
	clientMgr *ClientManager
	loader    *LazyLoader
}

// NewToolWrapper creates a new tool wrapper
func NewToolWrapper() *ToolWrapper {
	return &ToolWrapper{
		clientMgr: GetClientManager(),
	}
}

// SetLazyLoader sets the lazy loader after initialization
func (tw *ToolWrapper) SetLazyLoader(loader *LazyLoader) {
	tw.loader = loader
}

// GetLoadMCPServerTool returns a tool definition for loading MCP servers
func (tw *ToolWrapper) GetLoadMCPServerTool() ToolFunc {
	return func(ctx context.Context, args map[string]interface{}) (*tool.ToolResponse, error) {
		serverName, ok := args["server_name"].(string)
		if !ok || serverName == "" {
			return nil, fmt.Errorf("server_name is required")
		}

		// Check if server exists
		if !tw.loader.IsServerAvailable(serverName) {
			available := tw.loader.GetAvailableServers()
			return nil, fmt.Errorf("server '%s' not found. Available servers: %v", serverName, available)
		}

		// Check if already loaded
		if tw.clientMgr.IsConnected(serverName) {
			return tool.TextResponse(fmt.Sprintf("Server '%s' is already loaded", serverName)), nil
		}

		// Load the server
		result := tw.loader.LoadServer(ctx, serverName, false)
		if !result.Success {
			return nil, fmt.Errorf("failed to load server '%s': %v", serverName, result.Error)
		}

		return tool.TextResponse(fmt.Sprintf(
			"Server '%s' loaded successfully with %d tools",
			serverName,
			result.ToolsCount,
		)), nil
	}
}

// GetListMCPServersTool returns a tool for listing available MCP servers
func (tw *ToolWrapper) GetListMCPServersTool() ToolFunc {
	return func(ctx context.Context, args map[string]interface{}) (*tool.ToolResponse, error) {
		available := tw.loader.GetAvailableServers()
		connected := tw.clientMgr.GetConnectedServers()

		var result string
		result += "Available MCP Servers (not loaded):\n"
		for _, name := range available {
			if info, ok := tw.loader.GetServerInfo(name); ok {
				result += fmt.Sprintf("  - %s (%d tools, triggers: %v)\n",
					name,
					info["tool_count"],
					info["triggers"],
				)
			}
		}

		result += "\nConnected MCP Servers:\n"
		if len(connected) == 0 {
			result += "  (none)\n"
		} else {
			for _, name := range connected {
				result += fmt.Sprintf("  - %s\n", name)
			}
		}

		return tool.TextResponse(result), nil
	}
}

// ConvertMCPToolsToToolkit registers all connected MCP tools to a toolkit
func (tw *ToolWrapper) ConvertMCPToolsToToolkit(tk *tool.Toolkit) error {
	adapter := NewToolAdapter(tw.clientMgr.registry)

	// Create MCP tool group
	tk.CreateToolGroup("mcp", "MCP Server Tools", true, "")

	tools := adapter.GetAllTools()
	for _, toolInfo := range tools {
		toolDef := toolInfo.ToLucyBotTool()

		// Create wrapper function
		wrapper := func(ctx context.Context, kwargs map[string]interface{}) *tool.ToolResponse {
			result, err := adapter.Call(ctx, toolInfo.FullName(), kwargs)
			if err != nil {
				return tool.TextResponse(fmt.Sprintf("Error: %v", err))
			}
			return result
		}

		tk.Register(wrapper, &tool.RegisterOptions{
			GroupName:       "mcp",
			FuncName:        toolDef.Function.Name,
			FuncDescription: toolDef.Function.Description,
		})
	}

	return nil
}

// BuildMCPToolsMessage creates a message describing available MCP tools
func (tw *ToolWrapper) BuildMCPToolsMessage() string {
	available := tw.loader.GetAvailableServers()
	if len(available) == 0 {
		return ""
	}

	var msg string
	msg += "\n## Available MCP Servers (Not Loaded)\n\n"
	msg += "The following MCP servers can be loaded on-demand:\n\n"

	for _, name := range available {
		if info, ok := tw.loader.GetServerInfo(name); ok {
			msg += fmt.Sprintf("- **%s**: ", name)
			if triggers, ok := info["triggers"].([]string); ok && len(triggers) > 0 {
				msg += fmt.Sprintf("(triggers: %v)", triggers)
			}
			msg += "\n"
		}
	}

	msg += "\nTo load a server, use: `load_mcp_server(server_name=\"server_name\")`\n"
	return msg
}

// ToolFunc is the function signature for LucyBot tools
type ToolFunc func(ctx context.Context, args map[string]interface{}) (*tool.ToolResponse, error)

// Ensure Config types are available
var _ = config.MCPServerConfig{} // Compile-time check
```

- [ ] **Step 6.2: Modify tools init.go to add load_mcp_server tool**

Modify `lucybot/internal/tools/init.go`:

Add import:
```go
import (
	"context"
	"fmt"

	"github.com/tingly-dev/lucybot/internal/mcp"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
)
```

Add at the end of InitTools function (before `return registry`):

```go
	// MCP server management tools
	toolWrapper := mcp.NewToolWrapper()

	registry.Register(CreateToolInfo(
		"load_mcp_server",
		"Load an MCP server and register its tools. Use this when you need tools from a specific MCP server.",
		"MCP",
		toolWrapper.GetLoadMCPServerTool(),
		struct {
			ServerName string `json:"server_name" desc:"Name of the MCP server to load"`
		}{},
	))

	registry.Register(CreateToolInfo(
		"list_mcp_servers",
		"List all available and connected MCP servers.",
		"MCP",
		toolWrapper.GetListMCPServersTool(),
		struct{}{},
	))
```

- [ ] **Step 6.3: Commit**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes
git add lucybot/internal/mcp/tool_wrapper.go lucybot/internal/tools/init.go
git commit -m "feat(mcp): add tool wrapper and load_mcp_server tool"
```

---

## Task 7: Integrate MCP into Agent

**Files:**
- Modify: `lucybot/internal/agent/agent.go`
- Test: Verify integration works

- [ ] **Step 7.1: Modify agent.go to integrate MCP lazy loading**

Modify `lucybot/internal/agent/agent.go`:

Add imports:
```go
import (
	"fmt"
	"strings"

	"github.com/tingly-dev/lucybot/internal/config"
	"github.com/tingly-dev/lucybot/internal/mcp"
	"github.com/tingly-dev/lucybot/internal/tools"
	"github.com/tingly-dev/tingly-agentscope/pkg/agent"
	"github.com/tingly-dev/tingly-agentscope/pkg/formatter"
	"github.com/tingly-dev/tingly-agentscope/pkg/memory"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/model/anthropic"
	"github.com/tingly-dev/tingly-agentscope/pkg/model/openai"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
)
```

Add to LucyBotAgent struct (around line 18):
```go
// LucyBotAgent wraps ReActAgent with LucyBot-specific functionality
type LucyBotAgent struct {
	*agent.ReActAgent
	config      *config.Config
	toolkit     *tool.Toolkit
	workDir     string
	registry    *tools.Registry
	mcpLoader   *mcp.LazyLoader    // NEW: MCP lazy loader
	toolWrapper *mcp.ToolWrapper   // NEW: MCP tool wrapper
}
```

Modify NewLucyBotAgent function (around line 63-99):

```go
// NewLucyBotAgent creates a new LucyBotAgent from configuration
func NewLucyBotAgent(cfg *LucyBotAgentConfig) (*LucyBotAgent, error) {
	// Create model
	factory := NewModelFactory()
	chatModel, err := factory.CreateModel(&cfg.Config.Agent.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to create model: %w", err)
	}

	// Initialize tools
	registry := tools.InitTools(cfg.WorkDir)
	toolkit := registry.BuildToolkit()

	// Initialize MCP lazy loading if configured
	var mcpLoader *mcp.LazyLoader
	toolWrapper := mcp.NewToolWrapper()

	if len(cfg.Config.MCP.Servers) > 0 {
		// Load MCP config defaults
		cfg.Config.MCP.ApplyDefaults()

		// Create tool registry from config
		mcpToolReg := mcp.NewToolRegistry()
		mcpToolReg.LoadFromConfig(&cfg.Config.MCP)

		// Initialize client manager
		clientMgr := mcp.GetClientManager()
		if err := clientMgr.Initialize(&cfg.Config.MCP, mcpToolReg); err != nil {
			return nil, fmt.Errorf("failed to initialize MCP client manager: %w", err)
		}

		// Create lazy loader
		mcpLoader = mcp.NewLazyLoader(mcpToolReg, clientMgr, cfg.Config.MCP.LazyLoading)
		toolWrapper.SetLazyLoader(mcpLoader)

		// Load eager servers (lazy_load=false)
		ctx := context.Background()
		results := mcpLoader.LoadEagerServers(ctx)
		for _, result := range results {
			if result.Success {
				fmt.Printf("[MCP] Loaded eager server '%s' with %d tools\n", result.ServerName, result.ToolsCount)
			} else {
				fmt.Printf("[MCP] Failed to load eager server '%s': %v\n", result.ServerName, result.Error)
			}
		}

		// Convert MCP tools to toolkit
		if err := toolWrapper.ConvertMCPToolsToToolkit(toolkit); err != nil {
			return nil, fmt.Errorf("failed to convert MCP tools: %w", err)
		}
	}

	// Create memory
	mem := memory.NewHistory(100)

	// Create ReAct agent
	agentConfig := &agent.ReActAgentConfig{
		Name:          cfg.Config.Agent.Name,
		SystemPrompt:  buildSystemPrompt(cfg.Config, toolWrapper),
		Model:         chatModel,
		Toolkit:       toolkit,
		Memory:        mem,
		MaxIterations: cfg.Config.Agent.MaxIters,
	}

	reactAgent := agent.NewReActAgent(agentConfig)

	// Set formatter for rich output
	reactAgent.SetFormatter(formatter.NewTeaFormatter())

	return &LucyBotAgent{
		ReActAgent:  reactAgent,
		config:      cfg.Config,
		toolkit:     toolkit,
		workDir:     cfg.WorkDir,
		registry:    registry,
		mcpLoader:   mcpLoader,
		toolWrapper: toolWrapper,
	}, nil
}
```

Add helper function to build system prompt:

```go
// buildSystemPrompt builds the system prompt with MCP server information
func buildSystemPrompt(cfg *config.Config, toolWrapper *mcp.ToolWrapper) string {
	prompt := cfg.Agent.SystemPrompt

	// Add MCP server information if available
	if toolWrapper != nil && toolWrapper.BuildMCPToolsMessage != nil {
		mcpMsg := toolWrapper.BuildMCPToolsMessage()
		if mcpMsg != "" {
			prompt = prompt + mcpMsg
		}
	}

	return prompt
}
```

Add method to analyze input before reply:

```go
// AnalyzeInput analyzes user input for MCP lazy loading triggers
func (a *LucyBotAgent) AnalyzeInput(ctx context.Context, input string) error {
	if a.mcpLoader == nil || !a.config.MCP.LazyLoading.Enabled {
		return nil
	}

	// Analyze input for matching servers
	decision := a.mcpLoader.AnalyzeInput(input)
	if len(decision.ServersToLoad) > 0 {
		// Load matching servers
		results := a.mcpLoader.LoadDecision(ctx, decision)
		for _, result := range results {
			if result.Success {
				fmt.Printf("[MCP] Auto-loaded server '%s' with %d tools\n", result.ServerName, result.ToolsCount)
			}
		}
	}

	return nil
}
```

- [ ] **Step 7.2: Add context import if missing**

Check if `context` is imported in `lucybot/internal/agent/agent.go`. If not, add it to the imports.

- [ ] **Step 7.3: Build and test**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes/lucybot
go build ./cmd/lucybot 2>&1
```

Expected: Build succeeds

- [ ] **Step 7.4: Commit**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes
git add lucybot/internal/agent/agent.go
git commit -m "feat(agent): integrate MCP lazy loading into LucyBotAgent"
```

---

## Task 8: Final Integration Testing

**Files:**
- Test all components together
- Run full test suite

- [ ] **Step 8.1: Run all MCP tests**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes/lucybot
go test ./internal/mcp/... -v 2>&1 | head -100
```

Expected: All tests pass

- [ ] **Step 8.2: Build lucybot**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes/lucybot
go build ./cmd/lucybot 2>&1
```

Expected: Build succeeds

- [ ] **Step 8.3: Verify tools are available**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes/lucybot
./lucybot tools 2>&1 | grep -i mcp
```

Expected: Shows `load_mcp_server` and `list_mcp_servers` tools

- [ ] **Step 8.4: Run lint and typecheck**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes/lucybot
go vet ./internal/mcp/...
go vet ./internal/agent/...
go vet ./internal/tools/...
go vet ./internal/config/...
```

Expected: No errors

- [ ] **Step 8.5: Final commit**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes
git commit -m "feat(mcp): complete MCP tool registration and lazy loading implementation

Implements full MCP (Model Context Protocol) tool registration and lazy loading:

- Add MCP configuration with lazy loading support
- Add keyword extractor for intelligent tool matching
- Add tool registry with inverted keyword index
- Add client manager for managing MCP connections
- Add lazy loader for on-demand server loading
- Add tool wrapper for AgentScope integration
- Add load_mcp_server and list_mcp_servers tools
- Integrate MCP lazy loading into LucyBotAgent

Features:
- Keyword-based matching to auto-load relevant servers
- Preload chains for transitive dependencies
- Lightweight metadata registry (no active connections until needed)
- Explicit tool for LLM to load servers on-demand"
```

---

## Summary

This plan implements a complete MCP tool registration and lazy loading system:

1. **Configuration** - Extends lucybot config with MCP server settings and lazy loading options
2. **Keyword Extractor** - Extracts keywords from tool schemas for intelligent matching
3. **Tool Registry** - Maintains lightweight metadata with inverted keyword index
4. **Client Manager** - Manages active MCP connections (singleton pattern)
5. **Lazy Loader** - Orchestrates on-demand loading with callbacks
6. **Tool Wrapper** - Bridges MCP tools to AgentScope toolkit
7. **Agent Integration** - Wires everything together in LucyBotAgent

The system reduces initial token usage by only loading MCP servers when needed, either through:
- **Auto-detection**: Analyzing user input for keyword matches
- **Explicit loading**: LLM calls `load_mcp_server()` tool
- **Preload chains**: Loading dependent servers together
