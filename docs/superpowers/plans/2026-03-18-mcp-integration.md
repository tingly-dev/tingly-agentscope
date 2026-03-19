# MCP Tool Integration Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Integrate the existing MCP (Model Context Protocol) infrastructure into LucyBot agent, enabling dynamic loading of MCP servers based on user input.

**Architecture:** The existing MCP infrastructure (`lucybot/internal/mcp/`) provides lazy loading, keyword matching, and tool adaptation. This plan wires these components into the agent initialization flow, allowing MCP servers to be loaded on-demand when user input matches server keywords.

**Tech Stack:** Go, existing MCP infrastructure in `lucybot/internal/mcp/`

**Important Note:** The MCP infrastructure (client, registry, adapter, lazy_loader, keyword_extractor) already exists. This plan focuses on **integration** - connecting these components to the agent and config systems.

---

## File Structure

| File | Purpose |
|------|---------|
| `lucybot/internal/config/config.go` | MODIFY: Add MCPConfig to main Config struct |
| `lucybot/internal/mcp/config.go` | CREATE: MCP configuration types (MCPServerConfig, etc.) |
| `lucybot/internal/mcp/integration.go` | CREATE: High-level MCP integration helper |
| `lucybot/internal/tools/init.go` | MODIFY: Add `load_mcp_server` tool |
| `lucybot/internal/agent/agent.go` | MODIFY: Initialize MCP lazy loading in agent |

---

## Task 1: Add MCP Configuration Types

**Files:**
- Create: `lucybot/internal/mcp/config.go`
- Modify: `lucybot/internal/config/config.go:54-58`
- Test: `lucybot/internal/mcp/config_test.go`

- [ ] **Step 1.1: Write the failing test**

Create `lucybot/internal/mcp/config_test.go`:
```go
package mcp

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
}

func TestMCPConfig(t *testing.T) {
	config := MCPConfig{
		Servers: map[string]MCPServerConfig{
			"filesystem": {
				Name:     "filesystem",
				Command:  "npx",
				Args:     []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
				Enabled:  true,
				LazyLoad: boolPtr(true),
				Triggers: []string{"file", "read"},
			},
		},
	}

	servers := config.GetEnabledServers()
	if len(servers) != 1 || servers[0] != "filesystem" {
		t.Errorf("Expected enabled server 'filesystem', got %v", servers)
	}
}

func boolPtr(b bool) *bool {
	return &b
}
```

- [ ] **Step 1.2: Run test to verify it fails**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes/lucybot
go test ./internal/mcp -run TestMCPServerConfigValidation -v
```

Expected: FAIL with "undefined: MCPServerConfig"

- [ ] **Step 1.3: Write MCP config types**

Create `lucybot/internal/mcp/config.go`:
```go
package mcp

import "fmt"

