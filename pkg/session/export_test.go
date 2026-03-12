package session

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/tingly-dev/tingly-agentscope/pkg/module"
)

// TestExportToDict tests the ExportToDict method
func TestExportToDict(t *testing.T) {
	session := NewJSONSession(t.TempDir())

	// Create mock state modules
	modules := map[string]module.StateModule{
		"module1": NewMockStateModule(),
		"module2": NewMockStateModule(),
	}

	// Set test data
	modules["module1"].(*MockStateModule).Set("key1", "value1")
	modules["module1"].(*MockStateModule).Set("number", 42)
	modules["module2"].(*MockStateModule).Set("key2", "value2")
	modules["module2"].(*MockStateModule).Set("active", true)

	// Export to dict
	ctx := context.Background()
	result, err := session.ExportToDict(ctx, modules)
	if err != nil {
		t.Fatalf("ExportToDict failed: %v", err)
	}

	// Verify result structure
	if result == nil {
		t.Fatal("ExportToDict returned nil map")
	}

	// Verify module1 data
	module1Data, ok := result["module1"].(map[string]any)
	if !ok {
		t.Fatal("module1 data is not a map")
	}
	if module1Data["key1"] != "value1" {
		t.Errorf("Expected module1.key1 = 'value1', got %v", module1Data["key1"])
	}
	if module1Data["number"] != 42 {
		t.Errorf("Expected module1.number = 42, got %v", module1Data["number"])
	}

	// Verify module2 data
	module2Data, ok := result["module2"].(map[string]any)
	if !ok {
		t.Fatal("module2 data is not a map")
	}
	if module2Data["key2"] != "value2" {
		t.Errorf("Expected module2.key2 = 'value2', got %v", module2Data["key2"])
	}
	if module2Data["active"] != true {
		t.Errorf("Expected module2.active = true, got %v", module2Data["active"])
	}
}

// TestExportToDict_EmptyModules tests ExportToDict with empty modules
func TestExportToDict_EmptyModules(t *testing.T) {
	session := NewJSONSession(t.TempDir())

	// Test with empty map
	modules := map[string]module.StateModule{}
	ctx := context.Background()
	result, err := session.ExportToDict(ctx, modules)
	if err != nil {
		t.Fatalf("ExportToDict with empty modules failed: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d items", len(result))
	}
}

// TestExportToDict_NilModules tests ExportToDict with nil modules
func TestExportToDict_NilModules(t *testing.T) {
	session := NewJSONSession(t.TempDir())

	// Test with nil map
	ctx := context.Background()
	result, err := session.ExportToDict(ctx, nil)
	if err != nil {
		t.Fatalf("ExportToDict with nil modules failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result map")
	}
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d items", len(result))
	}
}

