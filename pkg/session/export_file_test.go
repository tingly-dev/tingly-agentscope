package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tingly-dev/tingly-agentscope/pkg/module"
)

// MockStateModule is a mock implementation of StateModule for testing
type MockStateModule struct {
	data map[string]any
}

func NewMockStateModule() *MockStateModule {
	return &MockStateModule{
		data: make(map[string]any),
	}
}

func (m *MockStateModule) StateDict() map[string]any {
	// Return a copy to prevent external modification
	result := make(map[string]any, len(m.data))
	for k, v := range m.data {
		result[k] = v
	}
	return result
}

func (m *MockStateModule) LoadStateDict(ctx context.Context, state map[string]any) error {
	// Copy the state into our internal data
	m.data = make(map[string]any, len(state))
	for k, v := range state {
		m.data[k] = v
	}
	return nil
}

func (m *MockStateModule) Set(key string, value any) {
	m.data[key] = value
}

func (m *MockStateModule) Get(key string) (any, bool) {
	val, ok := m.data[key]
	return val, ok
}

func TestExportToFilePath(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "subdir", "test_session.json")

	// Create JSONSession instance
	session := NewJSONSession(tempDir)

	// Create mock state modules
	modules := map[string]module.StateModule{
		"module1": NewMockStateModule(),
		"module2": NewMockStateModule(),
	}

	// Set some test data
	modules["module1"].(*MockStateModule).Set("key1", "value1")
	modules["module1"].(*MockStateModule).Set("number", 42)
	modules["module2"].(*MockStateModule).Set("key2", "value2")
	modules["module2"].(*MockStateModule).Set("active", true)

	// Test export
	ctx := context.Background()
	err := session.ExportToFilePath(ctx, testFile, modules)
	if err != nil {
		t.Fatalf("ExportToFilePath failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatalf("Export file was not created: %s", testFile)
	}

	// Verify file contents
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read exported file: %v", err)
	}

	var state map[string]map[string]any
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("Failed to unmarshal exported data: %v", err)
	}

	// Verify module1 data
	if state["module1"]["key1"] != "value1" {
		t.Errorf("Expected module1.key1 = 'value1', got %v", state["module1"]["key1"])
	}
	if state["module1"]["number"] != float64(42) {
		t.Errorf("Expected module1.number = 42, got %v", state["module1"]["number"])
	}

	// Verify module2 data
	if state["module2"]["key2"] != "value2" {
		t.Errorf("Expected module2.key2 = 'value2', got %v", state["module2"]["key2"])
	}
	if state["module2"]["active"] != true {
		t.Errorf("Expected module2.active = true, got %v", state["module2"]["active"])
	}
}

