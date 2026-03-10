package tools

import (
	"github.com/tingly-dev/tingly-agentscope/pkg/toolschema"
)

// Re-export ToolDescriptor and ToolRegistry from toolschema
type ToolDescriptor = toolschema.ToolDescriptor
type ToolRegistry = toolschema.ToolRegistry

// NewToolRegistry creates a new tool registry
var NewToolRegistry = toolschema.NewToolRegistry

// Global registry functions
var (
	RegisterTool        = toolschema.RegisterTool
	ListTools           = toolschema.ListTools
	ListToolsByCategory = toolschema.ListToolsByCategory
	GetTool             = toolschema.GetTool
	GetToolCategories   = toolschema.GetToolCategories
	FormatToolStatus    = toolschema.FormatToolStatus
)