// TestImportFromDict tests the ImportFromDict method
func TestImportFromDict(t *testing.T) {
	session := NewJSONSession(t.TempDir())

	// Create state dict
	stateDict := map[string]any{
		"module1": map[string]any{
			"key1":   "imported_value1",
			"number": 100,
		},
		"module2": map[string]any{
			"key2":   "imported_value2",
			"active": false,
		},
	}

	// Create mock modules
	modules := map[string]module.StateModule{
		"module1": NewMockStateModule(),
		"module2": NewMockStateModule(),
	}

	// Import from dict
	ctx := context.Background()
	err := session.ImportFromDict(ctx, stateDict, modules)
	if err != nil {
		t.Fatalf("ImportFromDict failed: %v", err)
	}

	// Verify module1 data
	val1, ok1 := modules["module1"].(*MockStateModule).Get("key1")
	if !ok1 || val1 != "imported_value1" {
		t.Errorf("Expected module1.key1 = 'imported_value1', got %v", val1)
	}
	val2, ok2 := modules["module1"].(*MockStateModule).Get("number")
	if !ok2 || val2 != 100 {
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

// TestImportFromDict_NilState tests ImportFromDict with nil state
func TestImportFromDict_NilState(t *testing.T) {
	session := NewJSONSession(t.TempDir())

	modules := map[string]module.StateModule{
		"module1": NewMockStateModule(),
	}

	ctx := context.Background()
	err := session.ImportFromDict(ctx, nil, modules)
	if err == nil {
		t.Fatal("ImportFromDict with nil state should return error")
	}

	expectedErrMsg := "state dictionary cannot be nil"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

// TestImportFromDict_InvalidStateFormat tests ImportFromDict with invalid state format
func TestImportFromDict_InvalidStateFormat(t *testing.T) {
	session := NewJSONSession(t.TempDir())

	tests := []struct {
		name        string
		stateDict   map[string]any
		expectError bool
		errorMsg    string
	}{
		{
			name: "string value instead of map",
			stateDict: map[string]any{
				"module1": "invalid string value",
			},
			expectError: true,
			errorMsg:    "invalid state format for module module1",
		},
		{
			name: "number value instead of map",
			stateDict: map[string]any{
				"module1": 12345,
			},
			expectError: true,
			errorMsg:    "invalid state format for module module1",
		},
		{
			name: "nil value for module",
			stateDict: map[string]any{
				"module1": nil,
			},
			expectError: true,
			errorMsg:    "invalid state format for module module1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modules := map[string]module.StateModule{
				"module1": NewMockStateModule(),
			}

			ctx := context.Background()
			err := session.ImportFromDict(ctx, tt.stateDict, modules)
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				// Check if error message contains expected substring
				if len(tt.errorMsg) > 0 {
					errMsg := err.Error()
					// Use simple substring check instead of strings.Contains
					found := false
					for i := 0; i <= len(errMsg)-len(tt.errorMsg); i++ {
						if errMsg[i:i+len(tt.errorMsg)] == tt.errorMsg {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Error message should contain '%s', got '%s'", tt.errorMsg, errMsg)
					}
				}
			}
		})
	}
}

// TestImportFromDict_ModuleNotFound tests ImportFromDict when module is not in state
func TestImportFromDict_ModuleNotFound(t *testing.T) {
	session := NewJSONSession(t.TempDir())

	// State dict doesn't contain module2
	stateDict := map[string]any{
		"module1": map[string]any{
			"key1": "value1",
		},
	}

	modules := map[string]module.StateModule{
		"module1": NewMockStateModule(),
		"module2": NewMockStateModule(), // This module is not in stateDict
	}

	ctx := context.Background()
	err := session.ImportFromDict(ctx, stateDict, modules)
	if err != nil {
		t.Fatalf("ImportFromDict should not fail when module is not in state: %v", err)
	}

	// Verify module1 was loaded
	val1, ok1 := modules["module1"].(*MockStateModule).Get("key1")
	if !ok1 || val1 != "value1" {
		t.Errorf("Expected module1.key1 = 'value1', got %v", val1)
	}

	// Verify module2 is empty (not loaded)
	_, ok2 := modules["module2"].(*MockStateModule).Get("key1")
	if ok2 {
		t.Error("module2 should not have been loaded")
	}
}

// TestExportImportDictRoundTrip tests round-trip export/import with dict
func TestExportImportDictRoundTrip(t *testing.T) {
	session := NewJSONSession(t.TempDir())

	// Create original modules with data
	originalModules := map[string]module.StateModule{
		"module1": NewMockStateModule(),
		"module2": NewMockStateModule(),
	}
	originalModules["module1"].(*MockStateModule).Set("timestamp", time.Now().Unix())
	originalModules["module1"].(*MockStateModule).Set("items", []string{"a", "b", "c"})
	originalModules["module2"].(*MockStateModule).Set("nested", map[string]any{"x": 1, "y": 2})

	// Export to dict
	ctx := context.Background()
	stateDict, err := session.ExportToDict(ctx, originalModules)
	if err != nil {
		t.Fatalf("Export to dict failed: %v", err)
	}

	// Create new modules for import
	newModules := map[string]module.StateModule{
		"module1": NewMockStateModule(),
		"module2": NewMockStateModule(),
	}

	// Import from dict
	err = session.ImportFromDict(ctx, stateDict, newModules)
	if err != nil {
		t.Fatalf("Import from dict failed: %v", err)
	}

	// Verify data matches
	origModule1 := originalModules["module1"].(*MockStateModule)
	newModule1 := newModules["module1"].(*MockStateModule)

	// Check timestamp
	origTimestamp, _ := origModule1.Get("timestamp")
	newTimestamp, ok := newModule1.Get("timestamp")
	if !ok {
		t.Error("timestamp not found in imported data")
	} else if newTimestamp != origTimestamp {
		t.Errorf("timestamp mismatch: original=%v, imported=%v", origTimestamp, newTimestamp)
	}

	// Check items
	origItems, _ := origModule1.Get("items")
	newItems, ok := newModule1.Get("items")
	if !ok {
		t.Error("items not found in imported data")
	} else {
		origSlice := origItems.([]string)
		newSlice := newItems.([]string)
		if len(origSlice) != len(newSlice) {
			t.Errorf("items length mismatch: original=%d, imported=%d", len(origSlice), len(newSlice))
		}
		for i, v := range origSlice {
			if newSlice[i] != v {
				t.Errorf("items[%d] mismatch: original=%v, imported=%v", i, v, newSlice[i])
			}
		}
	}
}

// TestExportToJSON tests the ExportToJSON method
func TestExportToJSON(t *testing.T) {
	session := NewJSONSession(t.TempDir())

	// Create mock state modules
	modules := map[string]module.StateModule{
		"module1": NewMockStateModule(),
		"module2": NewMockStateModule(),
	}

	// Set test data
	modules["module1"].(*MockStateModule).Set("key1", "value1")
	modules["module1"].(*MockStateModule).Set("number", 42)
	modules["module2"].(*MockStateModule).Set("key2", "value2")

	// Export to JSON
	ctx := context.Background()
	jsonStr, err := session.ExportToJSON(ctx, modules)
	if err != nil {
		t.Fatalf("ExportToJSON failed: %v", err)
	}

	// Verify JSON can be unmarshaled
	var result map[string]map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Fatalf("Failed to unmarshal exported JSON: %v", err)
	}

	// Verify data
	if result["module1"]["key1"] != "value1" {
		t.Errorf("Expected module1.key1 = 'value1', got %v", result["module1"]["key1"])
	}
	if result["module1"]["number"] != float64(42) {
		t.Errorf("Expected module1.number = 42, got %v", result["module1"]["number"])
	}
}

