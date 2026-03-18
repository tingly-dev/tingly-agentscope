package tools

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAskUserQuestionTool_Execute_NonInteractive(t *testing.T) {
	t.Run("with default value", func(t *testing.T) {
		tool := NewAskUserQuestionTool(nil, func() bool { return false })
		result, err := tool.Execute(context.Background(), AskUserQuestionParams{
			Question: "What is your name?",
			Default:  "Anonymous",
		})
		require.NoError(t, err)
		assert.Equal(t, "Anonymous", result)
	})

	t.Run("without default value", func(t *testing.T) {
		tool := NewAskUserQuestionTool(nil, func() bool { return false })
		_, err := tool.Execute(context.Background(), AskUserQuestionParams{
			Question: "What is your name?",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "non-interactive mode")
	})
}

func TestAskUserQuestionTool_Execute_Interactive(t *testing.T) {
	t.Run("with response channel", func(t *testing.T) {
		responseChan := make(ResponseChannel, 1)
		tool := NewAskUserQuestionTool(responseChan, func() bool { return true })

		// Send response asynchronously
		go func() {
			responseChan <- UserResponse{Answer: "John"}
		}()

		result, err := tool.Execute(context.Background(), AskUserQuestionParams{
			Question: "What is your name?",
		})
		require.NoError(t, err)
		assert.Equal(t, "John", result)
	})

	t.Run("user cancelled", func(t *testing.T) {
		responseChan := make(ResponseChannel, 1)
		tool := NewAskUserQuestionTool(responseChan, func() bool { return true })

		// Send cancelled response asynchronously
		go func() {
			responseChan <- UserResponse{Cancelled: true}
		}()

		_, err := tool.Execute(context.Background(), AskUserQuestionParams{
			Question: "What is your name?",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cancelled")
	})

	t.Run("context cancelled", func(t *testing.T) {
		responseChan := make(ResponseChannel)
		tool := NewAskUserQuestionTool(responseChan, func() bool { return true })

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		_, err := tool.Execute(ctx, AskUserQuestionParams{
			Question: "What is your name?",
		})
		require.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})

	t.Run("no response channel with default", func(t *testing.T) {
		tool := NewAskUserQuestionTool(nil, func() bool { return true })
		result, err := tool.Execute(context.Background(), AskUserQuestionParams{
			Question: "What is your name?",
			Default:  "Anonymous",
		})
		require.NoError(t, err)
		assert.Equal(t, "Anonymous", result)
	})

	t.Run("no response channel without default", func(t *testing.T) {
		tool := NewAskUserQuestionTool(nil, func() bool { return true })
		_, err := tool.Execute(context.Background(), AskUserQuestionParams{
			Question: "What is your name?",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no response channel configured")
	})
}

func TestAskUserQuestionTool_Execute_Validation(t *testing.T) {
	t.Run("empty question", func(t *testing.T) {
		tool := NewAskUserQuestionTool(nil, func() bool { return true })
		_, err := tool.Execute(context.Background(), AskUserQuestionParams{
			Question: "",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "question is required")
	})
}

func TestAskUserQuestionTool_Name(t *testing.T) {
	tool := NewAskUserQuestionTool(nil, nil)
	assert.Equal(t, "ask_user_question", tool.Name())
}

func TestAskUserQuestionTool_Description(t *testing.T) {
	tool := NewAskUserQuestionTool(nil, nil)
	assert.Contains(t, tool.Description(), "Ask the user")
}

func TestInformUserTool_Execute(t *testing.T) {
	t.Run("successful display", func(t *testing.T) {
		var receivedMessage string
		var receivedLevel string
		displayFunc := func(message, level string) error {
			receivedMessage = message
			receivedLevel = level
			return nil
		}

		tool := NewInformUserTool(displayFunc)
		err := tool.Execute(context.Background(), InformUserParams{
			Message: "Hello, user!",
			Level:   "info",
		})
		require.NoError(t, err)
		assert.Equal(t, "Hello, user!", receivedMessage)
		assert.Equal(t, "info", receivedLevel)
	})

	t.Run("default level", func(t *testing.T) {
		var receivedLevel string
		displayFunc := func(message, level string) error {
			receivedLevel = level
			return nil
		}

		tool := NewInformUserTool(displayFunc)
		err := tool.Execute(context.Background(), InformUserParams{
			Message: "Hello, user!",
		})
		require.NoError(t, err)
		assert.Equal(t, "info", receivedLevel)
	})

	t.Run("all valid levels", func(t *testing.T) {
		levels := []string{"info", "warning", "error", "success"}
		for _, level := range levels {
			t.Run(level, func(t *testing.T) {
				tool := NewInformUserTool(func(_, _ string) error { return nil })
				err := tool.Execute(context.Background(), InformUserParams{
					Message: "Test",
					Level:   level,
				})
				require.NoError(t, err)
			})
		}
	})

	t.Run("invalid level", func(t *testing.T) {
		tool := NewInformUserTool(nil)
		err := tool.Execute(context.Background(), InformUserParams{
			Message: "Test",
			Level:   "invalid",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid level")
	})

	t.Run("empty message", func(t *testing.T) {
		tool := NewInformUserTool(nil)
		err := tool.Execute(context.Background(), InformUserParams{
			Message: "",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "message is required")
	})

	t.Run("no display function", func(t *testing.T) {
		tool := NewInformUserTool(nil)
		err := tool.Execute(context.Background(), InformUserParams{
			Message: "Hello",
		})
		require.NoError(t, err)
	})
}

func TestInformUserTool_Name(t *testing.T) {
	tool := NewInformUserTool(nil)
	assert.Equal(t, "inform_user", tool.Name())
}

func TestInformUserTool_Description(t *testing.T) {
	tool := NewInformUserTool(nil)
	assert.Contains(t, tool.Description(), "Display an informational message")
}
