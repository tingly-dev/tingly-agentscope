package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
)

// TodoTools provides TODO.md file management
type TodoTools struct {
	workDir string
}

// NewTodoTools creates a new TodoTools instance
func NewTodoTools(workDir string) *TodoTools {
	return &TodoTools{workDir: workDir}
}

// getDefaultPath returns the default TODO.md path
func (tt *TodoTools) getDefaultPath() string {
	return filepath.Join(tt.workDir, "TODO.md")
}

// TodoReadParams holds parameters for todo_read tool
type TodoReadParams struct {
	Path string `json:"path,omitempty" description:"Path to TODO file (default: ./TODO.md)"`
}

// TodoRead reads a TODO.md file
func (tt *TodoTools) TodoRead(ctx context.Context, params TodoReadParams) (*tool.ToolResponse, error) {
	path := params.Path
	if path == "" {
		path = tt.getDefaultPath()
	}

	if !filepath.IsAbs(path) {
		path = filepath.Join(tt.workDir, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return tool.TextResponse("No TODO.md file found."), nil
		}
		return tool.TextResponse(fmt.Sprintf("Error: failed to read file: %v", err)), nil
	}

	return tool.TextResponse(string(data)), nil
}

// TodoWriteParams holds parameters for todo_write tool
type TodoWriteParams struct {
	Path    string `json:"path,omitempty" description:"Path to TODO file (default: ./TODO.md)"`
	Content string `json:"content" description:"The content to write to the TODO file"`
	Append  bool   `json:"append,omitempty" description:"Append to existing content instead of overwriting"`
}

// TodoWrite writes to a TODO.md file
func (tt *TodoTools) TodoWrite(ctx context.Context, params TodoWriteParams) (*tool.ToolResponse, error) {
	path := params.Path
	if path == "" {
		path = tt.getDefaultPath()
	}

	if !filepath.IsAbs(path) {
		path = filepath.Join(tt.workDir, path)
	}

	var content string
	if params.Append {
		// Read existing content
		existing, err := os.ReadFile(path)
		if err == nil {
			content = string(existing) + "\n" + params.Content
		} else {
			content = params.Content
		}
	} else {
		content = params.Content
	}

	// Ensure proper formatting
	content = strings.TrimSpace(content)
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return tool.TextResponse(fmt.Sprintf("Error: failed to write file: %v", err)), nil
	}

	return tool.TextResponse(fmt.Sprintf("TODO file '%s' has been updated.", path)), nil
}
