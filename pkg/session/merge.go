package session

import (
	"context"
	"fmt"
	"reflect"
)

// MergeStrategy defines how to merge conflicting states
type MergeStrategy int

const (
	// MergeStrategyOverride means last write wins - later states override earlier ones
	MergeStrategyOverride MergeStrategy = iota

	// MergeStrategyKeepFirst keeps the first value encountered - earlier states take precedence
	MergeStrategyKeepFirst

	// MergeStrategyMerge performs deep merge for maps, combining nested structures
	MergeStrategyMerge

	// MergeStrategyError returns an error when conflicts are detected
	MergeStrategyError
)

// String returns the string representation of the merge strategy
func (ms MergeStrategy) String() string {
	switch ms {
	case MergeStrategyOverride:
		return "override"
	case MergeStrategyKeepFirst:
		return "keep_first"
	case MergeStrategyMerge:
		return "merge"
	case MergeStrategyError:
		return "error"
	default:
		return "unknown"
	}
}

// MergeSessions merges multiple session states according to the specified strategy
// It handles nested structures properly and resolves conflicts based on the strategy
//
// Parameters:
//   - ctx: Context for cancellation (currently not used but kept for consistency)
//   - states: Slice of session state maps to merge
//   - strategy: The merge strategy to use for conflict resolution
//
// Returns:
//   - map[string]any: The merged session state
//   - error: Error if strategy is MergeStrategyError and conflicts are detected
//
// Strategies:
//   - Override: Last write wins - values from later states override earlier ones
//   - KeepFirst: Keep first value - values from earlier states take precedence
//   - Merge: Deep merge for maps - recursively merges nested maps
//   - Error: Returns error on conflict - detects when same key has different values
//
// Example:
//
//	state1 := map[string]any{"module1": {"key1": "value1"}}
//	state2 := map[string]any{"module1": {"key2": "value2"}, "module2": {"key3": "value3"}}
//	merged, _ := MergeSessions(context.Background(), []map[string]any{state1, state2}, MergeStrategyMerge)
//	// merged will be: {"module1": {"key1": "value1", "key2": "value2"}, "module2": {"key3": "value3"}}
func MergeSessions(ctx context.Context, states []map[string]any, strategy MergeStrategy) (map[string]any, error) {
	// Validate inputs
	if len(states) == 0 {
		return make(map[string]any), nil
	}

	if strategy < MergeStrategyOverride || strategy > MergeStrategyError {
		return nil, fmt.Errorf("invalid merge strategy: %d", strategy)
	}

	// Start with an empty result
	result := make(map[string]any)

	// Merge each state according to the strategy
	for i, state := range states {
		if state == nil {
			continue // Skip nil states
		}

		for key, value := range state {
			existingValue, exists := result[key]

			if !exists {
				// Key doesn't exist yet, add it
				result[key] = value
			} else {
				// Key exists, handle conflict based on strategy
				mergedValue, err := mergeValue(key, existingValue, value, strategy, i)
				if err != nil {
					return nil, err
				}
				if mergedValue != nil {
					result[key] = mergedValue
				}
				// If mergedValue is nil and strategy is KeepFirst, we keep the existing value
			}
		}
	}

	return result, nil
}

// mergeValue merges two values according to the strategy
// Returns the merged value, or nil if existing value should be kept (for KeepFirst)
func mergeValue(key string, existing, incoming any, strategy MergeStrategy, stateIndex int) (any, error) {
	switch strategy {
	case MergeStrategyOverride:
		// Last write wins - use incoming value
		return incoming, nil

	case MergeStrategyKeepFirst:
		// Keep first value - return nil to signal keeping existing
		return nil, nil

	case MergeStrategyMerge:
		// Try to deep merge if both are maps
		return mergeDeep(existing, incoming)

	case MergeStrategyError:
		// Check for conflicts
		if !deepEqual(existing, incoming) {
			return nil, fmt.Errorf("conflict detected for key '%s' at state index %d: existing value %#v differs from incoming %#v", key, stateIndex, existing, incoming)
		}
		// No conflict, keep existing
		return existing, nil

	default:
		return nil, fmt.Errorf("unknown merge strategy: %d", strategy)
	}
}

