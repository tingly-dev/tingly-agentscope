package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBashSession_ExecuteBash(t *testing.T) {
	session := NewBashSession()
	ctx := context.Background()

	// Test simple command - use internal method
	result, err := session.executeBashInternal(ctx, "echo 'hello world'", 0)

	if err != nil {
		t.Fatalf("executeBashInternal failed: %v", err)
	}

	if !strings.Contains(result, "hello world") {
		t.Errorf("Expected 'hello world' in result, got: %s", result)
	}
}

func TestBashSession_Timeout(t *testing.T) {
	session := NewBashSession()
	ctx := context.Background()

	// Test with timeout (sleep 3, timeout 1s)
	start := time.Now()
	result, err := session.executeBashInternal(ctx, "sleep 3", 1)

	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("executeBashInternal with timeout failed: %v", err)
	}

	if !strings.Contains(result, "timed out") {
		t.Errorf("Expected timeout message, got: %s", result)
	}

	if elapsed > 2*time.Second {
		t.Errorf("Timeout took too long: %v", elapsed)
	}
}

func TestBashSession_EnvVars(t *testing.T) {
	session := NewBashSession()
	ctx := context.Background()

	// Set environment variable
	session.SetEnv("TEST_VAR", "test_value")

	// Use environment variable in command
	result, err := session.executeBashInternal(ctx, "echo $TEST_VAR", 0)

	if err != nil {
		t.Fatalf("executeBashInternal with env var failed: %v", err)
	}

	if !strings.Contains(result, "test_value") {
		t.Errorf("Expected 'test_value' in result, got: %s", result)
	}

	// Verify GetEnv
	val, ok := session.GetEnv("TEST_VAR")
	if !ok || val != "test_value" {
		t.Errorf("GetEnv failed, expected 'test_value', got '%s'", val)
	}
}

func TestBashSession_InitCommands(t *testing.T) {
	// Configure with init commands
	ConfigureBash([]string{"export INIT_TEST=1"}, false)

	// Use global session since ConfigureBash affects the global session
	session := GetGlobalBashSession()

	ctx := context.Background()

	// Check that init command was run
	result, err := session.executeBashInternal(ctx, "echo $INIT_TEST", 0)

	if err != nil {
		t.Fatalf("executeBashInternal after init failed: %v", err)
	}

	if !strings.Contains(result, "1") {
		t.Errorf("Expected INIT_TEST to be set, got: %s", result)
	}
}

func TestBashSession_Reset(t *testing.T) {
	session := NewBashSession()

	session.SetEnv("RESET_TEST", "value")
	session.Reset()

	_, ok := session.GetEnv("RESET_TEST")
	if ok {
		t.Error("Expected env var to be cleared after reset")
	}

	if session.initialized {
		t.Error("Expected initialized to be false after reset")
	}
}

func TestBashTools_JobDone(t *testing.T) {
	bt := NewBashTools(nil)
	ctx := context.Background()

	result, err := bt.JobDone(ctx, JobDoneParams{})
	if err != nil {
		t.Fatalf("JobDone failed: %v", err)
	}

	if !strings.Contains(result, "completed successfully") {
		t.Errorf("Expected success message, got: %s", result)
	}
}

func TestGetGlobalBashSession(t *testing.T) {
	session1 := GetGlobalBashSession()
	session2 := GetGlobalBashSession()

	if session1 != session2 {
		t.Error("Expected singleton instance")
	}

	// Verify it's actually the same session
	session1.SetEnv("SINGLETON_TEST", "1")
	val, _ := session2.GetEnv("SINGLETON_TEST")
	if val != "1" {
		t.Error("Singleton not working correctly")
	}
}

func TestBashTools_ExecuteBash(t *testing.T) {
	bt := NewBashTools(nil)
	ctx := context.Background()

	result, err := bt.ExecuteBash(ctx, ExecuteBashParams{
		Command: "echo 'bash tools test'",
	})

	if err != nil {
		t.Fatalf("ExecuteBash failed: %v", err)
	}

	if !strings.Contains(result, "bash tools test") {
		t.Errorf("Expected output in result, got: %s", result)
	}
}

func TestBashTools_GetSession(t *testing.T) {
	bt := NewBashTools(nil)
	session := bt.GetSession()

	if session == nil {
		t.Error("Expected non-nil session")
	}
}

