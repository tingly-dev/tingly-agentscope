package session

import (
	"testing"
	"time"
)

func TestSessionWithMetadata(t *testing.T) {
	session := &Session{
		ID:            "test-id",
		Name:          "Test Session",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		AgentName:     "lucybot",
		WorkingDir:    "/home/user/project",
		ModelName:     "gpt-4o",
		LastMessage:   "Hello, world!",
		Messages:      []Message{},
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
