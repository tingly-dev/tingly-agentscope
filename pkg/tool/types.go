package tool

import (
	"reflect"

	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// DescriptiveTool is an interface for tools that can provide their own metadata.
// This allows tools to be self-documenting and more easily integrated with LLM systems.
//
// Example implementation:
//
//	type MyTool struct{}
//
//	func (t *MyTool) Name() string {
//	    return "my_tool"
//	}
//
//	func (t *MyTool) Description() string {
//	    return "Does something useful"
//	}
//
//	func (t *MyTool) Call(ctx context.Context, args *MyArgs) (*ToolResponse, error) {
//	    // implementation
//	}
type DescriptiveTool interface {
	// Name returns the tool's identifier. Should be snake_case for compatibility.
	// If not implemented, the method name (e.g., "MyMethod" -> "my_method") will be used.
	Name() string

	// Description returns a human-readable description of what the tool does.
	// This is used in the system prompt to help the LLM understand when to use the tool.
	Description() string
}

// TypedHandle wraps a typed tool for execution
type TypedHandle struct {
	Tool    any          // The tool struct
	ArgType reflect.Type // Expected argument type (e.g., SearchArgs)
}

// FunctionHandle wraps a simple function for execution
type FunctionHandle struct {
	Func reflect.Value // The function
}

// ToolDescriptor completely describes a registered tool
type ToolDescriptor struct {
	// Identification
	Name  string
	Group string

	// Schema
	Schema model.ToolDefinition

	// Execution (exactly one will be non-nil)
	Typed    *TypedHandle    // For RegisterTool - type-safe with struct args
	Function *FunctionHandle // For RegisterFunction - simple function with map args

	// Configuration
	PresetKwargs map[string]types.JSONSerializable
}
