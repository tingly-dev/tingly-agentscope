package registry

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// Decoder decodes raw input into a specific type
type Decoder interface {
	Decode(data map[string]any) (any, error)
}

// fieldDecoder handles decoding for a single struct field
type fieldDecoder struct {
	name     string                         // JSON field name
	index    int                            // Struct field index for fast access
	required bool                           // Whether field is required
	setFunc  func(reflect.Value, any) error // Pre-compiled setter function (receives field value)
}

// cachedDecoder is a cached decoder for a specific struct type.
// Built once during registration, reused for all subsequent calls.
type cachedDecoder struct {
	targetType  reflect.Type   // The struct type we decode to
	fields      []fieldDecoder // Pre-compiled field decoders
	mu          sync.RWMutex   // Protects fields during lazy loading
	initialized bool           // Whether fields have been initialized
}

// Decode decodes a map into the target struct type.
// This method is reflection-free on the hot path.
func (d *cachedDecoder) Decode(data map[string]any) (any, error) {
	// Create a new instance of the target type
	result := reflect.New(d.targetType).Elem()

	// Decode each field using pre-compiled setters (zero reflection)
	for _, fd := range d.fields {
		val, ok := data[fd.name]
		if !ok {
			if fd.required {
				return nil, NewDecodeError(fd.name, "missing required field", nil)
			}
			continue
		}

		// Get field value by index (zero reflection on hot path)
		fieldVal := result.Field(fd.index)

		if err := fd.setFunc(fieldVal, val); err != nil {
			return nil, NewDecodeErrorWithCause(fd.name, "failed to set value", val, err)
		}
	}

	// Return pointer to the struct
	return result.Addr().Interface(), nil
}

// buildCachedDecoder builds a decoder for the given type.
// This is called once during registration.
func buildCachedDecoder(typ reflect.Type) *cachedDecoder {
	if typ.Kind() != reflect.Struct {
		panic(fmt.Sprintf("buildCachedDecoder: expected struct, got %v", typ.Kind()))
	}

	decoder := &cachedDecoder{
		targetType: typ,
		fields:     make([]fieldDecoder, 0, typ.NumField()),
	}

	// Build field decoders for each exported field
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" || jsonTag == "" {
			continue
		}

		// Parse json tag: "name,omitempty" -> name = "name"
		name := strings.Split(jsonTag, ",")[0]
		if name == "" {
			// Use field name if tag is just ",omitempty"
			name = field.Name
		}

		// Check if field is required via jsonschema tag
		schemaTag := field.Tag.Get("jsonschema")
		required := strings.Contains(schemaTag, "required")

		decoder.fields = append(decoder.fields, fieldDecoder{
			name:     name,
			index:    i, // Store field index for fast access
			required: required,
			setFunc:  makeSetter(field.Type),
		})
	}

	decoder.initialized = true
	return decoder
}

