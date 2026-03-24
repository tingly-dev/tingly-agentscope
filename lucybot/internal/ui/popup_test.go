package ui

import (
	"testing"

	"github.com/tingly-dev/lucybot/internal/skills"
)

func TestSetCommandItems(t *testing.T) {
	popup := NewPopup("Commands", 10)

	// Set command items
	popup.SetCommandItems()

	// Check that built-in commands are present
	builtinFound := map[string]bool{
		"/help":   false,
		"/clear":  false,
		"/resume": false,
		"/compact": false,
		"/tools":  false,
		"/model":  false,
		"/quit":   false,
	}

	for _, item := range popup.allItems {
		if _, ok := builtinFound[item.Title]; ok {
			builtinFound[item.Title] = true
		}
	}

	// Verify all built-in commands are present
	for cmd, found := range builtinFound {
		if !found {
			t.Errorf("Built-in command %s not found in popup items", cmd)
		}
	}

	// Verify count
	expectedCount := 7 // help, clear, resume, compact, tools, model, quit
	if len(popup.allItems) != expectedCount {
		t.Errorf("Expected %d items, got %d", expectedCount, len(popup.allItems))
	}
}

func TestSetCommandItemsWithSkills(t *testing.T) {
	popup := NewPopup("Commands", 10)

	// Create test skills
	skill1 := &skills.Skill{
		Name:        "code-review",
		Description: "Review code for best practices",
	}
	skill2 := &skills.Skill{
		Name:        "debug",
		Description: "Help debug issues",
	}

	skillsList := []*skills.Skill{skill1, skill2}

	// Set command items with skills
	popup.SetCommandItemsWithSkills(skillsList)

	// Check that built-in commands are present
	builtinFound := map[string]bool{
		"/help":   false,
		"/clear":  false,
		"/resume": false,
		"/compact": false,
		"/tools":  false,
		"/model":  false,
		"/quit":   false,
	}

	// Check that skill commands are present
	skillFound := map[string]bool{
		"/code-review": false,
		"/debug":       false,
	}

	for _, item := range popup.allItems {
		if _, ok := builtinFound[item.Title]; ok {
			builtinFound[item.Title] = true
		}
		if _, ok := skillFound[item.Title]; ok {
			skillFound[item.Title] = true
		}
	}

	// Verify all built-in commands are present
	for cmd, found := range builtinFound {
		if !found {
			t.Errorf("Built-in command %s not found in popup items", cmd)
		}
	}

	// Verify all skill commands are present
	for cmd, found := range skillFound {
		if !found {
			t.Errorf("Skill command %s not found in popup items", cmd)
		}
	}

	// Verify total count
	expectedCount := len(builtinFound) + len(skillFound)
	if len(popup.allItems) != expectedCount {
		t.Errorf("Expected %d items, got %d", expectedCount, len(popup.allItems))
	}
}

func TestSetCommandItemsWithSkills_Empty(t *testing.T) {
	popup := NewPopup("Commands", 10)

	// Set command items with no skills
	popup.SetCommandItemsWithSkills([]*skills.Skill{})

	// Should still have built-in commands
	builtinCount := 7 // help, clear, resume, compact, tools, model, quit
	if len(popup.allItems) != builtinCount {
		t.Errorf("Expected %d built-in items, got %d", builtinCount, len(popup.allItems))
	}
}

func TestSetCommandItemsWithSkills_Nil(t *testing.T) {
	popup := NewPopup("Commands", 10)

	// Set command items with nil skills
	popup.SetCommandItemsWithSkills(nil)

	// Should still have built-in commands
	builtinCount := 7 // help, clear, resume, compact, tools, model, quit
	if len(popup.allItems) != builtinCount {
		t.Errorf("Expected %d built-in items, got %d", builtinCount, len(popup.allItems))
	}
}

func TestSetCommandItemsWithSkills_PreservesOriginal(t *testing.T) {
	popup := NewPopup("Commands", 10)

	// Set original command items
	popup.SetCommandItems()

	// Get initial count
	initialCount := len(popup.allItems)

	// Create test skills
	skill1 := &skills.Skill{
		Name:        "test",
		Description: "Test skill",
	}

	skillsList := []*skills.Skill{skill1}

	// Set command items with skills
	popup.SetCommandItemsWithSkills(skillsList)

	// Should have one more item (the skill command)
	expectedCount := initialCount + 1
	if len(popup.allItems) != expectedCount {
		t.Errorf("Expected %d items, got %d", expectedCount, len(popup.allItems))
	}
}