func TestConfigureBash(t *testing.T) {
	// This is a global operation, so we need to be careful
	originalInits := []string{}
	if globalBashSession != nil {
		originalInits = globalBashSession.initCommands
	}

	defer func() {
		ConfigureBash(originalInits, false)
	}()

	ConfigureBash([]string{"export CONFIGURE_TEST=1"}, false)

	if globalBashSession == nil {
		t.Error("Expected global session to exist")
	}

	// Note: We can't easily test the actual initialization without side effects
}

func TestBashSessionConcurrentAccess(t *testing.T) {
	session := NewBashSession()
	ctx := context.Background()

	// Test concurrent access
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			_, _ = session.executeBashInternal(ctx, "echo test", 0)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// If we got here without deadlock or race, the test passed
}

func TestBatchTool_Batch(t *testing.T) {
	bt := NewBatchTool()
	ctx := context.Background()

	// Register test functions
	bt.Register("tool1", func(ctx context.Context, kwargs map[string]any) (string, error) {
		return "result1", nil
	})
	bt.Register("tool2", func(ctx context.Context, kwargs map[string]any) (string, error) {
		return "result2", nil
	})

	invocations := []Invocation{
		{ToolName: "tool1", Input: map[string]any{}},
		{ToolName: "tool2", Input: map[string]any{}},
	}

	result, err := bt.Batch(ctx, "test batch", invocations)
	if err != nil {
		t.Fatalf("Batch failed: %v", err)
	}

	if !strings.Contains(result, "Completed: 2/2") {
		t.Errorf("Expected completion message, got: %s", result)
	}

	if !strings.Contains(result, "tool1:") {
		t.Errorf("Expected tool1 in results, got: %s", result)
	}

	if !strings.Contains(result, "tool2:") {
		t.Errorf("Expected tool2 in results, got: %s", result)
	}
}

func TestBatchTool_ToolNotFound(t *testing.T) {
	bt := NewBatchTool()
	ctx := context.Background()

	invocations := []Invocation{
		{ToolName: "nonexistent", Input: map[string]any{}},
	}

	result, err := bt.Batch(ctx, "test error", invocations)
	if err != nil {
		t.Fatalf("Batch should not error on tool not found: %v", err)
	}

	if !strings.Contains(result, "Errors") {
		t.Errorf("Expected error message, got: %s", result)
	}
}

func TestBatchTool_EmptyInvocations(t *testing.T) {
	bt := NewBatchTool()
	ctx := context.Background()

	result, err := bt.Batch(ctx, "empty test", []Invocation{})
	if err != nil {
		t.Fatalf("Batch failed: %v", err)
	}

	if !strings.Contains(result, "No invocations provided") {
		t.Errorf("Expected empty invocations message, got: %s", result)
	}
}

func TestNotebookTools_ReadNotebook(t *testing.T) {
	tmpDir := t.TempDir()
	nt := NewNotebookTools(tmpDir)
	ctx := context.Background()

	// Create a simple notebook JSON
	notebookJSON := `{
		"cells": [
			{
				"cell_type": "code",
				"source": ["print('hello')"],
				"outputs": [
					{
						"output_type": "stream",
						"text": ["hello\n"]
					}
				]
			},
			{
				"cell_type": "markdown",
				"source": ["# Test Notebook"]
			}
		],
		"nbformat": 4,
		"nbformat_minor": 2
	}`

	notebookPath := filepath.Join(tmpDir, "test.ipynb")
	if err := os.WriteFile(notebookPath, []byte(notebookJSON), 0644); err != nil {
		t.Fatalf("Failed to create notebook: %v", err)
	}

	result, err := nt.ReadNotebook(ctx, ReadNotebookParams{
		NotebookPath: "test.ipynb",
	})

	if err != nil {
		t.Fatalf("ReadNotebook failed: %v", err)
	}

	if !strings.Contains(result, "Cell 0 [code]") {
		t.Errorf("Expected cell header in result, got: %s", result)
	}

	if !strings.Contains(result, "print('hello')") {
		t.Errorf("Expected source code in result, got: %s", result)
	}

	if !strings.Contains(result, "hello") {
		t.Errorf("Expected output in result, got: %s", result)
	}
}