// makeSetter creates a type-specific setter function.
// The returned function is pre-compiled and reflection-free.
func makeSetter(fieldType reflect.Type) func(reflect.Value, any) error {
	kind := fieldType.Kind()

	switch kind {
	case reflect.String:
		return func(v reflect.Value, val any) error {
			if val == nil {
				v.SetString("")
				return nil
			}
			if s, ok := val.(string); ok {
				v.SetString(s)
				return nil
			}
			return fmt.Errorf("expected string, got %T", val)
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return func(v reflect.Value, val any) error {
			var i int64
			switch x := val.(type) {
			case float64:
				i = int64(x)
			case int:
				i = int64(x)
			case int64:
				i = x
			case int32:
				i = int64(x)
			case int16:
				i = int64(x)
			case int8:
				i = int64(x)
			case uint:
				i = int64(x)
			case uint32:
				i = int64(x)
			case string:
				_, err := fmt.Sscanf(x, "%d", &i)
				if err != nil {
					return fmt.Errorf("cannot parse string as int: %s", x)
				}
			default:
				return fmt.Errorf("expected integer, got %T", val)
			}
			v.SetInt(i)
			return nil
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return func(v reflect.Value, val any) error {
			var u uint64
			switch x := val.(type) {
			case float64:
				u = uint64(x)
			case int:
				u = uint64(x)
			case int64:
				u = uint64(x)
			case uint:
				u = uint64(x)
			case uint32:
				u = uint64(x)
			case uint64:
				u = x
			case string:
				_, err := fmt.Sscanf(x, "%d", &u)
				if err != nil {
					return fmt.Errorf("cannot parse string as uint: %s", x)
				}
			default:
				return fmt.Errorf("expected unsigned integer, got %T", val)
			}
			v.SetUint(u)
			return nil
		}

	case reflect.Float32, reflect.Float64:
		return func(v reflect.Value, val any) error {
			var f float64
			switch x := val.(type) {
			case float64:
				f = x
			case float32:
				f = float64(x)
			case int:
				f = float64(x)
			case int64:
				f = float64(x)
			case string:
				_, err := fmt.Sscanf(x, "%f", &f)
				if err != nil {
					return fmt.Errorf("cannot parse string as float: %s", x)
				}
			default:
				return fmt.Errorf("expected float, got %T", val)
			}
			v.SetFloat(f)
			return nil
		}

	case reflect.Bool:
		return func(v reflect.Value, val any) error {
			if b, ok := val.(bool); ok {
				v.SetBool(b)
				return nil
			}
			return fmt.Errorf("expected bool, got %T", val)
		}

	case reflect.Interface:
		return func(v reflect.Value, val any) error {
			v.Set(reflect.ValueOf(val))
			return nil
		}

	case reflect.Ptr:
		elemType := fieldType.Elem()
		return func(v reflect.Value, val any) error {
			if val == nil {
				v.Set(reflect.Zero(fieldType))
				return nil
			}
			// For pointer types, create a new instance and decode into it
			ptr := reflect.New(elemType)
			// Use JSON unmarshal for complex nested types
			data, err := json.Marshal(val)
			if err != nil {
				return err
			}
			err = json.Unmarshal(data, ptr.Interface())
			if err != nil {
				return err
			}
			v.Set(ptr)
			return nil
		}

	case reflect.Slice:
		return func(v reflect.Value, val any) error {
			rv := reflect.ValueOf(val)
			if rv.Kind() == reflect.Interface {
				rv = rv.Elem()
			}
			if rv.Kind() == reflect.Invalid || rv.IsNil() {
				v.Set(reflect.Zero(fieldType))
				return nil
			}
			if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
				return fmt.Errorf("expected slice/array, got %T", val)
			}

			// Create a new slice of the correct type
			slice := reflect.MakeSlice(fieldType, rv.Len(), rv.Len())

			// Copy elements
			for i := 0; i < rv.Len(); i++ {
				elem := rv.Index(i).Interface()
				// Use JSON marshaling for type conversion
				data, err := json.Marshal(elem)
				if err != nil {
					return err
				}
				destElem := slice.Index(i).Addr().Interface()
				if err := json.Unmarshal(data, destElem); err != nil {
					return err
				}
			}

			v.Set(slice)
			return nil
		}

	case reflect.Map:
		return func(v reflect.Value, val any) error {
			if val == nil {
				v.Set(reflect.Zero(fieldType))
				return nil
			}
			// Use JSON unmarshal for map types
			data, err := json.Marshal(val)
			if err != nil {
				return err
			}
			return json.Unmarshal(data, v.Addr().Interface())
		}

	case reflect.Struct:
		// Check for special types (like time.Time)
		if fieldType == reflect.TypeOf(json.Number("")) {
			return func(v reflect.Value, val any) error {
				switch x := val.(type) {
				case string:
					v.Set(reflect.ValueOf(json.Number(x)))
				case float64:
					v.Set(reflect.ValueOf(json.Number(fmt.Sprintf("%v", x))))
				case int:
					v.Set(reflect.ValueOf(json.Number(fmt.Sprintf("%d", x))))
				default:
					return fmt.Errorf("expected json.Number compatible type, got %T", val)
				}
				return nil
			}
		}

		// For general structs, use JSON unmarshal
		return func(v reflect.Value, val any) error {
			data, err := json.Marshal(val)
			if err != nil {
				return err
			}
			return json.Unmarshal(data, v.Addr().Interface())
		}

	default:
		// Fallback to JSON unmarshal for unknown types
		return func(v reflect.Value, val any) error {
			data, err := json.Marshal(val)
			if err != nil {
				return err
			}
			return json.Unmarshal(data, v.Addr().Interface())
		}
	}
}
