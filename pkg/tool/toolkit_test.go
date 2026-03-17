package tool

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// TestStructParams tests tool calling with struct parameters
type TestParams struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// testStructTool implements ToolCallable for testing
type testStructTool struct{}

func (t *testStructTool) Call(ctx context.Context, args any) (*ToolResponse, error) {
	params, ok := args.(*TestParams)
	if !ok {
		// If args is a map, convert it
		if m, ok := args.(map[string]any); ok {
			params = &TestParams{}
			if name, ok := m["name"].(string); ok {
				params.Name = name
			}
			if count, ok := m["count"].(int); ok {
				params.Count = count
			}
		} else {
			return TextResponse("Error: invalid arguments"), nil
		}
	}
	return TextResponse("name: " + params.Name + ", count: " + string(rune(params.Count))), nil
}

func TestCallWithStructParams(t *testing.T) {
	tk := NewToolkit()

	// Register a tool with struct parameters
	err := tk.Register(&testStructTool{}, &RegisterOptions{
		GroupName:       "basic",
		FuncName:        "test_struct_tool",
		FuncDescription: "A test tool with struct parameters",
		JSONSchema: &model.ToolDefinition{
			Type: "function",
			Function: model.FunctionDefinition{
				Name:        "test_struct_tool",
				Description: "A test tool with struct parameters",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{
							"type":        "string",
							"description": "name",
						},
						"count": map[string]any{
							"type":        "integer",
							"description": "count",
						},
					},
					"required": []string{"name", "count"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to register tool: %v", err)
	}

	// Call the tool
	ctx := context.Background()
	toolBlock := &message.ToolUseBlock{
		Name: "test_struct_tool",
		Input: map[string]types.JSONSerializable{
			"name":  "test",
			"count": 42,
		},
	}

	result, err := tk.Call(ctx, toolBlock)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Check result
	if len(result.Content) == 0 {
		t.Error("expected non-empty content")
	}
}

// testPointerStructTool implements ToolCallable for testing
type testPointerStructTool struct{}

func (t *testPointerStructTool) Call(ctx context.Context, args any) (*ToolResponse, error) {
	params, ok := args.(*TestParams)
	if !ok {
		// If args is a map, convert it
		if m, ok := args.(map[string]any); ok {
			params = &TestParams{}
			if name, ok := m["name"].(string); ok {
				params.Name = name
			}
			if count, ok := m["count"].(int); ok {
				params.Count = count
			}
		} else {
			return TextResponse("Error: invalid arguments"), nil
		}
	}
	if params == nil {
		return TextResponse("nil params"), nil
	}
	return TextResponse("name: " + params.Name + ", count: " + string(rune(params.Count))), nil
}

func TestCallWithPointerStructParams(t *testing.T) {
	tk := NewToolkit()

	// Register a tool with pointer struct parameters
	err := tk.Register(&testPointerStructTool{}, &RegisterOptions{
		GroupName:       "basic",
		FuncName:        "test_pointer_struct_tool",
		FuncDescription: "A test tool with pointer struct parameters",
		JSONSchema: &model.ToolDefinition{
			Type: "function",
			Function: model.FunctionDefinition{
				Name:        "test_pointer_struct_tool",
				Description: "A test tool with pointer struct parameters",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{
							"type":        "string",
							"description": "name",
						},
						"count": map[string]any{
							"type":        "integer",
							"description": "count",
						},
					},
					"required": []string{"name", "count"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to register tool: %v", err)
	}

	// Call the tool
	ctx := context.Background()
	toolBlock := &message.ToolUseBlock{
		Name: "test_pointer_struct_tool",
		Input: map[string]types.JSONSerializable{
			"name":  "test",
			"count": 42,
		},
	}

	result, err := tk.Call(ctx, toolBlock)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Check result
	if len(result.Content) == 0 {
		t.Error("expected non-empty content")
	}
}

// testToolResponseTool implements ToolCallable for testing
type testToolResponseTool struct{}

func (t *testToolResponseTool) Call(ctx context.Context, args any) (*ToolResponse, error) {
	params, ok := args.(*TestParams)
	if !ok {
		// If args is a map, convert it
		if m, ok := args.(map[string]any); ok {
			params = &TestParams{}
			if name, ok := m["name"].(string); ok {
				params.Name = name
			}
		}
	}
	return TextResponse("result: " + params.Name), nil
}

func TestCallWithToolResponseReturn(t *testing.T) {
	tk := NewToolkit()

	// Register a tool that returns ToolResponse
	err := tk.Register(&testToolResponseTool{}, &RegisterOptions{
		GroupName:       "basic",
		FuncName:        "test_response_tool",
		FuncDescription: "A test tool that returns ToolResponse",
		JSONSchema: &model.ToolDefinition{
			Type: "function",
			Function: model.FunctionDefinition{
				Name:        "test_response_tool",
				Description: "A test tool that returns ToolResponse",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{
							"type":        "string",
							"description": "name",
						},
					},
					"required": []string{"name"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to register tool: %v", err)
	}

	// Call the tool
	ctx := context.Background()
	toolBlock := &message.ToolUseBlock{
		Name: "test_response_tool",
		Input: map[string]types.JSONSerializable{
			"name": "test",
		},
	}

	result, err := tk.Call(ctx, toolBlock)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Check result has content
	if len(result.Content) == 0 {
		t.Error("expected non-empty content")
	}
}

// mockToolFunction is a simple mock tool for testing
type mockToolFunction struct{}

func (m *mockToolFunction) Call(ctx context.Context, args any) (*ToolResponse, error) {
	return TextResponse("mock result"), nil
}

func TestGetToolList_InternalStyle(t *testing.T) {
	tk := NewToolkit()

	// Register a test tool
	err := tk.Register(&mockToolFunction{}, &RegisterOptions{
		GroupName:       "basic",
		FuncName:        "test_tool",
		FuncDescription: "A test tool",
		JSONSchema: &model.ToolDefinition{
			Type: "function",
			Function: model.FunctionDefinition{
				Name:        "test_tool",
				Description: "A test tool",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"input": map[string]any{
							"type":        "string",
							"description": "test input",
						},
					},
					"required": []string{"input"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to register tool: %v", err)
	}

	// Test internal style
	result, err := tk.GetToolList(APIStyleInternal)
	if err != nil {
		t.Fatalf("GetToolList failed: %v", err)
	}

	schemas, ok := result.([]model.ToolDefinition)
	if !ok {
		t.Fatalf("expected []model.ToolDefinition, got %T", result)
	}

	if len(schemas) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(schemas))
	}

	if schemas[0].Function.Name != "test_tool" {
		t.Errorf("expected tool name 'test_tool', got '%s'", schemas[0].Function.Name)
	}
}

func TestGetToolList_AnthropicStyle(t *testing.T) {
	tk := NewToolkit()

	// Register a test tool
	err := tk.Register(&mockToolFunction{}, &RegisterOptions{
		GroupName:       "basic",
		FuncName:        "test_tool",
		FuncDescription: "A test tool",
		JSONSchema: &model.ToolDefinition{
			Type: "function",
			Function: model.FunctionDefinition{
				Name:        "test_tool",
				Description: "A test tool",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"input": map[string]any{
							"type":        "string",
							"description": "test input",
						},
					},
					"required": []string{"input"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to register tool: %v", err)
	}

	// Test Anthropic style
	result, err := tk.GetToolList(APIStyleAnthropic)
	if err != nil {
		t.Fatalf("GetToolList failed: %v", err)
	}

	// Check it's a valid JSON-serializable structure
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal result: %v", err)
	}

	var parsed []map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if len(parsed) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(parsed))
	}

	if parsed[0]["name"] != "test_tool" {
		t.Errorf("expected tool name 'test_tool', got '%v'", parsed[0]["name"])
	}

	if parsed[0]["description"] != "A test tool" {
		t.Errorf("expected description 'A test tool', got '%v'", parsed[0]["description"])
	}

	// Check input_schema exists
	if _, ok := parsed[0]["input_schema"]; !ok {
		t.Error("expected input_schema in result")
	}
}

func TestGetToolList_OpenAIStyle(t *testing.T) {
	tk := NewToolkit()

	// Register a test tool
	err := tk.Register(&mockToolFunction{}, &RegisterOptions{
		GroupName:       "basic",
		FuncName:        "test_tool",
		FuncDescription: "A test tool",
		JSONSchema: &model.ToolDefinition{
			Type: "function",
			Function: model.FunctionDefinition{
				Name:        "test_tool",
				Description: "A test tool",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"input": map[string]any{
							"type":        "string",
							"description": "test input",
						},
					},
					"required": []string{"input"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to register tool: %v", err)
	}

	// Test OpenAI style
	result, err := tk.GetToolList(APIStyleOpenAI)
	if err != nil {
		t.Fatalf("GetToolList failed: %v", err)
	}

	// Check it's a valid JSON-serializable structure
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal result: %v", err)
	}

	var parsed []map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if len(parsed) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(parsed))
	}

	if parsed[0]["type"] != "function" {
		t.Errorf("expected type 'function', got '%v'", parsed[0]["type"])
	}

	// Check function exists
	fn, ok := parsed[0]["function"].(map[string]any)
	if !ok {
		t.Fatal("expected function in result")
	}

	if fn["name"] != "test_tool" {
		t.Errorf("expected function name 'test_tool', got '%v'", fn["name"])
	}

	if fn["description"] != "A test tool" {
		t.Errorf("expected description 'A test tool', got '%v'", fn["description"])
	}

	// Check parameters exists
	if _, ok := fn["parameters"]; !ok {
		t.Error("expected parameters in function")
	}
}

func TestGetToolList_InvalidStyle(t *testing.T) {
	tk := NewToolkit()

	// Test invalid style
	_, err := tk.GetToolList(APIStyle("invalid"))
	if err == nil {
		t.Error("expected error for invalid API style")
	}

	if err.Error() != "unsupported API style: invalid" {
		t.Errorf("expected 'unsupported API style: invalid', got '%v'", err)
	}
}

func TestGetToolList_WithInactiveGroup(t *testing.T) {
	tk := NewToolkit()

	// Create a group
	err := tk.CreateToolGroup("test_group", "Test group", false, "")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	// Register a tool in the inactive group
	err = tk.Register(&mockToolFunction{}, &RegisterOptions{
		GroupName:       "test_group",
		FuncName:        "inactive_tool",
		FuncDescription: "An inactive tool",
		JSONSchema: &model.ToolDefinition{
			Type: "function",
			Function: model.FunctionDefinition{
				Name:        "inactive_tool",
				Description: "An inactive tool",
				Parameters:  map[string]any{"type": "object"},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to register tool: %v", err)
	}

	// Register a tool in basic group
	err = tk.Register(&mockToolFunction{}, &RegisterOptions{
		GroupName:       "basic",
		FuncName:        "active_tool",
		FuncDescription: "An active tool",
		JSONSchema: &model.ToolDefinition{
			Type: "function",
			Function: model.FunctionDefinition{
				Name:        "active_tool",
				Description: "An active tool",
				Parameters:  map[string]any{"type": "object"},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to register tool: %v", err)
	}

	// Test that only active tools are returned
	result, err := tk.GetToolList(APIStyleInternal)
	if err != nil {
		t.Fatalf("GetToolList failed: %v", err)
	}

	schemas, ok := result.([]model.ToolDefinition)
	if !ok {
		t.Fatalf("expected []model.ToolDefinition, got %T", result)
	}

	if len(schemas) != 1 {
		t.Fatalf("expected 1 active tool, got %d", len(schemas))
	}

	if schemas[0].Function.Name != "active_tool" {
		t.Errorf("expected 'active_tool', got '%s'", schemas[0].Function.Name)
	}
}

func TestGetToolInfo(t *testing.T) {
	tk := NewToolkit()

	// Create a group
	err := tk.CreateToolGroup("test_group", "Test group", true, "")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	// Register tools
	err = tk.Register(&mockToolFunction{}, &RegisterOptions{
		GroupName:       "test_group",
		FuncName:        "tool1",
		FuncDescription: "Tool 1",
		JSONSchema: &model.ToolDefinition{
			Type: "function",
			Function: model.FunctionDefinition{
				Name:        "tool1",
				Description: "Tool 1",
				Parameters:  map[string]any{"type": "object"},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to register tool: %v", err)
	}

	err = tk.Register(&mockToolFunction{}, &RegisterOptions{
		GroupName: "basic",
		FuncName:  "tool2",
		JSONSchema: &model.ToolDefinition{
			Type: "function",
			Function: model.FunctionDefinition{
				Name:        "tool2",
				Description: "Tool 2",
				Parameters:  map[string]any{"type": "object"},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to register tool: %v", err)
	}

	// Get tool info
	info := tk.GetToolInfo()

	// The info should be a map with "tool_list" key
	if _, ok := info["tool_list"]; !ok {
		t.Fatalf("expected 'tool_list' key in info, got keys: %v", info)
	}

	// Convert to JSON to check structure (this simulates API usage)
	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("failed to marshal info: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal info: %v", err)
	}

	toolList := parsed["tool_list"].(map[string]any)
	totalTools := int(toolList["total_tools"].(float64))
	if totalTools != 2 {
		t.Errorf("expected 2 total tools, got %d", totalTools)
	}

	tools := toolList["tools"].([]any)
	if len(tools) != 2 {
		t.Errorf("expected 2 tools in info, got %d", len(tools))
	}

	// Check active groups
	activeGroups := toolList["active_groups"].([]any)
	if len(activeGroups) != 1 || activeGroups[0].(string) != "test_group" {
		t.Errorf("expected active_groups to be ['test_group'], got %v", activeGroups)
	}
}