func TestNotebookTools_NotebookEditCell(t *testing.T) {
	tmpDir := t.TempDir()
	nt := NewNotebookTools(tmpDir)
	ctx := context.Background()

	// Create initial notebook
	notebookJSON := `{
		"cells": [
			{
				"cell_type": "code",
				"source": ["old source"],
				"metadata": {}
			}
		],
		"nbformat": 4,
		"nbformat_minor": 2
	}`

	notebookPath := filepath.Join(tmpDir, "edit.ipynb")
	if err := os.WriteFile(notebookPath, []byte(notebookJSON), 0644); err != nil {
		t.Fatalf("Failed to create notebook: %v", err)
	}

	// Test replace mode
	result, err := nt.NotebookEditCell(ctx, NotebookEditCellParams{
		NotebookPath: "edit.ipynb",
		CellNumber:   0,
		NewSource:    "new source",
		EditMode:     "replace",
	})

	if err != nil {
		t.Fatalf("NotebookEditCell failed: %v", err)
	}

	if !strings.Contains(result, "Successfully replaced") {
		t.Errorf("Expected success message, got: %s", result)
	}

	// Verify the edit
	data, err := os.ReadFile(notebookPath)
	if err != nil {
		t.Fatalf("Failed to read edited notebook: %v", err)
	}

	if !strings.Contains(string(data), "new source") {
		t.Error("Edit was not applied")
	}
}

func TestNotebookTools_NotebookEditCellInsert(t *testing.T) {
	tmpDir := t.TempDir()
	nt := NewNotebookTools(tmpDir)
	ctx := context.Background()

	// Create initial notebook
	notebookJSON := `{
		"cells": [
			{
				"cell_type": "code",
				"source": ["existing"],
				"metadata": {}
			}
		],
		"nbformat": 4,
		"nbformat_minor": 2
	}`

	notebookPath := filepath.Join(tmpDir, "insert.ipynb")
	if err := os.WriteFile(notebookPath, []byte(notebookJSON), 0644); err != nil {
		t.Fatalf("Failed to create notebook: %v", err)
	}

	// Test insert mode
	result, err := nt.NotebookEditCell(ctx, NotebookEditCellParams{
		NotebookPath: "insert.ipynb",
		CellNumber:   0,
		NewSource:    "new cell",
		EditMode:     "insert",
		CellType:     "code",
	})

	if err != nil {
		t.Fatalf("NotebookEditCell insert failed: %v", err)
	}

	if !strings.Contains(result, "Successfully inserted") {
		t.Errorf("Expected success message, got: %s", result)
	}

	// Verify the insert
	data, err := os.ReadFile(notebookPath)
	if err != nil {
		t.Fatalf("Failed to read edited notebook: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "new cell") {
		t.Error("Insert was not applied")
	}

	// Check that we now have 2 cells
	if strings.Count(content, `"cell_type"`) != 2 {
		t.Error("Expected 2 cells after insert")
	}
}

func TestNotebookTools_NotebookEditCellDelete(t *testing.T) {
	tmpDir := t.TempDir()
	nt := NewNotebookTools(tmpDir)
	ctx := context.Background()

	// Create notebook with 2 cells
	notebookJSON := `{
		"cells": [
			{
				"cell_type": "code",
				"source": ["cell 1"],
				"metadata": {}
			},
			{
				"cell_type": "code",
				"source": ["cell 2"],
				"metadata": {}
			}
		],
		"nbformat": 4,
		"nbformat_minor": 2
	}`

	notebookPath := filepath.Join(tmpDir, "delete.ipynb")
	if err := os.WriteFile(notebookPath, []byte(notebookJSON), 0644); err != nil {
		t.Fatalf("Failed to create notebook: %v", err)
	}

	// Test delete mode
	result, err := nt.NotebookEditCell(ctx, NotebookEditCellParams{
		NotebookPath: "delete.ipynb",
		CellNumber:   0,
		NewSource:    "",
		EditMode:     "delete",
	})

	if err != nil {
		t.Fatalf("NotebookEditCell delete failed: %v", err)
	}

	if !strings.Contains(result, "Successfully deleted") {
		t.Errorf("Expected success message, got: %s", result)
	}

	// Verify the delete
	data, err := os.ReadFile(notebookPath)
	if err != nil {
		t.Fatalf("Failed to read edited notebook: %v", err)
	}

	content := string(data)
	if strings.Contains(content, "cell 1") {
		t.Error("Delete was not applied - cell 1 still present")
	}

	if !strings.Contains(content, "cell 2") {
		t.Error("cell 2 should still be present")
	}
}
