package main

import (
	_ "embed"
	"strings"
)

// CheckCompletion checks if output contains completion signal
func CheckCompletion(output string) bool {
	return strings.Contains(output, CompletionSignal)
}

// CheckDiscussionComplete checks if output contains discussion complete signal
func CheckDiscussionComplete(output string) bool {
	return strings.Contains(output, DiscussionCompleteSignal)
}

// HasQuestions checks if output contains questions block
func HasQuestions(output string) bool {
	return strings.Contains(output, QuestionsMarker)
}

//go:embed prompts/loop_instructions.md
var defaultInstructions string

//go:embed prompts/spec_to_tasks.md
var specToTasksPrompt string

// CompletionSignal is the marker the agent outputs when all tasks are complete
const CompletionSignal = "<promise>COMPLETE</promise>"

// DiscussionCompleteSignal is the marker the agent outputs when discussion is done
const DiscussionCompleteSignal = "<discussion-complete/>"

// QuestionsMarker is used to wrap batch questions
const QuestionsMarker = "<questions>"