// mergeDeep performs deep merge of two values
// If both values are maps, it recursively merges them
// Otherwise, it returns the incoming value (last write wins)
func mergeDeep(existing, incoming any) (any, error) {
	// Handle nil values
	if incoming == nil {
		return existing, nil
	}
	if existing == nil {
		return incoming, nil
	}

	// Try to treat both as maps
	existingMap, existingIsMap := existing.(map[string]any)
	incomingMap, incomingIsMap := incoming.(map[string]any)

	if existingIsMap && incomingIsMap {
		// Both are maps, deep merge them
		return mergeMaps(existingMap, incomingMap), nil
	}

	// Not both maps, use incoming value (override)
	return incoming, nil
}

// mergeMaps merges two maps recursively
// It combines keys from both maps, with incoming values taking precedence for non-map values
// For nested maps, it recursively merges them
func mergeMaps(existing, incoming map[string]any) map[string]any {
	result := make(map[string]any)

	// Copy all existing values
	for key, value := range existing {
		result[key] = value
	}

	// Merge incoming values
	for key, incomingValue := range incoming {
		existingValue, exists := existing[key]

		if !exists {
			// Key doesn't exist in existing, add it
			result[key] = incomingValue
		} else {
			// Key exists in both, try to deep merge
			mergedValue := mergeMapValues(existingValue, incomingValue)
			result[key] = mergedValue
		}
	}

	return result
}

// mergeMapValues merges two values within a map context
// If both are maps, recursively merge them
// Otherwise, incoming value takes precedence
func mergeMapValues(existing, incoming any) any {
	// Handle nil values
	if incoming == nil {
		return existing
	}
	if existing == nil {
		return incoming
	}

	// Try to treat both as maps
	existingMap, existingIsMap := existing.(map[string]any)
	incomingMap, incomingIsMap := incoming.(map[string]any)

	if existingIsMap && incomingIsMap {
		// Both are maps, recursively merge them
		return mergeMaps(existingMap, incomingMap)
	}

	// Not both maps, incoming takes precedence
	return incoming
}

// MergeSessionsInto merges multiple session states into an existing state
// This is useful for incremental merging without creating a new map
//
// Parameters:
//   - ctx: Context for cancellation
//   - target: The target state to merge into (will be modified)
//   - states: Slice of session state maps to merge into target
//   - strategy: The merge strategy to use for conflict resolution
//
// Returns:
//   - error: Error if strategy is MergeStrategyError and conflicts are detected
//
// The target map is modified in-place and also returned for convenience
func MergeSessionsInto(ctx context.Context, target map[string]any, states []map[string]any, strategy MergeStrategy) error {
	if target == nil {
		return fmt.Errorf("target map cannot be nil")
	}

	if strategy < MergeStrategyOverride || strategy > MergeStrategyError {
		return fmt.Errorf("invalid merge strategy: %d", strategy)
	}

	for i, state := range states {
		if state == nil {
			continue // Skip nil states
		}

		for key, value := range state {
			existingValue, exists := target[key]

			if !exists {
				// Key doesn't exist yet, add it
				target[key] = value
			} else {
				// Key exists, handle conflict based on strategy
				mergedValue, err := mergeValue(key, existingValue, value, strategy, i)
				if err != nil {
					return err
				}
				if mergedValue != nil {
					target[key] = mergedValue
				}
				// If mergedValue is nil and strategy is KeepFirst, we keep the existing value
			}
		}
	}

	return nil
}

