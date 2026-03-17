package registry

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types
type TestArgs struct {
	Name     string  `json:"name" jsonschema:"required,description=The name"`
	Age      int     `json:"age" jsonschema:"description=The age"`
	Active   bool    `json:"active" jsonschema:"description=Whether active"`
	Score    float64 `json:"score" jsonschema:"description=The score"`
	Optional string  `json:"optional,omitempty"`
}

type NestedArgs struct {
	Name  string  `json:"name" jsonschema:"required"`
	Value float64 `json:"value" jsonschema:"required"`
}

type ComplexArgs struct {
	Name    string            `json:"name" jsonschema:"required"`
	Nested  NestedArgs        `json:"nested" jsonschema:"required"`
	Items   []string          `json:"items" jsonschema:"description=List of items"`
	Numbers []int             `json:"numbers,omitempty"`
	MapData map[string]string `json:"map_data,omitempty"`
}

func TestBuildCachedDecoder(t *testing.T) {
	typ := reflect.TypeOf(TestArgs{})
	decoder := buildCachedDecoder(typ)

	assert.NotNil(t, decoder)
	assert.Equal(t, typ, decoder.targetType)
	assert.True(t, decoder.initialized)
	assert.Len(t, decoder.fields, 5) // Name, Age, Active, Score, Optional
}

