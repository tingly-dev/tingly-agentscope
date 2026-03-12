package agent

import (
	"fmt"

	"github.com/tingly-dev/lucybot/internal/config"
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
	config   *config.Config
	toolkit  *tool.Toolkit
	workDir  string
	registry *tools.Registry
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

	// Initialize tools
	registry := tools.InitTools(cfg.WorkDir)
	toolkit := registry.BuildToolkit()

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

// SetWorkDir updates the working directory
func (a *LucyBotAgent) SetWorkDir(dir string) {
	a.workDir = dir
}
