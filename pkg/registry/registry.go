package registry

import (
	"fmt"
	"reflect"
	"sync"
)

// Registry manages type registrations and their decoders.
// Thread-safe with double-checked locking for performance.
type Registry struct {
	mu       sync.RWMutex
	decoders map[reflect.Type]*cachedDecoder
}

// globalRegistry is the default global registry instance.
var globalRegistry = &Registry{
	decoders: make(map[reflect.Type]*cachedDecoder),
}

// Default returns the global default registry.
func Default() *Registry {
	return globalRegistry
}

// Register registers a type and returns its decoder.
// If the type is already registered, returns the cached decoder.
// Thread-safe with double-checked locking pattern.
func (r *Registry) Register(typ reflect.Type) (*cachedDecoder, error) {
	// Fast path: read lock
	r.mu.RLock()
	decoder, ok := r.decoders[typ]
	r.mu.RUnlock()

	if ok {
		return decoder, nil
	}

	// Slow path: write lock
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if decoder, ok := r.decoders[typ]; ok {
		return decoder, nil
	}

	// Validate type
	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct type, got %s", typ.Kind())
	}

	// Build and cache the decoder
	decoder = buildCachedDecoder(typ)
	r.decoders[typ] = decoder

	return decoder, nil
}

// MustRegister registers a type and panics on error.
// Convenient for init() functions.
func (r *Registry) MustRegister(typ reflect.Type) *cachedDecoder {
	decoder, err := r.Register(typ)
	if err != nil {
		panic(err)
	}
	return decoder
}

// GetDecoder returns the cached decoder for a type, or nil if not registered.
func (r *Registry) GetDecoder(typ reflect.Type) *cachedDecoder {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.decoders[typ]
}

// Decode decodes input data into a new instance of the specified type.
// Returns a pointer to the decoded struct.
func (r *Registry) Decode(typ reflect.Type, data map[string]any) (any, error) {
	decoder := r.GetDecoder(typ)
	if decoder == nil {
		// Auto-register on first use
		var err error
		decoder, err = r.Register(typ)
		if err != nil {
			return nil, fmt.Errorf("failed to register type %s: %w", typ, err)
		}
	}

	return decoder.Decode(data)
}

// Clear removes all registered decoders.
// Primarily useful for testing.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.decoders = make(map[reflect.Type]*cachedDecoder)
}

// Size returns the number of registered types.
func (r *Registry) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.decoders)
}

// RegisterType is a convenience function to register a type by value.
// The type is inferred from the pointer to the value.
func RegisterType[T any](r *Registry) (*cachedDecoder, error) {
	var zero T
	typ := reflect.TypeOf(zero)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return r.Register(typ)
}

// MustRegisterType is a convenience function that panics on error.
func MustRegisterType[T any](r *Registry) *cachedDecoder {
	decoder, err := RegisterType[T](r)
	if err != nil {
		panic(err)
	}
	return decoder
}

// DecodeInto is a convenience function to decode data into a value of type T.
// Returns a pointer to the decoded value.
func DecodeInto[T any](r *Registry, data map[string]any) (*T, error) {
	var zero T
	typ := reflect.TypeOf(zero)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	result, err := r.Decode(typ, data)
	if err != nil {
		return nil, err
	}

	return result.(*T), nil
}

// Global convenience functions using the default registry

// Register registers a type in the global registry.
func Register(typ reflect.Type) (*cachedDecoder, error) {
	return Default().Register(typ)
}

// MustRegister registers a type in the global registry and panics on error.
func MustRegister(typ reflect.Type) *cachedDecoder {
	return Default().MustRegister(typ)
}

// GetDecoder returns the decoder for a type from the global registry.
func GetDecoder(typ reflect.Type) *cachedDecoder {
	return Default().GetDecoder(typ)
}

// Decode decodes data using the global registry.
func Decode(typ reflect.Type, data map[string]any) (any, error) {
	return Default().Decode(typ, data)
}

// ClearAll clears all registrations from the global registry.
func ClearAll() {
	Default().Clear()
}
