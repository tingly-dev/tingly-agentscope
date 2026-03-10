package formatter_test

import (
	"fmt"

	"github.com/tingly-dev/tingly-agentscope/pkg/formatter"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// ExampleConsoleFormatter demonstrates the console formatter output
func ExampleConsoleFormatter() {
	f := formatter.NewConsoleFormatter()

	// Example 1: Simple text message
	fmt.Println("=== Example 1: User Message ===")
	userMsg := message.NewMsg("User", "List all Go files in the current directory", types.RoleUser)
	fmt.Println(f.FormatMessage(userMsg))

	// Example 2: Assistant response with tool use
	fmt.Println("\n=== Example 2: Assistant with Tool Use ===")
	toolInput := map[string]types.JSONSerializable{
		"pattern": "*.go",
	}
	blocks := []message.ContentBlock{
		message.Text("I'll search for Go files for you."),
		&message.ToolUseBlock{
			ID:    "tool_123",
			Name:  "glob_files",
			Input: toolInput,
		},
	}
	assistantMsg := message.NewMsg("Assistant", blocks, types.RoleAssistant)
	fmt.Println(f.FormatMessage(assistantMsg))

	// Example 3: Tool result
	fmt.Println("\n=== Example 3: Tool Result ===")
	resultBlocks := []message.ContentBlock{
		&message.ToolResultBlock{
			ID:   "tool_123",
			Name: "glob_files",
			Output: []message.ContentBlock{
				message.Text("main.go\ntools.go\nutils.go\n"),
			},
		},
	}
	resultMsg := message.NewMsg("glob_files", resultBlocks, types.RoleUser)
	fmt.Println(f.FormatMessage(resultMsg))

	// Example 4: Complete tool call flow
	fmt.Println("\n=== Example 4: Complete Tool Call Flow ===")

	// Assistant decides to call a tool
	toolInput2 := map[string]types.JSONSerializable{
		"path":  "main.go",
		"limit": float64(10),
	}
	blocks2 := []message.ContentBlock{
		message.Text("I'll read the main.go file to show you the first 10 lines."),
		&message.ToolUseBlock{
			ID:    "tool_456",
			Name:  "view_file",
			Input: toolInput2,
		},
	}
	assistantMsg2 := message.NewMsg("Assistant", blocks2, types.RoleAssistant)
	fmt.Println(f.FormatMessage(assistantMsg2))

	// Tool returns result
	resultBlocks2 := []message.ContentBlock{
		&message.ToolResultBlock{
			ID:   "tool_456",
			Name: "view_file",
			Output: []message.ContentBlock{
				message.Text("    1: package main\n    2:\n    3: import \"fmt\"\n    4:\n    5: func main() {\n    6:     fmt.Println(\"Hello, World!\")\n    7: }\n    8: \n    9: // End of file\n   10: \n"),
			},
		},
	}
	resultMsg2 := message.NewMsg("view_file", resultBlocks2, types.RoleUser)
	fmt.Println(f.FormatMessage(resultMsg2))

	// Example 5: Non-verbose mode (compact)
	fmt.Println("\n=== Example 5: Non-Verbose Mode ===")
	f2 := formatter.NewConsoleFormatter()
	f2.Verbose = false
	f2.Compact = true
	fmt.Println(f2.FormatMessage(assistantMsg2))

	// Example 6: No colors
	fmt.Println("\n=== Example 6: No Colors ===")
	f3 := formatter.NewConsoleFormatter()
	f3.Colorize = false
	fmt.Println(f3.FormatMessage(assistantMsg2))
}

// ExampleConsoleFormatter_withThinking demonstrates formatter with thinking blocks
func ExampleConsoleFormatter_withThinking() {
	f := formatter.NewConsoleFormatter()

	fmt.Println("=== Example: Assistant with Thinking ===")
	blocks := []message.ContentBlock{
		&message.ThinkingBlock{Thinking: "The user wants to find Go files. I should use the glob_files tool with pattern *.go"},
		message.Text("I'll search for Go files in the current directory."),
		&message.ToolUseBlock{
			ID:    "tool_789",
			Name:  "glob_files",
			Input: map[string]types.JSONSerializable{"pattern": "*.go"},
		},
	}
	msg := message.NewMsg("Assistant", blocks, types.RoleAssistant)
	fmt.Println(f.FormatMessage(msg))
}
