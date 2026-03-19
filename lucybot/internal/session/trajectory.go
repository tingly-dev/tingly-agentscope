package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TrajectoryAnalyzer analyzes session patterns for insights
type TrajectoryAnalyzer struct {
	store Store
}

// NewTrajectoryAnalyzer creates a new trajectory analyzer
func NewTrajectoryAnalyzer(store Store) *TrajectoryAnalyzer {
	return &TrajectoryAnalyzer{store: store}
}

// ToolUsage tracks how often a tool was used
type ToolUsage struct {
	ToolName  string `json:"tool_name"`
	Count     int    `json:"count"`
	Successes int    `json:"successes"`
	Failures  int    `json:"failures"`
}

// SessionMetrics contains metrics for a single session
type SessionMetrics struct {
	SessionID         string                 `json:"session_id"`
	Duration          time.Duration          `json:"duration"`
	MessageCount      int                    `json:"message_count"`
	UserMessages      int                    `json:"user_messages"`
	AssistantMessages int                    `json:"assistant_messages"`
	ToolCalls         int                    `json:"tool_calls"`
	ToolUsage         []ToolUsage            `json:"tool_usage"`
	Patterns          []string               `json:"patterns"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// AggregateMetrics contains aggregated metrics across multiple sessions
type AggregateMetrics struct {
	TotalSessions         int            `json:"total_sessions"`
	TotalDuration         time.Duration  `json:"total_duration"`
	AvgDuration           time.Duration  `json:"avg_duration"`
	TotalMessages         int            `json:"total_messages"`
	AvgMessagesPerSession float64        `json:"avg_messages_per_session"`
	MostUsedTools         []ToolUsage    `json:"most_used_tools"`
	CommonPatterns        []string       `json:"common_patterns"`
	SessionsPerDay        map[string]int `json:"sessions_per_day"`
}

// AnalyzeSession analyzes a single session and returns metrics
func (a *TrajectoryAnalyzer) AnalyzeSession(sessionID string) (*SessionMetrics, error) {
	session, err := a.store.Load(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	metrics := &SessionMetrics{
		SessionID: sessionID,
		Duration:  session.UpdatedAt.Sub(session.CreatedAt),
		Metadata:  make(map[string]interface{}),
	}

	// Count messages
	for _, msg := range session.Messages {
		metrics.MessageCount++
		switch msg.Role {
		case "user":
			metrics.UserMessages++
		case "assistant":
			metrics.AssistantMessages++
		}

		// Analyze content for tool calls
		contentStr, _ := msg.Content.(string)
		if strings.Contains(contentStr, "ToolCall") || strings.Contains(contentStr, "tool_use") {
			metrics.ToolCalls++
		}
	}

	// Detect patterns
	metrics.Patterns = a.detectPatterns(session.Messages)

	return metrics, nil
}

// AnalyzeAllSessions analyzes all sessions and returns aggregate metrics
func (a *TrajectoryAnalyzer) AnalyzeAllSessions() (*AggregateMetrics, error) {
	sessions, err := a.store.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	metrics := &AggregateMetrics{
		TotalSessions:  len(sessions),
		SessionsPerDay: make(map[string]int),
	}

	if len(sessions) == 0 {
		return metrics, nil
	}

	toolUsage := make(map[string]*ToolUsage)
	patternCount := make(map[string]int)

	for _, info := range sessions {
		// Aggregate durations
		duration := info.UpdatedAt.Sub(info.CreatedAt)
		metrics.TotalDuration += duration
		metrics.TotalMessages += info.MessageCount

		// Count sessions per day
		day := info.CreatedAt.Format("2006-01-02")
		metrics.SessionsPerDay[day]++

		// Analyze individual session for tool usage and patterns
		session, err := a.store.Load(info.ID)
		if err != nil {
			continue // Skip sessions that can't be loaded
		}

		for _, msg := range session.Messages {
			// Extract tool usage
			contentStr, _ := msg.Content.(string)
			if toolName := extractToolName(contentStr); toolName != "" {
				if _, ok := toolUsage[toolName]; !ok {
					toolUsage[toolName] = &ToolUsage{ToolName: toolName}
				}
				toolUsage[toolName].Count++
			}

			// Count patterns
			for _, pattern := range a.detectPatterns([]Message{msg}) {
				patternCount[pattern]++
			}
		}
	}

	// Calculate averages
	metrics.AvgDuration = metrics.TotalDuration / time.Duration(metrics.TotalSessions)
	metrics.AvgMessagesPerSession = float64(metrics.TotalMessages) / float64(metrics.TotalSessions)

	// Convert tool usage map to slice
	for _, usage := range toolUsage {
		metrics.MostUsedTools = append(metrics.MostUsedTools, *usage)
	}

	// Sort tools by count (simple bubble sort)
	for i := 0; i < len(metrics.MostUsedTools); i++ {
		for j := i + 1; j < len(metrics.MostUsedTools); j++ {
			if metrics.MostUsedTools[j].Count > metrics.MostUsedTools[i].Count {
				metrics.MostUsedTools[i], metrics.MostUsedTools[j] = metrics.MostUsedTools[j], metrics.MostUsedTools[i]
			}
		}
	}

	// Get top patterns
	for pattern, count := range patternCount {
		if count > 1 {
			metrics.CommonPatterns = append(metrics.CommonPatterns, fmt.Sprintf("%s (%dx)", pattern, count))
		}
	}

	return metrics, nil
}

// detectPatterns detects common patterns in messages
func (a *TrajectoryAnalyzer) detectPatterns(messages []Message) []string {
	var patterns []string
	patternCounts := make(map[string]int)

	for _, msg := range messages {
		contentStr, _ := msg.Content.(string)

		// Detect patterns
		switch {
		case strings.Contains(contentStr, "edit_file"):
			patternCounts["file_editing"]++
		case strings.Contains(contentStr, "view_source") || strings.Contains(contentStr, "view_file"):
			patternCounts["code_reading"]++
		case strings.Contains(contentStr, "grep") || strings.Contains(contentStr, "search"):
			patternCounts["code_search"]++
		case strings.Contains(contentStr, "bash") || strings.Contains(contentStr, "command"):
			patternCounts["command_execution"]++
		case strings.Contains(contentStr, "error") || strings.Contains(contentStr, "fail"):
			patternCounts["error_handling"]++
		}
	}

	for pattern := range patternCounts {
		patterns = append(patterns, pattern)
	}

	return patterns
}

// extractToolName extracts a tool name from content if present
func extractToolName(content string) string {
	// Look for common tool call patterns
	patterns := []string{
		"edit_file", "view_file", "view_source", "create_file",
		"grep", "find_file", "list_directory", "bash",
		"ask_user_question", "inform_user",
	}

	for _, tool := range patterns {
		if strings.Contains(content, tool) {
			return tool
		}
	}
	return ""
}

// ExportSession exports a session to a file for analysis
func (a *TrajectoryAnalyzer) ExportSession(sessionID string, outputPath string) error {
	session, err := a.store.Load(sessionID)
	if err != nil {
		return err
	}

	// Add extension if not present
	if filepath.Ext(outputPath) == "" {
		outputPath += ".json"
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	return writeFile(outputPath, data)
}

// CompareSessions compares two sessions and returns differences
func (a *TrajectoryAnalyzer) CompareSessions(sessionID1, sessionID2 string) (*SessionComparison, error) {
	session1, err := a.store.Load(sessionID1)
	if err != nil {
		return nil, fmt.Errorf("failed to load session 1: %w", err)
	}

	session2, err := a.store.Load(sessionID2)
	if err != nil {
		return nil, fmt.Errorf("failed to load session 2: %w", err)
	}

	comparison := &SessionComparison{
		Session1: sessionID1,
		Session2: sessionID2,
	}

	// Compare message counts
	comparison.MessageCountDiff = len(session2.Messages) - len(session1.Messages)

	// Compare durations
	duration1 := session1.UpdatedAt.Sub(session1.CreatedAt)
	duration2 := session2.UpdatedAt.Sub(session2.CreatedAt)
	comparison.DurationDiff = duration2 - duration1

	return comparison, nil
}

// SessionComparison represents a comparison between two sessions
type SessionComparison struct {
	Session1         string        `json:"session1"`
	Session2         string        `json:"session2"`
	MessageCountDiff int           `json:"message_count_diff"`
	DurationDiff     time.Duration `json:"duration_diff"`
}

// writeFile is a helper to write data to a file
func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}