// MCPServerConfig represents an MCP server configuration with lazy loading support
type MCPServerConfig struct {
	Name        string            `toml:"name" json:"name"`
	Command     string            `toml:"command" json:"command"`
	Args        []string          `toml:"args" json:"args"`
	Env         map[string]string `toml:"env,omitempty" json:"env,omitempty"`
	Enabled     bool              `toml:"enabled" json:"enabled"`
	LazyLoad    *bool             `toml:"lazy_load,omitempty" json:"lazy_load,omitempty"`
	Triggers    []string          `toml:"triggers" json:"triggers"`
	PreloadWith []string          `toml:"preload_with" json:"preload_with"`
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

// MCPConfig holds all MCP-related configuration
type MCPConfig struct {
	Servers map[string]MCPServerConfig `toml:"servers" json:"servers"`
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

// GetServer returns a server configuration by name
func (c *MCPConfig) GetServer(name string) (MCPServerConfig, bool) {
	server, ok := c.Servers[name]
	return server, ok
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
	MCP     mcp.MCPConfig `toml:"mcp"`
}
```

Add import:
```go
import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/adrg/xdg"
	"github.com/tingly-dev/lucybot/internal/mcp"
)
```

- [ ] **Step 1.5: Run test to verify it passes**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes/lucybot
go test ./internal/mcp -run TestMCPServerConfigValidation -v
go test ./internal/mcp -run TestMCPConfig -v
```

Expected: PASS

- [ ] **Step 1.6: Commit**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes
git add lucybot/internal/mcp/config.go lucybot/internal/mcp/config_test.go lucybot/internal/config/config.go
git commit -m "feat(config): add MCP configuration types"
```

---

## Task 2: Create MCP Integration Helper

**Files:**
- Create: `lucybot/internal/mcp/integration.go`
- Test: `lucybot/internal/mcp/integration_test.go`

- [ ] **Step 2.1: Write the failing test**

Create `lucybot/internal/mcp/integration_test.go`:
```go
package mcp

import (
	"testing"
)

func TestMCPIntegrationHelper(t *testing.T) {
	// Test that helper can be created
	helper := NewIntegrationHelper()
	if helper == nil {
		t.Error("Expected helper to be created")
	}

	// Test config loading
	cfg := &MCPConfig{
		Servers: map[string]MCPServerConfig{
			"test": {
				Name:    "test",
				Command: "echo",
				Enabled: true,
			},
		},
	}

	helper.LoadConfig(cfg)

	if helper.config == nil {
		t.Error("Expected config to be loaded")
	}
}
```

- [ ] **Step 2.2: Run test to verify it fails**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes/lucybot
go test ./internal/mcp -run TestMCPIntegrationHelper -v
```

Expected: FAIL with "undefined: NewIntegrationHelper"

- [ ] **Step 2.3: Write integration helper**

Create `lucybot/internal/mcp/integration.go`:
```go
package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
)

// IntegrationHelper provides a high-level interface for integrating MCP with LucyBot
type IntegrationHelper struct {
	config     *MCPConfig
	registry   *Registry
	loader     *LazyLoader
	adapter    *ToolAdapter
	isInitialized bool
}

// NewIntegrationHelper creates a new MCP integration helper
func NewIntegrationHelper() *IntegrationHelper {
	return &IntegrationHelper{}
}

// LoadConfig loads MCP configuration and initializes components
func (h *IntegrationHelper) LoadConfig(cfg *MCPConfig) error {
	h.config = cfg

	// Create registry
	h.registry = NewRegistry()

	// Register all configured servers
	for name, serverCfg := range cfg.Servers {
		if !serverCfg.Enabled {
			continue
		}

		// Convert to registry format
		regConfig := &ServerConfig{
			Name:    serverCfg.Name,
			Command: serverCfg.Command,
			Args:    serverCfg.Args,
			Env:     serverCfg.Env,
			Enabled: serverCfg.Enabled,
		}

		if err := h.registry.Register(regConfig); err != nil {
			return fmt.Errorf("failed to register server '%s': %w", name, err)
		}
	}

	// Create lazy loader
	loaderConfig := DefaultLazyLoadingConfig()
	h.loader = NewLazyLoader(h.registry, loaderConfig)

	// Set up server keywords from triggers
	for name, serverCfg := range cfg.Servers {
		if len(serverCfg.Triggers) > 0 {
			h.loader.SetServerKeywords(name, serverCfg.Triggers)
		}
	}

	// Set up preload chains
	for name, serverCfg := range cfg.Servers {
		if len(serverCfg.PreloadWith) > 0 {
			h.loader.SetPreloadChain(name, serverCfg.PreloadWith)
		}
	}

	// Create tool adapter
	h.adapter = NewToolAdapter(h.registry)

	h.isInitialized = true
	return nil
}

// LoadEagerServers loads servers with lazy_load=false on startup
func (h *IntegrationHelper) LoadEagerServers(ctx context.Context) []LoadResult {
	if !h.isInitialized {
		return nil
	}

	var results []LoadResult
	for name, serverCfg := range h.config.Servers {
		if !serverCfg.Enabled {
			continue
		}

		// Load if not lazy
		if !serverCfg.ShouldLazyLoad(true) {
			result, err := h.loader.LoadServer(ctx, name, false)
			if err != nil {
				results = append(results, *result)
			} else {
				results = append(results, *result)
			}
		}
	}

	return results
}

// AnalyzeInput analyzes user input and loads matching servers
func (h *IntegrationHelper) AnalyzeInput(ctx context.Context, userInput string) ([]LoadResult, error) {
	if !h.isInitialized {
		return nil, nil
	}

	decision, err := h.loader.AnalyzeInput(ctx, userInput)
	if err != nil {
		return nil, err
	}

	if !decision.ShouldLoad {
		return nil, nil
	}

	// Load matching servers
	var results []LoadResult
	for _, serverName := range decision.ServersToLoad {
		result, err := h.loader.LoadServer(ctx, serverName, false)
		if err != nil {
			results = append(results, LoadResult{
				ServerName: serverName,
				Success:    false,
				Error:      err.Error(),
			})
		} else {
			results = append(results, *result)
		}
	}

	// Load preloads
	for _, preloadName := range decision.Preloads {
		result, err := h.loader.LoadServer(ctx, preloadName, true)
		if err != nil {
			results = append(results, LoadResult{
				ServerName: preloadName,
				Success:    false,
				Error:      err.Error(),
			})
		} else {
			results = append(results, *result)
		}
	}

	return results, nil
}

// RegisterTools registers all loaded MCP tools to a toolkit
func (h *IntegrationHelper) RegisterTools(tk *tool.Toolkit) error {
	if !h.isInitialized {
		return nil
	}

	// Create MCP tool group
	tk.CreateToolGroup("mcp", "MCP Server Tools", true, "")

	// Get all tools from loaded servers
	tools := h.adapter.GetAllTools()
	for _, toolInfo := range tools {
		toolDef := toolInfo.ToLucyBotTool()

		// Create wrapper function
		wrapper := func(ctx context.Context, kwargs map[string]interface{}) *tool.ToolResponse {
			result, err := h.adapter.Call(ctx, toolInfo.FullName(), kwargs)
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

// LoadServer explicitly loads a specific server
func (h *IntegrationHelper) LoadServer(ctx context.Context, serverName string) (*LoadResult, error) {
	if !h.isInitialized {
		return nil, fmt.Errorf("MCP helper not initialized")
	}

	return h.loader.LoadServer(ctx, serverName, false)
}

// IsServerLoaded checks if a server is loaded
func (h *IntegrationHelper) IsServerLoaded(serverName string) bool {
	if !h.isInitialized {
		return false
	}
	return h.loader.IsLoaded(serverName)
}

// GetAvailableServers returns all configured server names
func (h *IntegrationHelper) GetAvailableServers() []string {
	if !h.config {
		return nil
	}
	return h.config.GetEnabledServers()
}

// BuildSystemPromptAppendix builds the MCP section for the system prompt
func (h *IntegrationHelper) BuildSystemPromptAppendix() string {
	if !h.isInitialized || h.config == nil {
		return ""
	}

	var lazyServers []string
	var eagerServers []string

	for name, serverCfg := range h.config.Servers {
		if !serverCfg.Enabled {
			continue
		}
		if h.loader.IsLoaded(name) {
			continue // Already loaded
		}
		if serverCfg.ShouldLazyLoad(true) {
			lazyServers = append(lazyServers, name)
		} else {
			eagerServers = append(eagerServers, name)
		}
	}

	if len(lazyServers) == 0 && len(eagerServers) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n## Available MCP Servers\n\n")

	if len(lazyServers) > 0 {
		sb.WriteString("The following MCP servers can be loaded on-demand:\n")
		for _, name := range lazyServers {
			if serverCfg, ok := h.config.GetServer(name); ok {
				sb.WriteString(fmt.Sprintf("- **%s**", name))
				if len(serverCfg.Triggers) > 0 {
					sb.WriteString(fmt.Sprintf(" (keywords: %v)", serverCfg.Triggers))
				}
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\nTo load a server, use: `load_mcp_server(server_name=\"server_name\")`\n")
	}

	if len(eagerServers) > 0 {
		sb.WriteString("\nThe following servers will load automatically:\n")
		for _, name := range eagerServers {
			sb.WriteString(fmt.Sprintf("- **%s**\n", name))
		}
	}

	return sb.String()
}

