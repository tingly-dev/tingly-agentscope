package types

import "testing"

func TestBlockTypeErrorDefined(t *testing.T) {
	if BlockTypeError != "error" {
		t.Errorf("BlockTypeError should be 'error', got '%s'", BlockTypeError)
	}
}
