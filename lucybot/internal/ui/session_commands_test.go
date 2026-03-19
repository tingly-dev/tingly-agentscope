package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tingly-dev/lucybot/internal/config"
	"github.com/tingly-dev/lucybot/internal/session"
)

func TestHandleSessionsCommand(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.SessionConfig{
		Enabled:     true,
		StoragePath: tmpDir,
	}

	mgr, _ := session.NewManager(cfg, "test", "/work")

	// Create test sessions
	mgr.Create("1", "Session 1")
	mgr.Create("2", "Session 2")

	app := &App{
		config: &config.Config{Session: *cfg},
		input:  NewInput(),
	}

	// Handle /sessions command
	cmd := app.handleSessionsCommand()
	if cmd == nil {
		t.Error("Expected command to be returned")
	}
}

func TestHandleSessionsCommand_NoSessions(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.SessionConfig{
		Enabled:     true,
		StoragePath: tmpDir,
	}

	_, _ = session.NewManager(cfg, "test", "/work")

	app := &App{
		config: &config.Config{Session: *cfg},
		input:  NewInput(),
	}

	// Handle /sessions command with no sessions
	cmd := app.handleSessionsCommand()
	if cmd == nil {
		t.Error("Expected command to be returned")
	}

	// Execute the command to get the message
	msg := cmd()
	systemMsg, ok := msg.(SystemMsg)
	if !ok {
		t.Errorf("Expected SystemMsg, got %T", msg)
	}

	if systemMsg.Content == "" {
		t.Error("Expected content in system message")
	}
}

func TestHandleSessionsCommand_SessionNotEnabled(t *testing.T) {
	cfg := &config.SessionConfig{
		Enabled: false,
	}

	app := &App{
		config: &config.Config{Session: *cfg},
		input:  NewInput(),
	}

	// Handle /sessions command when sessions are not enabled
	cmd := app.handleSessionsCommand()
	if cmd == nil {
		t.Error("Expected command to be returned")
	}

	// Execute the command to get the message
	msg := cmd()
	systemMsg, ok := msg.(SystemMsg)
	if !ok {
		t.Errorf("Expected SystemMsg, got %T", msg)
	}

	expectedContent := "Session persistence is not enabled"
	if !strings.HasPrefix(systemMsg.Content, expectedContent) {
		t.Errorf("Expected message to start with %q, got %q", expectedContent, systemMsg.Content)
	}
}

func TestHandleResumeCommand_WithArgs(t *testing.T) {
	app := &App{
		input: NewInput(),
	}

	// Handle /resume command with session ID argument
	cmd := app.handleResumeCommand("session-123")
	if cmd == nil {
		t.Error("Expected command to be returned")
	}

	// Execute the command to get the message
	msg := cmd()
	resumeMsg, ok := msg.(ResumeSessionMsg)
	if !ok {
		t.Errorf("Expected ResumeSessionMsg, got %T", msg)
	}

	if resumeMsg.SessionID != "session-123" {
		t.Errorf("Expected session ID session-123, got %s", resumeMsg.SessionID)
	}
}

func TestHandleResumeCommand_NoArgs(t *testing.T) {
	app := &App{
		input: NewInput(),
	}

	// Handle /resume command with no arguments (should show picker)
	// Since there's no agent/session manager, it returns an error message
	cmd := app.handleResumeCommand("")
	if cmd == nil {
		t.Error("Expected command to be returned")
	}

	// Execute the command to get the message
	msg := cmd()
	// When there's no session manager, it returns an error SystemMsg
	systemMsg, ok := msg.(SystemMsg)
	if !ok {
		t.Errorf("Expected SystemMsg (error when no session manager), got %T", msg)
	}

	// Should contain an error message
	if !strings.Contains(systemMsg.Content, "Error") {
		t.Errorf("Expected error message when no session manager, got %q", systemMsg.Content)
	}
}

func TestHandleResumeCommand_NoArgs_WithSessions(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.SessionConfig{
		Enabled:     true,
		StoragePath: tmpDir,
	}

	// Create a session manager with test sessions
	mgr, _ := session.NewManager(cfg, "test", "/work")
	mgr.Create("1", "Session 1")

	// Create a mock agent with session manager
	// Note: This would require setting up a full agent, which is complex
	// For now, we just verify the command structure is correct
	app := &App{
		config: &config.Config{Session: *cfg},
		input:  NewInput(),
	}

	// Handle /resume command with no arguments
	cmd := app.handleResumeCommand("")
	if cmd == nil {
		t.Error("Expected command to be returned")
	}

	// The command should be non-nil (actual session picker will be tested in Task 10)
}

func TestFormatSessionItem(t *testing.T) {
	testTime, _ := time.Parse(time.RFC3339, "2026-03-20T10:00:00Z")
	info := &session.SessionInfo{
		ID:           "test-id",
		Name:         "Test Session",
		CreatedAt:    testTime,
		MessageCount: 42,
	}

	result := FormatSessionItem(info)

	// Check that it contains the key parts
	if !strings.Contains(result, "Test Session") {
		t.Errorf("Expected result to contain session name, got %q", result)
	}
	if !strings.Contains(result, "42 messages") {
		t.Errorf("Expected result to contain message count, got %q", result)
	}
}

func TestFormatSessionItem_EmptyName(t *testing.T) {
	testTime, _ := time.Parse(time.RFC3339, "2026-03-20T10:00:00Z")
	info := &session.SessionInfo{
		ID:           "test-id",
		Name:         "",
		CreatedAt:    testTime,
		MessageCount: 5,
	}

	result := FormatSessionItem(info)

	// Should use ID when name is empty
	if !strings.Contains(result, "test-id") {
		t.Errorf("Expected to use ID when name is empty, got %q", result)
	}
}

func TestSessionPickerIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.SessionConfig{
		Enabled:     true,
		StoragePath: tmpDir,
	}

	// Create a session manager with test sessions
	mgr, _ := session.NewManager(cfg, "test", "/work")
	mgr.Create("1", "Session 1")
	mgr.Create("2", "Session 2")

	// Get session info
	sessions, _ := mgr.List()

	app := &App{
		config: &config.Config{Session: *cfg},
		input:  NewInput(),
	}

	// Send ShowSessionPickerMsg to display picker
	msg := ShowSessionPickerMsg{Sessions: sessions}
	model, cmd := app.Update(msg)

	if cmd != nil {
		t.Error("Expected no command from ShowSessionPickerMsg")
	}

	// Check that the picker is now set
	updatedApp := model.(*App)
	if updatedApp.sessionPicker == nil {
		t.Fatal("Expected sessionPicker to be set after ShowSessionPickerMsg")
	}

	// Verify view shows picker instead of normal app
	view := updatedApp.View()
	if view == "" {
		t.Error("Expected non-empty view when picker is active")
	}

	// Test that ESC closes the picker
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	model, cmd = updatedApp.Update(escMsg)

	if cmd != nil {
		t.Error("Expected no command from ESC key")
	}

	// Picker should be closed
	updatedApp = model.(*App)
	if updatedApp.sessionPicker != nil {
		t.Error("Expected sessionPicker to be nil after ESC")
	}
}