func TestDecoderDecode(t *testing.T) {
	typ := reflect.TypeOf(TestArgs{})
	decoder := buildCachedDecoder(typ)

	tests := []struct {
		name    string
		input   map[string]any
		want    *TestArgs
		wantErr bool
		errMsg  string
	}{
		{
			name: "all fields",
			input: map[string]any{
				"name":     "Alice",
				"age":      float64(30),
				"active":   true,
				"score":    95.5,
				"optional": "present",
			},
			want: &TestArgs{
				Name:     "Alice",
				Age:      30,
				Active:   true,
				Score:    95.5,
				Optional: "present",
			},
		},
		{
			name: "required fields only",
			input: map[string]any{
				"name":   "Bob",
				"age":    float64(25),
				"active": false,
				"score":  87.3,
			},
			want: &TestArgs{
				Name:   "Bob",
				Age:    25,
				Active: false,
				Score:  87.3,
			},
		},
		{
			name: "missing required field",
			input: map[string]any{
				"age": float64(30),
			},
			wantErr: true,
			errMsg:  "missing required field",
		},
		{
			name: "wrong type for string",
			input: map[string]any{
				"name":   123,
				"age":    float64(30),
				"active": true,
				"score":  95.5,
			},
			wantErr: true,
			errMsg:  "expected string",
		},
		{
			name: "int as float64 (JSON default)",
			input: map[string]any{
				"name":   "Charlie",
				"age":    float64(42),
				"active": false,
				"score":  88.8,
			},
			want: &TestArgs{
				Name:   "Charlie",
				Age:    42,
				Active: false,
				Score:  88.8,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decoder.Decode(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			require.NoError(t, err)
			assert.IsType(t, &TestArgs{}, result)
			got := result.(*TestArgs)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRegistryRegister(t *testing.T) {
	reg := &Registry{decoders: make(map[reflect.Type]*cachedDecoder)}
	typ := reflect.TypeOf(TestArgs{})

	// First registration
	decoder, err := reg.Register(typ)
	require.NoError(t, err)
	assert.NotNil(t, decoder)

	// Second registration should return cached decoder
	decoder2, err := reg.Register(typ)
	require.NoError(t, err)
	assert.Same(t, decoder, decoder2)

	// Verify decoder is in cache
	cached := reg.GetDecoder(typ)
	assert.Same(t, decoder, cached)
}

func TestRegistryDecode(t *testing.T) {
	reg := &Registry{decoders: make(map[reflect.Type]*cachedDecoder)}
	typ := reflect.TypeOf(TestArgs{})

	input := map[string]any{
		"name":   "Test",
		"age":    float64(20),
		"active": true,
		"score":  100.0,
	}

	result, err := reg.Decode(typ, input)
	require.NoError(t, err)

	got, ok := result.(*TestArgs)
	require.True(t, ok)
	assert.Equal(t, "Test", got.Name)
	assert.Equal(t, 20, got.Age)
	assert.True(t, got.Active)
	assert.Equal(t, 100.0, got.Score)
}

func TestGlobalRegistry(t *testing.T) {
	// Use global registry
	ClearAll()
	typ := reflect.TypeOf(TestArgs{})

	decoder, err := Register(typ)
	require.NoError(t, err)
	assert.NotNil(t, decoder)

	// Verify it's cached
	cached := GetDecoder(typ)
	assert.Same(t, decoder, cached)
}

func TestDecodeInto(t *testing.T) {
	ClearAll()

	input := map[string]any{
		"name":   "Jane",
		"age":    float64(28),
		"active": true,
		"score":  92.5,
	}

	result, err := DecodeInto[TestArgs](Default(), input)
	require.NoError(t, err)

	assert.Equal(t, "Jane", result.Name)
	assert.Equal(t, 28, result.Age)
	assert.True(t, result.Active)
	assert.Equal(t, 92.5, result.Score)
}

func TestComplexTypes(t *testing.T) {
	typ := reflect.TypeOf(ComplexArgs{})
	decoder := buildCachedDecoder(typ)

	input := map[string]any{
		"name": "test",
		"nested": map[string]any{
			"name":  "nested",
			"value": float64(123.45),
		},
		"items":   []any{"a", "b", "c"},
		"numbers": []any{float64(1), float64(2), float64(3)},
		"map_data": map[string]any{
			"key1": "value1",
			"key2": "value2",
		},
	}

	result, err := decoder.Decode(input)
	require.NoError(t, err)

	got, ok := result.(*ComplexArgs)
	require.True(t, ok)

	assert.Equal(t, "test", got.Name)
	assert.Equal(t, "nested", got.Nested.Name)
	assert.Equal(t, 123.45, got.Nested.Value)
	assert.Equal(t, []string{"a", "b", "c"}, got.Items)
	assert.Equal(t, []int{1, 2, 3}, got.Numbers)
	assert.Equal(t, map[string]string{"key1": "value1", "key2": "value2"}, got.MapData)
}

func TestDecodeError(t *testing.T) {
	typ := reflect.TypeOf(TestArgs{})
	decoder := buildCachedDecoder(typ)

	input := map[string]any{
		"age": float64(30),
	}

	_, err := decoder.Decode(input)
	require.Error(t, err)

	var decodeErr *DecodeError
	require.ErrorAs(t, err, &decodeErr)
	assert.Equal(t, "name", decodeErr.Field)
	assert.Equal(t, "missing required field", decodeErr.Message)
}

func TestNonStructType(t *testing.T) {
	reg := &Registry{decoders: make(map[reflect.Type]*cachedDecoder)}
	typ := reflect.TypeOf("string") // Not a struct

	_, err := reg.Register(typ)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected struct type")
}

func TestRegistryClear(t *testing.T) {
	reg := &Registry{decoders: make(map[reflect.Type]*cachedDecoder)}
	typ := reflect.TypeOf(TestArgs{})

	reg.MustRegister(typ)
	assert.Equal(t, 1, reg.Size())

	reg.Clear()
	assert.Equal(t, 0, reg.Size())
	assert.Nil(t, reg.GetDecoder(typ))
}

func TestNilInput(t *testing.T) {
	typ := reflect.TypeOf(TestArgs{})
	decoder := buildCachedDecoder(typ)

	input := map[string]any{
		"name":     "NilTest",
		"age":      float64(30),
		"active":   true,
		"score":    90.0,
		"optional": nil,
	}

	result, err := decoder.Decode(input)
	require.NoError(t, err)

	got := result.(*TestArgs)
	assert.Equal(t, "NilTest", got.Name)
	// nil for optional string should be empty string
	assert.Equal(t, "", got.Optional)
}

func TestStringNumberParsing(t *testing.T) {
	typ := reflect.TypeOf(TestArgs{})
	decoder := buildCachedDecoder(typ)

	// Test parsing numbers from strings
	input := map[string]any{
		"name":   "StringTest",
		"age":    "25",   // String that should parse as int
		"active": "true", // String that should parse as bool (will fail, expected bool)
		"score":  "95.5",
	}

	_, err := decoder.Decode(input)
	assert.Error(t, err) // active field should fail
}
