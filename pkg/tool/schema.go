package tool

import (
	"reflect"
	"strings"
)

// StructToSchema converts a struct to JSON Schema using reflection.
// It reads `json` and `description` tags from struct fields.
func StructToSchema(v any) map[string]any {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}

	properties := make(map[string]any)
	required := []string{}

	for i := 0; i < val.NumField(); i++ {
		field := val.Type().Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		fieldValue := val.Field(i)
		jsonTag := field.Tag.Get("json")

		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Parse json tag
		name := field.Name
		if jsonTag != "" {
			// Check if jsonTag has any special directives (comma)
			idxComma := strings.Index(jsonTag, ",")
			if idxComma == -1 {
				// No comma, entire tag is the name
				name = jsonTag
			} else {
				// Has comma, extract name before comma
				name = jsonTag[:idxComma]
			}
		}

		// Check if required: explicit `required` tag takes priority, then fall back to omitempty inference
		isRequired := false
		if reqTag := field.Tag.Get("required"); reqTag != "" {
			isRequired = reqTag == "true"
		} else {
			isRequired = !strings.Contains(jsonTag, "omitempty")
		}

		prop := make(map[string]any)

		// Set type based on field kind
		switch fieldValue.Kind() {
		case reflect.String:
			prop["type"] = "string"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			prop["type"] = "integer"
		case reflect.Float32, reflect.Float64:
			prop["type"] = "number"
		case reflect.Bool:
			prop["type"] = "boolean"
		case reflect.Slice, reflect.Array:
			prop["type"] = "array"
		case reflect.Map:
			prop["type"] = "object"
		default:
			prop["type"] = "string"
		}

		// Add description if available
		if desc := getFieldTag(field, "description"); desc != "" {
			prop["description"] = desc
		}

		properties[name] = prop

		if isRequired {
			required = append(required, name)
		}
	}

	return map[string]any{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
}

// getFieldTag gets a custom tag from a struct field
func getFieldTag(field reflect.StructField, tagKey string) string {
	return field.Tag.Get(tagKey)
}

// ToSnakeCase converts PascalCase to snake_case
func ToSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}
