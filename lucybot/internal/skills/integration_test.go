package skills

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// TestSkillsIntegration_EndToEnd tests the complete skills workflow
// from discovery to command execution and memory preservation
func TestSkillsIntegration_EndToEnd(t *testing.T) {
	// Create temporary directory for skills
	tmpDir, err := os.MkdirTemp("", "skills-integration-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test skill file
	skillContent := `---
name: "test-code-review"
description: "Reviews code for best practices and potential issues"
version: "1.0.0"
author: "test"
tools: ["view_source", "grep"]
triggers: ["review", "code review", "check code"]
categories: ["code quality"]
---
# Code Review Skill

This skill helps review code for:
- Best practices
- Security issues
- Performance concerns
- Code style

## Instructions

When reviewing code:
1. Check for security vulnerabilities
2. Verify best practices are followed
3. Look for performance optimizations
4. Ensure consistent code style
5. Provide actionable feedback
`

	skillPath := filepath.Join(tmpDir, "code-review.md")
	if err := os.WriteFile(skillPath, []byte(skillContent), 0644); err != nil {
		t.Fatalf("Failed to write skill file: %v", err)
	}

	t.Run("discover_and_load_skill", func(t *testing.T) {
		// Create discovery
		discovery := NewDiscovery([]string{tmpDir})

		// Discover skills
		skillsList, err := discovery.Discover()
		if err != nil {
			t.Fatalf("Failed to discover skills: %v", err)
		}

		if len(skillsList) != 1 {
			t.Errorf("Expected 1 skill, got %d", len(skillsList))
		}

		skill := skillsList[0]
		if skill.Name != "test-code-review" {
			t.Errorf("Expected skill name 'test-code-review', got %q", skill.Name)
		}

		if skill.Description != "Reviews code for best practices and potential issues" {
			t.Errorf("Unexpected description: %q", skill.Description)
		}
	})

	t.Run("register_skill_and_command", func(t *testing.T) {
		// Create registry
		registry := NewRegistry()

		// Create discovery
		discovery := NewDiscovery([]string{tmpDir})

		// Load skills from discovery
		if err := registry.LoadFromDiscovery(discovery); err != nil {
			t.Fatalf("Failed to load skills: %v", err)
		}

		// Check skill is registered
		skill, ok := registry.Get("test-code-review")
		if !ok {
			t.Fatal("Skill not found in registry")
		}

		// Check command is registered
		cmdRegistry := registry.GetCommandRegistry()
		cmdName := skill.CommandName()
		retrievedSkill, ok := cmdRegistry.Get(cmdName)
		if !ok {
			t.Fatalf("Command %q not registered", cmdName)
		}

		if retrievedSkill.Name != skill.Name {
			t.Errorf("Retrieved skill name mismatch: got %q, want %q", retrievedSkill.Name, skill.Name)
		}
	})

	t.Run("inject_skill_into_message", func(t *testing.T) {
		// Load skill
		skill, err := LoadFromFile(skillPath)
		if err != nil {
			t.Fatalf("Failed to load skill: %v", err)
		}

		// Create injector
		injector := NewSkillInjector(skill)

		// Create user message
		userMsg := message.NewMsg(
			"user",
			[]message.ContentBlock{message.Text("Please review my code")},
			types.RoleUser,
		)

		// Inject skill content
		ctx := context.Background()
		injectedMsg := injector.Inject(ctx, userMsg)

		// Verify injection
		content := injectedMsg.GetTextContent()
		if !contains(content, "test-code-review") {
			t.Error("Injected message should contain skill name")
		}

		if !contains(content, "Reviews code for best practices") {
			t.Error("Injected message should contain skill description")
		}

		if !contains(content, "Please review my code") {
			t.Error("Injected message should contain original user content")
		}

		// Verify metadata
		if injectedMsg.Metadata == nil {
			t.Fatal("Injected message should have metadata")
		}

		if mark, ok := injectedMsg.Metadata[SystemPromptMark].(bool); !ok || !mark {
			t.Error("Injected message should have system_prompt_mark set to true")
		}

		if skillName, ok := injectedMsg.Metadata[SkillNameMark].(string); !ok || skillName != "test-code-review" {
			t.Errorf("Expected skill_name 'test-code-review', got %q", skillName)
		}
	})

	t.Run("check_if_skill_loaded_in_memory", func(t *testing.T) {
		// Load skill
		skill, err := LoadFromFile(skillPath)
		if err != nil {
			t.Fatalf("Failed to load skill: %v", err)
		}

		// Create injector
		injector := NewSkillInjector(skill)

		// Test with empty memory
		emptyMem := &mockMemory{messages: []*message.Msg{}}
		if injector.IsSkillLoaded(emptyMem) {
			t.Error("Skill should not be loaded in empty memory")
		}

		// Test with skill message in memory
		ctx := context.Background()
		userMsg := message.NewMsg(
			"user",
			[]message.ContentBlock{message.Text("Test")},
			types.RoleUser,
		)
		injectedMsg := injector.Inject(ctx, userMsg)

		memWithSkill := &mockMemory{messages: []*message.Msg{injectedMsg}}
		if !injector.IsSkillLoaded(memWithSkill) {
			t.Error("Skill should be loaded in memory with skill message")
		}

		// Test with different skill in memory
		differentSkillMsg := message.NewMsg("user", "Test", types.RoleUser)
		differentSkillMsg.Metadata = map[string]any{
			SystemPromptMark: true,
			SkillNameMark:    "different-skill",
		}

		memWithDifferent := &mockMemory{messages: []*message.Msg{differentSkillMsg}}
		if injector.IsSkillLoaded(memWithDifferent) {
			t.Error("Skill should not be considered loaded if different skill is in memory")
		}
	})

	t.Run("command_name_generation", func(t *testing.T) {
		// Load skill
		skill, err := LoadFromFile(skillPath)
		if err != nil {
			t.Fatalf("Failed to load skill: %v", err)
		}

		// Check command name
		cmd := skill.CommandName()
		expectedCmd := "/test-code-review"
		if cmd != expectedCmd {
			t.Errorf("Expected command name %q, got %q", expectedCmd, cmd)
		}
	})

	t.Run("skill_triggers_matching", func(t *testing.T) {
		// Load skill
		skill, err := LoadFromFile(skillPath)
		if err != nil {
			t.Fatalf("Failed to load skill: %v", err)
		}

		// Test trigger matching
		testCases := []struct {
			input    string
			expected bool
		}{
			{"please review my code", true},
			{"can you do a code review?", true},
			{"check code for issues", true}, // "check code" matches "check code" trigger
			{"help with debugging", false},
			{"write some code", false},
		}

		for _, tc := range testCases {
			t.Run(tc.input, func(t *testing.T) {
				result := skill.MatchesTrigger(tc.input)
				if result != tc.expected {
					t.Errorf("MatchesTrigger(%q) = %v, want %v", tc.input, result, tc.expected)
				}
			})
		}
	})
}

// TestSkillsIntegration_MultipleSkills tests handling multiple skills
func TestSkillsIntegration_MultipleSkills(t *testing.T) {
	// Create temporary directory for skills
	tmpDir, err := os.MkdirTemp("", "skills-multi-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create multiple skill files
	skills := []struct {
		name        string
		description string
		content     string
	}{
		{
			name:        "debug-helper",
			description: "Helps debug issues",
			content:     "# Debug Helper\n\nDebugging instructions here.",
		},
		{
			name:        "git-expert",
			description: "Git operations expert",
			content:     "# Git Expert\n\nGit instructions here.",
		},
		{
			name:        "test-writer",
			description: "Writes unit tests",
			content:     "# Test Writer\n\nTest writing instructions.",
		},
	}

	for _, skill := range skills {
		skillContent := "---\n"
		skillContent += "name: \"" + skill.name + "\"\n"
		skillContent += "description: \"" + skill.description + "\"\n"
		skillContent += "version: \"1.0.0\"\n"
		skillContent += "---\n"
		skillContent += skill.content

		skillPath := filepath.Join(tmpDir, skill.name+".md")
		if err := os.WriteFile(skillPath, []byte(skillContent), 0644); err != nil {
			t.Fatalf("Failed to write skill file %s: %v", skill.name, err)
		}
	}

	// Create registry and discovery
	registry := NewRegistry()
	discovery := NewDiscovery([]string{tmpDir})

	// Load all skills
	if err := registry.LoadFromDiscovery(discovery); err != nil {
		t.Fatalf("Failed to load skills: %v", err)
	}

	// Verify all skills are loaded
	if registry.Count() != len(skills) {
		t.Errorf("Expected %d skills, got %d", len(skills), registry.Count())
	}

	// Verify all commands are registered
	cmdRegistry := registry.GetCommandRegistry()
	commands := cmdRegistry.ListCommands()
	if len(commands) != len(skills) {
		t.Errorf("Expected %d commands, got %d", len(skills), len(commands))
	}

	// Verify each skill has a unique command
	cmdMap := make(map[string]bool)
	for _, cmd := range commands {
		if cmdMap[cmd] {
			t.Errorf("Duplicate command found: %s", cmd)
		}
		cmdMap[cmd] = true
	}
}

// TestSkillsIntegration_CommandRegistry tests the command registry integration
func TestSkillsIntegration_CommandRegistry(t *testing.T) {
	registry := NewRegistry()

	// Create test skills
	skill1 := &Skill{
		Name:        "skill-one",
		Description: "First skill",
		Content:     "Content 1",
	}

	skill2 := &Skill{
		Name:        "skill-two",
		Description: "Second skill",
		Content:     "Content 2",
	}

	// Register skills
	if err := registry.Register(skill1); err != nil {
		t.Fatalf("Failed to register skill1: %v", err)
	}

	if err := registry.Register(skill2); err != nil {
		t.Fatalf("Failed to register skill2: %v", err)
	}

	cmdRegistry := registry.GetCommandRegistry()

	// Test command retrieval
	skill, ok := cmdRegistry.Get("/skill-one")
	if !ok {
		t.Error("Failed to get /skill-one command")
	}

	if skill.Name != "skill-one" {
		t.Errorf("Expected skill name 'skill-one', got %q", skill.Name)
	}

	// Test command listing
	commands := cmdRegistry.ListCommands()
	if len(commands) != 2 {
		t.Errorf("Expected 2 commands, got %d", len(commands))
	}

	// Test unregistering
	registry.Unregister("skill-one")
	commands = cmdRegistry.ListCommands()
	if len(commands) != 1 {
		t.Errorf("Expected 1 command after unregister, got %d", len(commands))
	}

	_, ok = cmdRegistry.Get("/skill-one")
	if ok {
		t.Error("Skill-one should be unregistered")
	}
}
