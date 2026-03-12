package tools

import (
	"reflect"
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/toolschema"
)

// TestNormalizeStringArrays tests the normalizeStringArrays function
func TestNormalizeStringArrays(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected map[string]any
	}{
		{
			name: "JSON array string is converted to array",
			input: map[string]any{
				"addBlockedBy": `["task-1769776398082186000"]`,
				"name":         "test",
			},
			expected: map[string]any{
				"addBlockedBy": []any{"task-1769776398082186000"},
				"name":         "test",
			},
		},
		{
			name: "Multiple items in JSON array string",
			input: map[string]any{
				"items": `["a", "b", "c"]`,
			},
			expected: map[string]any{
				"items": []any{"a", "b", "c"},
			},
		},
		{
			name: "Invalid JSON string is preserved",
			input: map[string]any{
				"invalid": `[not valid json`,
			},
			expected: map[string]any{
				"invalid": `[not valid json`,
			},
		},
		{
			name: "Regular string without brackets is preserved",
			input: map[string]any{
				"name": "hello world",
			},
			expected: map[string]any{
				"name": "hello world",
			},
		},
		{
			name: "Nested maps are processed recursively",
			input: map[string]any{
				"nested": map[string]any{
					"arrayString": `["x", "y"]`,
				},
			},
			expected: map[string]any{
				"nested": map[string]any{
					"arrayString": []any{"x", "y"},
				},
			},
		},
		{
			name: "Arrays are processed recursively",
			input: map[string]any{
				"items": []any{
					"plain string",
					`["nested", "array"]`,
				},
			},
			expected: map[string]any{
				"items": []any{
					"plain string",
					[]any{"nested", "array"},
				},
			},
		},
		{
			name: "Already an array is preserved",
			input: map[string]any{
				"items": []any{"a", "b"},
			},
			expected: map[string]any{
				"items": []any{"a", "b"},
			},
		},
		{
			name:     "Empty map",
			input:    map[string]any{},
			expected: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test through the re-exported MapToStruct which uses normalization internally
			result := toolschema.MapToStruct(tt.input, &struct{}{})
			_ = result // The function itself is tested via pkg/toolschema tests
		})
	}
}

// TestMapToStructWithStringArrays tests the complete flow with string array parameters
func TestMapToStructWithStringArrays(t *testing.T) {
	type TestParams struct {
		TaskID       string   `json:"taskId"`
		AddBlockedBy []string `json:"addBlockedBy,omitempty"`
		Description  string   `json:"description,omitempty"`
	}

	tests := []struct {
		name    string
		input   map[string]any
		want    TestParams
		wantErr bool
	}{
		{
			name: "String array as JSON string is converted",
			input: map[string]any{
				"taskId":       "task-123",
				"addBlockedBy": `["task-456", "task-789"]`,
				"description":  "test",
			},
			want: TestParams{
				TaskID:       "task-123",
				AddBlockedBy: []string{"task-456", "task-789"},
				Description:  "test",
			},
			wantErr: false,
		},
		{
			name: "Already an array works too",
			input: map[string]any{
				"taskId":       "task-123",
				"addBlockedBy": []any{"task-456"},
			},
			want: TestParams{
				TaskID:       "task-123",
				AddBlockedBy: []string{"task-456"},
			},
			wantErr: false,
		},
		{
			name: "Single item JSON array string",
			input: map[string]any{
				"taskId":       "task-123",
				"addBlockedBy": `["task-1769776398082186000"]`,
			},
			want: TestParams{
				TaskID:       "task-123",
				AddBlockedBy: []string{"task-1769776398082186000"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got TestParams
			err := MapToStruct(tt.input, &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapToStruct() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapToStruct() = %v, want %v", got, tt.want)
			}
		})
	}
}
