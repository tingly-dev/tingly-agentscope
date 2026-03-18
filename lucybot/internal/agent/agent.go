package agent

import (
	"context"
	"fmt"

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

// LucyBotAgent wraps ReActAgent with LucyBot-specific functionality
type LucyBotAgent struct {
	*agent.ReActAgent
	config     *config.Config
	toolkit    *tool.Toolkit
	workDir    string
	registry   *tools.Registry
	mcpHelper  *mcp.IntegrationHelper
}

// LucyBotAgentConfig holds configuration for creating a LucyBotAgent
type LucyBotAgentConfig struct {
	Config  *config.Config
	WorkDir string
}

// ModelFactory creates ChatModel instances based on configuration
type ModelFactory struct{}

// NewModelFactory creates a new model factory
func NewModelFactory() *ModelFactory {
	return &ModelFactory{}
}

// CreateModel creates a model client from configuration
func (mf *ModelFactory) CreateModel(cfg *config.ModelConfig) (model.ChatModel, error) {
	switch cfg.ModelType {
	case "openai":
		return openai.NewClient(&openai.Config{
			APIKey:  cfg.APIKey,
			BaseURL: cfg.BaseURL,
			Model:   cfg.ModelName,
			Stream:  cfg.Stream,
		})
	case "anthropic":
		return anthropic.NewClient(&anthropic.Config{
			APIKey:  cfg.APIKey,
			BaseURL: cfg.BaseURL,
			Model:   cfg.ModelName,
			Stream:  cfg.Stream,
		})
	default:
		return nil, fmt.Errorf("unsupported model type: %s", cfg.ModelType)
	}
}

// NewLucyBotAgent creates a new LucyBotAgent from configuration
func NewLucyBotAgent(cfg *LucyBotAgentConfig) (*LucyBotAgent, error) {
	// Create model
	factory := NewModelFactory()
	chatModel, err := factory.CreateModel(&cfg.Config.Agent.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to create model: %w", err)
	}

	// Initialize MCP helper if MCP is configured
	var mcpHelper *mcp.IntegrationHelper
	if len(cfg.Config.MCP.Servers) > 0 {
		mcpHelper = mcp.NewIntegrationHelper()
		if err := mcpHelper.LoadConfig(&cfg.Config.MCP); err != nil {
			// Log warning but don't fail - MCP is optional
			fmt.Printf("Warning: failed to load MCP config: %v\n", err)
			mcpHelper = nil
		} else {
			// Load eager servers (those with lazy_load=false)
			ctx := context.Background()
			results := mcpHelper.LoadEagerServers(ctx)
			for _, result := range results {
				if result.Success {
					fmt.Printf("✓ Loaded MCP server: %s (%d tools)\n", result.ServerName, len(result.ToolsLoaded))
				} else {
					fmt.Printf("✗ Failed to load MCP server: %s - %s\n", result.ServerName, result.Error)
				}
			}
		}
	}

	// Initialize tools with MCP helper
	registry := tools.InitTools(cfg.WorkDir, mcpHelper)
	toolkit := registry.BuildToolkit()

	// Register MCP tools to toolkit if helper is available
	if mcpHelper != nil {
		if err := mcpHelper.RegisterTools(toolkit); err != nil {
			fmt.Printf("Warning: failed to register MCP tools: %v\n", err)
		}
	}

	// Create memory
	mem := memory.NewHistory(100)

	// Create ReAct agent
	agentConfig := &agent.ReActAgentConfig{
		Name:          cfg.Config.Agent.Name,
		SystemPrompt:  cfg.Config.Agent.SystemPrompt,
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

// GetConfig returns the agent configuration
func (a *LucyBotAgent) GetConfig() *config.Config {
	return a.config
}

// GetWorkDir returns the working directory
func (a *LucyBotAgent) GetWorkDir() string {
	return a.workDir
}

// GetRegistry returns the tool registry
func (a *LucyBotAgent) GetRegistry() *tools.Registry {
	return a.registry
}

// GetToolkit returns the toolkit
func (a *LucyBotAgent) GetToolkit() *tool.Toolkit {
	return a.toolkit
}

// GetMCPHelper returns the MCP integration helper
func (a *LucyBotAgent) GetMCPHelper() *mcp.IntegrationHelper {
	return a.mcpHelper
}

// SetWorkDir updates the working directory
func (a *LucyBotAgent) SetWorkDir(dir string) {
	a.workDir = dir
}

// SetStreamingConfig sets the streaming configuration on the underlying ReAct agent
func (a *LucyBotAgent) SetStreamingConfig(streaming *agent.StreamingConfig) {
	a.ReActAgent.SetStreamingConfig(streaming)
}
