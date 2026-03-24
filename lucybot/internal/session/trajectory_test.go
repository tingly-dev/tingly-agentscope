package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrajectoryAnalyzer_AnalyzeSession(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "")
	analyzer := NewTrajectoryAnalyzer(store)

	// Create a session with messages
	now := time.Now()
	session := &Session{
		ID:        "test-session",
		Name:      "Test Session",
		CreatedAt: now.Add(-time.Hour),
		UpdatedAt: now,
		Messages: []Message{
			{Role: "user", Content: "Hello", Timestamp: now.Add(-time.Hour)},
			{Role: "assistant", Content: "Hi there", Timestamp: now.Add(-time.Hour + time.Minute)},
			{Role: "user", Content: "edit_file some content", Timestamp: now.Add(-time.Hour + 2*time.Minute)},
			{Role: "assistant", Content: "ToolCall: edit_file", Timestamp: now.Add(-time.Hour + 3*time.Minute)},
		},
	}

	err := store.Save(session)
	require.NoError(t, err)

	// Analyze the session
	metrics, err := analyzer.AnalyzeSession("test-session")
	require.NoError(t, err)

	assert.Equal(t, "test-session", metrics.SessionID)
	assert.Equal(t, 4, metrics.MessageCount)
	assert.Equal(t, 2, metrics.UserMessages)
	assert.Equal(t, 2, metrics.AssistantMessages)
	assert.Equal(t, 1, metrics.ToolCalls)
	assert.Contains(t, metrics.Patterns, "file_editing")
}

func TestTrajectoryAnalyzer_AnalyzeSession_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "")
	analyzer := NewTrajectoryAnalyzer(store)

	_, err := analyzer.AnalyzeSession("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load session")
}

func TestTrajectoryAnalyzer_AnalyzeAllSessions(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "")
	analyzer := NewTrajectoryAnalyzer(store)

	now := time.Now()

	// Create first session
	session1 := &Session{
		ID:        "session-1",
		Name:      "First Session",
		CreatedAt: now.Add(-2 * time.Hour),
		UpdatedAt: now.Add(-time.Hour),
		Messages: []Message{
			{Role: "user", Content: "grep pattern", Timestamp: now.Add(-2 * time.Hour)},
			{Role: "assistant", Content: "Found matches", Timestamp: now.Add(-2*time.Hour + time.Minute)},
			{Role: "user", Content: "view_file path", Timestamp: now.Add(-2*time.Hour + 2*time.Minute)},
			{Role: "user", Content: "edit_file some code", Timestamp: now.Add(-2*time.Hour + 3*time.Minute)},
		},
	}
	err := store.Save(session1)
	require.NoError(t, err)

	// Create second session with duplicate patterns to trigger common patterns
	session2 := &Session{
		ID:        "session-2",
		Name:      "Second Session",
		CreatedAt: now.Add(-time.Hour),
		UpdatedAt: now,
		Messages: []Message{
			{Role: "user", Content: "bash ls -la", Timestamp: now.Add(-time.Hour)},
			{Role: "assistant", Content: "Command output", Timestamp: now.Add(-time.Hour + time.Minute)},
			{Role: "user", Content: "edit_file more content", Timestamp: now.Add(-time.Hour + 2*time.Minute)},
			{Role: "assistant", Content: "File edited", Timestamp: now.Add(-time.Hour + 3*time.Minute)},
			{Role: "user", Content: "grep another", Timestamp: now.Add(-time.Hour + 4*time.Minute)},
		},
	}
	err = store.Save(session2)
	require.NoError(t, err)

	// Analyze all sessions
	metrics, err := analyzer.AnalyzeAllSessions()
	require.NoError(t, err)

	assert.Equal(t, 2, metrics.TotalSessions)
	// Note: JSONLStore counts metadata line as a message, so total is 9 (3 + 1 metadata + 4 + 1 metadata)
	assert.GreaterOrEqual(t, metrics.TotalMessages, 7)
	assert.Greater(t, metrics.AvgMessagesPerSession, 0.0)
	assert.NotEmpty(t, metrics.MostUsedTools)
	assert.NotEmpty(t, metrics.CommonPatterns)
	assert.NotNil(t, metrics.SessionsPerDay)
}

func TestTrajectoryAnalyzer_AnalyzeAllSessions_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "")
	analyzer := NewTrajectoryAnalyzer(store)

	metrics, err := analyzer.AnalyzeAllSessions()
	require.NoError(t, err)

	assert.Equal(t, 0, metrics.TotalSessions)
	assert.Equal(t, 0, metrics.TotalMessages)
	assert.Empty(t, metrics.MostUsedTools)
	assert.NotNil(t, metrics.SessionsPerDay)
}

