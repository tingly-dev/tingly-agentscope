package session

import (
	"context"
	"testing"
	"time"
)

// TestValidateState_ValidState tests validation of a valid state
func TestValidateState_ValidState(t *testing.T) {
	state := map[string]any{
		"module1": map[string]any{
			"string": "value",
			"number": 42,
			"float":  3.14,
			"bool":   true,
			"null":   nil,
			"array":  []any{1, 2, 3},
			"nested": map[string]any{
				"deep": "value",
			},
		},
		"module2": map[string]any{
			"empty_array": []any{},
			"empty_map":   map[string]any{},
		},
	}

	err := ValidateState(state)
	if err != nil {
		t.Errorf("Expected valid state to pass validation, got error: %v", err)
	}
}

// TestValidateState_NilState tests that nil state returns an error
func TestValidateState_NilState(t *testing.T) {
	err := ValidateState(nil)
	if err == nil {
		t.Fatal("Expected error for nil state, got nil")
	}

	expectedError := "state cannot be nil"
	if err.Error() != expectedError {
		t.Errorf("Expected error message %q, got %q", expectedError, err.Error())
	}
}

// TestValidateState_NilModuleState tests that nil module state returns an error
func TestValidateState_NilModuleState(t *testing.T) {
	state := map[string]any{
		"module1": map[string]any{
			"valid": "value",
		},
		"module2": nil, // Invalid: nil module state
	}

	err := ValidateState(state)
	if err == nil {
		t.Fatal("Expected error for nil module state, got nil")
	}

	expectedError := "validation failed for module module2"
	if err.Error() != expectedError && len(err.Error()) < len(expectedError) {
		t.Errorf("Expected error containing %q, got %q", expectedError, err.Error())
	}
}

// TestValidateState_NestedNilValues tests that nested nil values are allowed
func TestValidateState_NestedNilValues(t *testing.T) {
	state := map[string]any{
		"module1": map[string]any{
			"nullable": nil,
			"nested": map[string]any{
				"also_nullable": nil,
			},
			"array_with_nil": []any{1, nil, 3},
		},
	}

	err := ValidateState(state)
	if err != nil {
		t.Errorf("Expected nested nil values to be valid, got error: %v", err)
	}
}

// TestValidateState_UnsupportedType tests that unsupported types return an error
func TestValidateState_UnsupportedType(t *testing.T) {
	// Create an unsupported type (chan is not supported)
	ch := make(chan int)
	state := map[string]any{
		"module1": map[string]any{
			"valid":   "value",
			"invalid": ch,
		},
	}

	err := ValidateState(state)
	if err == nil {
		t.Fatal("Expected error for unsupported type, got nil")
	}

	expectedError := "validation failed for module module1"
	if err.Error() != expectedError && len(err.Error()) < len(expectedError) {
		t.Errorf("Expected error containing %q, got %q", expectedError, err.Error())
	}
}

// TestValidateState_DeepNesting tests validation of deeply nested structures
func TestValidateState_DeepNesting(t *testing.T) {
	state := map[string]any{
		"module1": map[string]any{
			"level1": map[string]any{
				"level2": map[string]any{
					"level3": map[string]any{
						"level4": map[string]any{
							"deep": "value",
						},
					},
				},
			},
		},
	}

	err := ValidateState(state)
	if err != nil {
		t.Errorf("Expected deeply nested structure to be valid, got error: %v", err)
	}
}

// TestValidateState_StructValues tests that struct values (like time.Time) are allowed
func TestValidateState_StructValues(t *testing.T) {
	state := map[string]any{
		"module1": map[string]any{
			"string":    "value",
			"time":      time.Now(),
			"timestamp": time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		},
	}

	err := ValidateState(state)
	if err != nil {
		t.Errorf("Expected struct values to be valid, got error: %v", err)
	}
}

// TestValidateState_PointerValues tests that pointer values are allowed
func TestValidateState_PointerValues(t *testing.T) {
	str := "test"
	num := 42
	state := map[string]any{
		"module1": map[string]any{
			"string_ptr": &str,
			"int_ptr":    &num,
			"nil_ptr":    (*string)(nil),
		},
	}

	err := ValidateState(state)
	if err != nil {
		t.Errorf("Expected pointer values to be valid, got error: %v", err)
	}
}

// TestValidateState_ComplexArray tests validation of complex arrays
func TestValidateState_ComplexArray(t *testing.T) {
	state := map[string]any{
		"module1": map[string]any{
			"complex_array": []any{
				"string",
				42,
				3.14,
				true,
				nil,
				map[string]any{"nested": "value"},
				[]any{1, 2, 3},
			},
		},
	}

	err := ValidateState(state)
	if err != nil {
		t.Errorf("Expected complex array to be valid, got error: %v", err)
	}
}