// MergeStrategyValidator validates if a merge can be performed without conflicts
// It checks if merging with the given strategy would result in data loss or conflicts
//
// Parameters:
//   - ctx: Context for cancellation
//   - states: Slice of session state maps to validate
//   - strategy: The merge strategy to validate against
//
// Returns:
//   - []string: List of validation warnings (empty if no warnings)
//   - error: Error if validation fails (e.g., strategy is invalid)
//
// This is useful for pre-flight validation before performing a merge
func MergeStrategyValidator(ctx context.Context, states []map[string]any, strategy MergeStrategy) ([]string, error) {
	if strategy < MergeStrategyOverride || strategy > MergeStrategyError {
		return nil, fmt.Errorf("invalid merge strategy: %d", strategy)
	}

	var warnings []string

	if strategy == MergeStrategyError {
		// Check if there would be any conflicts
		for i := 1; i < len(states); i++ {
			if states[i] == nil {
				continue
			}

			for key, value := range states[i] {
				for j := 0; j < i; j++ {
					if states[j] == nil {
						continue
					}

					if existingValue, exists := states[j][key]; exists {
						if !deepEqual(existingValue, value) {
							warnings = append(warnings, fmt.Sprintf("potential conflict at key '%s' between state %d and state %d", key, j, i))
						}
					}
				}
			}
		}
	}

	if strategy == MergeStrategyKeepFirst && len(states) > 1 {
		// Warn that values from later states will be ignored
		warnings = append(warnings, fmt.Sprintf("using KeepFirst strategy with %d states: values from later states will be ignored", len(states)))
	}

	if strategy == MergeStrategyOverride && len(states) > 1 {
		// Warn that values from earlier states may be overridden
		warnings = append(warnings, fmt.Sprintf("using Override strategy with %d states: values from earlier states may be overridden", len(states)))
	}

	return warnings, nil
}

// GetMergeConflicts returns a list of conflicts that would occur if merging with Error strategy
// This is useful for UI feedback or pre-merge validation
//
// Parameters:
//   - ctx: Context for cancellation
//   - states: Slice of session state maps to check for conflicts
//
// Returns:
//   - []MergeConflict: List of conflicts detected
//   - error: Error if states cannot be analyzed
type MergeConflict struct {
	Key         string `json:"key"`
	StateIndex1 int    `json:"state_index_1"`
	StateIndex2 int    `json:"state_index_2"`
	Value1      any    `json:"value_1"`
	Value2      any    `json:"value_2"`
}

func GetMergeConflicts(ctx context.Context, states []map[string]any) ([]MergeConflict, error) {
	var conflicts []MergeConflict

	// Check each pair of states for conflicts
	for i := 1; i < len(states); i++ {
		if states[i] == nil {
			continue
		}

		for key, value := range states[i] {
			for j := 0; j < i; j++ {
				if states[j] == nil {
					continue
				}

				if existingValue, exists := states[j][key]; exists {
					if !deepEqual(existingValue, value) {
						conflicts = append(conflicts, MergeConflict{
							Key:         key,
							StateIndex1: j,
							StateIndex2: i,
							Value1:      existingValue,
							Value2:      value,
						})
					}
				}
			}
		}
	}

	return conflicts, nil
}

// CanMerge checks if a merge can be performed without errors for the given strategy
// This is a convenience method that returns true if the merge would succeed
//
// Parameters:
//   - ctx: Context for cancellation
//   - states: Slice of session state maps to check
//   - strategy: The merge strategy to check
//
// Returns:
//   - bool: True if merge can be performed without errors
//   - error: Error that would occur during merge (nil if merge would succeed)
func CanMerge(ctx context.Context, states []map[string]any, strategy MergeStrategy) (bool, error) {
	if strategy < MergeStrategyOverride || strategy > MergeStrategyError {
		return false, fmt.Errorf("invalid merge strategy: %d", strategy)
	}

	// Only Error strategy can fail
	if strategy != MergeStrategyError {
		return true, nil
	}

	// Check for conflicts
	conflicts, err := GetMergeConflicts(ctx, states)
	if err != nil {
		return false, err
	}

	return len(conflicts) == 0, nil
}

// deepEqualValues compares two values for deep equality
// This is a simplified version that handles the common cases in session states
func deepEqualValues(a, b any) bool {
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
		return false
	}

	switch valA.Kind() {
	case reflect.Map:
		return deepEqualMap(valA, valB)
	case reflect.Slice, reflect.Array:
		return deepEqualSlice(valA, valB)
	default:
		return reflect.DeepEqual(a, b)
	}
}
