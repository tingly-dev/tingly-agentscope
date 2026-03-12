package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// UserInteractionTools holds tools for user interaction
type UserInteractionTools struct{}

// NewUserInteractionTools creates a new UserInteractionTools instance
func NewUserInteractionTools() *UserInteractionTools {
	return &UserInteractionTools{}
}

// Tool descriptions for user interaction tools
const (
	ToolDescAskUserQuestion = `Use this tool when you need to ask the user questions during execution. This allows you to:
1. Gather user preferences or requirements
2. Clarify ambiguous instructions
3. Get decisions on implementation choices as you work
4. Offer choices to the user about what direction to take.

Usage notes:
- Users will always be able to select "Other" to provide custom text input
- Use multiSelect: true to allow multiple answers to be selected for a question
- If you recommend a specific option, make that the first option in the list and add "(Recommended)" at the end of the label

Plan mode note: In plan mode, use this tool to clarify requirements or choose between approaches BEFORE finalizing your plan. Do NOT use this tool to ask "Is my plan ready?" or "Should I proceed?" - use ExitPlanMode for plan approval.`
)

// QuestionOption represents an option for a user question
type QuestionOption struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// Question represents a single question to ask the user
type Question struct {
	Header      string           `json:"header"`
	Question    string           `json:"question"`
	Options     []QuestionOption `json:"options"`
	MultiSelect bool             `json:"multi_select"`
}

// AskUserQuestionParams holds parameters for AskUserQuestion
type AskUserQuestionParams struct {
	Questions []Question     `json:"questions" required:"true" description:"Questions to ask the user (1-4 questions)"`
	Metadata  map[string]any `json:"metadata,omitempty" description:"Optional metadata for tracking"`
}

// UserAnswers represents the user's answers
type UserAnswers struct {
	Answers map[string]string `json:"answers"`
}

// AskUserQuestion asks the user questions and returns their answers
//
// Note: In a real CLI environment, this would prompt the user interactively.
// For demonstration purposes, this implementation reads from environment variables
// or provides instructions on how answers should be provided.
func (uit *UserInteractionTools) AskUserQuestion(ctx context.Context, params AskUserQuestionParams) (string, error) {
	if len(params.Questions) == 0 || len(params.Questions) > 4 {
		return "Error: must provide 1-4 questions", nil
	}

	// In a real implementation, this would interact with the user via stdin/stdout
	// For now, we check for environment variables or return instructions

	var output []string
	output = append(output, "=== User Questions ===\n")

	answers := make(map[string]string)

	for i, q := range params.Questions {
		output = append(output, fmt.Sprintf("[%s] %s", q.Header, q.Question))
		output = append(output, "")

		if len(q.Options) > 0 {
			for j, opt := range q.Options {
				label := opt.Label
				if q.MultiSelect {
					output = append(output, fmt.Sprintf("  [%d] %s - %s", j+1, label, opt.Description))
				} else {
					output = append(output, fmt.Sprintf("  (%c) %s - %s", 'A'+j, label, opt.Description))
				}
			}
			output = append(output, "")
		}

		// Check for environment variable answer (format: QUESTION_<N>)
		envKey := fmt.Sprintf("QUESTION_%d", i)
		if answer := os.Getenv(envKey); answer != "" {
			answers[envKey] = answer
			output = append(output, fmt.Sprintf("  Answer (from env): %s\n", answer))
		} else {
			// Provide instruction for setting answer
			output = append(output, fmt.Sprintf("  [Set answer via: export %s=<your_answer>]\n", envKey))
		}
	}

	output = append(output, "=== Instructions ===")
	output = append(output, "To provide answers in this environment:")
	output = append(output, "1. Set environment variables: export QUESTION_0=<answer>")
	output = append(output, "2. Re-run the command")

	result := map[string]any{
		"questions": params.Questions,
		"answers":   answers,
		"message":   "In interactive mode, user would be prompted for answers",
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	output = append(output, "\n"+string(data))

	return fmt.Sprintf("%s", strings.Join(output, "\n")), nil
}

// InteractiveAskUserQuestion provides an interactive prompt for user questions
// This would be used in a terminal context
func (uit *UserInteractionTools) InteractiveAskUserQuestion(ctx context.Context, params AskUserQuestionParams) (map[string]string, error) {
	answers := make(map[string]string)

	// For each question, prompt the user
	for i, q := range params.Questions {
		fmt.Printf("\n[%s] %s\n", q.Header, q.Question)

		if len(q.Options) > 0 {
			fmt.Println("Options:")
			for j, opt := range q.Options {
				fmt.Printf("  %c. %s - %s\n", 'A'+j, opt.Label, opt.Description)
			}
			fmt.Println("  Other. Custom input")
		}

		// Prompt for answer
		fmt.Print("Your choice: ")
		var answer string
		fmt.Scanln(&answer)

		answers[fmt.Sprintf("question_%d", i)] = answer
	}

	return answers, nil
}

func init() {
	// Register user interaction tools in the global registry
	RegisterTool("ask_user_question", ToolDescAskUserQuestion, "User Interaction", true)
}
