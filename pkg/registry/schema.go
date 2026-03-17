package registry

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/tingly-dev/tingly-agentscope/pkg/model"
)

// schemaCache holds generated schemas with their metadata.
type schemaCache struct {
	mu      sync.RWMutex
	schemas map[reflect.Type]*model.ToolDefinition
}

// globalSchemaCache is the default schema cache.
var globalSchemaCache = &schemaCache{
	schemas: make(map[reflect.Type]*model.ToolDefinition),
}

// GenerateSchema generates a JSON Schema for the given type.
// The result is cached for subsequent calls.
func GenerateSchema(typ reflect.Type, name, description string) *model.ToolDefinition {
	return globalSchemaCache.Generate(typ, name, description)
}

// Generate generates and caches a tool definition schema.
func (c *schemaCache) Generate(typ reflect.Type, name, description string) *model.ToolDefinition {
	// Fast path: read lock
	c.mu.RLock()
	if schema, ok := c.schemas[typ]; ok {
		c.mu.RUnlock()
		return schema
	}
	c.mu.RUnlock()

	// Slow path: write lock
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if schema, ok := c.schemas[typ]; ok {
		return schema
	}

	schema := &model.ToolDefinition{
		Type: "function",
		Function: model.FunctionDefinition{
			Name:        name,
			Description: description,
			Parameters:  buildParameters(typ),
		},
	}

	c.schemas[typ] = schema
	return schema
}

// GetCached returns a cached schema if available, without generating.
func (c *schemaCache) GetCached(typ reflect.Type) (*model.ToolDefinition, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	schema, ok := c.schemas[typ]
	return schema, ok
}

// Clear removes all cached schemas.
func (c *schemaCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.schemas = make(map[reflect.Type]*model.ToolDefinition)
}

// buildParameters builds the parameters object for a struct type.
func buildParameters(typ reflect.Type) map[string]any {
	properties := make(map[string]any)
	required := []string{}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" || jsonTag == "" {
			continue
		}

		// Parse json tag to get field name
		name := strings.Split(jsonTag, ",")[0]
		if name == "" {
			name = field.Name
		}

		// Build field schema from tags and type
		schemaTag := field.Tag.Get("jsonschema")
		properties[name] = buildFieldSchema(field, schemaTag)

		// Check if field is required
		if strings.Contains(schemaTag, "required") {
			required = append(required, name)
		} else {
			// Check if omitempty is NOT present in json tag
			if !strings.Contains(jsonTag, "omitempty") {
				required = append(required, name)
			}
		}
	}

	return map[string]any{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
}

// buildFieldSchema builds the schema for a single field.
func buildFieldSchema(field reflect.StructField, schemaTag string) map[string]any {
	schema := make(map[string]any)

	// Parse jsonschema tag: "required,description=Some text,default=10"
	for _, part := range strings.Split(schemaTag, ",") {
		if kv := strings.SplitN(part, "=", 2); len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			val := strings.TrimSpace(kv[1])
			schema[key] = val
		} else if part := strings.TrimSpace(part); part != "required" && part != "" {
			// Boolean flags
			schema[part] = true
		}
	}

	// Auto-detect type if not specified
	if _, ok := schema["type"]; !ok {
		schema["type"] = jsonSchemaType(field.Type)
	}

	// Add enum if specified in tag
	if enum := field.Tag.Get("enum"); enum != "" {
		enums := []any{}
		for _, e := range strings.Split(enum, ",") {
			enums = append(enums, parseEnumValue(e, field.Type))
		}
		schema["enum"] = enums
	}

	// Handle array items
	if schema["type"] == "array" {
		if _, ok := schema["items"]; !ok {
			schema["items"] = map[string]any{
				"type": jsonSchemaType(field.Type.Elem()),
			}
		}
	}

	return schema
}

// jsonSchemaType returns the JSON Schema type for a Go type.
func jsonSchemaType(typ reflect.Type) string {
	kind := typ.Kind()

	// Handle pointer types
	if kind == reflect.Ptr {
		return jsonSchemaType(typ.Elem())
	}

	switch kind {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "integer"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map:
		return "object"
	case reflect.Struct:
		// Check for special types
		if typ == reflect.TypeOf([]any{}).Elem() {
			return "object"
		}
		return "object"
	case reflect.Interface:
		return "object"
	default:
		return "string"
	}
}

// parseEnumValue parses an enum value string to the correct type.
func parseEnumValue(s string, typ reflect.Type) any {
	kind := typ.Kind()

	// Handle pointer types
	if kind == reflect.Ptr {
		kind = typ.Elem().Kind()
	}

	switch kind {
	case reflect.String:
		return s
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var i int64
		fmt.Sscanf(s, "%d", &i)
		return int(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var u uint64
		fmt.Sscanf(s, "%d", &u)
		return uint(u)
	case reflect.Float32, reflect.Float64:
		var f float64
		fmt.Sscanf(s, "%f", &f)
		return f
	case reflect.Bool:
		return strings.ToLower(s) == "true"
	default:
		return s
	}
}

// ClearSchemaCache clears all cached schemas.
func ClearSchemaCache() {
	globalSchemaCache.Clear()
}

// GetCachedSchema returns a cached schema if available.
func GetCachedSchema(typ reflect.Type) (*model.ToolDefinition, bool) {
	return globalSchemaCache.GetCached(typ)
}
