package tool

import (
	"encoding/json"
	"fmt"
	"strings"
)

// MapToStruct converts a map to a struct using JSON unmarshaling.
// It normalizes string representations of booleans, numbers, and arrays.
func MapToStruct(m map[string]any, target interface{}) error {
	// Normalize string booleans to actual booleans
	normalized := normalizeBoolStrings(m)

	// Normalize string arrays to actual arrays
	normalized = normalizeStringArrays(normalized)

	data, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

// normalizeBoolStrings converts string representations of booleans and numbers to actual types
func normalizeBoolStrings(m map[string]any) map[string]any {
	result := make(map[string]any)
	for k, v := range m {
		// Convert string "true"/"false" to bool
		if str, ok := v.(string); ok {
			switch strings.ToLower(str) {
			case "true":
				result[k] = true
			case "false":
				result[k] = false
			default:
				// Try to convert to integer
				if intVal, err := parseInt(str); err == nil {
					result[k] = intVal
				} else if floatVal, err := parseFloat(str); err == nil {
					result[k] = floatVal
				} else {
					result[k] = v
				}
			}
		} else if nestedMap, ok := v.(map[string]any); ok {
			result[k] = normalizeBoolStrings(nestedMap)
		} else if slice, ok := v.([]any); ok {
			result[k] = normalizeSlice(slice)
		} else {
			result[k] = v
		}
	}
	return result
}

// normalizeSlice recursively normalizes elements in a slice
func normalizeSlice(slice []any) []any {
	result := make([]any, len(slice))
	for i, v := range slice {
		if str, ok := v.(string); ok {
			switch strings.ToLower(str) {
			case "true":
				result[i] = true
			case "false":
				result[i] = false
			default:
				if intVal, err := parseInt(str); err == nil {
					result[i] = intVal
				} else if floatVal, err := parseFloat(str); err == nil {
					result[i] = floatVal
				} else {
					result[i] = v
				}
			}
		} else if nestedMap, ok := v.(map[string]any); ok {
			result[i] = normalizeBoolStrings(nestedMap)
		} else if nestedSlice, ok := v.([]any); ok {
			result[i] = normalizeSlice(nestedSlice)
		} else {
			result[i] = v
		}
	}
	return result
}

// normalizeStringArrays converts JSON array strings to actual arrays
// This handles cases where LLM sends "[\"a\",\"b\"]" as a string instead of an array
func normalizeStringArrays(m map[string]any) map[string]any {
	result := make(map[string]any, len(m))

	for k, v := range m {
		switch val := v.(type) {
		case string:
			// Check if it looks like a JSON array
			if len(val) > 1 && val[0] == '[' && val[len(val)-1] == ']' {
				var arr []any
				if err := json.Unmarshal([]byte(val), &arr); err == nil {
					result[k] = arr
				} else {
					result[k] = v
				}
			} else {
				result[k] = v
			}
		default:
			// Recursively handle nested maps
			if nestedMap, ok := v.(map[string]any); ok {
				result[k] = normalizeStringArrays(nestedMap)
			} else if slice, ok := v.([]any); ok {
				result[k] = normalizeSliceInSlice(slice)
			} else {
				result[k] = v
			}
		}
	}

	return result
}

// normalizeSliceInSlice recursively normalizes elements in a slice
func normalizeSliceInSlice(slice []any) []any {
	result := make([]any, len(slice))
	for i, v := range slice {
		if str, ok := v.(string); ok {
			// Check if it looks like a JSON array
			if len(str) > 1 && str[0] == '[' && str[len(str)-1] == ']' {
				var arr []any
				if err := json.Unmarshal([]byte(str), &arr); err == nil {
					result[i] = arr
				} else {
					result[i] = v
				}
			} else {
				result[i] = v
			}
		} else if nestedMap, ok := v.(map[string]any); ok {
			result[i] = normalizeStringArrays(nestedMap)
		} else if nestedSlice, ok := v.([]any); ok {
			result[i] = normalizeSliceInSlice(nestedSlice)
		} else {
			result[i] = v
		}
	}
	return result
}

// parseInt attempts to parse a string to an integer
// Returns the parsed int and nil error if successful
func parseInt(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}

	// Check for decimal point or exponent - if present, it's a float
	if strings.Contains(s, ".") || strings.Contains(s, "e") || strings.Contains(s, "E") {
		return 0, fmt.Errorf("float format")
	}

	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	if err != nil {
		return 0, err
	}

	// Verify the entire string was consumed (no trailing chars)
	var verify int
	if n, _ := fmt.Sscanf(s, "%d%c", &verify, new(rune)); n != 1 {
		return 0, fmt.Errorf("trailing characters")
	}

	return result, nil
}

// parseFloat attempts to parse a string to a float64
// Returns the parsed float64 and nil error if successful
func parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}

	var result float64
	_, err := fmt.Sscanf(s, "%f", &result)
	if err != nil {
		return 0, err
	}

	// Verify the entire string was consumed
	var verify float64
	if n, _ := fmt.Sscanf(s, "%f%c", &verify, new(rune)); n != 1 {
		return 0, fmt.Errorf("trailing characters")
	}

	return result, nil
}