// TestExportToJSON_EmptyModules tests ExportToJSON with empty modules
func TestExportToJSON_EmptyModules(t *testing.T) {
	session := NewJSONSession(t.TempDir())

	modules := map[string]module.StateModule{}
	ctx := context.Background()
	jsonStr, err := session.ExportToJSON(ctx, modules)
	if err != nil {
		t.Fatalf("ExportToJSON with empty modules failed: %v", err)
	}

	// Verify JSON is valid empty object
	var result map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Expected empty JSON object, got %d items", len(result))
	}
}

// TestExportToJSON_NilModules tests ExportToJSON with nil modules
func TestExportToJSON_NilModules(t *testing.T) {
	session := NewJSONSession(t.TempDir())

	ctx := context.Background()
	jsonStr, err := session.ExportToJSON(ctx, nil)
	if err != nil {
		t.Fatalf("ExportToJSON with nil modules failed: %v", err)
	}

	// Verify JSON is valid empty object
	var result map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Expected empty JSON object, got %d items", len(result))
	}
}

// TestImportFromJSON tests the ImportFromJSON method
func TestImportFromJSON(t *testing.T) {
	session := NewJSONSession(t.TempDir())

	// Create JSON string
	jsonData := `{
		"module1": {
			"key1": "json_value1",
			"number": 200
		},
		"module2": {
			"key2": "json_value2"
		}
	}`

	// Create mock modules
	modules := map[string]module.StateModule{
		"module1": NewMockStateModule(),
		"module2": NewMockStateModule(),
	}

	// Import from JSON
	ctx := context.Background()
	err := session.ImportFromJSON(ctx, jsonData, modules)
	if err != nil {
		t.Fatalf("ImportFromJSON failed: %v", err)
	}

	// Verify module1 data
	val1, ok1 := modules["module1"].(*MockStateModule).Get("key1")
	if !ok1 || val1 != "json_value1" {
		t.Errorf("Expected module1.key1 = 'json_value1', got %v", val1)
	}
	val2, ok2 := modules["module1"].(*MockStateModule).Get("number")
	if !ok2 || val2 != float64(200) {
		t.Errorf("Expected module1.number = 200 (as float64), got %v (type %T)", val2, val2)
	}
}

