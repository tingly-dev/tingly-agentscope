package message

import (
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
	"testing"
)

func TestErrorBlockType(t *testing.T) {
	block := &ErrorBlock{
		ErrorType: ErrorTypeAPI,
		Message:   "test error",
	}
	if block.Type() != types.BlockTypeError {
		t.Errorf("ErrorBlock.Type() should return BlockTypeError, got '%s'", block.Type())
	}
}

func TestErrorTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		errType  ErrorType
		expected string
	}{
		{"API", ErrorTypeAPI, "api"},
		{"Panic", ErrorTypePanic, "panic"},
		{"Warning", ErrorTypeWarning, "warning"},
		{"System", ErrorTypeSystem, "system"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.errType) != tt.expected {
				t.Errorf("ErrorType %s should be '%s', got '%s'", tt.name, tt.expected, tt.errType)
			}
		})
	}
}
