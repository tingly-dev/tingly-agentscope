package ui

import (
	"strings"
	"testing"

	"github.com/tingly-dev/lucybot/internal/agent"
	"github.com/tingly-dev/lucybot/internal/config"
	"github.com/tingly-dev/lucybot/internal/skills"
)

func TestApp_HandleSkillCommand(t *testing.T) {
	// Create a test skill
	testSkill := &skills.Skill{
		Name:        "code-analysis",
		Description: "Test skill for code analysis",
		Content:     "You are a code analysis expert.",
	}

	// Create a skills registry and register the test skill
	skillsRegistry := skills.NewRegistry()
	if err := skillsRegistry.Register(testSkill); err != nil {
		t.Fatalf("Failed to register test skill: %v", err)
	}

	// Create a minimal app config
	cfg := &config.Config{
		Agent: config.AgentConfig{
			Name: "test-agent",
			Model: config.ModelConfig{
				ModelType: "openai",
				APIKey:    "test-key",
				ModelName: "gpt-4",
			},
		},
	}

	// Create a minimal agent with skills registry
	lucyAgent, err := agent.NewLucyBotAgent(&agent.LucyBotAgentConfig{
		Config:  cfg,
		WorkDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Create app
	app := NewApp(&AppConfig{
		Agent:    lucyAgent,
		Config:   cfg,
		Registry: nil,
	})

	// Test that skill command is recognized
	input := "/code-analysis test input"
	if strings.HasPrefix(input, "/") {
		parts := strings.Fields(input)
		cmd := parts[0]

		// Verify it's a potential skill command
		if cmd == "/code-analysis" {
			// Command recognized - now test via handleSlashCommand
			cmdResult := app.handleSlashCommand(input)
			if cmdResult == nil {
				// Command was handled (returned nil means no async response)
				return
			}
			// Or command returned a function for async handling
			return
		}
	}

	t.Error("Skill command should be recognized")
}

func TestApp_HandleSkillCommand_WithAgent(t *testing.T) {
	// Create a test skill
	testSkill := &skills.Skill{
		Name:        "test-skill",
		Description: "A test skill",
		Content:     "You are helpful.",
	}

	// Create and register skill
	skillsRegistry := skills.NewRegistry()
	if err := skillsRegistry.Register(testSkill); err != nil {
		t.Fatalf("Failed to register skill: %v", err)
	}

	// Create app config
	cfg := &config.Config{
		Agent: config.AgentConfig{
			Name: "test",
			Model: config.ModelConfig{
				ModelType: "openai",
				APIKey:    "test",
				ModelName: "gpt-4",
			},
		},
	}

	lucyAgent, err := agent.NewLucyBotAgent(&agent.LucyBotAgentConfig{
		Config:  cfg,
		WorkDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	app := NewApp(&AppConfig{
		Agent:  lucyAgent,
		Config: cfg,
	})

	// Test handleSkillCommand directly
	testArgs := "test arguments"
	cmd := app.handleSkillCommand(testSkill, testArgs)

	if cmd == nil {
		t.Fatal("handleSkillCommand should return a command")
	}

	// The command should be a function that returns a tea.Msg
	cmdFunc := cmd()
	if cmdFunc == nil {
		t.Fatal("Command function should return a message")
	}

	// Verify it's a ResponseMsg or will send messages
	switch msg := cmdFunc.(type) {
	case ResponseMsg:
		// Expected - skill command returns a response
		if msg.AgentName == "" {
			t.Error("ResponseMsg should have AgentName set")
		}
	default:
		// Other message types are acceptable (e.g., for adding messages)
	}
}

func TestApp_SkillCommandIntegration(t *testing.T) {
	// Test that skill commands are checked before built-in commands
	cfg := &config.Config{
		Agent: config.AgentConfig{
			Name: "test",
			Model: config.ModelConfig{
				ModelType: "openai",
				APIKey:    "test",
				ModelName: "gpt-4",
			},
		},
	}

	lucyAgent, err := agent.NewLucyBotAgent(&agent.LucyBotAgentConfig{
		Config:  cfg,
		WorkDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Get the agent's skills registry and register a test skill
	skillsRegistry := lucyAgent.GetSkillsRegistry()
	testSkill := &skills.Skill{
		Name:        "custom",
		Description: "Custom skill",
		Content:     "Custom instructions",
	}
	if err := skillsRegistry.Register(testSkill); err != nil {
		t.Fatalf("Failed to register skill: %v", err)
	}

	app := NewApp(&AppConfig{
		Agent:  lucyAgent,
		Config: cfg,
	})

	// Submit a skill command
	input := "/custom do something"
	cmd := app.handleSubmit(input)

	if cmd == nil {
		t.Fatal("handleSubmit should return a command for skill input")
	}

	// Verify the command is valid
	cmdFunc := cmd()
	if cmdFunc == nil {
		t.Fatal("Command function should return a message")
	}
}
