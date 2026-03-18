package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
)

// IntegrationHelper provides a high-level interface for integrating MCP with LucyBot
type IntegrationHelper struct {
	config        *MCPConfig
	registry      *Registry
	loader        *LazyLoader
	adapter       *ToolAdapter
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

		// Convert to registry format - adapt based on actual ServerConfig struct
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

	// Create lazy loader - use DefaultLazyLoadingConfig() or create config
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
	if h.config == nil {
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
