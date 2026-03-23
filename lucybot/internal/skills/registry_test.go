package skills

import (
	"testing"
)

func TestRegistry_CommandRegistry(t *testing.T) {
	registry := NewRegistry()

	skill := &Skill{
		Name:        "test-skill",
		Description: "Test",
		Content:     "Content",
	}

	registry.Register(skill)

	cmdRegistry := registry.GetCommandRegistry()
	if cmdRegistry == nil {
		t.Fatal("GetCommandRegistry() should not return nil")
	}

	skillFromCmd, ok := cmdRegistry.Get("/test-skill")
	if !ok {
		t.Fatal("Command should be registered")
	}

	if skillFromCmd.Name != skill.Name {
		t.Errorf("Skill name = %v, want %v", skillFromCmd.Name, skill.Name)
	}
}

func TestRegistry_CommandRegistryDuplicate(t *testing.T) {
	registry := NewRegistry()

	skill1 := &Skill{
		Name:        "code-analysis",
		Description: "Test",
		Content:     "Content",
	}

	skill2 := &Skill{
		Name:        "Code Analysis", // Different name, same command
		Description: "Test",
		Content:     "Content",
	}

	err := registry.Register(skill1)
	if err != nil {
		t.Fatalf("Register() first skill error = %v", err)
	}

	err = registry.Register(skill2)
	if err == nil {
		t.Error("Register() should return error for duplicate command")
	}

	// Verify first skill is still registered
	skill, ok := registry.Get("code-analysis")
	if !ok {
		t.Error("First skill should still be registered after failed duplicate")
	}
	if skill.Name != skill1.Name {
		t.Errorf("Got skill name = %v, want %v", skill.Name, skill1.Name)
	}
}
