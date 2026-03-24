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

func TestSkillInjector_IsSkillLoaded(t *testing.T) {
	skill := &Skill{
		Name:        "test-skill",
		Description: "Test skill",
		Content:     "Follow these instructions",
	}

	injector := NewSkillInjector(skill)

	t.Run("memory is nil", func(t *testing.T) {
		loaded := injector.IsSkillLoaded(nil)
		if loaded {
			t.Error("IsSkillLoaded() should return false when memory is nil")
		}
	})

	t.Run("memory is empty", func(t *testing.T) {
		mem := &mockMemory{messages: []*message.Msg{}}
		loaded := injector.IsSkillLoaded(mem)
		if loaded {
			t.Error("IsSkillLoaded() should return false when memory is empty")
		}
	})

	t.Run("skill not in memory", func(t *testing.T) {
		// Create a message with a different skill
		msg := message.NewMsg("user", "Help me", types.RoleUser)
		msg.Metadata = map[string]any{
			SystemPromptMark: true,
			SkillNameMark:    "different-skill",
		}

		mem := &mockMemory{messages: []*message.Msg{msg}}
		loaded := injector.IsSkillLoaded(mem)
		if loaded {
			t.Error("IsSkillLoaded() should return false when skill is not in memory")
		}
	})

	t.Run("skill is in memory", func(t *testing.T) {
		// Create a message with the same skill
		msg := message.NewMsg("user", "Help me", types.RoleUser)
		msg.Metadata = map[string]any{
			SystemPromptMark: true,
			SkillNameMark:    "test-skill",
		}

		mem := &mockMemory{messages: []*message.Msg{msg}}
		loaded := injector.IsSkillLoaded(mem)
		if !loaded {
			t.Error("IsSkillLoaded() should return true when skill is in memory")
		}
	})

	t.Run("message without metadata", func(t *testing.T) {
		msg := message.NewMsg("user", "Help me", types.RoleUser)
		// No metadata

		mem := &mockMemory{messages: []*message.Msg{msg}}
		loaded := injector.IsSkillLoaded(mem)
		if loaded {
			t.Error("IsSkillLoaded() should return false when message has no metadata")
		}
	})

	t.Run("message with system_prompt_mark but no skill name", func(t *testing.T) {
		msg := message.NewMsg("user", "Help me", types.RoleUser)
		msg.Metadata = map[string]any{
			SystemPromptMark: true,
			// No skill name
		}

		mem := &mockMemory{messages: []*message.Msg{msg}}
		loaded := injector.IsSkillLoaded(mem)
		if loaded {
			t.Error("IsSkillLoaded() should return false when message has no skill name")
		}
	})
}

// mockMemory is a simple mock implementation of Memory for testing
type mockMemory struct {
	messages []*message.Msg
}

func (m *mockMemory) Add(ctx context.Context, msg *message.Msg) error {
	m.messages = append(m.messages, msg)
	return nil
}

func (m *mockMemory) GetMessages() []*message.Msg {
	return m.messages
}

func (m *mockMemory) GetLastN(n int) []*message.Msg {
	if n >= len(m.messages) {
		return m.messages
	}
	return m.messages[len(m.messages)-n:]
}

func (m *mockMemory) Clear() {
	m.messages = []*message.Msg{}
}

func (m *mockMemory) Size() int {
	return len(m.messages)
}
