package ui

import (
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
)

func TestMessages_AddTurn(t *testing.T) {
	messages := NewMessages()

	// Add a user turn
	userTurn := NewInteractionTurn("user", "")
	userTurn.AddContentBlock(&message.TextBlock{Text: "Hello"})
	messages.AddTurn(userTurn)

	// Add an assistant turn
	asstTurn := NewInteractionTurn("assistant", "Lucy")
	asstTurn.AddContentBlock(&message.TextBlock{Text: "Hi there"})
	messages.AddTurn(asstTurn)

	// Should have 2 turns
	if len(messages.turns) != 2 {
		t.Errorf("Expected 2 turns, got %d", len(messages.turns))
	}
}

func TestMessages_GetOrCreateCurrentTurn(t *testing.T) {
	messages := NewMessages()

	// Get current turn (should create new assistant turn)
	turn := messages.GetOrCreateCurrentTurn("assistant", "Lucy")

	// Should have 1 turn
	if len(messages.turns) != 1 {
		t.Errorf("Expected 1 turn, got %d", len(messages.turns))
	}

	// Add content to turn
	turn.AddContentBlock(&message.TextBlock{Text: "Hello"})

	// Get current turn again (should return same incomplete turn)
	turn2 := messages.GetOrCreateCurrentTurn("assistant", "Lucy")
	if turn2 != turn {
		t.Error("Should return same incomplete turn")
	}

	// Mark turn complete
	turn.Complete = true

	// Get current turn (should create new turn)
	turn3 := messages.GetOrCreateCurrentTurn("assistant", "Lucy")
	if turn3 == turn {
		t.Error("Should create new turn after previous is complete")
	}
}
