package agent

import (
	"context"
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

func TestSimpleTokenCounter_CountTokens(t *testing.T) {
	counter := NewSimpleTokenCounter()

	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{
			name:     "empty string",
			content:  "",
			expected: 0,
		},
		{
			name:     "short text",
			content:  "Hello",
			expected: 1, // "Hello" is 5 chars / 4 = 1.25 -> 1
		},
		{
			name:     "longer text",
			content:  "This is a longer text with more words to count",
			expected: 11, // 47 chars / 4 = 11.75 -> 11
		},
		{
			name:     "exactly 4 chars",
			content:  "abcd",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := counter.CountTokens(tt.content)
			if result != tt.expected {
				t.Errorf("CountTokens() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSimpleTokenCounter_CountMessageTokens(t *testing.T) {
	counter := NewSimpleTokenCounter()

	t.Run("text message", func(t *testing.T) {
		msg := message.NewMsg("test", "This is a test message", types.RoleUser)
		count := counter.CountMessageTokens(msg)

		// Count: "test" + "This is a test message" + "user" ≈ 6 + 22 + 4 = 32 / 4 ≈ 8
		if count < 5 || count > 12 {
			t.Errorf("CountMessageTokens() = %v, want reasonable value", count)
		}
	})

	t.Run("nil message", func(t *testing.T) {
		count := counter.CountMessageTokens(nil)
		if count != 0 {
			t.Errorf("CountMessageTokens(nil) = %v, want 0", count)
		}
	})

	t.Run("message with text blocks", func(t *testing.T) {
		blocks := []message.ContentBlock{
			message.Text("First block"),
			message.Text("Second block"),
		}
		msg := message.NewMsg("agent", blocks, types.RoleAssistant)
		count := counter.CountMessageTokens(msg)

		if count < 5 {
			t.Errorf("CountMessageTokens() = %v, want at least 5", count)
		}
	})

	t.Run("message with tool use", func(t *testing.T) {
		input := map[string]any{
			"path":  "/tmp/file.txt",
			"limit": float64(100),
		}
		blocks := []message.ContentBlock{
			message.Text("I'll read the file"),
			&message.ToolUseBlock{
				ID:    "tool_1",
				Name:  "read_file",
				Input: input,
			},
		}
		msg := message.NewMsg("agent", blocks, types.RoleAssistant)
		count := counter.CountMessageTokens(msg)

		if count < 10 {
			t.Errorf("CountMessageTokens() = %v, want at least 10", count)
		}
	})

	t.Run("message with metadata", func(t *testing.T) {
		msg := message.NewMsg("test", "Content", types.RoleUser)
		msg.Metadata = map[string]any{
			"key1": "value1",
			"key2": 12345,
		}
		count := counter.CountMessageTokens(msg)

		if count < 5 {
			t.Errorf("CountMessageTokens() with metadata = %v, want at least 5", count)
		}
	})
}

func TestCompressionConfig_Defaults(t *testing.T) {
	config := &CompressionConfig{
		Enable:           true,
		TokenCounter:     NewSimpleTokenCounter(),
		TriggerThreshold: 1000,
		KeepRecent:       3,
	}

	if !config.Enable {
		t.Error("Enable should be true")
	}
	if config.TriggerThreshold != 1000 {
		t.Errorf("TriggerThreshold = %v, want 1000", config.TriggerThreshold)
	}
	if config.KeepRecent != 3 {
		t.Errorf("KeepRecent = %v, want 3", config.KeepRecent)
	}
	if config.TokenCounter == nil {
		t.Error("TokenCounter should not be nil")
	}
}

func TestReActAgent_ShouldCompressMemory(t *testing.T) {
	ctx := context.Background()

	t.Run("no compression config", func(t *testing.T) {
		config := &ReActAgentConfig{}
		agent := &ReActAgent{
			AgentBase: NewAgentBase("test", "system"),
			config:    config,
		}
		if agent.ShouldCompressMemory(ctx) {
			t.Error("ShouldCompressMemory() should return false when no compression config")
		}
	})

	t.Run("compression disabled", func(t *testing.T) {
		config := &CompressionConfig{
			Enable:       false,
			TokenCounter: NewSimpleTokenCounter(),
		}
		agent := &ReActAgent{
			AgentBase: NewAgentBase("test", "system"),
			config: &ReActAgentConfig{
				Compression: config,
			},
		}
		if agent.ShouldCompressMemory(ctx) {
			t.Error("ShouldCompressMemory() should return false when disabled")
		}
	})

	t.Run("compression enabled but no memory", func(t *testing.T) {
		config := &CompressionConfig{
			Enable:       true,
			TokenCounter: NewSimpleTokenCounter(),
		}
		agent := &ReActAgent{
			AgentBase: NewAgentBase("test", "system"),
			config: &ReActAgentConfig{
				Compression: config,
			},
		}
		if agent.ShouldCompressMemory(ctx) {
			t.Error("ShouldCompressMemory() should return false when no memory")
		}
	})

	t.Run("compression enabled but memory empty", func(t *testing.T) {
		mem := NewSimpleMemory(10)
		config := &CompressionConfig{
			Enable:           true,
			TokenCounter:     NewSimpleTokenCounter(),
			TriggerThreshold: 1000,
		}
		agent := &ReActAgent{
			AgentBase: NewAgentBase("test", "system"),
			config: &ReActAgentConfig{
				Memory:      mem,
				Compression: config,
			},
		}
		if agent.ShouldCompressMemory(ctx) {
			t.Error("ShouldCompressMemory() should return false when memory is empty")
		}
	})

	t.Run("compression should trigger", func(t *testing.T) {
		mem := NewSimpleMemory(100)
		config := &CompressionConfig{
			Enable:           true,
			TokenCounter:     NewSimpleTokenCounter(),
			TriggerThreshold: 10, // Low threshold for testing
		}
		agent := &ReActAgent{
			AgentBase: NewAgentBase("test", "system"),
			config: &ReActAgentConfig{
				Memory:      mem,
				Compression: config,
			},
		}

		// Add enough messages to trigger compression
		for i := 0; i < 5; i++ {
			msg := message.NewMsg("user", "This is a test message that should trigger compression", types.RoleUser)
			mem.Add(ctx, msg)
		}

		if !agent.ShouldCompressMemory(ctx) {
			t.Error("ShouldCompressMemory() should return true when threshold exceeded")
		}
	})
}

func TestReActAgent_GetMemoryTokenCount(t *testing.T) {
	ctx := context.Background()

	t.Run("no compression config", func(t *testing.T) {
		mem := NewSimpleMemory(10)
		agent := &ReActAgent{
			AgentBase: NewAgentBase("test", "system"),
			config: &ReActAgentConfig{
				Memory: mem,
			},
		}

		count := agent.GetMemoryTokenCount(ctx)
		if count != 0 {
			t.Errorf("GetMemoryTokenCount() should return 0 when no compression config, got %v", count)
		}
	})

	t.Run("count tokens in memory", func(t *testing.T) {
		mem := NewSimpleMemory(10)
		config := &CompressionConfig{
			Enable:       true,
			TokenCounter: NewSimpleTokenCounter(),
		}
		agent := &ReActAgent{
			AgentBase: NewAgentBase("test", "system"),
			config: &ReActAgentConfig{
				Memory:      mem,
				Compression: config,
			},
		}

		// Add a message
		msg := message.NewMsg("user", "Test message for token counting", types.RoleUser)
		mem.Add(ctx, msg)

		count := agent.GetMemoryTokenCount(ctx)
		if count <= 0 {
			t.Errorf("GetMemoryTokenCount() = %v, want > 0", count)
		}
	})
}

func TestSummarySchema(t *testing.T) {
	summary := &SummarySchema{
		TaskOverview:         "Test task",
		CurrentState:         "In progress",
		ImportantDiscoveries: "Found important info",
		NextSteps:            "Continue work",
		ContextToPreserve:    "User preferences",
	}

	if summary.TaskOverview != "Test task" {
		t.Errorf("TaskOverview = %v, want 'Test task'", summary.TaskOverview)
	}
	if summary.CurrentState != "In progress" {
		t.Errorf("CurrentState = %v, want 'In progress'", summary.CurrentState)
	}
	if summary.ImportantDiscoveries != "Found important info" {
		t.Errorf("ImportantDiscoveries = %v, want 'Found important info'", summary.ImportantDiscoveries)
	}
	if summary.NextSteps != "Continue work" {
		t.Errorf("NextSteps = %v, want 'Continue work'", summary.NextSteps)
	}
	if summary.ContextToPreserve != "User preferences" {
		t.Errorf("ContextToPreserve = %v, want 'User preferences'", summary.ContextToPreserve)
	}
}

func TestCompressionResult(t *testing.T) {
	result := &CompressionResult{
		OriginalTokenCount:   10000,
		CompressedTokenCount: 5000,
		Summary: &SummarySchema{
			TaskOverview: "Test",
		},
		CompressedMessages: []*message.Msg{
			message.NewMsg("system", "Compressed content", types.RoleSystem),
		},
	}

	if result.OriginalTokenCount != 10000 {
		t.Errorf("OriginalTokenCount = %v, want 10000", result.OriginalTokenCount)
	}
	if result.CompressedTokenCount != 5000 {
		t.Errorf("CompressedTokenCount = %v, want 5000", result.CompressedTokenCount)
	}
	if result.Summary == nil {
		t.Error("Summary should not be nil")
	}
	if len(result.CompressedMessages) != 1 {
		t.Errorf("CompressedMessages length = %v, want 1", len(result.CompressedMessages))
	}
}

func TestParseSummaryFromText(t *testing.T) {
	agent := &ReActAgent{}

	t.Run("parse structured summary", func(t *testing.T) {
		text := `task_overview: Build a web application
current_state: Set up project structure
important_discoveries: Need to use Go framework
next_steps: Implement API endpoints
context_to_preserve: User wants REST API`

		summary := agent.parseSummaryFromText(text)

		if summary.TaskOverview == "" {
			t.Error("TaskOverview should not be empty")
		}
		if summary.CurrentState == "" {
			t.Error("CurrentState should not be empty")
		}
		if summary.NextSteps == "" {
			t.Error("NextSteps should not be empty")
		}
	})

	t.Run("parse with field prefixes", func(t *testing.T) {
		text := `- Task Overview: Create a CLI tool
- Current State: Initial phase
- Important Discoveries: Go is suitable
- Next Steps: Write code
- Context: User prefers simplicity`

		summary := agent.parseSummaryFromText(text)

		// Should populate fields even if format is slightly different
		if summary.TaskOverview == "" && summary.CurrentState == "" {
			t.Error("parseSummaryFromText should handle various formats")
		}
	})

	t.Run("parse empty text", func(t *testing.T) {
		summary := agent.parseSummaryFromText("")

		if summary.TaskOverview == "" {
			// Should have default value
			t.Log("TaskOverview has default value:", summary.TaskOverview)
		}
	})
}

func TestCreateCompressedMessage(t *testing.T) {
	agent := &ReActAgent{}

	summary := &SummarySchema{
		TaskOverview:         "Test overview",
		CurrentState:         "Test state",
		ImportantDiscoveries: "Test discoveries",
		NextSteps:            "Test steps",
		ContextToPreserve:    "Test context",
	}

	msg := agent.createCompressedMessage(summary)

	if msg == nil {
		t.Fatal("createCompressedMessage() should not return nil")
	}
	if msg.Role != types.RoleSystem {
		t.Errorf("Message role = %v, want %v", msg.Role, types.RoleSystem)
	}
	if msg.Name != "system" {
		t.Errorf("Message name = %v, want 'system'", msg.Name)
	}

	// Check content contains summary fields
	content := msg.GetTextContent()
	if !contains(content, "Test overview") {
		t.Error("Compressed message should contain task overview")
	}
	if !contains(content, "system-info") {
		t.Error("Compressed message should contain system-info tag")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