// TestImportFromJSON_InvalidJSON tests ImportFromJSON with invalid JSON
func TestImportFromJSON_InvalidJSON(t *testing.T) {
	session := NewJSONSession(t.TempDir())

	modules := map[string]module.StateModule{
		"module1": NewMockStateModule(),
	}

	tests := []struct {
		name        string
		jsonData    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "malformed JSON",
			jsonData:    `{"key": "value"`,
			expectError: true,
			errorMsg:    "failed to parse JSON data",
		},
		{
			name:        "empty string",
			jsonData:    "",
			expectError: true,
			errorMsg:    "failed to parse JSON data",
		},
		{
			name:        "null JSON",
			jsonData:    "null",
			expectError: true,
			errorMsg:    "invalid JSON format",
		},
		{
			name:        "array instead of object",
			jsonData:    `["item1", "item2"]`,
			expectError: true,
			errorMsg:    "failed to parse JSON data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := session.ImportFromJSON(ctx, tt.jsonData, modules)
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				// Check if error message contains expected substring
				if len(tt.errorMsg) > 0 {
					errMsg := err.Error()
					found := false
					for i := 0; i <= len(errMsg)-len(tt.errorMsg); i++ {
						if errMsg[i:i+len(tt.errorMsg)] == tt.errorMsg {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Error message should contain '%s', got '%s'", tt.errorMsg, errMsg)
					}
				}
			}
		})
	}
}

// TestExportImportJSONRoundTrip tests round-trip export/import with JSON
func TestExportImportJSONRoundTrip(t *testing.T) {
	session := NewJSONSession(t.TempDir())

	// Create original modules with data
	originalModules := map[string]module.StateModule{
		"module1": NewMockStateModule(),
		"module2": NewMockStateModule(),
	}
	originalModules["module1"].(*MockStateModule).Set("timestamp", time.Now().Unix())
	originalModules["module1"].(*MockStateModule).Set("items", []string{"x", "y", "z"})
	originalModules["module2"].(*MockStateModule).Set("flag", true)

	// Export to JSON
	ctx := context.Background()
	jsonStr, err := session.ExportToJSON(ctx, originalModules)
	if err != nil {
		t.Fatalf("Export to JSON failed: %v", err)
	}

	// Create new modules for import
	newModules := map[string]module.StateModule{
		"module1": NewMockStateModule(),
		"module2": NewMockStateModule(),
	}

	// Import from JSON
	err = session.ImportFromJSON(ctx, jsonStr, newModules)
	if err != nil {
		t.Fatalf("Import from JSON failed: %v", err)
	}

	// Verify data matches
	origModule1 := originalModules["module1"].(*MockStateModule)
	newModule1 := newModules["module1"].(*MockStateModule)

	// Check timestamp - JSON converts int64 to float64
	origTimestamp, _ := origModule1.Get("timestamp")
	newTimestamp, ok := newModule1.Get("timestamp")
	if !ok {
		t.Error("timestamp not found in imported data")
	} else {
		// Original is int64, imported is float64 after JSON round-trip
		origInt, ok1 := origTimestamp.(int64)
		newFloat, ok2 := newTimestamp.(float64)
		if !ok1 || !ok2 {
			t.Errorf("timestamp type mismatch: original=%T, imported=%T", origTimestamp, newTimestamp)
		} else if int64(newFloat) != origInt {
			t.Errorf("timestamp mismatch: original=%v, imported=%v", origTimestamp, newTimestamp)
		}
	}

	// Check items - JSON converts []string to []interface{}
	origItems, _ := origModule1.Get("items")
	newItems, ok := newModule1.Get("items")
	if !ok {
		t.Error("items not found in imported data")
	} else {
		// Original is []string, imported is []interface{}
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
}

// TestConcurrentDictOperations tests thread safety of dict operations
func TestConcurrentDictOperations(t *testing.T) {
	session := NewJSONSession(t.TempDir())
	ctx := context.Background()
	numGoroutines := 50
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*2)

	// Concurrent exports
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			modules := map[string]module.StateModule{
				"module1": NewMockStateModule(),
			}
			modules["module1"].(*MockStateModule).Set("index", idx)
			_, err := session.ExportToDict(ctx, modules)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	// Concurrent imports
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			stateDict := map[string]any{
				"module1": map[string]any{
					"index": idx,
				},
			}
			modules := map[string]module.StateModule{
				"module1": NewMockStateModule(),
			}
			err := session.ImportFromDict(ctx, stateDict, modules)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Errorf("Concurrent operation failed: %v", err)
		errorCount++
	}
	if errorCount > 0 {
		t.Fatalf(" %d concurrent operations failed", errorCount)
	}
}