// TestValidateState_AllPrimitiveTypes tests all supported primitive types
func TestValidateState_AllPrimitiveTypes(t *testing.T) {
	state := map[string]any{
		"module1": map[string]any{
			"string":  "value",
			"int":     42,
			"int8":    int8(8),
			"int16":   int16(16),
			"int32":   int32(32),
			"int64":   int64(64),
			"uint":    uint(42),
			"uint8":   uint8(8),
			"uint16":  uint16(16),
			"uint32":  uint32(32),
			"uint64":  uint64(64),
			"float32": float32(3.14),
			"float64": 3.14,
			"bool":    true,
		},
	}

	err := ValidateState(state)
	if err != nil {
		t.Errorf("Expected all primitive types to be valid, got error: %v", err)
	}
}

// TestDiffSessions_IdenticalStates tests diff of identical states
func TestDiffSessions_IdenticalStates(t *testing.T) {
	state := map[string]any{
		"module1": map[string]any{
			"key1": "value1",
			"key2": 42,
		},
		"module2": map[string]any{
			"nested": map[string]any{
				"deep": "value",
			},
		},
	}

	ctx := context.Background()
	diff, err := DiffSessions(ctx, state, state)
	if err != nil {
		t.Fatalf("Failed to diff identical states: %v", err)
	}

	if len(diff.Added) != 0 {
		t.Errorf("Expected no added fields, got %d", len(diff.Added))
	}

	if len(diff.Removed) != 0 {
		t.Errorf("Expected no removed fields, got %d", len(diff.Removed))
	}

	if len(diff.Modified) != 0 {
		t.Errorf("Expected no modified fields, got %d", len(diff.Modified))
	}
}

// TestDiffSessions_AddedFields tests detecting added fields
func TestDiffSessions_AddedFields(t *testing.T) {
	state1 := map[string]any{
		"module1": map[string]any{
			"key1": "value1",
		},
	}

	state2 := map[string]any{
		"module1": map[string]any{
			"key1": "value1",
			"key2": "value2", // Added - this will mark module1 as modified
		},
		"module2": map[string]any{ // Added module
			"new": "module",
		},
	}

	ctx := context.Background()
	diff, err := DiffSessions(ctx, state1, state2)
	if err != nil {
		t.Fatalf("Failed to diff states: %v", err)
	}

	if len(diff.Added) != 1 {
		t.Errorf("Expected 1 added module, got %d", len(diff.Added))
	}

	if len(diff.Removed) != 0 {
		t.Errorf("Expected no removed modules, got %d", len(diff.Removed))
	}

	if len(diff.Modified) != 1 {
		t.Errorf("Expected 1 modified module, got %d", len(diff.Modified))
	}

	// Verify module2 was added
	if _, ok := diff.Added["module2"]; !ok {
		t.Error("Expected module2 to be in added fields")
	}

	// Verify module1 was modified (has a new field)
	if _, ok := diff.Modified["module1"]; !ok {
		t.Error("Expected module1 to be in modified fields")
	}
}

// TestDiffSessions_RemovedFields tests detecting removed fields
func TestDiffSessions_RemovedFields(t *testing.T) {
	state1 := map[string]any{
		"module1": map[string]any{
			"key1": "value1",
			"key2": "value2",
		},
		"module2": map[string]any{
			"old": "module",
		},
	}

	state2 := map[string]any{
		"module1": map[string]any{
			"key1": "value1",
			// key2 removed - this will mark module1 as modified
		},
		// module2 removed
	}

	ctx := context.Background()
	diff, err := DiffSessions(ctx, state1, state2)
	if err != nil {
		t.Fatalf("Failed to diff states: %v", err)
	}

	if len(diff.Added) != 0 {
		t.Errorf("Expected no added modules, got %d", len(diff.Added))
	}

	if len(diff.Removed) != 1 {
		t.Errorf("Expected 1 removed module, got %d", len(diff.Removed))
	}

	if len(diff.Modified) != 1 {
		t.Errorf("Expected 1 modified module, got %d", len(diff.Modified))
	}

	// Verify module2 was removed
	if _, ok := diff.Removed["module2"]; !ok {
		t.Error("Expected module2 to be in removed fields")
	}

	// Verify module1 was modified (lost a field)
	if _, ok := diff.Modified["module1"]; !ok {
		t.Error("Expected module1 to be in modified fields")
	}
}

