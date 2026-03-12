package tools

import (
	"context"

	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
)

// Tool descriptions for bash tools
const (
	ToolDescExecuteBash = `Executes a given bash command with optional timeout. Working directory persists between commands; shell state (everything else) does not. The shell environment is initialized from the user's profile (bash or zsh).

IMPORTANT: This tool is for terminal operations like git, npm, docker, etc. DO NOT use it for file operations (reading, writing, searching) - use the specialized tools for those types of tasks.

Directory Verification:
- If the command will create new directories or files, first use ls to verify the parent directory exists and is the correct location
- For example, before running "mkdir foo/bar", first use ls foo to check that "foo" exists and is the intended parent directory

Command Execution:
- Always quote file paths that contain spaces with double quotes (e.g., cd "path with spaces/file.txt")
- Examples of proper quoting: cd "/Users/name/My Documents" (correct) vs cd /Users/name/My Documents (incorrect - will fail)

For simple commands (git, npm, standard CLI tools), keep descriptions brief (5-10 words):
- ls → "List files in current directory"
- git status → "Show working tree status"
- npm install → "Install package dependencies"

For commands that are harder to parse at a glance (piped commands, obscure flags, etc.), add enough context to clarify what it does:
- find . -name "*.tmp" -exec rm {} \; → "Find and delete all .tmp files recursively"
- git reset --hard origin/main → "Discard all local changes and match remote main"
- curl -s url | jq '.data[]' → "Fetch JSON from URL and extract data array elements"

You can call multiple tools in a single response. Maximize use of parallel tool calls where possible.`
	ToolDescJobDone = "Mark the task as complete when finished"
)

// BashTools wraps bash-related tools
type BashTools struct {
	session *BashSession
}

// NewBashTools creates a new BashTools instance
func NewBashTools(session *BashSession) *BashTools {
	if session == nil {
		session = GetGlobalBashSession()
	}
	return &BashTools{
		session: session,
	}
}

// ExecuteBashParams holds parameters for ExecuteBash
type ExecuteBashParams struct {
	Command string `json:"command" required:"true" description="Shell command to execute"`
	Timeout int    `json:"timeout,omitempty" description:"Timeout in seconds (default: 120)"`
}

// ExecuteBash runs a shell command with timeout
func (bt *BashTools) ExecuteBash(ctx context.Context, params ExecuteBashParams) (string, error) {
	return bt.session.executeBashInternal(ctx, params.Command, params.Timeout)
}

// JobDoneParams holds parameters for JobDone
type JobDoneParams struct{}

// JobDone marks the task as complete
func (bt *BashTools) JobDone(ctx context.Context, params JobDoneParams) (string, error) {
	return "Task completed successfully", nil
}

// GetSession returns the bash session
func (bt *BashTools) GetSession() *BashSession {
	return bt.session
}

// Constraint returns the output constraint for bash tools
// Implements the ConstrainedTool interface
func (bt *BashTools) Constraint() tool.OutputConstraint {
	// Bash commands can produce very large output (logs, build output, etc.)
	// Use moderate byte limit but high line limit for structured output
	return tool.NewDefaultConstraint(50*1024, 5000, 0, 120) // 50KB, 5000 lines, no item limit, 120s timeout
}

func init() {
	// Register bash tools in the global registry
	RegisterTool("execute_bash", ToolDescExecuteBash, "Bash Execution", true)
	RegisterTool("job_done", ToolDescJobDone, "Bash Execution", true)
}
