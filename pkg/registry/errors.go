package registry

import (
	"fmt"
)

// DecodeError represents an error that occurred during decoding
type DecodeError struct {
	Field   string // Field name where the error occurred
	Value   any    // The value that caused the error
	Message string // Error message
	Cause   error  // Underlying error, if any
}

// Error returns the error message
func (e *DecodeError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("decode error on field '%s': %s: %v", e.Field, e.Message, e.Cause)
	}
	return fmt.Sprintf("decode error on field '%s': %s", e.Field, e.Message)
}

// Unwrap returns the underlying error
func (e *DecodeError) Unwrap() error {
	return e.Cause
}

// NewDecodeError creates a new DecodeError
func NewDecodeError(field, message string, value any) *DecodeError {
	return &DecodeError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// NewDecodeErrorWithCause creates a new DecodeError with an underlying cause
func NewDecodeErrorWithCause(field, message string, value any, cause error) *DecodeError {
	return &DecodeError{
		Field:   field,
		Value:   value,
		Message: message,
		Cause:   cause,
	}
}