// TestDiffSessions_ModifiedFields tests detecting modified fields
func TestDiffSessions_ModifiedFields(t *testing.T) {
	state1 := map[string]any{
		"module1": map[string]any{
			"string": "old value",
			"number": 10,
		},
	}

	state2 := map[string]any{
		"module1": map[string]any{
			"string": "new value",
			"number": 20,
		},
	}

	ctx := context.Background()
	diff, err := DiffSessions(ctx, state1, state2)
	if err != nil {
		t.Fatalf("Failed to diff states: %v", err)
	}

	if len(diff.Added) != 0 {
		t.Errorf("Expected no added fields, got %d", len(diff.Added))
	}

	if len(diff.Removed) != 0 {
		t.Errorf("Expected no removed fields, got %d", len(diff.Removed))
	}

	if len(diff.Modified) != 1 {
		t.Fatalf("Expected 1 modified module, got %d", len(diff.Modified))
	}

	moduleDiff, ok := diff.Modified["module1"]
	if !ok {
		t.Fatal("Expected module1 to be in modified fields")
	}

	// Verify the module itself was modified
	module1State1 := state1["module1"].(map[string]any)
	module1State2 := state2["module1"].(map[string]any)
	if !deepEqual(moduleDiff.OldValue, module1State1) {
		t.Error("Old value doesn't match")
	}
	if !deepEqual(moduleDiff.NewValue, module1State2) {
		t.Error("New value doesn't match")
	}
}

// TestDiffSessions_NilStates tests that nil states return errors
func TestDiffSessions_NilStates(t *testing.T) {
	ctx := context.Background()
	validState := map[string]any{
		"module": map[string]any{
			"key": "value",
		},
	}

	// Test nil state1
	_, err := DiffSessions(ctx, nil, validState)
	if err == nil {
		t.Fatal("Expected error for nil state1, got nil")
	}

	// Test nil state2
	_, err = DiffSessions(ctx, validState, nil)
	if err == nil {
		t.Fatal("Expected error for nil state2, got nil")
	}
}

// TestDiffSessions_EmptyStates tests diff with empty states
func TestDiffSessions_EmptyStates(t *testing.T) {
	ctx := context.Background()

	// Empty to populated
	emptyState := map[string]any{}
	populatedState := map[string]any{
		"module1": map[string]any{
			"key": "value",
		},
	}

	diff, err := DiffSessions(ctx, emptyState, populatedState)
	if err != nil {
		t.Fatalf("Failed to diff empty to populated: %v", err)
	}

	if len(diff.Added) != 1 {
		t.Errorf("Expected 1 added field, got %d", len(diff.Added))
	}

	// Populated to empty
	diff, err = DiffSessions(ctx, populatedState, emptyState)
	if err != nil {
		t.Fatalf("Failed to diff populated to empty: %v", err)
	}

	if len(diff.Removed) != 1 {
		t.Errorf("Expected 1 removed field, got %d", len(diff.Removed))
	}
}

// TestDiffSessions_NestedMapChanges tests detecting changes in nested maps
func TestDiffSessions_NestedMapChanges(t *testing.T) {
	state1 := map[string]any{
		"module1": map[string]any{
			"nested": map[string]any{
				"level1": map[string]any{
					"level2": "value",
				},
			},
		},
	}

	state2 := map[string]any{
		"module1": map[string]any{
			"nested": map[string]any{
				"level1": map[string]any{
					"level2": "changed value",
				},
			},
		},
	}

	ctx := context.Background()
	diff, err := DiffSessions(ctx, state1, state2)
	if err != nil {
		t.Fatalf("Failed to diff nested states: %v", err)
	}

	if len(diff.Modified) != 1 {
		t.Errorf("Expected 1 modified field, got %d", len(diff.Modified))
	}
}

// TestDiffSessions_ArrayChanges tests detecting changes in arrays
func TestDiffSessions_ArrayChanges(t *testing.T) {
	state1 := map[string]any{
		"module1": map[string]any{
			"array": []any{1, 2, 3},
		},
	}

	state2 := map[string]any{
		"module1": map[string]any{
			"array": []any{1, 2, 4}, // Changed last element
		},
	}

	ctx := context.Background()
	diff, err := DiffSessions(ctx, state1, state2)
	if err != nil {
		t.Fatalf("Failed to diff array states: %v", err)
	}

	if len(diff.Modified) != 1 {
		t.Errorf("Expected 1 modified field, got %d", len(diff.Modified))
	}
}

