package builtin

import (
	"context"
	"encoding/json"

	"github.com/tingly-dev/tingly-agentscope/pkg/model"
)

// FinishInput is the input for the finish tool
type FinishInput struct {
	Summary string `json:"summary" jsonschema:"description=A summary of what was accomplished and the final answer to the user"`
}

// FinishResult is the result of the finish tool
type FinishResult struct {
	Status  string `json:"status"`
	Summary string `json:"summary"`
}

// FinishTool allows the agent to signal completion
type FinishTool struct {
	Name        string
	Description string
}

// NewFinishTool creates a new finish tool
func NewFinishTool() *FinishTool {
	return &FinishTool{
		Name:        "finish",
		Description: "Signal that the task is complete and provide a final summary. Use this when you have finished all necessary work and have a complete answer for the user.",
	}
}

// Execute runs the finish tool
func (f *FinishTool) Execute(ctx context.Context, input FinishInput) (*FinishResult, error) {
	return &FinishResult{
		Status:  "finished",
		Summary: input.Summary,
	}, nil
}

// GetSchema returns the tool schema for registration
func (f *FinishTool) GetSchema() model.ToolDefinition {
	return model.ToolDefinition{
		Type: "function",
		Function: model.FunctionDefinition{
			Name:        f.Name,
			Description: f.Description,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"summary": map[string]any{
						"type":        "string",
						"description": "A summary of what was accomplished and the final answer to the user",
					},
				},
				"required": []string{"summary"},
			},
		},
	}
}

// ToDescriptor converts the tool to a tool descriptor
func (f *FinishTool) ToDescriptor() *DescriptiveToolImpl {
	return &DescriptiveToolImpl{
		ToolName:        f.Name,
		ToolDescription: f.Description,
		ToolParameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"summary": map[string]any{
					"type":        "string",
					"description": "A summary of what was accomplished and the final answer to the user",
				},
			},
			"required": []string{"summary"},
		},
		ExecuteFunc: func(ctx context.Context, input json.RawMessage) (any, error) {
			var finishInput FinishInput
			if err := json.Unmarshal(input, &finishInput); err != nil {
				return nil, err
			}
			return f.Execute(ctx, finishInput)
		},
	}
}

// DescriptiveToolImpl is a helper type for tool registration
type DescriptiveToolImpl struct {
	ToolName        string
	ToolDescription string
	ToolParameters  map[string]any
	ExecuteFunc     func(context.Context, json.RawMessage) (any, error)
}

func (d *DescriptiveToolImpl) Name() string        { return d.ToolName }
func (d *DescriptiveToolImpl) Description() string { return d.ToolDescription }
func (d *DescriptiveToolImpl) Parameters() map[string]any {
	params := d.ToolParameters
	if params == nil {
		params = map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}
	return params
}
