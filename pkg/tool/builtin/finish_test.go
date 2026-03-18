package builtin

import (
	"context"
	"testing"
)

func TestFinishTool(t *testing.T) {
	tool := NewFinishTool()

	// Test that the tool is registered with correct name
	if tool.Name != "finish" {
		t.Errorf("Expected name 'finish', got %s", tool.Name)
	}

	// Test successful finish
	result, err := tool.Execute(context.Background(), FinishInput{
		Summary: "Task completed successfully",
	})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result.Status != "finished" {
		t.Errorf("Expected status 'finished', got %s", result.Status)
	}

	if result.Summary != "Task completed successfully" {
		t.Errorf("Expected summary 'Task completed successfully', got %s", result.Summary)
	}
}

func TestFinishToolSchema(t *testing.T) {
	tool := NewFinishTool()

	schema := tool.GetSchema()
	if schema.Function.Name != "finish" {
		t.Errorf("Expected schema name 'finish', got %s", schema.Function.Name)
	}

	// Check that summary parameter exists
	if schema.Function.Parameters == nil {
		t.Error("Expected parameters in schema")
	}
}
