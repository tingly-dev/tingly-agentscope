package ui

import "testing"

func TestStylesDefined(t *testing.T) {
	// Verify all key styles are defined
	if UserStyle.GetForeground() == nil {
		t.Error("UserStyle should have foreground color")
	}
	if AssistantStyle.GetForeground() == nil {
		t.Error("AssistantStyle should have foreground color")
	}
	if ToolSymbol != "●" {
		t.Errorf("ToolSymbol should be '●', got %q", ToolSymbol)
	}
}