// GetLoadServerTool returns a tool function for loading MCP servers
func (h *IntegrationHelper) GetLoadServerTool() func(ctx context.Context, args map[string]interface{}) (*tool.ToolResponse, error) {
	return func(ctx context.Context, args map[string]interface{}) (*tool.ToolResponse, error) {
		serverName, ok := args["server_name"].(string)
		if !ok || serverName == "" {
			return nil, fmt.Errorf("server_name is required")
		}

		// Check if server exists
		_, exists := h.config.GetServer(serverName)
		if !exists {
			available := h.GetAvailableServers()
			return nil, fmt.Errorf("server '%s' not found. Available: %v", serverName, available)
		}

		// Check if already loaded
		if h.IsServerLoaded(serverName) {
			return tool.TextResponse(fmt.Sprintf("Server '%s' is already loaded", serverName)), nil
		}

		// Load the server
		result, err := h.LoadServer(ctx, serverName)
		if err != nil {
			return nil, fmt.Errorf("failed to load server '%s': %w", serverName, err)
		}

		if !result.Success {
			return nil, fmt.Errorf("failed to load server '%s': %s", serverName, result.Error)
		}

		return tool.TextResponse(fmt.Sprintf(
			"Server '%s' loaded successfully with %d tools",
			serverName,
			len(result.ToolsLoaded),
		)), nil
	}
}

