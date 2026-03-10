package tools

import (
	"encoding/json"
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/model"
)

// TestJSONTagUnmarshal tests that JSON unmarshaling works correctly with snake_case tags
func TestJSONTagUnmarshal(t *testing.T) {
	type TestParams struct {
		FilePath string `json:"file_path" required:"true"`
	}

	tests := []struct {
		name    string
		input   string
		wantErr bool
		wantVal string
	}{
		{
			name:    "correct snake_case key",
			input:   `{"file_path": "/test/path"}`,
			wantErr: false,
			wantVal: "/test/path",
		},
		{
			name:    "incorrect camelCase key",
			input:   `{"FilePath": "/test/path"}`,
			wantErr: false, // JSON unmarshal doesn't error, just doesn't populate the field
			wantVal: "",    // empty because key doesn't match
		},
		{
			name:    "both keys - snake_case takes precedence",
			input:   `{"file_path": "/test/snake", "FilePath": "/test/camel"}`,
			wantErr: false,
			wantVal: "/test/snake",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var params TestParams
			err := json.Unmarshal([]byte(tt.input), &params)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if params.FilePath != tt.wantVal {
				t.Errorf("FilePath = %q, want %q", params.FilePath, tt.wantVal)
			}
		})
	}
}

// TestMapToStructWithJSONTags tests MapToStruct with different key formats
func TestMapToStructWithJSONTags(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]any
		wantVal string
	}{
		{
			name:    "correct snake_case key",
			input:   map[string]any{"file_path": "/test/path"},
			wantVal: "/test/path",
		},
		{
			name:    "incorrect camelCase key",
			input:   map[string]any{"FilePath": "/test/path"},
			wantVal: "", // empty because key doesn't match JSON tag
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := ViewFileParams{}
			err := MapToStruct(tt.input, &params)
			if err != nil {
				t.Errorf("MapToStruct() error = %v", err)
				return
			}
			if params.FilePath != tt.wantVal {
				t.Errorf("FilePath = %q, want %q", params.FilePath, tt.wantVal)
			}
		})
	}
}

// TestStructToSchema verifies that the schema generation uses JSON tags correctly
func TestStructToSchema(t *testing.T) {
	schema := StructToSchema(ViewFileParams{})

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("schema.properties is not a map")
	}

	// Check that file_path (not FilePath) is in the schema
	if _, ok := props["file_path"]; !ok {
		t.Error("schema should contain 'file_path' property")
	}

	// Check that FilePath is NOT in the schema (it should use the JSON tag)
	if _, ok := props["FilePath"]; ok {
		t.Error("schema should NOT contain 'FilePath' property (should use JSON tag 'file_path')")
	}

	// Print schema for debugging
	t.Logf("Generated schema: %+v", schema)
}

// TestViewFileParamsJSONRoundTrip tests JSON serialization round-trip
func TestViewFileParamsJSONRoundTrip(t *testing.T) {
	original := ViewFileParams{
		FilePath: "/test/django/db/models/aggregates.py",
		Limit:    100,
		Offset:   10,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	t.Logf("JSON output: %s", string(jsonData))

	// Unmarshal back
	var decoded ViewFileParams
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.FilePath != original.FilePath {
		t.Errorf("FilePath = %q, want %q", decoded.FilePath, original.FilePath)
	}
	if decoded.Limit != original.Limit {
		t.Errorf("Limit = %d, want %d", decoded.Limit, original.Limit)
	}
	if decoded.Offset != original.Offset {
		t.Errorf("Offset = %d, want %d", decoded.Offset, original.Offset)
	}

	// Also test that unmarshaling with wrong key name doesn't work
	var wrongDecoded ViewFileParams
	wrongJSON := `{"FilePath": "/test/path", "Limit": 100, "Offset": 10}`
	if err := json.Unmarshal([]byte(wrongJSON), &wrongDecoded); err != nil {
		t.Fatalf("json.Unmarshal() with wrong keys error = %v", err)
	}

	t.Logf("After unmarshal with wrong keys: FilePath=%q, Limit=%d, Offset=%d",
		wrongDecoded.FilePath, wrongDecoded.Limit, wrongDecoded.Offset)

	if wrongDecoded.FilePath != "" {
		t.Errorf("FilePath should be empty when using wrong key, got %q", wrongDecoded.FilePath)
	}
}

// TestTaskGetParamsSchema tests the schema generation for TaskGetParams
func TestTaskGetParamsSchema(t *testing.T) {
	schema := StructToSchema(TaskGetParams{})

	// Print schema for debugging
	schemaJSON, _ := json.MarshalIndent(schema, "", "  ")
	t.Logf("TaskGetParams schema: %s", string(schemaJSON))

	// Verify the schema structure
	if schema["type"] != "object" {
		t.Errorf("schema type should be 'object', got %v", schema["type"])
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("schema.properties is not a map")
	}

	// Check that taskId is in the schema
	taskIDProp, ok := props["taskId"]
	if !ok {
		t.Fatal("schema should contain 'taskId' property")
	}

	taskIDPropMap, ok := taskIDProp.(map[string]any)
	if !ok {
		t.Fatalf("taskId property should be a map, got %T", taskIDProp)
	}

	if taskIDPropMap["type"] != "string" {
		t.Errorf("taskId type should be 'string', got %v", taskIDPropMap["type"])
	}

	// Check required fields
	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("schema.required should be a string array")
	}

	if len(required) != 1 || required[0] != "taskId" {
		t.Errorf("required should be ['taskId'], got %v", required)
	}
}

