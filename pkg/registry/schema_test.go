package registry

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildParameters(t *testing.T) {
	typ := reflect.TypeOf(TestArgs{})
	params := buildParameters(typ)

	assert.Equal(t, "object", params["type"])
	assert.NotNil(t, params["properties"])
	assert.NotNil(t, params["required"])

	props := params["properties"].(map[string]any)
	assert.Contains(t, props, "name")
	assert.Contains(t, props, "age")
	assert.Contains(t, props, "active")
	assert.Contains(t, props, "score")
	assert.Contains(t, props, "optional")

	// Check name field schema
	nameSchema := props["name"].(map[string]any)
	assert.Equal(t, "string", nameSchema["type"])
	assert.Equal(t, "The name", nameSchema["description"])

	// Check age field schema
	ageSchema := props["age"].(map[string]any)
	assert.Equal(t, "integer", ageSchema["type"])
	assert.Equal(t, "The age", ageSchema["description"])

	// Check required fields
	required := params["required"].([]string)
	assert.Contains(t, required, "name")
	assert.Contains(t, required, "age")
	assert.Contains(t, required, "active")
	assert.Contains(t, required, "score")
	assert.NotContains(t, required, "optional") // Has omitempty
}

func TestGenerateSchema(t *testing.T) {
	ClearSchemaCache()

	typ := reflect.TypeOf(TestArgs{})
	schema := GenerateSchema(typ, "test_tool", "A test tool")

	require.NotNil(t, schema)
	assert.Equal(t, "function", schema.Type)
	assert.Equal(t, "test_tool", schema.Function.Name)
	assert.Equal(t, "A test tool", schema.Function.Description)

	params := schema.Function.Parameters
	assert.Equal(t, "object", params["type"])

	// Verify caching
	cached, ok := GetCachedSchema(typ)
	assert.True(t, ok)
	assert.Same(t, schema, cached)
}

func TestJsonSchemaType(t *testing.T) {
	tests := []struct {
		typeStr string
		want    string
	}{
		{"string", "string"},
		{"int", "integer"},
		{"int8", "integer"},
		{"int16", "integer"},
		{"int32", "integer"},
		{"int64", "integer"},
		{"uint", "integer"},
		{"uint8", "integer"},
		{"uint16", "integer"},
		{"uint32", "integer"},
		{"uint64", "integer"},
		{"float32", "number"},
		{"float64", "number"},
		{"bool", "boolean"},
	}

	for _, tt := range tests {
		t.Run(tt.typeStr, func(t *testing.T) {
			typ := reflect.TypeOf(0)
			switch tt.typeStr {
			case "string":
				typ = reflect.TypeOf("")
			case "int":
				typ = reflect.TypeOf(0)
			case "int8":
				typ = reflect.TypeOf(int8(0))
			case "int16":
				typ = reflect.TypeOf(int16(0))
			case "int32":
				typ = reflect.TypeOf(int32(0))
			case "int64":
				typ = reflect.TypeOf(int64(0))
			case "uint":
				typ = reflect.TypeOf(uint(0))
			case "uint8":
				typ = reflect.TypeOf(uint8(0))
			case "uint16":
				typ = reflect.TypeOf(uint16(0))
			case "uint32":
				typ = reflect.TypeOf(uint32(0))
			case "uint64":
				typ = reflect.TypeOf(uint64(0))
			case "float32":
				typ = reflect.TypeOf(float32(0))
			case "float64":
				typ = reflect.TypeOf(float64(0))
			case "bool":
				typ = reflect.TypeOf(false)
			}

			got := jsonSchemaType(typ)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildFieldSchema(t *testing.T) {
	typ := reflect.TypeOf(TestArgs{})
	field, _ := typ.FieldByName("Name")

	schema := buildFieldSchema(field, "required,description=The name")

	assert.Equal(t, "string", schema["type"])
	assert.Equal(t, "The name", schema["description"])

	// Check that required flag is in the tag but not in schema (it's handled separately)
	// The schema function handles required separately
}

func TestSchemaCacheClear(t *testing.T) {
	typ := reflect.TypeOf(TestArgs{})

	// Generate first schema
	schema1 := GenerateSchema(typ, "tool1", "desc1")
	require.NotNil(t, schema1)

	// Clear cache
	ClearSchemaCache()

	// Generate new schema with same type but different metadata
	schema2 := GenerateSchema(typ, "tool2", "desc2")
	require.NotNil(t, schema2)

	// After clear, should generate new schema
	assert.Equal(t, "tool2", schema2.Function.Name)
	assert.Equal(t, "desc2", schema2.Function.Description)
}

func TestArraySchemaType(t *testing.T) {
	type ArrayArgs struct {
		Items []string `json:"items" jsonschema:"description=List of items"`
	}

	typ := reflect.TypeOf(ArrayArgs{})
	params := buildParameters(typ)

	props := params["properties"].(map[string]any)
	itemsSchema := props["items"].(map[string]any)

	assert.Equal(t, "array", itemsSchema["type"])
	assert.Equal(t, "List of items", itemsSchema["description"])

	// Check items type
	itemsItems := itemsSchema["items"].(map[string]any)
	assert.Equal(t, "string", itemsItems["type"])
}

func TestNestedStructSchema(t *testing.T) {
	typ := reflect.TypeOf(ComplexArgs{})
	params := buildParameters(typ)

	props := params["properties"].(map[string]any)
	nestedSchema := props["nested"].(map[string]any)

	assert.Equal(t, "object", nestedSchema["type"])
}

func TestRequiredFieldDetection(t *testing.T) {
	type TestRequired struct {
		Required1 string `json:"required1"`
		Required2 string `json:"required2" jsonschema:"required"`
		Optional1 string `json:"optional1,omitempty"`
		Optional2 string `json:"optional2,omitempty" jsonschema:"required"`
	}

	typ := reflect.TypeOf(TestRequired{})
	params := buildParameters(typ)

	required := params["required"].([]string)

	// required1 and required2 should be required (no omitempty)
	assert.Contains(t, required, "required1")
	assert.Contains(t, required, "required2")
	// optional1 has omitempty, so not required
	assert.NotContains(t, required, "optional1")
	// optional2 has explicit required tag
	assert.Contains(t, required, "optional2")
}

func TestParseEnumValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		typ      reflect.Type
		expected any
	}{
		{
			name:     "string enum",
			input:    "value1",
			typ:      reflect.TypeOf(""),
			expected: "value1",
		},
		{
			name:     "int enum",
			input:    "42",
			typ:      reflect.TypeOf(0),
			expected: 42,
		},
		{
			name:     "float enum",
			input:    "3.14",
			typ:      reflect.TypeOf(0.0),
			expected: 3.14,
		},
		{
			name:     "bool enum",
			input:    "true",
			typ:      reflect.TypeOf(false),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseEnumValue(tt.input, tt.typ)
			assert.Equal(t, tt.expected, got)
		})
	}
}
