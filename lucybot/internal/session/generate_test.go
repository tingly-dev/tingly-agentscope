package session

import (
	"testing"
	"time"
)

func TestGenerateSessionID(t *testing.T) {
	id := GenerateSessionID("lucybot", "Hello, world!")

	if len(id) != 32 {
		t.Errorf("Expected session ID length of 32, got %d", len(id))
	}

	// Test hex format
	for _, c := range id {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("Session ID contains non-hex character: %c", c)
		}
	}
}

func TestGenerateSessionIDUniqueness(t *testing.T) {
	agent := "lucybot"
	query := "Test query"

	id1 := GenerateSessionID(agent, query)
	time.Sleep(time.Microsecond)
	id2 := GenerateSessionID(agent, query)

	if id1 == id2 {
		t.Error("Session IDs should be unique even with same inputs")
	}
}

func TestGenerateSessionIDWithEmptyQuery(t *testing.T) {
	id := GenerateSessionID("lucybot", "")

	if len(id) != 32 {
		t.Errorf("Expected session ID length of 32 with empty query, got %d", len(id))
	}
}

func TestGenerateSessionIDQueryTruncation(t *testing.T) {
	longQuery := "This is a very long query that exceeds 128 characters. " +
		"It should be truncated to 128 characters when generating " +
		"the session ID to ensure consistent behavior and avoid " +
		"excessively long input strings that could affect performance."

	id1 := GenerateSessionID("lucybot", longQuery)
	id2 := GenerateSessionID("lucybot", longQuery[:128])

	// Both should be valid 32-char hex strings
	if len(id1) != 32 || len(id2) != 32 {
		t.Error("Both IDs should be valid 32-character hex strings")
	}
}
