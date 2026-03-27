package tools

import (
	"fmt"

	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
)

// ToolkitOptions configures which tools to register and their options.
type ToolkitOptions struct {
	ReadOptions  func(*ReadTool)
	WriteOptions func(*WriteTool)
	EditOptions  func(*EditTool)
	BashOptions  func(*BashTool)
}

// NewToolkit creates a *tool.Toolkit with all extension tools registered.
// Pass nil for default options. Use Register*Tool for selective registration.
func NewToolkit(opts *ToolkitOptions) (*tool.Toolkit, error) {
	tk := tool.NewToolkit()

	if opts == nil {
		opts = &ToolkitOptions{}
	}

	var readOpts []func(*ReadTool)
	if opts.ReadOptions != nil {
		readOpts = append(readOpts, opts.ReadOptions)
	}
	if err := RegisterReadTool(tk, readOpts...); err != nil {
		return nil, fmt.Errorf("failed to register read tool: %w", err)
	}

	var writeOpts []func(*WriteTool)
	if opts.WriteOptions != nil {
		writeOpts = append(writeOpts, opts.WriteOptions)
	}
	if err := RegisterWriteTool(tk, writeOpts...); err != nil {
		return nil, fmt.Errorf("failed to register write tool: %w", err)
	}

	var editOpts []func(*EditTool)
	if opts.EditOptions != nil {
		editOpts = append(editOpts, opts.EditOptions)
	}
	if err := RegisterEditTool(tk, editOpts...); err != nil {
		return nil, fmt.Errorf("failed to register edit tool: %w", err)
	}

	var bashOpts []func(*BashTool)
	if opts.BashOptions != nil {
		bashOpts = append(bashOpts, opts.BashOptions)
	}
	if err := RegisterBashTool(tk, bashOpts...); err != nil {
		return nil, fmt.Errorf("failed to register bash tool: %w", err)
	}

	return tk, nil
}

// ToolDefinition is an alias for model.ToolDefinition
type ToolDefinition = model.ToolDefinition
