package main

import (
	"fmt"

	"github.com/tingly-dev/tingly-agentscope/pkg/formatter"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

func main() {
	f := formatter.NewTeaFormatter()

	fmt.Println("╔═══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║              TeaFormatter Demonstration (Advanced Output)              ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════════╝")

	// Example 1: User message
	fmt.Println("\n📝 Example 1: User Message")
	fmt.Println("─────────────────────────────────────────────────────────────────")
	userMsg := message.NewMsg("User", "Find all Go files and show their line counts", types.RoleUser)
	fmt.Print(f.FormatMessage(userMsg))

	// Example 2: Assistant with tool use
	fmt.Println("🤖 Example 2: Assistant with Tool Use")
	fmt.Println("─────────────────────────────────────────────────────────────────")
	toolInput := map[string]types.JSONSerializable{
		"pattern": "*.go",
	}
	blocks := []message.ContentBlock{
		message.Text("I'll search for Go files and count lines."),
		&message.ToolUseBlock{
			ID:    "tool_abc123def",
			Name:  "glob_files",
			Input: toolInput,
		},
	}
	assistantMsg := message.NewMsg("Assistant", blocks, types.RoleAssistant)
	fmt.Print(f.FormatMessage(assistantMsg))

	// Example 3: Tool result
	fmt.Println("✓ Example 3: Tool Result")
	fmt.Println("─────────────────────────────────────────────────────────────────")
	resultBlocks := []message.ContentBlock{
		&message.ToolResultBlock{
			ID:   "tool_abc123def",
			Name: "glob_files",
			Output: []message.ContentBlock{
				message.Text("main.go - 150 lines\ntools.go - 320 lines\nutils.go - 85 lines\n"),
			},
		},
	}
	resultMsg := message.NewMsg("glob_files", resultBlocks, types.RoleUser)
	fmt.Print(f.FormatMessage(resultMsg))

	// Example 4: Complete tool call flow
	fmt.Println("🔄 Example 4: Complete Tool Call Flow")
	fmt.Println("─────────────────────────────────────────────────────────────────")
	toolInput2 := map[string]types.JSONSerializable{
		"path":  "main.go",
		"limit": float64(5),
	}
	blocks2 := []message.ContentBlock{
		message.Text("Let me read the first 5 lines of main.go."),
		&message.ToolUseBlock{
			ID:    "tool_xyz789",
			Name:  "view_file",
			Input: toolInput2,
		},
	}
	assistantMsg2 := message.NewMsg("Assistant", blocks2, types.RoleAssistant)
	fmt.Print(f.FormatMessage(assistantMsg2))

	resultBlocks2 := []message.ContentBlock{
		&message.ToolResultBlock{
			ID:   "tool_xyz789",
			Name: "view_file",
			Output: []message.ContentBlock{
				message.Text("    1: package main\n    2:\n    3: import \"fmt\"\n    4:\n    5: func main() {\n"),
			},
		},
	}
	resultMsg2 := message.NewMsg("view_file", resultBlocks2, types.RoleUser)
	fmt.Print(f.FormatMessage(resultMsg2))

	// Example 5: Compact mode
	fmt.Println("📊 Example 5: Compact Mode")
	fmt.Println("─────────────────────────────────────────────────────────────────")
	f2 := formatter.NewCompactTeaFormatter()
	fmt.Print(f2.FormatMessage(assistantMsg2))

	// Example 6: Monochrome mode
	fmt.Println("⚪ Example 6: Monochrome Mode (No Colors)")
	fmt.Println("─────────────────────────────────────────────────────────────────")
	f3 := formatter.NewMonochromeTeaFormatter()
	fmt.Print(f3.FormatMessage(assistantMsg2))

	fmt.Println("═══════════════════════════════════════════════════════════════════")
	fmt.Println("Demo complete!")
}
