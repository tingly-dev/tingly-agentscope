package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Tasks represents the task list for the loop
type Tasks struct {
	Project     string      `json:"project"`
	BranchName  string      `json:"branchName"`
	Description string      `json:"description"`
	UserStories []UserStory `json:"userStories"`
}

// UserStory represents a single user story in the task list
type UserStory struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	AcceptanceCriteria []string `json:"acceptanceCriteria"`
	Priority           int      `json:"priority"`
	Passes             bool     `json:"passes"`
	Notes              string   `json:"notes"`
}

// LoadTasks loads tasks from a JSON file
func LoadTasks(path string) (*Tasks, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read tasks file: %w", err)
	}

	var tasks Tasks
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse tasks JSON: %w", err)
	}

	return &tasks, nil
}

// SaveTasks saves tasks to a JSON file
func SaveTasks(path string, tasks *Tasks) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tasks: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// GetNextStory returns the highest priority story that hasn't passed yet
// Returns nil if all stories have passed
func (t *Tasks) GetNextStory() *UserStory {
	// Filter stories that haven't passed
	var pending []UserStory
	for _, story := range t.UserStories {
		if !story.Passes {
			pending = append(pending, story)
		}
	}

	if len(pending) == 0 {
		return nil
	}

	// Sort by priority (ascending - lower number = higher priority)
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].Priority < pending[j].Priority
	})

	return &pending[0]
}

// AllStoriesPassed returns true if all stories have passed
func (t *Tasks) AllStoriesPassed() bool {
	for _, story := range t.UserStories {
		if !story.Passes {
			return false
		}
	}
	return true
}

// GetPassedCount returns the number of passed stories
func (t *Tasks) GetPassedCount() int {
	count := 0
	for _, story := range t.UserStories {
		if story.Passes {
			count++
		}
	}
	return count
}

// GetTotalCount returns the total number of stories
func (t *Tasks) GetTotalCount() int {
	return len(t.UserStories)
}

// MarkStoryPassed marks a story as passed by ID
func (t *Tasks) MarkStoryPassed(storyID string) bool {
	for i := range t.UserStories {
		if t.UserStories[i].ID == storyID {
			t.UserStories[i].Passes = true
			return true
		}
	}
	return false
}

// FormatStoryList returns a formatted string of all stories with their status
func (t *Tasks) FormatStoryList() string {
	result := "User Stories:\n"
	for _, story := range t.UserStories {
		status := "pending"
		if story.Passes {
			status = "completed"
		}
		result += fmt.Sprintf("  [%s] %s (Priority %d): %s\n",
			status, story.ID, story.Priority, story.Title)
	}
	return result
}