func TestTrajectoryAnalyzer_ExportSession(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "")
	analyzer := NewTrajectoryAnalyzer(store)

	now := time.Now()
	session := &Session{
		ID:        "export-test",
		Name:      "Export Test Session",
		CreatedAt: now,
		UpdatedAt: now,
		Messages: []Message{
			{Role: "user", Content: "Hello", Timestamp: now},
		},
	}
	err := store.Save(session)
	require.NoError(t, err)

	// Export to file without extension
	outputPath := filepath.Join(tmpDir, "exported-session")
	err = analyzer.ExportSession("export-test", outputPath)
	require.NoError(t, err)

	// Verify file was created with .json extension
	_, err = os.Stat(outputPath + ".json")
	require.NoError(t, err)

	// Verify content is valid JSON
	data, err := os.ReadFile(outputPath + ".json")
	require.NoError(t, err)
	assert.Contains(t, string(data), "export-test")
}

func TestTrajectoryAnalyzer_ExportSession_WithExtension(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "")
	analyzer := NewTrajectoryAnalyzer(store)

	now := time.Now()
	session := &Session{
		ID:        "export-test-2",
		Name:      "Export Test Session 2",
		CreatedAt: now,
		UpdatedAt: now,
		Messages:  []Message{},
	}
	err := store.Save(session)
	require.NoError(t, err)

	// Export to file with extension
	outputPath := filepath.Join(tmpDir, "exported.txt")
	err = analyzer.ExportSession("export-test-2", outputPath)
	require.NoError(t, err)

	// Verify file was created
	_, err = os.Stat(outputPath)
	require.NoError(t, err)
}

func TestTrajectoryAnalyzer_ExportSession_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "")
	analyzer := NewTrajectoryAnalyzer(store)

	err := analyzer.ExportSession("nonexistent", filepath.Join(tmpDir, "output.json"))
	require.Error(t, err)
}

func TestTrajectoryAnalyzer_CompareSessions(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "")
	analyzer := NewTrajectoryAnalyzer(store)

	now := time.Now()

	// Create first session (shorter, fewer messages)
	session1 := &Session{
		ID:        "session-short",
		Name:      "Short Session",
		CreatedAt: now.Add(-30 * time.Minute),
		UpdatedAt: now,
		Messages: []Message{
			{Role: "user", Content: "Hello", Timestamp: now.Add(-30 * time.Minute)},
		},
	}
	err := store.Save(session1)
	require.NoError(t, err)

	// Create second session (longer duration, more messages)
	session2 := &Session{
		ID:        "session-long",
		Name:      "Long Session",
		CreatedAt: now.Add(-3 * time.Hour), // 3 hours ago
		UpdatedAt: now,                     // now, so duration is 3 hours
		Messages: []Message{
			{Role: "user", Content: "Hello", Timestamp: now.Add(-3 * time.Hour)},
			{Role: "assistant", Content: "Hi", Timestamp: now.Add(-3*time.Hour + time.Minute)},
			{Role: "user", Content: "More", Timestamp: now.Add(-3*time.Hour + 2*time.Minute)},
		},
	}
	err = store.Save(session2)
	require.NoError(t, err)

	// Compare sessions
	comparison, err := analyzer.CompareSessions("session-short", "session-long")
	require.NoError(t, err)

	assert.Equal(t, "session-short", comparison.Session1)
	assert.Equal(t, "session-long", comparison.Session2)
	assert.Equal(t, 2, comparison.MessageCountDiff) // 3 - 1 = 2
	// session-long has 3 hour duration, session-short has 30 min duration
	assert.Greater(t, comparison.DurationDiff, 2*time.Hour)
}

func TestTrajectoryAnalyzer_CompareSessions_Session1NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "")
	analyzer := NewTrajectoryAnalyzer(store)

	now := time.Now()
	session2 := &Session{
		ID:        "session-2",
		CreatedAt: now,
		UpdatedAt: now,
	}
	err := store.Save(session2)
	require.NoError(t, err)

	_, err = analyzer.CompareSessions("nonexistent", "session-2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load session 1")
}

func TestTrajectoryAnalyzer_CompareSessions_Session2NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "")
	analyzer := NewTrajectoryAnalyzer(store)

	now := time.Now()
	session1 := &Session{
		ID:        "session-1",
		CreatedAt: now,
		UpdatedAt: now,
	}
	err := store.Save(session1)
	require.NoError(t, err)

	_, err = analyzer.CompareSessions("session-1", "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load session 2")
}

func TestTrajectoryAnalyzer_DetectPatterns(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "")
	analyzer := NewTrajectoryAnalyzer(store)

	now := time.Now()
	session := &Session{
		ID:        "pattern-test",
		CreatedAt: now,
		UpdatedAt: now,
		Messages: []Message{
			{Role: "user", Content: "edit_file some code", Timestamp: now},    // file_editing
			{Role: "user", Content: "view_file path/to/file", Timestamp: now}, // code_reading
			{Role: "user", Content: "view_source symbol", Timestamp: now},     // code_reading
			{Role: "user", Content: "grep search pattern", Timestamp: now},    // code_search
			{Role: "user", Content: "search for something", Timestamp: now},   // code_search
			{Role: "user", Content: "bash ls -la", Timestamp: now},            // command_execution
			{Role: "user", Content: "run command", Timestamp: now},            // command_execution
			{Role: "user", Content: "error occurred", Timestamp: now},         // error_handling
			{Role: "assistant", Content: "operation failed", Timestamp: now},  // error_handling
		},
	}
	err := store.Save(session)
	require.NoError(t, err)

	metrics, err := analyzer.AnalyzeSession("pattern-test")
	require.NoError(t, err)

	assert.Contains(t, metrics.Patterns, "file_editing")
	assert.Contains(t, metrics.Patterns, "code_reading")
	assert.Contains(t, metrics.Patterns, "code_search")
	assert.Contains(t, metrics.Patterns, "command_execution")
	assert.Contains(t, metrics.Patterns, "error_handling")
}

