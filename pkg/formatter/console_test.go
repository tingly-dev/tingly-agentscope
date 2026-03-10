package formatter

import (
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

func TestConsoleFormatter_FormatMessage(t *testing.T) {
	f := NewConsoleFormatter()

	t.Run("text only message", func(t *testing.T) {
		msg := message.NewMsg("test", "Hello, world!", types.RoleUser)
		output := f.FormatMessage(msg)

		// Check that role and content are present
		// Note: when colorized, [user] becomes [36muser[0m
		if !contains(output, "user") && !contains(output, "[user]") {
			t.Errorf("Expected role 'user' in output, got: %s", output)
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
				ID:    "tool_123",
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
			&message.ToolResultBlock{ID: "t1", Name: "view_file", Output: []message.ContentBlock{message.Text("content")}},
		}

		msg := message.NewMsg("agent", blocks, types.RoleAssistant)
		output := f.FormatMessage(msg)

		if !contains(output, "view_file") {
			t.Errorf("Expected 'view_file' in compact output, got: %s", output)
		}
	})

	t.Run("verbose mode shows tool input", func(t *testing.T) {
		f.Verbose = true
		f.Compact = false

		input := map[string]types.JSONSerializable{
			"path":    "/tmp/test.txt",
			"limit":   float64(100),
			"context": true,
		}

		blocks := []message.ContentBlock{
			&message.ToolUseBlock{ID: "t1", Name: "view_file", Input: input},
		}

		msg := message.NewMsg("agent", blocks, types.RoleAssistant)
		output := f.FormatMessage(msg)

		// Verbose mode should show input parameters
		if !contains(output, "path:") {
			t.Errorf("Expected 'path:' in verbose output, got: %s", output)
		}
		if !contains(output, "limit:") {
			t.Errorf("Expected 'limit:' in verbose output, got: %s", output)
		}
	})

	t.Run("non-verbose mode hides tool input", func(t *testing.T) {
		f.Verbose = false

		input := map[string]types.JSONSerializable{
			"path":  "/tmp/test.txt",
			"limit": float64(100),
		}

		blocks := []message.ContentBlock{
			&message.ToolUseBlock{ID: "t1", Name: "view_file", Input: input},
		}

		msg := message.NewMsg("agent", blocks, types.RoleAssistant)
		output := f.FormatMessage(msg)

		// Non-verbose mode should not show input parameters
		if contains(output, "path:") {
			t.Errorf("Did not expect 'path:' in non-verbose output, got: %s", output)
		}
	})
}

func TestConsoleFormatter_Colorize(t *testing.T) {
	f := NewConsoleFormatter()

	t.Run("colorized output contains ANSI codes", func(t *testing.T) {
		f.Colorize = true
		f.Verbose = false

		blocks := []message.ContentBlock{
			&message.ToolUseBlock{ID: "t1", Name: "test_tool", Input: map[string]types.JSONSerializable{}},
		}

		msg := message.NewMsg("agent", blocks, types.RoleAssistant)
		output := f.FormatMessage(msg)

		// Colorized output should contain ANSI escape sequences
		if !contains(output, "\x1b[") {
			t.Errorf("Expected ANSI codes in colorized output, got: %s", output)
		}
	})

	t.Run("non-colorized output has no ANSI codes", func(t *testing.T) {
		f.Colorize = false

		blocks := []message.ContentBlock{
			&message.ToolUseBlock{ID: "t1", Name: "test_tool", Input: map[string]types.JSONSerializable{}},
		}

		msg := message.NewMsg("agent", blocks, types.RoleAssistant)
		output := f.FormatMessage(msg)

		// Non-colorized output should not contain ANSI escape sequences
		if contains(output, "\x1b[") {
			t.Errorf("Did not expect ANSI codes in non-colorized output, got: %s", output)
		}
	})
}

func TestConsoleFormatter_FormatContentBlock(t *testing.T) {
	f := NewConsoleFormatter()

	t.Run("text block", func(t *testing.T) {
		block := message.Text("Hello, world!")
		output := f.FormatContentBlock(block)

		if !contains(output, "Hello, world!") {
			t.Errorf("Expected text content in output, got: %s", output)
		}
	})

	t.Run("thinking block in verbose mode", func(t *testing.T) {
		f.Verbose = true
		block := &message.ThinkingBlock{Thinking: "Let me think..."}
		output := f.FormatContentBlock(block)

		if !contains(output, "Thinking:") {
			t.Errorf("Expected 'Thinking:' in verbose output, got: %s", output)
		}
		if !contains(output, "Let me think...") {
			t.Errorf("Expected thinking content in verbose output, got: %s", output)
		}
	})

	t.Run("thinking block in non-verbose mode", func(t *testing.T) {
		f.Verbose = false
		block := &message.ThinkingBlock{Thinking: "Let me think..."}
		output := f.FormatContentBlock(block)

		if !contains(output, "thinking...") {
			t.Errorf("Expected 'thinking...' indicator in non-verbose output, got: %s", output)
		}
		// Should not show full thinking content
		if contains(output, "Let me think...") {
			t.Errorf("Did not expect full thinking content in non-verbose output, got: %s", output)
		}
	})
}

func TestConsoleFormatter_ToolResultDetection(t *testing.T) {
	f := NewConsoleFormatter()
	f.Colorize = false // Disable color for easier testing

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

		// Should show "ToolResult" instead of "user"
		if !contains(output, "ToolResult") {
			t.Errorf("Expected 'ToolResult' in output, got: %s", output)
		}
		// Should not show "user" for pure tool result
		if contains(output, "[user]") {
			t.Errorf("Did not expect '[user]' in pure tool result output, got: %s", output)
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

		// Should show "user" for mixed content
		if !contains(output, "user") {
			t.Errorf("Expected 'user' in mixed content output, got: %s", output)
		}
		// Should not show "ToolResult" for mixed content
		if contains(output, "ToolResult") {
			t.Errorf("Did not expect 'ToolResult' in mixed content output, got: %s", output)
		}
	})
}

func TestConsoleFormatter_Counter(t *testing.T) {
	f := NewConsoleFormatter()
	f.Colorize = false

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
