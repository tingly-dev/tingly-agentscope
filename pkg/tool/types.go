package tool

import (
	"reflect"

	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

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
