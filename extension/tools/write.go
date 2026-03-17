package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
)

// WriteTool provides file writing capabilities
type WriteTool struct {
	allowedDirs    []string
	allowOverwrite bool
	maxWriteSize   int64
}

// WriteOptions configures the WriteTool
func WriteOptions(allowedDirs []string, allowOverwrite bool) func(*WriteTool) {
	return func(w *WriteTool) {
		w.allowedDirs = allowedDirs
		w.allowOverwrite = allowOverwrite
	}
}

// WriteMaxSize configures the maximum write size for WriteTool
func WriteMaxSize(maxSize int64) func(*WriteTool) {
	return func(w *WriteTool) {
		w.maxWriteSize = maxSize
	}
}

// NewWriteTool creates a new write tool instance
func NewWriteTool(options ...func(*WriteTool)) *WriteTool {
	wt := &WriteTool{
		allowedDirs:    []string{}, // Empty means allow all
		allowOverwrite: true,
		maxWriteSize:   10 * 1024 * 1024, // 10MB default
	}
	for _, opt := range options {
		opt(wt)
	}
	return wt
}

// WriteParams defines the parameters for the write tool
type WriteParams struct {
	Path    string `json:"path" jsonschema:"description=Path to the file to write (relative or absolute)"`
	Content string `json:"content" jsonschema:"description=Content to write to the file"`
}

// Write writes content to a file. Creates the file if it doesn't exist, overwrites if it does.
func (w *WriteTool) Write(ctx context.Context, params WriteParams) (*tool.ToolResponse, error) {
	// Check write size
	contentSize := int64(len(params.Content))
	if contentSize > w.maxWriteSize {
		return tool.TextResponse(fmt.Sprintf("Error: content too large (%d bytes, max %d bytes)", contentSize, w.maxWriteSize)), nil
	}

	// Validate path
	if err := validatePath(params.Path, w.allowedDirs); err != nil {
		return tool.TextResponse(fmt.Sprintf("Error: %v", err)), nil
	}

	// Get absolute path
	absPath, err := filepath.Abs(params.Path)
	if err != nil {
		return tool.TextResponse(fmt.Sprintf("Error: failed to resolve path: %v", err)), nil
	}

	// Check if file exists
	exists := false
	if _, err := os.Stat(absPath); err == nil {
		exists = true
	}

	if exists && !w.allowOverwrite {
		return tool.TextResponse(fmt.Sprintf("Error: file already exists and overwrite is not allowed: %s", params.Path)), nil
	}

	// Create parent directories if needed
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return tool.TextResponse(fmt.Sprintf("Error: failed to create parent directories: %v", err)), nil
	}

	// Write file
	if err := os.WriteFile(absPath, []byte(params.Content), 0644); err != nil {
		return tool.TextResponse(fmt.Sprintf("Error: failed to write file: %v", err)), nil
	}

	action := "created"
	if exists {
		action = "overwritten"
	}

	return tool.TextResponse(fmt.Sprintf("Successfully %s file: %s (%d bytes)", action, params.Path, len(params.Content))), nil
}

// RegisterWriteTool registers the write tool with the toolkit
func RegisterWriteTool(tk *tool.Toolkit, options ...func(*WriteTool)) error {
	wt := NewWriteTool(options...)
	return tk.Register(wt.Write, &tool.RegisterOptions{
		GroupName:       "basic",
		FuncName:        "write",
		FuncDescription: "Write content to a file. Creates the file if it doesn't exist, overwrites if it does. Automatically creates parent directories.",
	})
}

// ToToolUseBlock converts parameters to a ToolUseBlock for agent use
func (w *WriteTool) ToToolUseBlock(params WriteParams) *message.ToolUseBlock {
	return &message.ToolUseBlock{
		Name: "write",
		Input: map[string]any{
			"path":    params.Path,
			"content": params.Content,
		},
	}
}
