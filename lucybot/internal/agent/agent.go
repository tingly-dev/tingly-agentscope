package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/tingly-dev/lucybot/internal/config"
	"github.com/tingly-dev/lucybot/internal/mcp"
	"github.com/tingly-dev/lucybot/internal/session"
	"github.com/tingly-dev/lucybot/internal/tools"
	agentscopeAgent "github.com/tingly-dev/tingly-agentscope/pkg/agent"
	"github.com/tingly-dev/tingly-agentscope/pkg/formatter"
	"github.com/tingly-dev/tingly-agentscope/pkg/memory"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/model/anthropic"
	"github.com/tingly-dev/tingly-agentscope/pkg/model/openai"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
)

// LucyBotAgent wraps ReActAgent with LucyBot-specific functionality
type LucyBotAgent struct {
	*agentscopeAgent.ReActAgent
	config         *config.Config
	toolkit        *tool.Toolkit
	workDir        string
	registry       *tools.Registry
	mcpHelper      *mcp.IntegrationHelper
	sessionManager *session.Manager // Session manager for persistence
	sessionID      string           // Current session ID
	memory         memory.Memory    // Agent memory for session resumption
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

// buildSystemPrompt builds the system prompt with MCP information
func buildSystemPrompt(cfg *config.Config, mcpHelper *mcp.IntegrationHelper) string {
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

	// Build system prompt with MCP information
	systemPrompt := buildSystemPrompt(cfg.Config, mcpHelper)

	// Create ReAct agent
	agentConfig := &agentscopeAgent.ReActAgentConfig{
		Name:          cfg.Config.Agent.Name,
		SystemPrompt:  systemPrompt,
		Model:         chatModel,
		Toolkit:       toolkit,
		Memory:        mem,
		MaxIterations: cfg.Config.Agent.MaxIters,
	}

	reactAgent := agentscopeAgent.NewReActAgent(agentConfig)

	// Set formatter for rich output
	reactAgent.SetFormatter(formatter.NewTeaFormatter())

	lucyAgent := &LucyBotAgent{
		ReActAgent: reactAgent,
		config:     cfg.Config,
		toolkit:    toolkit,
		workDir:    cfg.WorkDir,
		registry:   registry,
		mcpHelper:  mcpHelper,
		memory:     mem,
	}

	// Initialize session manager if enabled
	if cfg.Config.Session.Enabled {
		mgr, err := session.NewManager(
			&cfg.Config.Session,
			cfg.Config.Agent.Name,
			cfg.WorkDir,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create session manager: %w", err)
		}
		lucyAgent.sessionManager = mgr

		// Generate or use provided session ID
		sessionID := cfg.Config.Session.SessionID
		if sessionID == "" {
			sessionID = generateSessionID()
		}

		// Initialize session
		if _, err := mgr.GetOrCreate(sessionID, cfg.Config.Agent.Name); err != nil {
			return nil, fmt.Errorf("failed to initialize session: %w", err)
		}

		lucyAgent.sessionID = sessionID

		// Wrap memory with recording memory to capture all messages
		recorder := mgr.GetRecorder()
		recordingMem := session.NewRecordingMemory(mem, recorder, sessionID)

		// Update the agent's memory reference
		lucyAgent.memory = recordingMem

		// Update the ReActAgent's memory reference
		reactAgent.SetMemory(recordingMem)
	}

	// Setup compression configuration from LucyBot config
	lucyAgent.SetupCompression()

	return lucyAgent, nil
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

// GetSessionManager returns the session manager
func (a *LucyBotAgent) GetSessionManager() *session.Manager {
	return a.sessionManager
}

// GetSessionID returns the current session ID
func (a *LucyBotAgent) GetSessionID() string {
	return a.sessionID
}

// SetSessionManager sets the session manager and session ID
func (a *LucyBotAgent) SetSessionManager(mgr *session.Manager, sessionID string) {
	a.sessionManager = mgr
	a.sessionID = sessionID
}

// SetSessionIDForRecording updates the session ID for recording
// This is used when resuming a session to append new messages to the resumed session
func (a *LucyBotAgent) SetSessionIDForRecording(sessionID string) {
	a.sessionID = sessionID
	// Update RecordingMemory's session ID if memory is RecordingMemory
	if recordingMem, ok := a.memory.(*session.RecordingMemory); ok {
		recordingMem.SetSessionID(sessionID)
	}

	// Also update the recorder's session ID directly
	if a.sessionManager != nil {
		recorder := a.sessionManager.GetRecorder()
		// Load the session to get its name
		if sess, err := a.sessionManager.Load(sessionID); err == nil {
			recorder.SetSessionID(sessionID, sess.Name)
		} else {
			// If we can't load the session, just update the ID
			recorder.SetSessionID(sessionID, "")
		}
	}
}

// GetMemory returns the agent's memory (needed for session resumption)
func (a *LucyBotAgent) GetMemory() memory.Memory {
	return a.memory
}

// SetWorkDir updates the working directory
func (a *LucyBotAgent) SetWorkDir(dir string) {
	a.workDir = dir
}

// SetStreamingConfig sets the streaming configuration on the underlying ReAct agent
func (a *LucyBotAgent) SetStreamingConfig(streaming *agentscopeAgent.StreamingConfig) {
	a.ReActAgent.SetStreamingConfig(streaming)
}

// CompactMemory manually triggers memory compression.
// Returns (wasCompressed, originalTokens, compressedTokens, error).
func (a *LucyBotAgent) CompactMemory(ctx context.Context) (bool, int, int, error) {
	result, err := a.ReActAgent.CompressMemory(ctx)
	if err != nil {
		return false, 0, 0, err
	}
	if result != nil {
		return true, result.OriginalTokenCount, result.CompressedTokenCount, nil
	}

	count := a.ReActAgent.GetMemoryTokenCount(ctx)
	return false, count, count, nil
}

// GetMemoryTokenCount returns the total token count of all messages in memory.
func (a *LucyBotAgent) GetMemoryTokenCount(ctx context.Context) int {
	return a.ReActAgent.GetMemoryTokenCount(ctx)
}

// SetupCompression initializes compression configuration from LucyBot config.
func (a *LucyBotAgent) SetupCompression() {
	cfg := a.config.Agent.Compression

	threshold := cfg.Threshold
	if threshold == 0 && cfg.ContextWindow > 0 && cfg.TriggerThresholdPercent > 0 {
		threshold = cfg.ContextWindow * cfg.TriggerThresholdPercent / 100
	}

	compressionCfg := &agentscopeAgent.CompressionConfig{
		Enable:           cfg.Enabled,
		TokenCounter:     agentscopeAgent.NewSimpleTokenCounter(),
		TriggerThreshold: threshold,
		KeepRecent:       cfg.KeepRecent,
	}

	a.ReActAgent.SetCompressionConfig(compressionCfg)
}

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

// Reply processes a user message and returns the agent's response
// Note: When sessions are enabled, RecordingMemory wrapper automatically records all messages
func (a *LucyBotAgent) Reply(ctx context.Context, msg *message.Msg) (*message.Msg, error) {
	// Call underlying ReActAgent's Reply
	// RecordingMemory wrapper will automatically record all messages to session
	return a.ReActAgent.Reply(ctx, msg)
}

// generateSessionID generates a unique session ID based on timestamp
func generateSessionID() string {
	return fmt.Sprintf("%08x", time.Now().UnixNano())[:8]
}
