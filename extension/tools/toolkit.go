package tools

import (
	"context"
	"fmt"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
)

// ExtensionToolkit provides a pre-configured toolkit with extension tools
type ExtensionToolkit struct {
	tk       *tool.Toolkit
	readTool *ReadTool
	writeTool *WriteTool
	editTool  *EditTool
	bashTool  *BashTool
}

// ExtensionOptions configures the ExtensionToolkit
type ExtensionOptions struct {
	ReadOptions  func(*ReadTool)
	WriteOptions func(*WriteTool)
	EditOptions  func(*EditTool)
	BashOptions  func(*BashTool)
}

// NewExtensionToolkit creates a new extension toolkit with all tools registered
func NewExtensionToolkit(opts *ExtensionOptions) (*ExtensionToolkit, error) {
	tk := tool.NewToolkit()

	et := &ExtensionToolkit{
		tk: tk,
	}

	// Configure and register read tool
	readOpts := []func(*ReadTool){}
	if opts != nil && opts.ReadOptions != nil {
		readOpts = append(readOpts, opts.ReadOptions)
	}
	et.readTool = NewReadTool(readOpts...)
	if err := tk.RegisterAll(et.readTool); err != nil {
		return nil, fmt.Errorf("failed to register read tool: %w", err)
	}

	// Configure and register write tool
	writeOpts := []func(*WriteTool){}
	if opts != nil && opts.WriteOptions != nil {
		writeOpts = append(writeOpts, opts.WriteOptions)
	}
	et.writeTool = NewWriteTool(writeOpts...)
	if err := tk.RegisterAll(et.writeTool); err != nil {
		return nil, fmt.Errorf("failed to register write tool: %w", err)
	}

	// Configure and register edit tool
	editOpts := []func(*EditTool){}
	if opts != nil && opts.EditOptions != nil {
		editOpts = append(editOpts, opts.EditOptions)
	}
	et.editTool = NewEditTool(editOpts...)
	if err := tk.RegisterAll(et.editTool); err != nil {
		return nil, fmt.Errorf("failed to register edit tool: %w", err)
	}

	// Configure and register bash tool
	bashOpts := []func(*BashTool){}
	if opts != nil && opts.BashOptions != nil {
		bashOpts = append(bashOpts, opts.BashOptions)
	}
	et.bashTool = NewBashTool(bashOpts...)
	if err := tk.RegisterAll(et.bashTool); err != nil {
		return nil, fmt.Errorf("failed to register bash tool: %w", err)
	}

	return et, nil
}

// GetToolkit returns the underlying toolkit for use with agents
func (et *ExtensionToolkit) GetToolkit() *tool.Toolkit {
	return et.tk
}

// GetSchemas returns tool definitions for the model
func (et *ExtensionToolkit) GetSchemas() []model.ToolDefinition {
	return et.tk.GetSchemas()
}

// Call executes a tool by name with the given parameters
func (et *ExtensionToolkit) Call(ctx context.Context, toolBlock *message.ToolUseBlock) (*tool.ToolResponse, error) {
	return et.tk.Call(ctx, toolBlock)
}

// Read provides direct access to the read tool
func (et *ExtensionToolkit) Read(ctx context.Context, path string, offset, limit int) (*tool.ToolResponse, error) {
	return et.readTool.Read(ctx, ReadParams{
		Path:   path,
		Offset: offset,
		Limit:  limit,
	})
}

// Write provides direct access to the write tool
func (et *ExtensionToolkit) Write(ctx context.Context, path, content string) (*tool.ToolResponse, error) {
	return et.writeTool.Write(ctx, WriteParams{
		Path:    path,
		Content: content,
	})
}

// Edit provides direct access to the edit tool
func (et *ExtensionToolkit) Edit(ctx context.Context, path, oldText, newText string) (*tool.ToolResponse, error) {
	return et.editTool.Edit(ctx, EditParams{
		Path:    path,
		OldText: oldText,
		NewText: newText,
	})
}

// Bash provides direct access to the bash tool
func (et *ExtensionToolkit) Bash(ctx context.Context, command string, timeout int) (*tool.ToolResponse, error) {
	return et.bashTool.Bash(ctx, BashParams{
		Command: command,
		Timeout: timeout,
	})
}

// ToolDefinition is an alias for model.ToolDefinition
type ToolDefinition = model.ToolDefinition
