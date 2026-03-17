package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// BashTool provides bash command execution capabilities
type BashTool struct {
	allowedCommands []string
	blockedCommands []string
	timeout         time.Duration
	workingDir      string
	allowChaining   bool
}

// BashOptions configures the BashTool
func BashOptions(allowedCommands, blockedCommands []string, timeout time.Duration, workingDir string) func(*BashTool) {
	return func(b *BashTool) {
		b.allowedCommands = allowedCommands
		b.blockedCommands = blockedCommands
		b.timeout = timeout
		b.workingDir = workingDir
	}
}

// BashAllowChaining enables or disables command chaining (default: false)
func BashAllowChaining(allow bool) func(*BashTool) {
	return func(b *BashTool) {
		b.allowChaining = allow
	}
}

// NewBashTool creates a new bash tool instance
func NewBashTool(options ...func(*BashTool)) *BashTool {
	bt := &BashTool{
		allowedCommands: []string{}, // Empty means allow all (unless blocked)
		blockedCommands: []string{"rm -rf /", "rm -rf /*", "> /dev/sda", "mkfs", "dd if=/dev/zero"},
		timeout:         120 * time.Second,
		workingDir:      "",
		allowChaining:   false,
	}
	for _, opt := range options {
		opt(bt)
	}
	return bt
}

// BashParams defines the parameters for the bash tool
type BashParams struct {
	Command string `json:"command" description:"Bash command to execute"`
	Timeout int    `json:"timeout,omitempty" description:"Timeout in seconds (optional, no default timeout)"`
}

// Bash executes a bash command in the current working directory
func (b *BashTool) Bash(ctx context.Context, params BashParams) (*tool.ToolResponse, error) {
	// Validate command
	if err := b.validateCommand(params.Command); err != nil {
		return tool.TextResponse(fmt.Sprintf("Error: %v", err)), nil
	}

	// Use provided timeout or default
	timeout := b.timeout
	if params.Timeout > 0 {
		timeout = time.Duration(params.Timeout) * time.Second
	}

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute command using bash -c
	cmd := exec.CommandContext(execCtx, "bash", "-c", params.Command)

	// Set working directory if specified
	if b.workingDir != "" {
		cmd.Dir = b.workingDir
	}

	// Capture output
	output, err := cmd.CombinedOutput()

	// Handle timeout
	if execCtx.Err() == context.DeadlineExceeded {
		return tool.TextResponse(fmt.Sprintf("Error: command timed out after %v", timeout)), nil
	}

	result := string(output)

	// Handle error
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return tool.TextResponse(fmt.Sprintf("Error: command exited with code %d\n%s", exitErr.ExitCode(), result)), nil
		}
		return tool.TextResponse(fmt.Sprintf("Error: failed to execute command: %v\n%s", err, result)), nil
	}

	return tool.TextResponse(result), nil
}

// validateCommand checks if the command is allowed
func (b *BashTool) validateCommand(command string) error {
	trimmed := strings.TrimSpace(command)

	// Check for command chaining if not allowed
	if !b.allowChaining {
		// Check for common command chaining patterns
		chainingPatterns := []string{"&&", "||", ";", "|", "`", "$("}
		for _, pattern := range chainingPatterns {
			if strings.Contains(trimmed, pattern) {
				return fmt.Errorf("command chaining not allowed: pattern '%s' detected", pattern)
			}
		}
	}

	// Check blocked commands (trimmed to avoid whitespace bypass)
	for _, blocked := range b.blockedCommands {
		blockedTrimmed := strings.TrimSpace(blocked)
		if strings.Contains(trimmed, blockedTrimmed) {
			return fmt.Errorf("command contains blocked pattern: %s", blockedTrimmed)
		}
	}

	// If allowed commands list is empty, allow all (except blocked)
	if len(b.allowedCommands) == 0 {
		return nil
	}

	// Check if command starts with an allowed command
	for _, allowed := range b.allowedCommands {
		if strings.HasPrefix(trimmed, allowed) {
			return nil
		}
	}

	return fmt.Errorf("command not in allowed list")
}

// RegisterBashTool registers the bash tool with the toolkit
// Note: This helper is provided for convenience, but NewExtensionToolkit
// automatically registers all tools. Use this if you want to register
// the bash tool separately.
func RegisterBashTool(tk *tool.Toolkit, options ...func(*BashTool)) error {
	bt := NewBashTool(options...)
	// Use RegisterAll to auto-register the Bash method
	descriptions := map[string]string{
		"Bash": "Execute a bash command in the current working directory. Returns stdout and stderr. Optionally provide a timeout in seconds.",
	}
	return tk.RegisterAll(bt, descriptions)
}

// Call implements the ToolCallable interface for programmatic use
func (b *BashTool) Call(ctx context.Context, kwargs map[string]any) (*tool.ToolResponse, error) {
	params := BashParams{}
	if command, ok := kwargs["command"].(string); ok {
		params.Command = command
	}
	if timeout, ok := kwargs["timeout"].(float64); ok {
		params.Timeout = int(timeout)
	}
	return b.Bash(ctx, params)
}

// ToToolUseBlock converts parameters to a ToolUseBlock for agent use
func (b *BashTool) ToToolUseBlock(params BashParams) *message.ToolUseBlock {
	input := map[string]types.JSONSerializable{
		"command": params.Command,
	}
	if params.Timeout > 0 {
		input["timeout"] = params.Timeout
	}
	return &message.ToolUseBlock{
		Name:  "bash",
		Input: input,
	}
}