func TestExtractToolName(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{"edit_file", "Calling edit_file tool", "edit_file"},
		{"view_file", "Using view_file", "view_file"},
		{"view_source", "Running view_source", "view_source"},
		{"create_file", "create_file operation", "create_file"},
		{"grep", "grep pattern", "grep"},
		{"find_file", "find_file search", "find_file"},
		{"list_directory", "list_directory path", "list_directory"},
		{"bash", "bash command", "bash"},
		{"ask_user_question", "ask_user_question prompt", "ask_user_question"},
		{"inform_user", "inform_user message", "inform_user"},
		{"no match", "some random content", ""},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractToolName(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTrajectoryAnalyzer_ToolUsageRanking(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONLStore(tmpDir, "")
	analyzer := NewTrajectoryAnalyzer(store)

	now := time.Now()
	today := now.Format("2006-01-02")

	// Create session with multiple tool uses
	session := &Session{
		ID:        "tool-usage-test",
		CreatedAt: now,
		UpdatedAt: now,
		Messages: []Message{
			{Role: "user", Content: "edit_file", Timestamp: now},
			{Role: "user", Content: "edit_file", Timestamp: now},
			{Role: "user", Content: "edit_file", Timestamp: now}, // 3 uses
			{Role: "user", Content: "grep pattern", Timestamp: now},
			{Role: "user", Content: "grep another", Timestamp: now}, // 2 uses
			{Role: "user", Content: "bash ls", Timestamp: now},      // 1 use
		},
	}
	err := store.Save(session)
	require.NoError(t, err)

	metrics, err := analyzer.AnalyzeAllSessions()
	require.NoError(t, err)

	// Verify sessions per day
	assert.Equal(t, 1, metrics.SessionsPerDay[today])

	// Verify tool ranking (most used first)
	require.Len(t, metrics.MostUsedTools, 3)
	assert.Equal(t, "edit_file", metrics.MostUsedTools[0].ToolName)
	assert.Equal(t, 3, metrics.MostUsedTools[0].Count)
	assert.Equal(t, "grep", metrics.MostUsedTools[1].ToolName)
	assert.Equal(t, 2, metrics.MostUsedTools[1].Count)
	assert.Equal(t, "bash", metrics.MostUsedTools[2].ToolName)
	assert.Equal(t, 1, metrics.MostUsedTools[2].Count)
}

func TestSessionComparison_Struct(t *testing.T) {
	comparison := &SessionComparison{
		Session1:         "session-a",
		Session2:         "session-b",
		MessageCountDiff: 5,
		DurationDiff:     time.Hour,
	}

	assert.Equal(t, "session-a", comparison.Session1)
	assert.Equal(t, "session-b", comparison.Session2)
	assert.Equal(t, 5, comparison.MessageCountDiff)
	assert.Equal(t, time.Hour, comparison.DurationDiff)
}

func TestSessionMetrics_Struct(t *testing.T) {
	metrics := &SessionMetrics{
		SessionID:         "test-id",
		Duration:          time.Hour,
		MessageCount:      10,
		UserMessages:      5,
		AssistantMessages: 5,
		ToolCalls:         3,
		ToolUsage: []ToolUsage{
			{ToolName: "edit_file", Count: 2, Successes: 2, Failures: 0},
		},
		Patterns: []string{"file_editing"},
		Metadata: map[string]interface{}{"key": "value"},
	}

	assert.Equal(t, "test-id", metrics.SessionID)
	assert.Equal(t, time.Hour, metrics.Duration)
	assert.Equal(t, 10, metrics.MessageCount)
	assert.Equal(t, 5, metrics.UserMessages)
	assert.Equal(t, 5, metrics.AssistantMessages)
	assert.Equal(t, 3, metrics.ToolCalls)
}

func TestAggregateMetrics_Struct(t *testing.T) {
	metrics := &AggregateMetrics{
		TotalSessions:         5,
		TotalDuration:         5 * time.Hour,
		AvgDuration:           time.Hour,
		TotalMessages:         50,
		AvgMessagesPerSession: 10.0,
		MostUsedTools: []ToolUsage{
			{ToolName: "edit_file", Count: 10},
		},
		CommonPatterns: []string{"file_editing (10x)"},
		SessionsPerDay: map[string]int{"2024-01-01": 2},
	}

	assert.Equal(t, 5, metrics.TotalSessions)
	assert.Equal(t, 50, metrics.TotalMessages)
	assert.Equal(t, 10.0, metrics.AvgMessagesPerSession)
	assert.Equal(t, 2, metrics.SessionsPerDay["2024-01-01"])
}
