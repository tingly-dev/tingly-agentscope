package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSkill_MatchesTrigger(t *testing.T) {
	skill := &Skill{
		Name:     "Git Helper",
		Triggers: []string{"git", "commit", "branch"},
	}

	tests := []struct {
		input    string
		expected bool
	}{
		{"help with git", true},
		{"how to commit", true},
		{"create a branch", true},
		{"help with python", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := skill.MatchesTrigger(tt.input)
			if result != tt.expected {
				t.Errorf("MatchesTrigger(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSkill_HasTool(t *testing.T) {
	skill := &Skill{
		Name:  "Test Skill",
		Tools: []string{"bash", "view_file", "edit_file"},
	}

	tests := []struct {
		toolName string
		expected bool
	}{
		{"bash", true},
		{"view_file", true},
		{"grep", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			result := skill.HasTool(tt.toolName)
			if result != tt.expected {
				t.Errorf("HasTool(%q) = %v, want %v", tt.toolName, result, tt.expected)
			}
		})
	}
}

func TestSkill_Validate(t *testing.T) {
	tests := []struct {
		name    string
		skill   *Skill
		wantErr bool
	}{
		{
			name:    "valid skill",
			skill:   &Skill{Name: "Test", Description: "A test skill"},
			wantErr: false,
		},
		{
			name:    "missing name",
			skill:   &Skill{Name: "", Description: "A test skill"},
			wantErr: true,
		},
		{
			name:    "missing description",
			skill:   &Skill{Name: "Test", Description: ""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.skill.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSkill_FormatPrompt(t *testing.T) {
	skill := &Skill{
		Name:        "Git Helper",
		Description: "Helper for git operations",
		Tools:       []string{"bash", "view_file"},
		Triggers:    []string{"git", "commit"},
		Content:     "When helping with git...",
	}

	prompt := skill.FormatPrompt()

	if !strings.Contains(prompt, "Git Helper") {
		t.Error("Expected prompt to contain skill name")
	}
	if !strings.Contains(prompt, "Helper for git operations") {
		t.Error("Expected prompt to contain description")
	}
	if !strings.Contains(prompt, "bash") {
		t.Error("Expected prompt to contain tools")
	}
	if !strings.Contains(prompt, "When helping with git...") {
		t.Error("Expected prompt to contain content")
	}
}

func TestLoadFromMarkdown(t *testing.T) {
	// Create temp file
	tmpDir, err := os.MkdirTemp("", "lucybot-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	content := `---
name: "Test Skill"
description: "A test skill"
version: "1.0.0"
tools:
  - bash
  - view_file
triggers:
  - test
  - example
---

# Test Skill Instructions

This is the skill content.
`

	testFile := filepath.Join(tmpDir, "SKILL.md")
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	skill, err := LoadFromFile(testFile)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if skill.Name != "Test Skill" {
		t.Errorf("Expected name 'Test Skill', got %q", skill.Name)
	}
	if skill.Description != "A test skill" {
		t.Errorf("Expected description 'A test skill', got %q", skill.Description)
	}
	if len(skill.Tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(skill.Tools))
	}
	if !strings.Contains(skill.Content, "Test Skill Instructions") {
		t.Error("Expected content to contain instructions")
	}
}

func TestRegistry(t *testing.T) {
	reg := NewRegistry()

	skill1 := &Skill{Name: "Skill 1", Description: "First skill"}
	skill2 := &Skill{Name: "Skill 2", Description: "Second skill"}

	// Register skills
	if err := reg.Register(skill1); err != nil {
		t.Fatalf("Failed to register skill1: %v", err)
	}
	if err := reg.Register(skill2); err != nil {
		t.Fatalf("Failed to register skill2: %v", err)
	}

	// Try to register duplicate
	if err := reg.Register(skill1); err == nil {
		t.Error("Expected error when registering duplicate skill")
	}

	// Get skill
	s, exists := reg.Get("Skill 1")
	if !exists {
		t.Error("Expected skill1 to exist")
	}
	if s.Name != "Skill 1" {
		t.Errorf("Expected name 'Skill 1', got %q", s.Name)
	}

	// List skills
	names := reg.List()
	if len(names) != 2 {
		t.Errorf("Expected 2 skills, got %d", len(names))
	}

	// Count
	if reg.Count() != 2 {
		t.Errorf("Expected count 2, got %d", reg.Count())
	}

	// Unregister
	reg.Unregister("Skill 1")
	if reg.Count() != 1 {
		t.Errorf("Expected count 1 after unregister, got %d", reg.Count())
	}

	// Clear
	reg.Clear()
	if reg.Count() != 0 {
		t.Errorf("Expected count 0 after clear, got %d", reg.Count())
	}
}

func TestRegistry_FindByTrigger(t *testing.T) {
	reg := NewRegistry()

	reg.Register(&Skill{Name: "Git", Description: "Git skill", Triggers: []string{"git"}})
	reg.Register(&Skill{Name: "Docker", Description: "Docker skill", Triggers: []string{"docker"}})
	reg.Register(&Skill{Name: "Git Advanced", Description: "Advanced git", Triggers: []string{"git", "advanced"}})

	matches := reg.FindByTrigger("help with git")
	if len(matches) != 2 {
		t.Errorf("Expected 2 matches for 'git', got %d", len(matches))
	}

	matches = reg.FindByTrigger("docker compose")
	if len(matches) != 1 {
		t.Errorf("Expected 1 match for 'docker', got %d", len(matches))
	}

	matches = reg.FindByTrigger("python")
	if len(matches) != 0 {
		t.Errorf("Expected 0 matches for 'python', got %d", len(matches))
	}
}

func TestSkill_CommandName(t *testing.T) {
	skill := &Skill{
		Name:        "code-analysis",
		Description: "Code analysis helper",
		Content:     "Analyze code patterns",
	}

	expectedCmd := "/code-analysis"
	if skill.CommandName() != expectedCmd {
		t.Errorf("CommandName() = %v, want %v", skill.CommandName(), expectedCmd)
	}
}
