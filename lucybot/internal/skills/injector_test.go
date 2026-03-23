package skills

import (
	"context"
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

func TestSkillInjector_Inject(t *testing.T) {
	ctx := context.Background()

	skill := &Skill{
		Name:        "test-skill",
		Description: "Test skill",
		Content:     "Follow these instructions",
	}

	injector := NewSkillInjector(skill)

	inputMsg := message.NewMsg(
		"user",
		[]message.ContentBlock{message.Text("Help me with something")},
		types.RoleUser,
	)

	result := injector.Inject(ctx, inputMsg)

	if result == inputMsg {
		t.Error("Inject() should return a new message, not the original")
	}

	content := result.GetTextContent()
	if !contains(content, "test-skill") {
		t.Errorf("Injected message should contain skill name, got: %s", content)
	}

	// Check metadata for system prompt mark
	if result.Metadata == nil {
		t.Fatal("Injected message should have metadata")
	}

	if _, hasMark := result.Metadata[SystemPromptMark]; !hasMark {
		t.Error("Injected message should have system_prompt_mark metadata")
	}

	if _, hasName := result.Metadata[SkillNameMark]; !hasName {
		t.Error("Injected message should have skill_name metadata")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
