package agent

import (
	"testing"

	"github.com/tingly-dev/lucybot/internal/config"
)

func TestAgentSessionIntegration(t *testing.T) {
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

	// Verify session manager is attached
	if agent.GetSessionManager() == nil {
		t.Error("Expected session manager to be attached")
	}
}
