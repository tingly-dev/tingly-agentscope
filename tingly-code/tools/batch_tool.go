package tools

import (
	"github.com/tingly-dev/tingly-agentscope/pkg/toolschema"
)

// Re-export BatchTool and related types from toolschema
type BatchTool = toolschema.BatchTool
type Invocation = toolschema.Invocation
type InvocationResult = toolschema.InvocationResult

// NewBatchTool creates a new BatchTool instance
var NewBatchTool = toolschema.NewBatchTool

// GetGlobalBatchTool returns the global batch tool (singleton)
var GetGlobalBatchTool = toolschema.GetGlobalBatchTool
