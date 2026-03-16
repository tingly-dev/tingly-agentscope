package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// WriteTool provides file writing capabilities
type WriteTool struct {
	allowedDirs []string
	allowOverwrite bool
}

// WriteOptions configures the WriteTool
func WriteOptions(allowedDirs []string, allowOverwrite bool) func(*WriteTool) {
	return func(w *WriteTool) {
		w.allowedDirs = allowedDirs
		w.allowOverwrite = allowOverwrite
	}
}

// NewWriteTool creates a new write tool instance
func NewWriteTool(options ...func(*WriteTool)) *WriteTool {
	wt := &WriteTool{
		allowedDirs:    []string{}, // Empty means allow all
		allowOverwrite: true,
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
	// Validate path
	if err := w.validatePath(params.Path); err != nil {
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

// validatePath checks if the path is allowed
func (w *WriteTool) validatePath(path string) error {
	if len(w.allowedDirs) == 0 {
		return nil // No restrictions
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	for _, allowedDir := range w.allowedDirs {
		absAllowedDir, err := filepath.Abs(allowedDir)
		if err != nil {
			continue
		}
		if strings.HasPrefix(absPath, absAllowedDir) {
			return nil
		}
	}

	return fmt.Errorf("path not allowed: %s", path)
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

// Call implements the ToolCallable interface for programmatic use
func (w *WriteTool) Call(ctx context.Context, kwargs map[string]any) (*tool.ToolResponse, error) {
	params := WriteParams{}
	if path, ok := kwargs["path"].(string); ok {
		params.Path = path
	}
	if content, ok := kwargs["content"].(string); ok {
		params.Content = content
	}
	return w.Write(ctx, params)
}

// ToToolUseBlock converts parameters to a ToolUseBlock for agent use
func (w *WriteTool) ToToolUseBlock(params WriteParams) *message.ToolUseBlock {
	return &message.ToolUseBlock{
		Name: "write",
		Input: map[string]types.JSONSerializable{
			"path":    params.Path,
			"content": params.Content,
		},
	}
}