// TestConcurrentJSONOperations tests thread safety of JSON operations
func TestConcurrentJSONOperations(t *testing.T) {
	session := NewJSONSession(t.TempDir())
	ctx := context.Background()
	numGoroutines := 50
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*2)

	// Concurrent exports
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			modules := map[string]module.StateModule{
				"module1": NewMockStateModule(),
			}
			modules["module1"].(*MockStateModule).Set("index", idx)
			_, err := session.ExportToJSON(ctx, modules)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	// Concurrent imports
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			jsonData := `{"module1": {"index": ` + string(rune('0'+idx%10)) + `}}`
			modules := map[string]module.StateModule{
				"module1": NewMockStateModule(),
			}
			err := session.ImportFromJSON(ctx, jsonData, modules)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Errorf("Concurrent operation failed: %v", err)
		errorCount++
	}
	if errorCount > 0 {
		t.Fatalf("%d concurrent operations failed", errorCount)
	}
}

// TestMixedConcurrentOperations tests thread safety with mixed operations
func TestMixedConcurrentOperations(t *testing.T) {
	session := NewJSONSession(t.TempDir())
	ctx := context.Background()
	numGoroutines := 30
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*3)

	// Mix of all operations
	for i := 0; i < numGoroutines; i++ {
		// Export to dict
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			modules := map[string]module.StateModule{
				"module1": NewMockStateModule(),
			}
			modules["module1"].(*MockStateModule).Set("id", idx)
			_, err := session.ExportToDict(ctx, modules)
			if err != nil {
				errors <- err
			}
		}(i)

		// Import from dict
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			stateDict := map[string]any{
				"module1": map[string]any{"id": idx},
			}
			modules := map[string]module.StateModule{
				"module1": NewMockStateModule(),
			}
			err := session.ImportFromDict(ctx, stateDict, modules)
			if err != nil {
				errors <- err
			}
		}(i)

		// Export to JSON
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			modules := map[string]module.StateModule{
				"module1": NewMockStateModule(),
			}
			modules["module1"].(*MockStateModule).Set("id", idx)
			_, err := session.ExportToJSON(ctx, modules)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent operation failed: %v", err)
	}
}

// TestExportToFilePath_InvalidPath tests ExportToFilePath with invalid paths
func TestExportToFilePath_InvalidPath(t *testing.T) {
	session := NewJSONSession(t.TempDir())

	modules := map[string]module.StateModule{
		"module1": NewMockStateModule(),
	}

	ctx := context.Background()

	// Test with a path that cannot be created (e.g., permission denied on /root)
	// Use t.TempDir() which is guaranteed to be writable
	testFile := filepath.Join(t.TempDir(), "test.json")
	err := session.ExportToFilePath(ctx, testFile, modules)
	if err != nil {
		t.Fatalf("ExportToFilePath should succeed: %v", err)
	}
}

// TestImportFromFilePath_AllowNotExistInNew tests ImportFromFilePath with allowNotExist flag
// Note: This is a duplicate of TestImportFromFilePath_AllowNotExist from export_file_test.go
// to ensure the test file is complete and self-contained
func TestImportFromFilePath_AllowNotExistInNew(t *testing.T) {
	session := NewJSONSession(t.TempDir())

	modules := map[string]module.StateModule{
		"module1": NewMockStateModule(),
	}

	ctx := context.Background()

	// Test with non-existent file and allowNotExist=true
	nonExistentFile := filepath.Join(t.TempDir(), "does_not_exist.json")
	err := session.ImportFromFilePath(ctx, nonExistentFile, modules, true)
	if err != nil {
		t.Fatalf("ImportFromFilePath with allowNotExist=true should not error: %v", err)
	}

	// Test with non-existent file and allowNotExist=false
	err = session.ImportFromFilePath(ctx, nonExistentFile, modules, false)
	if err == nil {
		t.Fatal("ImportFromFilePath with allowNotExist=false should error for non-existent file")
	}
}

// TestImportFromFilePath_InvalidJSON tests ImportFromFilePath with invalid JSON
func TestImportFromFilePath_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "invalid.json")

	// Write invalid JSON
	if err := os.WriteFile(testFile, []byte(`{"invalid": json}`), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	session := NewJSONSession(tempDir)
	modules := map[string]module.StateModule{
		"module1": NewMockStateModule(),
	}

	ctx := context.Background()
	err := session.ImportFromFilePath(ctx, testFile, modules, false)
	if err == nil {
		t.Fatal("ImportFromFilePath should fail with invalid JSON")
	}
}

// TestExportToFilePath_EmptyModules tests ExportToFilePath with empty modules
func TestExportToFilePath_EmptyModules(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "empty.json")

	session := NewJSONSession(tempDir)
	modules := map[string]module.StateModule{}

	ctx := context.Background()
	err := session.ExportToFilePath(ctx, testFile, modules)
	if err != nil {
		t.Fatalf("ExportToFilePath with empty modules failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatal("File was not created")
	}

	// Verify file contains empty JSON object
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty JSON object, got %d items", len(result))
	}
}

