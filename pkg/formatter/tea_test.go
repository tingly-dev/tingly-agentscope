package formatter

import (
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

func TestTeaFormatter_FormatMessage(t *testing.T) {
	f := NewTeaFormatter()

	t.Run("text only message", func(t *testing.T) {
		msg := message.NewMsg("test", "Hello, world!", types.RoleUser)
		output := f.FormatMessage(msg)

		// Check that role and content are present
		if !contains(output, "USER") {
			t.Errorf("Expected 'USER' role in output, got: %s", output)
		}
		if !contains(output, "Hello, world!") {
			t.Errorf("Expected 'Hello, world!' in output, got: %s", output)
		}
	})

	t.Run("message with tool use", func(t *testing.T) {
		input := map[string]types.JSONSerializable{
			"path":    "/tmp/test.txt",
			"limit":   float64(10),
			"pattern": "*.go",
		}

		blocks := []message.ContentBlock{
			message.Text("I'll search for Go files"),
			&message.ToolUseBlock{
				ID:    "tool_123456",
				Name:  "grep_files",
				Input: input,
			},
		}

		msg := message.NewMsg("agent", blocks, types.RoleAssistant)
		output := f.FormatMessage(msg)

		// Check that tool use is formatted
		if !contains(output, "grep_files") {
			t.Errorf("Expected 'grep_files' in output, got: %s", output)
		}
		if !contains(output, "▶") {
			t.Errorf("Expected tool use indicator '▶' in output, got: %s", output)
		}
	})

	t.Run("message with tool result", func(t *testing.T) {
		blocks := []message.ContentBlock{
			&message.ToolResultBlock{
				ID:     "tool_123",
				Name:   "grep_files",
				Output: []message.ContentBlock{message.Text("Found 5 files")},
			},
		}

		msg := message.NewMsg("grep_files", blocks, types.RoleUser)
		output := f.FormatMessage(msg)

		// Check that tool result is formatted
		if !contains(output, "grep_files") {
			t.Errorf("Expected 'grep_files' in output, got: %s", output)
		}
		if !contains(output, "◀") {
			t.Errorf("Expected tool result indicator '◀' in output, got: %s", output)
		}
	})

	t.Run("compact mode", func(t *testing.T) {
		f.Compact = true

		input := map[string]types.JSONSerializable{"path": "/tmp/test.txt"}
		blocks := []message.ContentBlock{
			message.Text("Searching..."),
			&message.ToolUseBlock{ID: "t1", Name: "view_file", Input: input},
		}

		msg := message.NewMsg("agent", blocks, types.RoleAssistant)
		output := f.FormatMessage(msg)

		if !contains(output, "view_file") {
			t.Errorf("Expected 'view_file' in compact output, got: %s", output)
		}
	})

	t.Run("no color mode", func(t *testing.T) {
		f.NoColor = true

		blocks := []message.ContentBlock{
			&message.ToolUseBlock{ID: "t1", Name: "test_tool", Input: map[string]types.JSONSerializable{}},
		}

		msg := message.NewMsg("agent", blocks, types.RoleAssistant)
		output := f.FormatMessage(msg)

		// No color mode should not contain ANSI codes
		if contains(output, "\x1b[") {
			t.Errorf("Did not expect ANSI codes in no-color output, got: %s", output)
		}
	})
}

func TestTeaFormatter_FormatContentBlock(t *testing.T) {
	f := NewTeaFormatter()

	t.Run("text block", func(t *testing.T) {
		block := message.Text("Hello, world!")
		output := f.FormatContentBlock(block)

		if !contains(output, "Hello, world!") {
			t.Errorf("Expected text content in output, got: %s", output)
		}
	})

	t.Run("thinking block", func(t *testing.T) {
		block := &message.ThinkingBlock{Thinking: "Let me think..."}
		output := f.FormatContentBlock(block)

		if !contains(output, "💭") {
			t.Errorf("Expected thinking icon in output, got: %s", output)
		}
	})
}

func TestNewCompactTeaFormatter(t *testing.T) {
	f := NewCompactTeaFormatter()

	if !f.Compact {
		t.Error("Expected compact mode to be enabled")
	}
	if f.ShowToolIDs {
		t.Error("Expected tool IDs to be hidden in compact mode")
	}
	if f.ShowTimestamps {
		t.Error("Expected timestamps to be hidden in compact mode")
	}
}

func TestNewMonochromeTeaFormatter(t *testing.T) {
	f := NewMonochromeTeaFormatter()

	if !f.NoColor {
		t.Error("Expected no color mode to be enabled")
	}
}

func TestTeaFormatter_ToolResultDetection(t *testing.T) {
	f := NewTeaFormatter()
	f.NoColor = true // Disable color for easier testing

	t.Run("pure tool result message shows ToolResult", func(t *testing.T) {
		blocks := []message.ContentBlock{
			&message.ToolResultBlock{
				ID:     "tool_123",
				Name:   "grep_files",
				Output: []message.ContentBlock{message.Text("Found 5 files")},
			},
		}

		msg := message.NewMsg("grep_files", blocks, types.RoleUser)
		output := f.FormatMessage(msg)

		// Should show "TOOLRESULT" instead of "USER"
		if !contains(output, "TOOLRESULT") {
			t.Errorf("Expected 'TOOLRESULT' in output, got: %s", output)
		}
		// Should show checkmark icon for ToolResult
		if !contains(output, "✓") {
			t.Errorf("Expected checkmark icon in ToolResult output, got: %s", output)
		}
	})

	t.Run("mixed content message shows user role", func(t *testing.T) {
		blocks := []message.ContentBlock{
			message.Text("Here are the results:"),
			&message.ToolResultBlock{
				ID:     "tool_123",
				Name:   "grep_files",
				Output: []message.ContentBlock{message.Text("Found 5 files")},
			},
		}

		msg := message.NewMsg("mixed", blocks, types.RoleUser)
		output := f.FormatMessage(msg)

		// Should show "USER" for mixed content
		if !contains(output, "USER") {
			t.Errorf("Expected 'USER' in mixed content output, got: %s", output)
		}
		// Should not show "TOOLRESULT" for mixed content
		if contains(output, "TOOLRESULT") {
			t.Errorf("Did not expect 'TOOLRESULT' in mixed content output, got: %s", output)
		}
	})
}

func TestTeaFormatter_Counter(t *testing.T) {
	f := NewTeaFormatter()
	f.NoColor = true
	f.ShowTimestamps = false // Hide timestamps for cleaner output

	t.Run("counter is displayed when set", func(t *testing.T) {
		f.SetRound(1)
		f.SetStep(1)

		msg := message.NewMsg("test", "Hello", types.RoleUser)
		output := f.FormatMessage(msg)

		// Should show counter [1.1]
		if !contains(output, "[1.1]") {
			t.Errorf("Expected '[1.1]' counter in output, got: %s", output)
		}
	})

	t.Run("NextStep increments counter", func(t *testing.T) {
		f.SetRound(1)
		f.SetStep(1)
		f.NextStep()

		msg := message.NewMsg("test", "Hello", types.RoleUser)
		output := f.FormatMessage(msg)

		// Should show counter [1.2]
		if !contains(output, "[1.2]") {
			t.Errorf("Expected '[1.2]' counter in output, got: %s", output)
		}
	})

	t.Run("NextRound increments round and resets step", func(t *testing.T) {
		f.SetRound(1)
		f.SetStep(5)
		f.NextRound()

		msg := message.NewMsg("test", "Hello", types.RoleUser)
		output := f.FormatMessage(msg)

		// Should show counter [2.1]
		if !contains(output, "[2.1]") {
			t.Errorf("Expected '[2.1]' counter in output, got: %s", output)
		}
	})

	t.Run("no counter when not set", func(t *testing.T) {
		f.ResetCounters()

		msg := message.NewMsg("test", "Hello", types.RoleUser)
		output := f.FormatMessage(msg)

		// Should not show counter pattern [x.y]
		if contains(output, "[0.") || contains(output, "[1.") {
			t.Errorf("Did not expect counter in output when not set, got: %s", output)
		}
	})
}