func TestImportFromFilePath(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_session.json")

	// Create test data
	testData := map[string]map[string]any{
		"module1": {
			"key1":   "imported_value1",
			"number": 100,
		},
		"module2": {
			"key2":   "imported_value2",
			"active": false,
		},
	}

	// Write test data to file
	data, _ := json.MarshalIndent(testData, "", "  ")
	if err := os.WriteFile(testFile, data, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create JSONSession instance
	session := NewJSONSession(tempDir)

	// Create mock state modules
	modules := map[string]module.StateModule{
		"module1": NewMockStateModule(),
		"module2": NewMockStateModule(),
	}

	// Test import
	ctx := context.Background()
	err := session.ImportFromFilePath(ctx, testFile, modules, false)
	if err != nil {
		t.Fatalf("ImportFromFilePath failed: %v", err)
	}

	// Verify module1 data
	val1, ok1 := modules["module1"].(*MockStateModule).Get("key1")
	if !ok1 || val1 != "imported_value1" {
		t.Errorf("Expected module1.key1 = 'imported_value1', got %v", val1)
	}
	val2, ok2 := modules["module1"].(*MockStateModule).Get("number")
	if !ok2 || val2 != float64(100) {
		t.Errorf("Expected module1.number = 100, got %v", val2)
	}

	// Verify module2 data
	val3, ok3 := modules["module2"].(*MockStateModule).Get("key2")
	if !ok3 || val3 != "imported_value2" {
		t.Errorf("Expected module2.key2 = 'imported_value2', got %v", val3)
	}
	val4, ok4 := modules["module2"].(*MockStateModule).Get("active")
	if !ok4 || val4 != false {
		t.Errorf("Expected module2.active = false, got %v", val4)
	}
}

func TestImportFromFilePath_AllowNotExist(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()
	nonExistentFile := filepath.Join(tempDir, "does_not_exist.json")

	// Create JSONSession instance
	session := NewJSONSession(tempDir)

	// Create mock state modules
	modules := map[string]module.StateModule{
		"module1": NewMockStateModule(),
	}

	// Test import with allowNotExist=true
	ctx := context.Background()
	err := session.ImportFromFilePath(ctx, nonExistentFile, modules, true)
	if err != nil {
		t.Fatalf("ImportFromFilePath with allowNotExist=true should not return error: %v", err)
	}

	// Test import with allowNotExist=false
	err = session.ImportFromFilePath(ctx, nonExistentFile, modules, false)
	if err == nil {
		t.Fatal("ImportFromFilePath with allowNotExist=false should return error for non-existent file")
	}
}

func TestExportImportRoundTrip(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "roundtrip_test.json")

	// Create JSONSession instance
	session := NewJSONSession(tempDir)

	// Create original modules with data
	originalModules := map[string]module.StateModule{
		"module1": NewMockStateModule(),
		"module2": NewMockStateModule(),
	}
	originalModules["module1"].(*MockStateModule).Set("timestamp", time.Now().Unix())
	originalModules["module1"].(*MockStateModule).Set("items", []string{"a", "b", "c"})
	originalModules["module2"].(*MockStateModule).Set("nested", map[string]any{"x": 1, "y": 2})

	// Export
	ctx := context.Background()
	err := session.ExportToFilePath(ctx, testFile, originalModules)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Create new modules for import
	newModules := map[string]module.StateModule{
		"module1": NewMockStateModule(),
		"module2": NewMockStateModule(),
	}

	// Import
	err = session.ImportFromFilePath(ctx, testFile, newModules, false)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify data matches
	origModule1 := originalModules["module1"].(*MockStateModule)
	newModule1 := newModules["module1"].(*MockStateModule)

	// Check timestamp (JSON unmarshaling converts int to float64)
	origTimestamp, _ := origModule1.Get("timestamp")
	newTimestamp, ok := newModule1.Get("timestamp")
	if !ok {
		t.Error("timestamp not found in imported data")
	} else if int64(newTimestamp.(float64)) != origTimestamp.(int64) {
		t.Errorf("timestamp mismatch: original=%v, imported=%v", origTimestamp, newTimestamp)
	}

	// Check items slice
	origItems, _ := origModule1.Get("items")
	newItems, ok := newModule1.Get("items")
	if !ok {
		t.Error("items not found in imported data")
	} else {
		origSlice := origItems.([]string)
		newSlice := newItems.([]interface{})
		if len(origSlice) != len(newSlice) {
			t.Errorf("items length mismatch: original=%d, imported=%d", len(origSlice), len(newSlice))
		}
		for i, v := range origSlice {
			if newSlice[i] != v {
				t.Errorf("items[%d] mismatch: original=%v, imported=%v", i, v, newSlice[i])
			}
		}
	}

	// Check nested map in module2
	origModule2 := originalModules["module2"].(*MockStateModule)
	newModule2 := newModules["module2"].(*MockStateModule)
	origNested, _ := origModule2.Get("nested")
	newNested, ok := newModule2.Get("nested")
	if !ok {
		t.Error("nested not found in imported data")
	} else {
		origMap := origNested.(map[string]any)
		newMap := newNested.(map[string]any)
		for k, v := range origMap {
			newVal, ok := newMap[k]
			if !ok {
				t.Errorf("nested key %s not found in imported data", k)
			} else if int(newVal.(float64)) != v.(int) {
				t.Errorf("nested[%s] mismatch: original=%v, imported=%v", k, v, newVal)
			}
		}
	}
}

func TestExportToFilePath_CreatesParentDirectories(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()
	deepPath := filepath.Join(tempDir, "level1", "level2", "level3", "test.json")

	// Create JSONSession instance
	session := NewJSONSession(tempDir)

	// Create mock module
	modules := map[string]module.StateModule{
		"module1": NewMockStateModule(),
	}
	modules["module1"].(*MockStateModule).Set("test", "value")

	// Export to deep path
	ctx := context.Background()
	err := session.ExportToFilePath(ctx, deepPath, modules)
	if err != nil {
		t.Fatalf("ExportToFilePath failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(deepPath); os.IsNotExist(err) {
		t.Fatalf("File was not created at deep path: %s", deepPath)
	}

	// Verify parent directories were created
	parentDir := filepath.Dir(deepPath)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		t.Fatalf("Parent directories were not created: %s", parentDir)
	}
}

func TestConcurrentExportImport(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Create JSONSession instance
	session := NewJSONSession(tempDir)

	ctx := context.Background()
	done := make(chan bool, 10)
	errors := make(chan error, 10)

	// Run concurrent operations
	for i := 0; i < 5; i++ {
		go func(idx int) {
			testFile := filepath.Join(tempDir, fmt.Sprintf("concurrent_%d.json", idx))
			modules := map[string]module.StateModule{
				"module1": NewMockStateModule(),
			}
			modules["module1"].(*MockStateModule).Set("index", idx)

			err := session.ExportToFilePath(ctx, testFile, modules)
			if err != nil {
				errors <- err
			}
			done <- true
		}(i)
	}

	for i := 0; i < 5; i++ {
		go func(idx int) {
			testFile := filepath.Join(tempDir, fmt.Sprintf("concurrent_%d.json", idx))
			modules := map[string]module.StateModule{
				"module1": NewMockStateModule(),
			}

			err := session.ImportFromFilePath(ctx, testFile, modules, true)
			if err != nil {
				errors <- err
			}
			done <- true
		}(i)
	}

	// Wait for all operations to complete
	for i := 0; i < 10; i++ {
		select {
		case <-done:
			// Operation completed
		case err := <-errors:
			t.Fatalf("Concurrent operation failed: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("Concurrent operations timed out")
		}
	}
}
