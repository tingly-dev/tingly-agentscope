package main

import (
	"fmt"

	"github.com/tingly-dev/tingly-agentscope/pkg/formatter"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

func main() {
	f := formatter.NewConsoleFormatter()

	fmt.Println("╔═══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║           Console Formatter Demonstration                            ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════════╝")

	// Example 1: User message
	fmt.Println("\n📝 Example 1: User Message")
	fmt.Println("─────────────────────────────────────────────────────────────────")
	userMsg := message.NewMsg("User", "List all Go files in the current directory", types.RoleUser)
	fmt.Println(f.FormatMessage(userMsg))

	// Example 2: Assistant with tool use
	fmt.Println("\n🤖 Example 2: Assistant with Tool Use")
	fmt.Println("─────────────────────────────────────────────────────────────────")
	toolInput := map[string]any{
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
	fmt.Println("\n✓ Example 3: Tool Result")
	fmt.Println("─────────────────────────────────────────────────────────────────")
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

	// Example 4: Complete tool call flow with parameters
	fmt.Println("\n🔄 Example 4: Complete Tool Call Flow")
	fmt.Println("─────────────────────────────────────────────────────────────────")
	toolInput2 := map[string]any{
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

	resultBlocks2 := []message.ContentBlock{
		&message.ToolResultBlock{
			ID:   "tool_456",
			Name: "view_file",
			Output: []message.ContentBlock{
				message.Text("    1: package main\n    2:\n    3: import \"fmt\"\n    4:\n    5: func main() {\n    6:     fmt.Println(\"Hello!\")\n    7: }\n"),
			},
		},
	}
	resultMsg2 := message.NewMsg("view_file", resultBlocks2, types.RoleUser)
	fmt.Println(f.FormatMessage(resultMsg2))

	// Example 5: Non-verbose mode
	fmt.Println("\n📊 Example 5: Non-Verbose Mode (Compact)")
	fmt.Println("─────────────────────────────────────────────────────────────────")
	f2 := formatter.NewConsoleFormatter()
	f2.Verbose = false
	f2.Compact = true
	fmt.Println(f2.FormatMessage(assistantMsg2))

	// Example 6: Without colors
	fmt.Println("\n⚪ Example 6: Without Colors")
	fmt.Println("─────────────────────────────────────────────────────────────────")
	f3 := formatter.NewConsoleFormatter()
	f3.Colorize = false
	fmt.Println(f3.FormatMessage(assistantMsg2))

	fmt.Println("\n═══════════════════════════════════════════════════════════════════")
	fmt.Println("Demo complete!")
}
