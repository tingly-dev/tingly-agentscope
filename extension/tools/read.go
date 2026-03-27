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
		allowedDirs: []string{},       // Empty means allow all
		maxFileSize: 10 * 1024 * 1024, // 10MB default
	}
	for _, opt := range options {
		opt(rt)
	}
	return rt
}

// ReadParams defines the parameters for the read tool
type ReadParams struct {
	Path   string `json:"path" required:"true" description:"Path to the file to read (relative or absolute)"`
	Offset int    `json:"offset,omitempty" description:"Line number to start reading from (1-indexed)"`
	Limit  int    `json:"limit,omitempty" description:"Maximum number of lines to read"`
}

// Name implements DescriptiveTool interface
func (r *ReadTool) Name() string {
	return "read"
}

// Description implements DescriptiveTool interface
func (r *ReadTool) Description() string {
	return "Read the contents of a file. Supports text files. Defaults to first 2000 lines. Use offset/limit for large files."
}

// Call reads the contents of a file
func (r *ReadTool) Call(ctx context.Context, params ReadParams) (*tool.ToolResponse, error) {
	// Validate offset and limit
	if params.Offset < 0 {
		params.Offset = 0
	}
	if params.Limit < 0 {
		return tool.TextResponse("Error: limit must be non-negative"), nil
	}

	// Validate path
	if err := validatePath(params.Path, r.allowedDirs); err != nil {
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

// RegisterReadTool registers the read tool with the toolkit
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

// RegisterReadTool registers the read tool with the given toolkit.
func RegisterReadTool(tk *tool.Toolkit, options ...func(*ReadTool)) error {
	rt := NewReadTool(options...)
	// RegisterAll uses the DescriptiveTool interface for name/description
	return tk.RegisterAll(rt)
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
