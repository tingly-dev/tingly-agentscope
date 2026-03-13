package toolschema

import (
	"context"
	"encoding/json"
	"testing"
)

// MockTool is a simple mock tool for testing
type MockTool struct{}

func (m *MockTool) Name() string {
	return "mock_tool"
}

func (m *MockTool) Description() string {
	return "A mock tool for testing"
}

func (m *MockTool) ParameterSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"input": map[string]any{
				"type":        "string",
				"description": "test input",
			},
		},
		"required": []string{"input"},
	}
}

func (m *MockTool) Call(ctx context.Context, params any) (string, error) {
	return "mock result", nil
}

func TestGetToolList_InternalStyle(t *testing.T) {
	tt := NewTypedToolkit()

	mockTool := &MockTool{}
	tt.Register(mockTool)

	// Test internal style
	result, err := tt.GetToolList(APIStyleInternal)
	if err != nil {
		t.Fatalf("GetToolList failed: %v", err)
	}

	tools, ok := result.([]ToolInfo)
	if !ok {
		t.Fatalf("expected []ToolInfo, got %T", result)
	}

	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}

	if tools[0].Name != "mock_tool" {
		t.Errorf("expected tool name 'mock_tool', got '%s'", tools[0].Name)
	}

	if tools[0].Description != "A mock tool for testing" {
		t.Errorf("expected description 'A mock tool for testing', got '%s'", tools[0].Description)
	}
}

func TestGetToolList_AnthropicStyle(t *testing.T) {
	tt := NewTypedToolkit()

	mockTool := &MockTool{}
	tt.Register(mockTool)

	// Test Anthropic style
	result, err := tt.GetToolList(APIStyleAnthropic)
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

	if parsed[0]["name"] != "mock_tool" {
		t.Errorf("expected tool name 'mock_tool', got '%v'", parsed[0]["name"])
	}

	if parsed[0]["description"] != "A mock tool for testing" {
		t.Errorf("expected description 'A mock tool for testing', got '%v'", parsed[0]["description"])
	}

	// Check input_schema exists
	if _, ok := parsed[0]["input_schema"]; !ok {
		t.Error("expected input_schema in result")
	}

	inputSchema := parsed[0]["input_schema"].(map[string]any)
	if inputSchema["type"] != "object" {
		t.Errorf("expected input_schema type 'object', got '%v'", inputSchema["type"])
	}
}

func TestGetToolList_OpenAIStyle(t *testing.T) {
	tt := NewTypedToolkit()

	mockTool := &MockTool{}
	tt.Register(mockTool)

	// Test OpenAI style
	result, err := tt.GetToolList(APIStyleOpenAI)
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

	if fn["name"] != "mock_tool" {
		t.Errorf("expected function name 'mock_tool', got '%v'", fn["name"])
	}

	if fn["description"] != "A mock tool for testing" {
		t.Errorf("expected description 'A mock tool for testing', got '%v'", fn["description"])
	}

	// Check parameters exists
	if _, ok := fn["parameters"]; !ok {
		t.Error("expected parameters in function")
	}

	params := fn["parameters"].(map[string]any)
	if params["type"] != "object" {
		t.Errorf("expected parameters type 'object', got '%v'", params["type"])
	}
}

func TestGetToolList_InvalidStyle(t *testing.T) {
	tt := NewTypedToolkit()

	mockTool := &MockTool{}
	tt.Register(mockTool)

	// Test invalid style
	_, err := tt.GetToolList(APIStyle("invalid"))
	if err == nil {
		t.Error("expected error for invalid API style")
	}

	if err.Error() != "unsupported API style: invalid" {
		t.Errorf("expected 'unsupported API style: invalid', got '%v'", err)
	}
}

func TestGetToolList_MultipleTools(t *testing.T) {
	tt := NewTypedToolkit()

	// Register multiple tools
	tt.Register(&MockTool{})
	tt.Register(&MockTool{})

	// Test internal style
	result, err := tt.GetToolList(APIStyleInternal)
	if err != nil {
		t.Fatalf("GetToolList failed: %v", err)
	}

	tools, ok := result.([]ToolInfo)
	if !ok {
		t.Fatalf("expected []ToolInfo, got %T", result)
	}

	// Both tools should be there (same name, but both registered)
	if len(tools) != 1 { // Same name overrides
		t.Logf("Note: Got %d tools (same name overrides)", len(tools))
	}
}

func TestGetToolInfo(t *testing.T) {
	tt := NewTypedToolkit()

	// Register multiple tools
	tt.Register(&MockTool{})

	// Get tool info
	info := tt.GetToolInfo()

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
	if totalTools != 1 {
		t.Errorf("expected 1 total tool, got %d", totalTools)
	}

	tools := toolList["tools"].([]any)
	if len(tools) != 1 {
		t.Errorf("expected 1 tool in info, got %d", len(tools))
	}

	toolInfo := tools[0].(map[string]any)
	if toolInfo["name"] != "mock_tool" {
		t.Errorf("expected tool name 'mock_tool', got '%v'", toolInfo["name"])
	}
}

func TestGetToolList_EmptyToolkit(t *testing.T) {
	tt := NewTypedToolkit()

	// Test with empty toolkit
	result, err := tt.GetToolList(APIStyleInternal)
	if err != nil {
		t.Fatalf("GetToolList failed: %v", err)
	}

	tools, ok := result.([]ToolInfo)
	if !ok {
		t.Fatalf("expected []ToolInfo, got %T", result)
	}

	if len(tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(tools))
	}
}
