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

func TestTreeSymbols(t *testing.T) {
	// Verify tree symbols are defined
	if TreeBranch == "" {
		t.Error("TreeBranch should not be empty")
	}
	if TreeVertical == "" {
		t.Error("TreeVertical should not be empty")
	}
	if TreeEnd == "" {
		t.Error("TreeEnd should not be empty")
	}
	if ModelSymbol == "" {
		t.Error("ModelSymbol should not be empty")
	}
	if ToolSymbol == "" {
		t.Error("ToolSymbol should not be empty")
	}
}

func TestResultStyles(t *testing.T) {
	// Verify result styles are defined
	if ResultTruncatedStyle.GetForeground() == nil {
		t.Error("ResultTruncatedStyle should have foreground color")
	}
}
