package skills

import (
	"testing"
)

func TestCommandRegistry_Register(t *testing.T) {
	registry := NewCommandRegistry()

	skill := &Skill{
		Name:        "code-analysis",
		Description: "Analyze code",
		Content:     "Code analysis instructions",
	}

	err := registry.Register(skill)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	cmds := registry.ListCommands()
	if len(cmds) != 1 {
		t.Errorf("ListCommands() = %v, want 1 command", len(cmds))
	}

	if cmds[0] != "/code-analysis" {
		t.Errorf("ListCommands()[0] = %v, want /code-analysis", cmds[0])
	}
}

func TestCommandRegistry_Get(t *testing.T) {
	registry := NewCommandRegistry()

	skill := &Skill{
		Name:        "test-skill",
		Description: "Test",
		Content:     "Content",
	}

	registry.Register(skill)

	retrieved, ok := registry.Get("/test-skill")
	if !ok {
		t.Fatal("Get() should return true for registered command")
	}

	if retrieved.Name != skill.Name {
		t.Errorf("Get() name = %v, want %v", retrieved.Name, skill.Name)
	}
}
