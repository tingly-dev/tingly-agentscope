package ui

import (
	"testing"
	"time"

	"github.com/tingly-dev/lucybot/internal/session"
	tea "github.com/charmbracelet/bubbletea"
)

func TestSessionPickerModel(t *testing.T) {
	sessions := []*session.SessionInfo{
		{ID: "1", Name: "Session 1", CreatedAt: time.Now(), MessageCount: 10},
		{ID: "2", Name: "Session 2", CreatedAt: time.Now(), MessageCount: 20},
	}

	model := newSessionPickerModel(sessions)

	// Initial state - list should be initialized
	if model.list.Items() == nil {
		t.Error("Expected list items to be initialized")
	}

	// Initial cursor should be at index 0
	if model.list.Cursor() != 0 {
		t.Errorf("Expected cursor at 0, got %d", model.list.Cursor())
	}
}

func TestSessionPickerSelection(t *testing.T) {
	sessions := []*session.SessionInfo{
		{ID: "1", Name: "Session 1", CreatedAt: time.Now(), MessageCount: 10},
		{ID: "2", Name: "Session 2", CreatedAt: time.Now(), MessageCount: 20},
	}

	model := newSessionPickerModel(sessions)

	// Test selection with Enter key
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := model.Update(msg)

	if cmd == nil {
		t.Fatal("Expected a command from Enter key")
	}

	// Execute the command to get the message
	msgChan := make(chan tea.Msg)
	go func() {
		msgChan <- cmd()
	}()

	result := <-msgChan
	if pickerMsg, ok := result.(SessionPickerMsg); ok {
		if pickerMsg.SessionID != "1" {
			t.Errorf("Expected SessionID '1', got '%s'", pickerMsg.SessionID)
		}
	} else {
		t.Errorf("Expected SessionPickerMsg, got %T", result)
	}
}

func TestSessionPickerQuit(t *testing.T) {
	sessions := []*session.SessionInfo{
		{ID: "1", Name: "Session 1", CreatedAt: time.Now(), MessageCount: 10},
	}

	model := newSessionPickerModel(sessions)

	// Test quit with Ctrl+C
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	newModel, cmd := model.Update(msg)

	if !newModel.(*sessionPickerModel).quitting {
		t.Error("Expected quitting to be true")
	}

	quitCmd := cmd()
	if quitCmd != tea.Quit() {
		t.Error("Expected tea.Quit command")
	}

	// View should return empty string when quitting
	if newModel.View() != "" {
		t.Error("Expected empty view when quitting")
	}
}

func TestSessionPickerDelete(t *testing.T) {
	sessions := []*session.SessionInfo{
		{ID: "1", Name: "Session 1", CreatedAt: time.Now(), MessageCount: 10},
		{ID: "2", Name: "Session 2", CreatedAt: time.Now(), MessageCount: 20},
	}

	model := newSessionPickerModel(sessions)

	// Test delete key - delete functionality was removed, so it should do nothing
	msg := tea.KeyMsg{Type: tea.KeyDelete}
	_, cmd := model.Update(msg)

	// Delete key should not trigger any command (functionality removed)
	if cmd != nil {
		t.Error("Expected no command from Delete key (delete functionality removed)")
	}
}

func TestSessionPickerEmptySessions(t *testing.T) {
	model := newSessionPickerModel([]*session.SessionInfo{})

	// Should not panic on empty sessions
	msg := tea.KeyMsg{Type: tea.KeyDown}
	newModel, _ := model.Update(msg)

	// Model should still be valid
	if newModel == nil {
		t.Error("Expected model to remain valid with empty sessions")
	}

	// View should handle empty sessions
	view := model.View()
	if view == "" {
		t.Error("Expected non-empty view even with no sessions")
	}
}

func TestSessionPickerNavigation(t *testing.T) {
	sessions := []*session.SessionInfo{
		{ID: "1", Name: "Session 1", CreatedAt: time.Now(), MessageCount: 10},
		{ID: "2", Name: "Session 2", CreatedAt: time.Now(), MessageCount: 20},
		{ID: "3", Name: "Session 3", CreatedAt: time.Now(), MessageCount: 30},
	}

	model := newSessionPickerModel(sessions)

	// The list component handles its own navigation
	// Test that the model can handle navigation messages without crashing
	msg := tea.KeyMsg{Type: tea.KeyDown}
	newModel, _ := model.Update(msg)
	if newModel == nil {
		t.Error("Expected model to remain valid after navigation")
	}

	msg = tea.KeyMsg{Type: tea.KeyUp}
	newModel, _ = model.Update(msg)
	if newModel == nil {
		t.Error("Expected model to remain valid after navigation")
	}

	// Verify the list still has items
	if newModel.(*sessionPickerModel).list.Items() == nil {
		t.Error("Expected list items to remain after navigation")
	}
}

func TestSessionItem(t *testing.T) {
	tests := []struct {
		name      string
		item      sessionItem
		wantTitle string
	}{
		{
			name: "with agent name",
			item: sessionItem{SessionInfo: session.SessionInfo{
				AgentName: "test-agent",
				Name:      "My Session",
				ID:        "123",
			}},
			wantTitle: "test-agent - My Session",
		},
		{
			name: "without agent name",
			item: sessionItem{SessionInfo: session.SessionInfo{
				Name: "My Session",
				ID:   "123",
			}},
			wantTitle: "My Session",
		},
		{
			name: "with empty name",
			item: sessionItem{SessionInfo: session.SessionInfo{
				ID: "123",
			}},
			wantTitle: "123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.item.Title(); got != tt.wantTitle {
				t.Errorf("Title() = %v, want %v", got, tt.wantTitle)
			}
		})
	}
}

func TestFormatLastMessage(t *testing.T) {
	tests := []struct {
		name string
		msg  string
		want string
	}{
		{
			name: "empty message",
			msg:  "",
			want: "No messages",
		},
		{
			name: "short message",
			msg:  "Hello",
			want: "Hello",
		},
		{
			name: "long message",
			msg:  "This is a very long message that should be truncated",
			want: "This is a very long message that should be trun...",
		},
		{
			name: "exactly 50 chars",
			msg:  string(make([]byte, 50)),
			want: string(make([]byte, 50)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatLastMessage(tt.msg); got != tt.want {
				t.Errorf("formatLastMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatDate(t *testing.T) {
	tests := []struct {
		name string
		t    time.Time
		want string
	}{
		{
			name: "zero time",
			t:    time.Time{},
			want: "unknown",
		},
		{
			name: "valid time",
			t:    time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC),
			want: "2024-03-15 14:30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatDate(tt.t); got != tt.want {
				t.Errorf("formatDate() = %v, want %v", got, tt.want)
			}
		})
	}
}
