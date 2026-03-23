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
