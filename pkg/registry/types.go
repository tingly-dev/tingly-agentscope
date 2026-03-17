package registry

import (
	"fmt"
	"reflect"
)

// DetectArgType detects the argument type from a tool function.
// It looks for a method with signature: Call(ctx context.Context, args *T) (*tool.ToolResponse, error)
//
// The function supports both:
// 1. Generic interface: TypedTool[T] with Call(ctx, args T)
// 2. Standard interface: ToolCallable with Call(ctx, args any)
func DetectArgType(fn reflect.Type) (reflect.Type, error) {
	// Check if it's a pointer to a struct
	if fn.Kind() != reflect.Ptr && fn.Kind() != reflect.Struct {
		return nil, fmt.Errorf("tool must be a struct or pointer to struct, got %v", fn.Kind())
	}

	// For pointer types, try both pointer and element for method lookup
	// because methods defined with pointer receivers can only be found on pointer types
	var method reflect.Method
	var ok bool

	if fn.Kind() == reflect.Ptr {
		// Try pointer first (for pointer receiver methods)
		method, ok = fn.MethodByName("Call")
		if !ok {
			// Try element (for value receiver methods)
			method, ok = fn.Elem().MethodByName("Call")
		}
	} else {
		// For struct type, only look for value receiver methods
		method, ok = fn.MethodByName("Call")
	}

	if !ok {
		return nil, fmt.Errorf("tool must have a Call method")
	}

	methodType := method.Type

	// Check method signature: func (receiver) Call(ctx context.Context, args *T) (*tool.ToolResponse, error)
	// Method type has receiver as first argument, so:
	// In(0) = receiver, In(1) = context.Context, In(2) = *T

	if methodType.NumIn() != 3 {
		return nil, fmt.Errorf("Call method must have exactly 2 parameters (ctx, args), got %d", methodType.NumIn()-1)
	}

	// Check first parameter is context.Context
	ctxType := methodType.In(1)
	if ctxType.String() != "context.Context" {
		return nil, fmt.Errorf("first parameter must be context.Context, got %v", ctxType)
	}

	// Get second parameter (args)
	argsType := methodType.In(2)

	// Args must be a pointer (for mutable output)
	if argsType.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("args parameter must be a pointer (*T), got %v", argsType)
	}

	// Return the element type (T)
	return argsType.Elem(), nil
}

// DetectArgTypeFromValue is a convenience function that detects type from a value.
func DetectArgTypeFromValue(fn any) (reflect.Type, error) {
	if fn == nil {
		return nil, fmt.Errorf("tool is nil")
	}
	return DetectArgType(reflect.TypeOf(fn))
}

// IsTypedTool checks if a function implements the typed tool pattern.
func IsTypedTool(fn any) bool {
	_, err := DetectArgTypeFromValue(fn)
	return err == nil
}

// MustDetectArgType detects the argument type or panics.
// Convenient for init() functions.
func MustDetectArgType(fn reflect.Type) reflect.Type {
	typ, err := DetectArgType(fn)
	if err != nil {
		panic(err)
	}
	return typ
}

// MustDetectArgTypeFromValue detects the argument type from a value or panics.
func MustDetectArgTypeFromValue(fn any) reflect.Type {
	typ, err := DetectArgTypeFromValue(fn)
	if err != nil {
		panic(err)
	}
	return typ
}
