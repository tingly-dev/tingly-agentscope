package tools

import (
	"context"
	"errors"
	"fmt"
)

// UserResponse represents a user's response to a question
type UserResponse struct {
	Answer    string
	Cancelled bool
}

// ResponseChannel is used to communicate between tool and UI
type ResponseChannel chan UserResponse

// AskUserQuestionParams holds parameters for asking the user a question
type AskUserQuestionParams struct {
	Question string   `json:"question" description:"The question to ask the user"`
	Options  []string `json:"options,omitempty" description:"Optional list of options for the user to choose from"`
	Default  string   `json:"default,omitempty" description:"Default value to return in non-interactive mode"`
}

// AskUserQuestionTool allows the agent to ask the user a question
type AskUserQuestionTool struct {
	responseChan  ResponseChannel
	isInteractive func() bool
}

// NewAskUserQuestionTool creates a new AskUserQuestionTool
func NewAskUserQuestionTool(responseChan ResponseChannel, isInteractive func() bool) *AskUserQuestionTool {
	return &AskUserQuestionTool{
		responseChan:  responseChan,
		isInteractive: isInteractive,
	}
}

// Name returns the tool name
func (t *AskUserQuestionTool) Name() string {
	return "ask_user_question"
}

// Description returns the tool description
func (t *AskUserQuestionTool) Description() string {
	return "Ask the user a question and wait for their response. Supports multiple choice or free text input."
}

// Execute asks the user a question and waits for their response
func (t *AskUserQuestionTool) Execute(ctx context.Context, params AskUserQuestionParams) (string, error) {
	// Check if we're in interactive mode
	if t.isInteractive != nil && !t.isInteractive() {
		// Non-interactive mode: return default if provided
		if params.Default != "" {
			return params.Default, nil
		}
		return "", errors.New("cannot ask question in non-interactive mode without a default value")
	}

	// Validate parameters
	if params.Question == "" {
		return "", errors.New("question is required")
	}

	// If we have a response channel, use it to communicate with the UI
	if t.responseChan != nil {
		select {
		case response := <-t.responseChan:
			if response.Cancelled {
				return "", errors.New("user cancelled the question")
			}
			return response.Answer, nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	// If no response channel, we can't ask the question
	if params.Default != "" {
		return params.Default, nil
	}

	return "", errors.New("no response channel configured for interactive mode")
}

// AskUserQuestionResult represents the result of asking the user a question
type AskUserQuestionResult struct {
	Answer    string `json:"answer"`
	Cancelled bool   `json:"cancelled"`
}

// InformUserParams holds parameters for informing the user
type InformUserParams struct {
	Message string `json:"message" description:"The message to display to the user"`
	Level   string `json:"level,omitempty" description:"Message level: info, warning, error, success (default: info)"`
}

// InformUserTool allows the agent to inform the user of something without asking a question
type InformUserTool struct {
	displayFunc func(message string, level string) error
}

// NewInformUserTool creates a new InformUserTool
func NewInformUserTool(displayFunc func(message string, level string) error) *InformUserTool {
	return &InformUserTool{
		displayFunc: displayFunc,
	}
}

// Name returns the tool name
func (t *InformUserTool) Name() string {
	return "inform_user"
}

// Description returns the tool description
func (t *InformUserTool) Description() string {
	return "Display an informational message to the user without waiting for a response"
}

// Execute displays a message to the user
func (t *InformUserTool) Execute(ctx context.Context, params InformUserParams) error {
	if params.Message == "" {
		return errors.New("message is required")
	}

	level := params.Level
	if level == "" {
		level = "info"
	}

	// Validate level
	validLevels := map[string]bool{"info": true, "warning": true, "error": true, "success": true}
	if !validLevels[level] {
		return fmt.Errorf("invalid level: %s (must be info, warning, error, or success)", level)
	}

	if t.displayFunc != nil {
		return t.displayFunc(params.Message, level)
	}

	// If no display function, just return success (message would be logged elsewhere)
	return nil
}
