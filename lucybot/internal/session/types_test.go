package session

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSessionWithMetadata(t *testing.T) {
	session := &Session{
		ID:          "test-id",
		Name:        "Test Session",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		AgentName:   "lucybot",
		WorkingDir:  "/home/user/project",
		ModelName:   "gpt-4o",
		LastMessage: "Hello, world!",
		Messages:    []Message{},
	}

	if session.AgentName != "lucybot" {
		t.Errorf("AgentName not set correctly")
	}
	if session.WorkingDir != "/home/user/project" {
		t.Errorf("WorkingDir not set correctly")
	}
	if session.LastMessage != "Hello, world!" {
		t.Errorf("LastMessage not set correctly")
	}
}

func TestMessageContentTypes(t *testing.T) {
	now := time.Now()

	// Test string content
	msg1 := Message{
		Role:      "user",
		Content:   "simple text content",
		Timestamp: now,
	}

	data, err := json.Marshal(msg1)
	if err != nil {
		t.Fatalf("Failed to marshal message with string content: %v", err)
	}

	var decoded1 Message
	if err := json.Unmarshal(data, &decoded1); err != nil {
		t.Fatalf("Failed to unmarshal message with string content: %v", err)
	}

	// Verify content is preserved
	contentStr, ok := decoded1.Content.(string)
	if !ok {
		t.Errorf("Expected string content after round-trip, got %T", decoded1.Content)
	}
	if contentStr != "simple text content" {
		t.Errorf("Content mismatch: got %q, want %q", contentStr, "simple text content")
	}

	// Test structured content (map)
	msg2 := Message{
		Role: "assistant",
		Content: map[string]interface{}{
			"text":    "structured data",
			"tool":    "search",
			"results": []int{1, 2, 3},
		},
		Timestamp: now,
	}

	data, err = json.Marshal(msg2)
	if err != nil {
		t.Fatalf("Failed to marshal message with structured content: %v", err)
	}

	// Verify JSON contains the structured fields
	dataStr := string(data)
	if !contains(dataStr, `"text"`) || !contains(dataStr, `"tool"`) {
		t.Errorf("Marshaled JSON missing structured fields: %s", dataStr)
	}

	var decoded2 Message
	if err := json.Unmarshal(data, &decoded2); err != nil {
		t.Fatalf("Failed to unmarshal message with structured content: %v", err)
	}

	// Verify content is a map
	contentMap, ok := decoded2.Content.(map[string]interface{})
	if !ok {
		t.Errorf("Expected map[string]interface{} content after round-trip, got %T", decoded2.Content)
	} else {
		if contentMap["text"] != "structured data" {
			t.Errorf("Structured content text mismatch: got %v, want 'structured data'", contentMap["text"])
		}
		if contentMap["tool"] != "search" {
			t.Errorf("Structured content tool mismatch: got %v, want 'search'", contentMap["tool"])
		}
	}

	// Test omitempty behavior with empty/zero values
	msg3 := Message{
		Role:      "system",
		Content:   "", // Empty string should be preserved
		Timestamp: now,
	}

	data, err = json.Marshal(msg3)
	if err != nil {
		t.Fatalf("Failed to marshal message with empty content: %v", err)
	}

	var decoded3 Message
	if err := json.Unmarshal(data, &decoded3); err != nil {
		t.Fatalf("Failed to unmarshal message with empty content: %v", err)
	}

	contentStr3, ok := decoded3.Content.(string)
	if !ok {
		t.Errorf("Expected empty string content after round-trip, got %T", decoded3.Content)
	}
	if contentStr3 != "" {
		t.Errorf("Empty content not preserved: got %q", contentStr3)
	}
}

func TestSessionWithStructuredContentMessages(t *testing.T) {
	now := time.Now()

	session := &Session{
		ID:        "test-structured",
		Name:      "Session with Structured Content",
		CreatedAt: now,
		UpdatedAt: now,
		Messages: []Message{
			{
				Role:      "user",
				Content:   "What files are in this directory?",
				Timestamp: now,
			},
			{
				Role: "assistant",
				Content: map[string]interface{}{
					"tool":    "list_directory",
					"path":    "/home/user/project",
					"files":   []string{"main.go", "README.md"},
					"count":   2,
					"success": true,
				},
				Timestamp: now.Add(time.Second),
			},
			{
				Role:      "user",
				Content:   "Show me main.go",
				Timestamp: now.Add(2 * time.Second),
			},
		},
	}

	// Test JSON serialization
	data, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("Failed to marshal session with structured content: %v", err)
	}

	// Test JSON deserialization
	var decoded Session
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal session with structured content: %v", err)
	}

	// Verify message count
	if len(decoded.Messages) != 3 {
		t.Errorf("Message count mismatch: got %d, want 3", len(decoded.Messages))
	}

	// Verify first message (string content)
	contentStr, ok := decoded.Messages[0].Content.(string)
	if !ok {
		t.Errorf("First message content should be string, got %T", decoded.Messages[0].Content)
	}
	if contentStr != "What files are in this directory?" {
		t.Errorf("First message content mismatch: got %q", contentStr)
	}

	// Verify second message (structured content)
	contentMap, ok := decoded.Messages[1].Content.(map[string]interface{})
	if !ok {
		t.Errorf("Second message content should be map, got %T", decoded.Messages[1].Content)
	}
	if contentMap["tool"] != "list_directory" {
		t.Errorf("Tool mismatch: got %v, want 'list_directory'", contentMap["tool"])
	}
	if contentMap["success"] != true {
		t.Errorf("Success flag mismatch: got %v, want true", contentMap["success"])
	}
}

func TestTypeAssertionHandling(t *testing.T) {
	// This test ensures the code properly handles type assertions
	// for the interface{} Content field

	msgString := Message{
		Role:    "user",
		Content: "string content",
	}

	// Test successful type assertion
	if content, ok := msgString.Content.(string); ok {
		if content != "string content" {
			t.Errorf("String content mismatch: %s", content)
		}
	} else {
		t.Error("Type assertion to string should succeed for string content")
	}

	// Test failed type assertion (should not panic)
	msgMap := Message{
		Role:    "user",
		Content: map[string]interface{}{"key": "value"},
	}

	if content, ok := msgMap.Content.(string); ok {
		t.Errorf("Type assertion to string should fail for map content, but got: %s", content)
	}

	// Verify map type assertion succeeds
	if content, ok := msgMap.Content.(map[string]interface{}); ok {
		if content["key"] != "value" {
			t.Errorf("Map content mismatch: %v", content)
		}
	} else {
		t.Error("Type assertion to map should succeed for map content")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
