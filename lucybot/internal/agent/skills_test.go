package agent

import (
	"testing"

	"github.com/tingly-dev/lucybot/internal/config"
	"github.com/tingly-dev/lucybot/internal/skills"
)

func TestLucyBotAgent_SkillsRegistry(t *testing.T) {
	cfg := &config.Config{
		Agent: config.AgentConfig{
			Name:         "test",
			SystemPrompt: "Test",
			Model: config.ModelConfig{
				ModelType: "anthropic",
				APIKey:    "test-key",
				ModelName: "claude-3-haiku-20240307",
			},
		},
	}

	agentCfg := &LucyBotAgentConfig{
		Config:  cfg,
		WorkDir: "/tmp/test",
	}

	agent, err := NewLucyBotAgent(agentCfg)
	if err != nil {
		t.Fatalf("NewLucyBotAgent() error = %v", err)
	}

	skillsRegistry := agent.GetSkillsRegistry()
	if skillsRegistry == nil {
		t.Fatal("GetSkillsRegistry() should not return nil")
	}

	// Test registering a skill
	skill := &skills.Skill{
		Name:        "test-skill",
		Description: "Test",
		Content:     "Content",
	}

	err = skillsRegistry.Register(skill)
	if err != nil {
		t.Errorf("Register() error = %v", err)
	}

	// Verify command registry works
	cmdRegistry := skillsRegistry.GetCommandRegistry()
	_, ok := cmdRegistry.Get("/test-skill")
	if !ok {
		t.Error("Skill should be registered as command")
	}
}