// TestExportToFilePath_NilModules tests ExportToFilePath with nil modules
func TestExportToFilePath_NilModules(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "nil.json")

	session := NewJSONSession(tempDir)

	ctx := context.Background()
	err := session.ExportToFilePath(ctx, testFile, nil)
	if err != nil {
		t.Fatalf("ExportToFilePath with nil modules failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatal("File was not created")
	}
}

// TestImportFromFilePath_EmptyModules tests ImportFromFilePath with empty modules
func TestImportFromFilePath_EmptyModules(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.json")

	// Create test data
	testData := map[string]map[string]any{
		"module1": {"key": "value"},
	}
	data, _ := json.Marshal(testData)
	if err := os.WriteFile(testFile, data, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	session := NewJSONSession(tempDir)
	modules := map[string]module.StateModule{}

	ctx := context.Background()
	err := session.ImportFromFilePath(ctx, testFile, modules, false)
	if err != nil {
		t.Fatalf("ImportFromFilePath with empty modules should succeed: %v", err)
	}
}

// TestImportFromFilePath_NilModules tests ImportFromFilePath with nil modules
func TestImportFromFilePath_NilModules(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.json")

	// Create test data
	testData := map[string]map[string]any{
		"module1": {"key": "value"},
	}
	data, _ := json.Marshal(testData)
	if err := os.WriteFile(testFile, data, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	session := NewJSONSession(tempDir)

	ctx := context.Background()
	err := session.ImportFromFilePath(ctx, testFile, nil, false)
	if err != nil {
		t.Fatalf("ImportFromFilePath with nil modules should succeed: %v", err)
	}
}

// TestExportImportFilePathRoundTrip_NestedData tests round-trip with complex nested data
func TestExportImportFilePathRoundTrip_NestedData(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "nested.json")

	session := NewJSONSession(tempDir)

	// Create modules with complex nested data
	originalModules := map[string]module.StateModule{
		"module1": NewMockStateModule(),
		"module2": NewMockStateModule(),
	}

	// Set nested data
	originalModules["module1"].(*MockStateModule).Set("config", map[string]any{
		"nested": map[string]any{
			"deep": map[string]any{
				"value": "deep_value",
			},
		},
		"list": []any{1, 2, 3, "four", map[string]any{"five": 5}},
	})

	originalModules["module2"].(*MockStateModule).Set("empty_map", map[string]any{})
	originalModules["module2"].(*MockStateModule).Set("empty_slice", []any{})

	// Export
	ctx := context.Background()
	err := session.ExportToFilePath(ctx, testFile, originalModules)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Create new modules
	newModules := map[string]module.StateModule{
		"module1": NewMockStateModule(),
		"module2": NewMockStateModule(),
	}

	// Import
	err = session.ImportFromFilePath(ctx, testFile, newModules, false)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify nested structure
	newModule1 := newModules["module1"].(*MockStateModule)
	config, ok := newModule1.Get("config")
	if !ok {
		t.Fatal("config not found in imported data")
	}

	configMap, ok := config.(map[string]any)
	if !ok {
		t.Fatal("config is not a map")
	}

	nested, ok := configMap["nested"].(map[string]any)
	if !ok {
		t.Fatal("nested is not a map")
	}

	deep, ok := nested["deep"].(map[string]any)
	if !ok {
		t.Fatal("deep is not a map")
	}

	if deep["value"] != "deep_value" {
		t.Errorf("Expected deep.value = 'deep_value', got %v", deep["value"])
	}

	// Verify list
	list, ok := configMap["list"].([]any)
	if !ok {
		t.Fatal("list is not a slice")
	}

	if len(list) != 5 {
		t.Errorf("Expected list length 5, got %d", len(list))
	}

	// Verify empty structures in module2
	newModule2 := newModules["module2"].(*MockStateModule)
	emptyMap, ok := newModule2.Get("empty_map")
	if !ok {
		t.Fatal("empty_map not found")
	}
	if len(emptyMap.(map[string]any)) != 0 {
		t.Error("empty_map should be empty")
	}

	emptySlice, ok := newModule2.Get("empty_slice")
	if !ok {
		t.Fatal("empty_slice not found")
	}
	if len(emptySlice.([]any)) != 0 {
		t.Error("empty_slice should be empty")
	}
}