// TestTaskManagementToolsRegistration tests the full registration flow for task tools
func TestTaskManagementToolsRegistration(t *testing.T) {
	// Create a task store
	taskStore := NewTaskStore("/tmp/test-tasks.json")
	defer taskStore.Clear()

	// Create task management tools
	taskTools := NewTaskManagementTools(taskStore)

	// Create a typed toolkit and register the tools
	tt := NewTypedToolkit()
	descriptions := map[string]string{
		"TaskCreate": ToolDescTaskCreate,
		"TaskGet":    ToolDescTaskGet,
		"TaskUpdate": ToolDescTaskUpdate,
		"TaskList":   ToolDescTaskList,
	}

	err := tt.RegisterAll(taskTools, descriptions)
	if err != nil {
		t.Fatalf("RegisterAll failed: %v", err)
	}

	// Get the schemas
	schemas := tt.GetSchemas()

	// Find the task_get schema
	var taskGetSchema *ToolInfo
	for i := range schemas {
		if schemas[i].Name == "task_get" {
			taskGetSchema = &schemas[i]
			break
		}
	}

	if taskGetSchema == nil {
		t.Fatal("task_get tool not found in schemas")
	}

	// Print the schema for debugging
	schemaJSON, _ := json.MarshalIndent(taskGetSchema, "", "  ")
	t.Logf("task_get tool info: %s", string(schemaJSON))

	// Verify parameters is a proper map (it's already map[string]any in ToolInfo)
	params := taskGetSchema.Parameters

	// Verify the schema structure
	if params["type"] != "object" {
		t.Errorf("parameters type should be 'object', got %v", params["type"])
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("parameters.properties should be a map")
	}

	// Check that taskId is in the schema
	if _, ok := props["taskId"]; !ok {
		t.Fatal("parameters should contain 'taskId' property")
	}

	// Now test with GetModelSchemas which is used by the agent
	modelSchemas := tt.GetModelSchemas()

	// Find the task_get model schema
	var taskGetModelSchema *model.ToolDefinition
	for i := range modelSchemas {
		if modelSchemas[i].Function.Name == "task_get" {
			taskGetModelSchema = &modelSchemas[i]
			break
		}
	}

	if taskGetModelSchema == nil {
		t.Fatal("task_get tool not found in model schemas")
	}

	// Print the model schema for debugging
	modelSchemaJSON, _ := json.MarshalIndent(taskGetModelSchema, "", "  ")
	t.Logf("task_get model schema: %s", string(modelSchemaJSON))

	// Verify the function parameters is a proper map (it's already map[string]any in FunctionDefinition)
	funcParams := taskGetModelSchema.Function.Parameters

	// Verify the schema structure
	if funcParams["type"] != "object" {
		t.Errorf("function parameters type should be 'object', got %v", funcParams["type"])
	}
}