// TestDiffSessions_NilValueChanges tests detecting changes to/from nil
func TestDiffSessions_NilValueChanges(t *testing.T) {
	state1 := map[string]any{
		"module1": map[string]any{
			"nullable": "value",
		},
	}

	state2 := map[string]any{
		"module1": map[string]any{
			"nullable": nil,
		},
	}

	ctx := context.Background()
	diff, err := DiffSessions(ctx, state1, state2)
	if err != nil {
		t.Fatalf("Failed to diff nil value states: %v", err)
	}

	if len(diff.Modified) != 1 {
		t.Errorf("Expected 1 modified field, got %d", len(diff.Modified))
	}

	fieldDiff := diff.Modified["module1"]
	module2State := fieldDiff.NewValue.(map[string]any)
	if module2State["nullable"] != nil {
		t.Error("Expected new value to have nullable field set to nil")
	}
}

// TestDeepEqual_Primitives tests deep equality for primitive types
func TestDeepEqual_Primitives(t *testing.T) {
	tests := []struct {
		name string
		a, b any
		want bool
	}{
		{"strings equal", "hello", "hello", true},
		{"strings different", "hello", "world", false},
		{"ints equal", 42, 42, true},
		{"ints different", 42, 43, false},
		{"floats equal", 3.14, 3.14, true},
		{"floats different", 3.14, 2.71, false},
		{"bools equal", true, true, true},
		{"bools different", true, false, false},
		{"nil and nil", nil, nil, true},
		{"nil and value", nil, "value", false},
		{"value and nil", "value", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deepEqual(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("deepEqual(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// TestDeepEqual_Maps tests deep equality for maps
func TestDeepEqual_Maps(t *testing.T) {
	map1 := map[string]any{
		"key1": "value1",
		"key2": 42,
		"nested": map[string]any{
			"deep": "value",
		},
	}

	map2 := map[string]any{
		"key1": "value1",
		"key2": 42,
		"nested": map[string]any{
			"deep": "value",
		},
	}

	map3 := map[string]any{
		"key1": "value1",
		"key2": 43, // Different value
		"nested": map[string]any{
			"deep": "value",
		},
	}

	if !deepEqual(map1, map2) {
		t.Error("Expected equal maps to be equal")
	}

	if deepEqual(map1, map3) {
		t.Error("Expected different maps to be different")
	}
}

// TestDeepEqual_Slices tests deep equality for slices
func TestDeepEqual_Slices(t *testing.T) {
	slice1 := []any{1, 2, 3, "four"}
	slice2 := []any{1, 2, 3, "four"}
	slice3 := []any{1, 2, 3, "five"}
	slice4 := []any{1, 2, 3} // Different length

	if !deepEqual(slice1, slice2) {
		t.Error("Expected equal slices to be equal")
	}

	if deepEqual(slice1, slice3) {
		t.Error("Expected different slices to be different")
	}

	if deepEqual(slice1, slice4) {
		t.Error("Expected slices of different lengths to be different")
	}
}

// TestDeepEqual_TypeCoercion tests numeric type coercion
func TestDeepEqual_TypeCoercion(t *testing.T) {
	tests := []struct {
		name string
		a, b any
		want bool
	}{
		{"int and float64 equal", 42, 42.0, true},
		{"float64 and int equal", 42.0, 42, true},
		{"int32 and float64 equal", int32(42), 42.0, true},
		{"int64 and float64 equal", int64(42), 42.0, true},
		{"uint and float64 equal", uint(42), 42.0, true},
		{"int and float64 different", 42, 43.0, false},
		{"string and number", "42", 42, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deepEqual(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("deepEqual(%v (%T), %v (%T)) = %v, want %v",
					tt.a, tt.a, tt.b, tt.b, got, tt.want)
			}
		})
	}
}

// TestDiffSessions_ComplexScenario tests a complex diff scenario
func TestDiffSessions_ComplexScenario(t *testing.T) {
	state1 := map[string]any{
		"module1": map[string]any{
			"unchanged": "same",
			"modified":  "old",
			"removed":   "will be removed",
		},
		"module2": map[string]any{
			"nested": map[string]any{
				"value": 10,
			},
		},
		"module3": map[string]any{
			"delete": "this module",
		},
	}

	state2 := map[string]any{
		"module1": map[string]any{
			"unchanged": "same",
			"modified":  "new",
			"added":     "new field",
		},
		"module2": map[string]any{
			"nested": map[string]any{
				"value": 20, // Changed
			},
		},
		"module4": map[string]any{ // New module
			"new": "module",
		},
	}

	ctx := context.Background()
	diff, err := DiffSessions(ctx, state1, state2)
	if err != nil {
		t.Fatalf("Failed to diff complex states: %v", err)
	}

	// Verify added (module4)
	if len(diff.Added) != 1 {
		t.Errorf("Expected 1 added module, got %d", len(diff.Added))
	}

	// Verify removed (module3)
	if len(diff.Removed) != 1 {
		t.Errorf("Expected 1 removed module, got %d", len(diff.Removed))
	}

	// Verify modified (module1 and module2 - both changed internally)
	if len(diff.Modified) != 2 {
		t.Errorf("Expected 2 modified modules, got %d", len(diff.Modified))
	}

	// Verify specific modules
	if _, ok := diff.Added["module4"]; !ok {
		t.Error("Expected module4 to be added")
	}

	if _, ok := diff.Removed["module3"]; !ok {
		t.Error("Expected module3 to be removed")
	}

	if _, ok := diff.Modified["module1"]; !ok {
		t.Error("Expected module1 to be modified")
	}

	if _, ok := diff.Modified["module2"]; !ok {
		t.Error("Expected module2 to be modified")
	}
}

// TestValidateValue_AllTypes tests validateValue with all supported types
func TestValidateValue_AllTypes(t *testing.T) {
	tests := []struct {
		name      string
		value     any
		wantError bool
	}{
		{"string", "test", false},
		{"int", 42, false},
		{"float", 3.14, false},
		{"bool", true, false},
		{"nil", nil, false},
		{"map", map[string]any{"key": "value"}, false},
		{"slice", []any{1, 2, 3}, false},
		{"array", [3]int{1, 2, 3}, false},
		{"struct", time.Now(), false},
		{"ptr", new(string), false},
		{"interface", "test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateValue("test_key", tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("validateValue() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// TestValidateModuleState_NonMapState tests module state that's not a map
func TestValidateModuleState_NonMapState(t *testing.T) {
	// Some modules might use custom types that aren't maps
	tests := []struct {
		name      string
		state     any
		wantError bool
	}{
		{"string state", "custom string state", false},
		{"int state", 42, false},
		{"bool state", true, false},
		{"nil state", nil, true}, // nil is not allowed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateModuleState("test_module", tt.state)
			if (err != nil) != tt.wantError {
				t.Errorf("validateModuleState() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// TestDiffSessions_FieldDiffStructure tests FieldDiff structure
func TestDiffSessions_FieldDiffStructure(t *testing.T) {
	state1 := map[string]any{
		"module": map[string]any{
			"key": "old_value",
		},
	}

	state2 := map[string]any{
		"module": map[string]any{
			"key": "new_value",
		},
	}

	ctx := context.Background()
	diff, err := DiffSessions(ctx, state1, state2)
	if err != nil {
		t.Fatalf("Failed to diff: %v", err)
	}

	if len(diff.Modified) != 1 {
		t.Fatalf("Expected 1 modified field, got %d", len(diff.Modified))
	}

	fieldDiff := diff.Modified["module"]

	// Verify FieldDiff has both OldValue and NewValue
	if fieldDiff.OldValue == nil {
		t.Error("Expected OldValue to be set")
	}

	if fieldDiff.NewValue == nil {
		t.Error("Expected NewValue to be set")
	}

	// Verify the values are correct
	oldMap := fieldDiff.OldValue.(map[string]any)
	newMap := fieldDiff.NewValue.(map[string]any)

	if oldMap["key"] != "old_value" {
		t.Errorf("Expected OldValue key to be 'old_value', got %v", oldMap["key"])
	}

	if newMap["key"] != "new_value" {
		t.Errorf("Expected NewValue key to be 'new_value', got %v", newMap["key"])
	}
}

// TestToFloat64 tests toFloat64 conversion function
func TestToFloat64(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		want     float64
		wantBool bool
	}{
		{"float64", 3.14, 3.14, true},
		{"float32", float32(3.0), 3.0, true}, // Use exact value to avoid precision issues
		{"int", 42, 42.0, true},
		{"int64", int64(42), 42.0, true},
		{"int32", int32(42), 42.0, true},
		{"int16", int16(42), 42.0, true},
		{"int8", int8(42), 42.0, true},
		{"uint", uint(42), 42.0, true},
		{"uint64", uint64(42), 42.0, true},
		{"uint32", uint32(42), 42.0, true},
		{"uint16", uint16(42), 42.0, true},
		{"uint8", uint8(42), 42.0, true},
		{"string", "42", 0, false},
		{"bool", true, 0, false},
		{"nil", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := toFloat64(tt.input)
			if ok != tt.wantBool {
				t.Errorf("toFloat64() ok = %v, wantBool %v", ok, tt.wantBool)
				return
			}
			if ok && got != tt.want {
				t.Errorf("toFloat64() = %v, want %v", got, tt.want)
			}
		})
	}
}
