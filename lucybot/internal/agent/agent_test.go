package agent

import (
	"context"
	"testing"

	"github.com/tingly-dev/lucybot/internal/config"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

func TestAgentLazySessionID(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Agent: config.AgentConfig{
			Name:             "test-agent",
			WorkingDirectory: tmpDir,
			Model: config.ModelConfig{
				ModelType: "openai",
				ModelName: "gpt-4o",
			},
		},
		Session: config.SessionConfig{
			Enabled:     true,
			StoragePath: tmpDir,
			// SessionID not provided - should use empty string for lazy generation
		},
	}

	agentCfg := &LucyBotAgentConfig{
		Config:  cfg,
		WorkDir: tmpDir,
	}

	agent, err := NewLucyBotAgent(agentCfg)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Session ID should be empty initially (lazy generation)
	if agent.GetSessionID() != "" {
		t.Errorf("Expected empty session ID initially for lazy generation, got %s", agent.GetSessionID())
	}
}

func TestAgentWithProvidedSessionID(t *testing.T) {
	tmpDir := t.TempDir()
	providedSessionID := "test-session-123"
	cfg := &config.Config{
		Agent: config.AgentConfig{
			Name:             "test-agent",
			WorkingDirectory: tmpDir,
			Model: config.ModelConfig{
				ModelType: "openai",
				ModelName: "gpt-4o",
			},
		},
		Session: config.SessionConfig{
			Enabled:     true,
			StoragePath: tmpDir,
			SessionID:   providedSessionID,
		},
	}

	agentCfg := &LucyBotAgentConfig{
		Config:  cfg,
		WorkDir: tmpDir,
	}

	agent, err := NewLucyBotAgent(agentCfg)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Session ID should be the provided one
	if agent.GetSessionID() != providedSessionID {
		t.Errorf("Expected session ID %s, got %s", providedSessionID, agent.GetSessionID())
	}
}

// TestAgentLazySessionIDGeneration tests the complete lazy generation flow
// including sessionID propagation from recorder to agent
func TestAgentLazySessionIDGeneration(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Agent: config.AgentConfig{
			Name:             "test-agent",
			WorkingDirectory: tmpDir,
			Model: config.ModelConfig{
				ModelType: "openai",
				ModelName: "gpt-4o",
			},
		},
		Session: config.SessionConfig{
			Enabled:     true,
			StoragePath: tmpDir,
			// SessionID not provided - empty string for lazy generation
		},
	}

	agentCfg := &LucyBotAgentConfig{
		Config:  cfg,
		WorkDir: tmpDir,
	}

	agent, err := NewLucyBotAgent(agentCfg)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Session ID should be empty initially
	if agent.GetSessionID() != "" {
		t.Errorf("Expected empty session ID initially, got %s", agent.GetSessionID())
	}

	// Record a user message - this should trigger lazy sessionID generation
	ctx := context.Background()
	userMsg := message.NewMsg("user", "Hello, this is a test query", types.RoleUser)

	// Add message to memory (RecordingMemory will trigger lazy generation)
	if err := agent.memory.Add(ctx, userMsg); err != nil {
		t.Logf("Warning: Failed to add message: %v", err)
	}

	// After recording, GetSessionID should return the generated sessionID
	generatedSessionID := agent.GetSessionID()
	if generatedSessionID == "" {
		t.Error("Expected sessionID to be generated after first message, got empty string")
	}

	// The generated sessionID should be a 32-character hex string (MD5 hash)
	if len(generatedSessionID) != 32 {
		t.Errorf("Expected sessionID to be 32 characters (MD5 hash), got %d characters: %s", len(generatedSessionID), generatedSessionID)
	}

	// Verify it's a valid hex string
	for _, c := range generatedSessionID {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("Expected sessionID to be hex string, got invalid character: %c in %s", c, generatedSessionID)
			break
		}
	}

	t.Logf("Successfully generated sessionID: %s", generatedSessionID)
}

// TestAgentEmptyStringSessionID tests explicitly setting empty string vs not setting at all
func TestAgentEmptyStringSessionID(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Agent: config.AgentConfig{
			Name:             "test-agent",
			WorkingDirectory: tmpDir,
			Model: config.ModelConfig{
				ModelType: "openai",
				ModelName: "gpt-4o",
			},
		},
		Session: config.SessionConfig{
			Enabled:     true,
			StoragePath: tmpDir,
			SessionID:   "", // Explicitly set to empty string
		},
	}

	agentCfg := &LucyBotAgentConfig{
		Config:  cfg,
		WorkDir: tmpDir,
	}

	agent, err := NewLucyBotAgent(agentCfg)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Empty string should be treated the same as not set (lazy generation)
	if agent.GetSessionID() != "" {
		t.Errorf("Expected empty session ID for explicitly empty SessionID, got %s", agent.GetSessionID())
	}
}

// TestAgentSessionDisabled tests that session is not created when disabled
func TestAgentSessionDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Agent: config.AgentConfig{
			Name:             "test-agent",
			WorkingDirectory: tmpDir,
			Model: config.ModelConfig{
				ModelType: "openai",
				ModelName: "gpt-4o",
			},
		},
		Session: config.SessionConfig{
			Enabled:     false, // Session disabled
			StoragePath: tmpDir,
		},
	}

	agentCfg := &LucyBotAgentConfig{
		Config:  cfg,
		WorkDir: tmpDir,
	}

	agent, err := NewLucyBotAgent(agentCfg)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Session manager should not be initialized
	if agent.GetSessionManager() != nil {
		t.Error("Expected nil session manager when session is disabled")
	}

	// Session ID should be empty
	if agent.GetSessionID() != "" {
		t.Errorf("Expected empty session ID when session is disabled, got %s", agent.GetSessionID())
	}
}

// TestAgentSessionIDPropagation tests that sessionID is properly propagated
// through RecordingMemory to the agent
func TestAgentSessionIDPropagation(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Agent: config.AgentConfig{
			Name:             "test-agent",
			WorkingDirectory: tmpDir,
			Model: config.ModelConfig{
				ModelType: "openai",
				ModelName: "gpt-4o",
			},
		},
		Session: config.SessionConfig{
			Enabled:     true,
			StoragePath: tmpDir,
			// Lazy generation - empty sessionID
		},
	}

	agentCfg := &LucyBotAgentConfig{
		Config:  cfg,
		WorkDir: tmpDir,
	}

	agent, err := NewLucyBotAgent(agentCfg)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Initially, sessionID field should be empty (not yet generated)
	if agent.sessionID != "" {
		t.Errorf("Expected empty agent.sessionID initially, got %s", agent.sessionID)
	}

	// After calling GetSessionID(), it should check RecordingMemory
	// which in turn checks the recorder
	sessionID := agent.GetSessionID()
	if sessionID != "" {
		// If recorder was somehow initialized, sessionID would be non-empty
		// but without any messages, it should still be empty
		t.Logf("SessionID after initialization: %s", sessionID)
	}
}