// GetListServersTool returns a tool function for listing MCP servers
func (h *IntegrationHelper) GetListServersTool() func(ctx context.Context, args map[string]interface{}) (*tool.ToolResponse, error) {
	return func(ctx context.Context, args map[string]interface{}) (*tool.ToolResponse, error) {
		available := h.GetAvailableServers()
		loaded := h.loader.GetLoadedServers()

		var sb strings.Builder
		sb.WriteString("Available MCP Servers:\n")
		for _, name := range available {
			isLoaded := false
			for _, l := range loaded {
				if l == name {
					isLoaded = true
					break
				}
			}
			if isLoaded {
				sb.WriteString(fmt.Sprintf("  - %s (loaded)\n", name))
			} else {
				sb.WriteString(fmt.Sprintf("  - %s (not loaded)\n", name))
			}
		}

		return tool.TextResponse(sb.String()), nil
	}
}
```

- [ ] **Step 2.4: Run test to verify it passes**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes/lucybot
go test ./internal/mcp -run TestMCPIntegrationHelper -v
```

Expected: PASS

- [ ] **Step 2.5: Commit**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes
git add lucybot/internal/mcp/integration.go lucybot/internal/mcp/integration_test.go
git commit -m "feat(mcp): add integration helper for agent integration"
```

---

## Task 3: Add load_mcp_server Tool

**Files:**
- Modify: `lucybot/internal/tools/init.go`

- [ ] **Step 3.1: Modify tools init.go to support MCP tools**

Modify `lucybot/internal/tools/init.go`:

Add parameter to InitTools function (around line 11):

Find:
```go
// InitTools initializes and registers all LucyBot tools
func InitTools(workDir string) *Registry {
```

Replace with:
```go
// InitTools initializes and registers all LucyBot tools
// mcpHelper is optional and can be nil if MCP is not configured
func InitTools(workDir string, mcpHelper *mcp.IntegrationHelper) *Registry {
```

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

Add MCP tools at the end of InitTools (before `return registry`):

```go
	// MCP server management tools
	if mcpHelper != nil {
		registry.Register(CreateToolInfo(
			"load_mcp_server",
			"Load an MCP server and register its tools. Use this when you need tools from a specific MCP server.",
			"MCP",
			mcpHelper.GetLoadServerTool(),
			struct {
				ServerName string `json:"server_name" desc:"Name of the MCP server to load"`
			}{},
		))

		registry.Register(CreateToolInfo(
			"list_mcp_servers",
			"List all available MCP servers and their load status.",
			"MCP",
			mcpHelper.GetListServersTool(),
			struct{}{},
		))
	}
```

- [ ] **Step 3.2: Commit**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes
git add lucybot/internal/tools/init.go
git commit -m "feat(tools): add load_mcp_server and list_mcp_servers tools"
```

---

## Task 4: Integrate MCP into Agent

**Files:**
- Modify: `lucybot/internal/agent/agent.go`

- [ ] **Step 4.1: Modify agent.go to integrate MCP**

Modify `lucybot/internal/agent/agent.go`:

Add import:
```go
import (
	"context"
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

Add MCP field to LucyBotAgent struct (around line 18):

```go
// LucyBotAgent wraps ReActAgent with LucyBot-specific functionality
type LucyBotAgent struct {
	*agent.ReActAgent
	config     *config.Config
	toolkit    *tool.Toolkit
	workDir    string
	registry   *tools.Registry
	mcpHelper  *mcp.IntegrationHelper  // NEW: MCP integration helper
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

	// Initialize MCP if configured
	var mcpHelper *mcp.IntegrationHelper
	if len(cfg.Config.MCP.Servers) > 0 {
		mcpHelper = mcp.NewIntegrationHelper()
		if err := mcpHelper.LoadConfig(&cfg.Config.MCP); err != nil {
			return nil, fmt.Errorf("failed to load MCP config: %w", err)
		}

		// Load eager servers
		ctx := context.Background()
		results := mcpHelper.LoadEagerServers(ctx)
		for _, result := range results {
			if result.Success {
				fmt.Printf("[MCP] Loaded server '%s' with %d tools\n", result.ServerName, len(result.ToolsLoaded))
			} else {
				fmt.Printf("[MCP] Failed to load server '%s': %s\n", result.ServerName, result.Error)
			}
		}
	}

	// Initialize tools (pass MCP helper)
	registry := tools.InitTools(cfg.WorkDir, mcpHelper)
	toolkit := registry.BuildToolkit()

	// Register MCP tools to toolkit if helper exists
	if mcpHelper != nil {
		if err := mcpHelper.RegisterTools(toolkit); err != nil {
			return nil, fmt.Errorf("failed to register MCP tools: %w", err)
		}
	}

	// Create memory
	mem := memory.NewHistory(100)

	// Build system prompt with MCP info
	systemPrompt := buildSystemPrompt(cfg.Config, mcpHelper)

	// Create ReAct agent
	agentConfig := &agent.ReActAgentConfig{
		Name:          cfg.Config.Agent.Name,
		SystemPrompt:  systemPrompt,
		Model:         chatModel,
		Toolkit:       toolkit,
		Memory:        mem,
		MaxIterations: cfg.Config.Agent.MaxIters,
	}

	reactAgent := agent.NewReActAgent(agentConfig)

	// Set formatter for rich output
	reactAgent.SetFormatter(formatter.NewTeaFormatter())

	return &LucyBotAgent{
		ReActAgent: reactAgent,
		config:     cfg.Config,
		toolkit:    toolkit,
		workDir:    cfg.WorkDir,
		registry:   registry,
		mcpHelper:  mcpHelper,
	}, nil
}
```

Add helper function:

```go
// buildSystemPrompt builds the system prompt with MCP information
func buildSystemPrompt(cfg config.Config, mcpHelper *mcp.IntegrationHelper) string {
	prompt := cfg.Agent.SystemPrompt

	// Add MCP server information if available
	if mcpHelper != nil {
		mcpSection := mcpHelper.BuildSystemPromptAppendix()
		if mcpSection != "" {
			prompt = prompt + mcpSection
		}
	}

	return prompt
}
```

Add method to analyze input:

```go
// AnalyzeInput analyzes user input for MCP lazy loading triggers
func (a *LucyBotAgent) AnalyzeInput(ctx context.Context, input string) error {
	if a.mcpHelper == nil {
		return nil
	}

	results, err := a.mcpHelper.AnalyzeInput(ctx, input)
	if err != nil {
		return err
	}

	// Re-register tools if new servers were loaded
	if len(results) > 0 {
		hasNew := false
		for _, result := range results {
			if result.Success {
				hasNew = true
				fmt.Printf("[MCP] Auto-loaded server '%s' with %d tools\n", result.ServerName, len(result.ToolsLoaded))
			}
		}

		if hasNew {
			if err := a.mcpHelper.RegisterTools(a.toolkit); err != nil {
				return fmt.Errorf("failed to register new MCP tools: %w", err)
			}
		}
	}

	return nil
}
```

- [ ] **Step 4.2: Commit**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes
git add lucybot/internal/agent/agent.go
git commit -m "feat(agent): integrate MCP lazy loading into LucyBotAgent"
```

---

## Task 5: Final Integration Testing

**Files:**
- Test all components together

- [ ] **Step 5.1: Run all MCP tests**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes/lucybot
go test ./internal/mcp/... -v 2>&1 | head -100
```

Expected: All tests pass

- [ ] **Step 5.2: Build lucybot**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes/lucybot
go build ./cmd/lucybot 2>&1
```

Expected: Build succeeds

- [ ] **Step 5.3: Verify tools are available**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes/lucybot
./lucybot tools 2>&1 | grep -i mcp
```

Expected: Shows `load_mcp_server` and `list_mcp_servers` tools

- [ ] **Step 5.4: Run lint and typecheck**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes/lucybot
go vet ./internal/mcp/...
go vet ./internal/agent/...
go vet ./internal/tools/...
go vet ./internal/config/...
```

Expected: No errors

- [ ] **Step 5.5: Final commit**

```bash
cd /home/xiao/program/tingly-agentscope/.worktrees/react-agent-fixes
git commit -m "feat(mcp): integrate MCP tool registration and lazy loading

Integrates existing MCP infrastructure into LucyBot agent:

- Add MCP configuration types (MCPServerConfig, MCPConfig)
- Add IntegrationHelper to coordinate MCP components
- Add load_mcp_server and list_mcp_servers tools
- Integrate MCP lazy loading into LucyBotAgent initialization
- Auto-load servers based on user input keyword matching
- Add MCP server info to system prompt

The existing MCP infrastructure (client, registry, adapter, lazy_loader,
keyword_extractor) is now fully wired into the agent."
```

---

## Summary

This plan integrates the **existing** MCP infrastructure into LucyBot:

1. **Config** - Add MCP config types and wire into main Config
2. **Integration Helper** - High-level coordinator for MCP components
3. **Tools** - Add `load_mcp_server` and `list_mcp_servers` tools
4. **Agent Integration** - Initialize MCP in agent, analyze input for triggers

The existing MCP code provides:
- `client.go` - MCP protocol implementation
- `registry.go` - Server management
- `adapter.go` - Tool conversion
- `lazy_loader.go` - On-demand loading with keyword matching
- `keyword_extractor.go` - Input analysis

This plan connects these pieces to the agent lifecycle.
