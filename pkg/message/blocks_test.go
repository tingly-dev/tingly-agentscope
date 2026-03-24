package message

import (
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
	"testing"
)

func TestErrorConstructor(t *testing.T) {
	block := Error(ErrorTypeAPI, "rate limit exceeded")

	if block == nil {
		t.Fatal("Error() should return non-nil block")
	}
	if block.ErrorType != ErrorTypeAPI {
		t.Errorf("Expected ErrorTypeAPI, got '%s'", block.ErrorType)
	}
	if block.Message != "rate limit exceeded" {
		t.Errorf("Expected 'rate limit exceeded', got '%s'", block.Message)
	}
	if block.Type() != types.BlockTypeError {
		t.Errorf("Type() should return BlockTypeError, got '%s'", block.Type())
	}
}

func TestErrorAllTypes(t *testing.T) {
	types := []ErrorType{ErrorTypeAPI, ErrorTypePanic, ErrorTypeWarning, ErrorTypeSystem}
	for _, errType := range types {
		block := Error(errType, "test")
		if block.ErrorType != errType {
			t.Errorf("Expected '%s', got '%s'", errType, block.ErrorType)
		}
	}
}
