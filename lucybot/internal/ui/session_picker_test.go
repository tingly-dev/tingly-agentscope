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

	// Initial state
	if model.cursor != 0 {
		t.Errorf("Expected cursor at 0, got %d", model.cursor)
	}
	if model.selected != nil {
		t.Error("Expected no selection initially")
	}

	// Test update with key down
	msg := tea.KeyMsg{Type: tea.KeyDown}
	_, cmd := model.Update(msg)
	if cmd != nil {
		t.Error("Expected no command from key down")
	}
	if model.cursor != 1 {
		t.Errorf("Expected cursor at 1 after down, got %d", model.cursor)
	}

	// Test update with key up
	msg = tea.KeyMsg{Type: tea.KeyUp}
	_, cmd = model.Update(msg)
	if cmd != nil {
		t.Error("Expected no command from key up")
	}
	if model.cursor != 0 {
		t.Errorf("Expected cursor at 0 after up, got %d", model.cursor)
	}

	// Test cursor bounds (can't go below 0)
	msg = tea.KeyMsg{Type: tea.KeyUp}
	_, cmd = model.Update(msg)
	if cmd != nil {
		t.Error("Expected no command from key up at boundary")
	}
	if model.cursor != 0 {
		t.Errorf("Expected cursor to stay at 0, got %d", model.cursor)
	}

	// Test cursor bounds (can't go past last item)
	model.cursor = 1
	msg = tea.KeyMsg{Type: tea.KeyDown}
	_, cmd = model.Update(msg)
	if cmd != nil {
		t.Error("Expected no command from key down at boundary")
	}
	if model.cursor != 1 {
		t.Errorf("Expected cursor to stay at 1, got %d", model.cursor)
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

	if model.selected == nil {
		t.Error("Expected selection to be set")
	}
	if model.selected.ID != "1" {
		t.Errorf("Expected selected ID '1', got '%s'", model.selected.ID)
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

	// Test delete key
	msg := tea.KeyMsg{Type: tea.KeyDelete}
	_, cmd := model.Update(msg)

	if cmd == nil {
		t.Fatal("Expected a command from Delete key")
	}

	// Execute the command to get the message
	msgChan := make(chan tea.Msg)
	go func() {
		msgChan <- cmd()
	}()

	result := <-msgChan
	if deleteMsg, ok := result.(DeleteSessionMsg); ok {
		if deleteMsg.SessionID != "1" {
			t.Errorf("Expected SessionID '1', got '%s'", deleteMsg.SessionID)
		}
	} else {
		t.Errorf("Expected DeleteSessionMsg, got %T", result)
	}
}

func TestSessionPickerEmptySessions(t *testing.T) {
	model := newSessionPickerModel([]*session.SessionInfo{})

	// Should not panic on empty sessions
	msg := tea.KeyMsg{Type: tea.KeyDown}
	newModel, _ := model.Update(msg)

	// Cursor should stay at 0
	if newModel.(*sessionPickerModel).cursor != 0 {
		t.Errorf("Expected cursor to stay at 0 with empty sessions, got %d", newModel.(*sessionPickerModel).cursor)
	}

	// View should handle empty sessions
	view := model.View()
	if view == "" {
		t.Error("Expected non-empty view even with no sessions")
	}
}

func TestSessionItem(t *testing.T) {
	tests := []struct {
		name     string
		item     sessionItem
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
