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

// ReadTool provides file reading capabilities
type ReadTool struct {
	allowedDirs []string
	maxFileSize int64
}

// ReadOptions configures the ReadTool
func ReadOptions(allowedDirs []string, maxFileSize int64) func(*ReadTool) {
	return func(r *ReadTool) {
		r.allowedDirs = allowedDirs
		r.maxFileSize = maxFileSize
	}
}

// NewReadTool creates a new read tool instance
func NewReadTool(options ...func(*ReadTool)) *ReadTool {
	rt := &ReadTool{
		allowedDirs: []string{}, // Empty means allow all
		maxFileSize: 10 * 1024 * 1024, // 10MB default
	}
	for _, opt := range options {
		opt(rt)
	}
	return rt
}

// ReadParams defines the parameters for the read tool
type ReadParams struct {
	Path   string `json:"path" jsonschema:"description=Path to the file to read (relative or absolute)"`
	Offset int    `json:"offset,omitempty" jsonschema:"description=Line number to start reading from (1-indexed)"`
	Limit  int    `json:"limit,omitempty" jsonschema:"description=Maximum number of lines to read"`
}

// Read reads the contents of a file
func (r *ReadTool) Read(ctx context.Context, params ReadParams) (*tool.ToolResponse, error) {
	// Validate path
	if err := r.validatePath(params.Path); err != nil {
		return tool.TextResponse(fmt.Sprintf("Error: %v", err)), nil
	}

	// Get absolute path
	absPath, err := filepath.Abs(params.Path)
	if err != nil {
		return tool.TextResponse(fmt.Sprintf("Error: failed to resolve path: %v", err)), nil
	}

	// Check file info
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return tool.TextResponse(fmt.Sprintf("Error: file not found: %s", params.Path)), nil
		}
		return tool.TextResponse(fmt.Sprintf("Error: failed to stat file: %v", err)), nil
	}

	if info.IsDir() {
		return tool.TextResponse(fmt.Sprintf("Error: path is a directory, not a file: %s", params.Path)), nil
	}

	// Check file size
	if info.Size() > r.maxFileSize {
		return tool.TextResponse(fmt.Sprintf("Error: file too large (%d bytes, max %d bytes)", info.Size(), r.maxFileSize)), nil
	}

	// Read file content
	content, err := os.ReadFile(absPath)
	if err != nil {
		return tool.TextResponse(fmt.Sprintf("Error: failed to read file: %v", err)), nil
	}

	// Handle offset and limit
	result := string(content)
	if params.Offset > 0 || params.Limit > 0 {
		result = applyLineRange(result, params.Offset, params.Limit)
	}

	return tool.TextResponse(result), nil
}

// validatePath checks if the path is allowed
func (r *ReadTool) validatePath(path string) error {
	if len(r.allowedDirs) == 0 {
		return nil // No restrictions
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	for _, allowedDir := range r.allowedDirs {
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

// applyLineRange applies offset and limit to content
func applyLineRange(content string, offset, limit int) string {
	lines := strings.Split(content, "\n")

	// Adjust offset (1-indexed to 0-indexed)
	start := offset - 1
	if start < 0 {
		start = 0
	}
	if start >= len(lines) {
		return ""
	}

	// Calculate end
	end := len(lines)
	if limit > 0 {
		end = start + limit
		if end > len(lines) {
			end = len(lines)
		}
	}

	return strings.Join(lines[start:end], "\n")
}

// RegisterReadTool registers the read tool with the toolkit
func RegisterReadTool(tk *tool.Toolkit, options ...func(*ReadTool)) error {
	rt := NewReadTool(options...)
	return tk.Register(rt.Read, &tool.RegisterOptions{
		GroupName:       "basic",
		FuncName:        "read",
		FuncDescription: "Read the contents of a file. Supports text files. Defaults to first 2000 lines. Use offset/limit for large files.",
	})
}

// Call implements the ToolCallable interface for programmatic use
func (r *ReadTool) Call(ctx context.Context, kwargs map[string]any) (*tool.ToolResponse, error) {
	params := ReadParams{}
	if path, ok := kwargs["path"].(string); ok {
		params.Path = path
	}
	if offset, ok := kwargs["offset"].(float64); ok {
		params.Offset = int(offset)
	}
	if limit, ok := kwargs["limit"].(float64); ok {
		params.Limit = int(limit)
	}
	return r.Read(ctx, params)
}

// ToToolUseBlock converts parameters to a ToolUseBlock for agent use
func (r *ReadTool) ToToolUseBlock(params ReadParams) *message.ToolUseBlock {
	input := map[string]types.JSONSerializable{
		"path": params.Path,
	}
	if params.Offset > 0 {
		input["offset"] = params.Offset
	}
	if params.Limit > 0 {
		input["limit"] = params.Limit
	}
	return &message.ToolUseBlock{
		Name:  "read",
		Input: input,
	}
}
