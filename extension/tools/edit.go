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

// EditTool provides file editing capabilities
type EditTool struct {
	allowedDirs []string
}

// EditOptions configures the EditTool
func EditOptions(allowedDirs []string) func(*EditTool) {
	return func(e *EditTool) {
		e.allowedDirs = allowedDirs
	}
}

// NewEditTool creates a new edit tool instance
func NewEditTool(options ...func(*EditTool)) *EditTool {
	et := &EditTool{
		allowedDirs: []string{}, // Empty means allow all
	}
	for _, opt := range options {
		opt(et)
	}
	return et
}

// EditParams defines the parameters for the edit tool
type EditParams struct {
	Path    string `json:"path" jsonschema:"description=Path to the file to edit (relative or absolute)"`
	OldText string `json:"oldText" jsonschema:"description=Exact text to find and replace (must match exactly including whitespace)"`
	NewText string `json:"newText" jsonschema:"description=New text to replace the old text with"`
}

// Edit edits a file by replacing exact text
func (e *EditTool) Edit(ctx context.Context, params EditParams) (*tool.ToolResponse, error) {
	// Validate path
	if err := e.validatePath(params.Path); err != nil {
		return tool.TextResponse(fmt.Sprintf("Error: %v", err)), nil
	}

	// Get absolute path
	absPath, err := filepath.Abs(params.Path)
	if err != nil {
		return tool.TextResponse(fmt.Sprintf("Error: failed to resolve path: %v", err)), nil
	}

	// Check if file exists
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

	// Read file content
	content, err := os.ReadFile(absPath)
	if err != nil {
		return tool.TextResponse(fmt.Sprintf("Error: failed to read file: %v", err)), nil
	}

	contentStr := string(content)

	// Check if oldText exists
	if !strings.Contains(contentStr, params.OldText) {
		return tool.TextResponse(fmt.Sprintf("Error: oldText not found in file. The text must match exactly (including whitespace).")), nil
	}

	// Count occurrences
	occurrences := strings.Count(contentStr, params.OldText)
	if occurrences > 1 {
		return tool.TextResponse(fmt.Sprintf("Error: oldText appears %d times in the file. Please provide more context to make it unique.", occurrences)), nil
	}

	// Replace text
	newContent := strings.Replace(contentStr, params.OldText, params.NewText, 1)

	// Write back
	if err := os.WriteFile(absPath, []byte(newContent), info.Mode()); err != nil {
		return tool.TextResponse(fmt.Sprintf("Error: failed to write file: %v", err)), nil
	}

	return tool.TextResponse(fmt.Sprintf("Successfully edited file: %s", params.Path)), nil
}

// validatePath checks if the path is allowed
func (e *EditTool) validatePath(path string) error {
	if len(e.allowedDirs) == 0 {
		return nil // No restrictions
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	for _, allowedDir := range e.allowedDirs {
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

// RegisterEditTool registers the edit tool with the toolkit
func RegisterEditTool(tk *tool.Toolkit, options ...func(*EditTool)) error {
	et := NewEditTool(options...)
	return tk.Register(et.Edit, &tool.RegisterOptions{
		GroupName:       "basic",
		FuncName:        "edit",
		FuncDescription: "Edit a file by replacing exact text. The oldText must match exactly (including whitespace). Use this for precise, surgical edits.",
	})
}

// Call implements the ToolCallable interface for programmatic use
func (e *EditTool) Call(ctx context.Context, kwargs map[string]any) (*tool.ToolResponse, error) {
	params := EditParams{}
	if path, ok := kwargs["path"].(string); ok {
		params.Path = path
	}
	if oldText, ok := kwargs["oldText"].(string); ok {
		params.OldText = oldText
	}
	if newText, ok := kwargs["newText"].(string); ok {
		params.NewText = newText
	}
	return e.Edit(ctx, params)
}

// ToToolUseBlock converts parameters to a ToolUseBlock for agent use
func (e *EditTool) ToToolUseBlock(params EditParams) *message.ToolUseBlock {
	return &message.ToolUseBlock{
		Name: "edit",
		Input: map[string]types.JSONSerializable{
			"path":    params.Path,
			"oldText": params.OldText,
			"newText": params.NewText,
		},
	}
}
