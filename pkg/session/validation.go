package session

import (
	"context"
	"fmt"
	"reflect"
)

// SessionDiff represents differences between two sessions
type SessionDiff struct {
	Added    map[string]any       `json:"added"`    // Fields in state2 but not in state1
	Removed  map[string]any       `json:"removed"`  // Fields in state1 but not in state2
	Modified map[string]FieldDiff `json:"modified"` // Fields that changed
}

// FieldDiff represents a single field change
type FieldDiff struct {
	OldValue any `json:"old_value"`
	NewValue any `json:"new_value"`
}

// ValidateState validates the structure of a session state
// It checks that the state is a valid map[string]any and performs basic validation
func ValidateState(state map[string]any) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}

	// Validate each module's state
	for moduleName, moduleState := range state {
		if err := validateModuleState(moduleName, moduleState); err != nil {
			return fmt.Errorf("validation failed for module %s: %w", moduleName, err)
		}
	}

	return nil
}

// validateModuleState validates a single module's state
func validateModuleState(moduleName string, moduleState any) error {
	if moduleState == nil {
		return fmt.Errorf("module state cannot be nil")
	}

	// Module state should be a map[string]any for deep inspection
	stateMap, ok := moduleState.(map[string]any)
	if !ok {
		// If it's not a map, we can't deeply validate it, but it might still be valid
		// Some modules might use custom types
		return nil
	}

	// Check for nested maps and validate recursively
	for key, value := range stateMap {
		if err := validateValue(key, value); err != nil {
			return fmt.Errorf("field %s: %w", key, err)
		}
	}

	return nil
}

// validateValue validates a single value in the state
func validateValue(key string, value any) error {
	// nil values are allowed
	if value == nil {
		return nil
	}

	val := reflect.ValueOf(value)
	switch val.Kind() {
	case reflect.Map:
		// Validate all values in the map
		for _, v := range val.MapKeys() {
			mapValue := val.MapIndex(v)
			if err := validateValue(fmt.Sprintf("%s.%v", key, v.Interface()), mapValue.Interface()); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		// Validate all elements in the slice/array
		for i := 0; i < val.Len(); i++ {
			elem := val.Index(i).Interface()
			if err := validateValue(fmt.Sprintf("%s[%d]", key, i), elem); err != nil {
				return err
			}
		}
	case reflect.Struct:
		// Structs are allowed (e.g., time.Time)
		return nil
	case reflect.Ptr, reflect.Interface:
		// Pointers and interfaces are allowed
		return nil
	case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Bool:
		// Primitive types are allowed
		return nil
	default:
		return fmt.Errorf("unsupported type %T for value", value)
	}

	return nil
}

// DiffSessions compares two session states and returns their differences
// It performs deep comparison for nested structures
func DiffSessions(ctx context.Context, state1, state2 map[string]any) (*SessionDiff, error) {
	// Validate inputs
	if state1 == nil {
		return nil, fmt.Errorf("state1 cannot be nil")
	}
	if state2 == nil {
		return nil, fmt.Errorf("state2 cannot be nil")
	}

	diff := &SessionDiff{
		Added:    make(map[string]any),
		Removed:  make(map[string]any),
		Modified: make(map[string]FieldDiff),
	}

	// Find added and modified fields
	for key, value2 := range state2 {
		value1, exists := state1[key]
		if !exists {
			// Field was added
			diff.Added[key] = value2
		} else {
			// Check if field was modified
			if !deepEqual(value1, value2) {
				diff.Modified[key] = FieldDiff{
					OldValue: value1,
					NewValue: value2,
				}
			}
		}
	}

	// Find removed fields
	for key, value1 := range state1 {
		if _, exists := state2[key]; !exists {
			// Field was removed
			diff.Removed[key] = value1
		}
	}

	return diff, nil
}

// deepEqual performs deep comparison of two values
// It handles maps, slices, and nested structures
func deepEqual(a, b any) bool {
	// Handle nil values
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Use reflection for type checking
	valA := reflect.ValueOf(a)
	valB := reflect.ValueOf(b)

	// Check types match
	if valA.Type() != valB.Type() {
		// Try to handle type coercion for common cases
		return tryTypeCoercionEqual(a, b)
	}

	switch valA.Kind() {
	case reflect.Map:
		return deepEqualMap(valA, valB)
	case reflect.Slice, reflect.Array:
		return deepEqualSlice(valA, valB)
	case reflect.Struct:
		// For structs, use reflect.DeepEqual
		return reflect.DeepEqual(a, b)
	case reflect.Ptr:
		if valA.IsNil() && valB.IsNil() {
			return true
		}
		if valA.IsNil() || valB.IsNil() {
			return false
		}
		return deepEqual(valA.Elem().Interface(), valB.Elem().Interface())
	case reflect.Interface:
		return deepEqual(valA.Elem().Interface(), valB.Elem().Interface())
	default:
		// For primitives and other types, use ==
		return reflect.DeepEqual(a, b)
	}
}

// deepEqualMap compares two map values deeply
func deepEqualMap(a, b reflect.Value) bool {
	if a.Len() != b.Len() {
		return false
	}

	for _, key := range a.MapKeys() {
		valA := a.MapIndex(key)
		valB := b.MapIndex(key)

		if !valB.IsValid() {
			return false
		}

		if !deepEqual(valA.Interface(), valB.Interface()) {
			return false
		}
	}

	return true
}

// deepEqualSlice compares two slice/array values deeply
func deepEqualSlice(a, b reflect.Value) bool {
	if a.Len() != b.Len() {
		return false
	}

	for i := 0; i < a.Len(); i++ {
		if !deepEqual(a.Index(i).Interface(), b.Index(i).Interface()) {
			return false
		}
	}

	return true
}

// tryTypeCoercionEqual handles type coercion for common numeric types
func tryTypeCoercionEqual(a, b any) bool {
	// Handle float vs int comparisons (common in JSON unmarshaling)
	floatA, okA := toFloat64(a)
	floatB, okB := toFloat64(b)

	if okA && okB {
		return floatA == floatB
	}

	return false
}

// toFloat64 attempts to convert a value to float64
func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	case int16:
		return float64(val), true
	case int8:
		return float64(val), true
	case uint:
		return float64(val), true
	case uint64:
		return float64(val), true
	case uint32:
		return float64(val), true
	case uint16:
		return float64(val), true
	case uint8:
		return float64(val), true
	default:
		return 0, false
	}
}
