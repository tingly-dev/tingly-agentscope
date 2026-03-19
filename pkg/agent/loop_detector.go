package agent

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
)

// LoopDetector detects repeated tool calls to prevent infinite loops
type LoopDetector struct {
	maxOccurrences int
	toolCounts     map[string]int // signature -> count
}

// NewLoopDetector creates a new loop detector
// maxOccurrences is the maximum number of times the same tool with same params can be called
func NewLoopDetector(maxOccurrences int) *LoopDetector {
	if maxOccurrences <= 0 {
		maxOccurrences = 3 // Default
	}
	return &LoopDetector{
		maxOccurrences: maxOccurrences,
		toolCounts:     make(map[string]int),
	}
}

// DetectLoop checks if calling this tool would create a loop
// Returns true if the same tool has been called too many times with the same parameters
func (l *LoopDetector) DetectLoop(toolBlock *message.ToolUseBlock) bool {
	if toolBlock == nil {
		return false
	}

	signature := l.getToolSignature(toolBlock)
	l.toolCounts[signature]++
	count := l.toolCounts[signature]

	fmt.Fprintf(os.Stderr, "[LOOP] Tool=%s, Signature=%s, Count=%d/%d\n", toolBlock.Name, signature[:8], count, l.maxOccurrences)

	return count > l.maxOccurrences
}

// Reset clears the detection history
func (l *LoopDetector) Reset() {
	l.toolCounts = make(map[string]int)
}

// getToolSignature generates a unique signature for a tool call
// This includes the tool name and normalized input parameters
func (l *LoopDetector) getToolSignature(toolBlock *message.ToolUseBlock) string {
	// Extract input as map[string]any
	var inputMap map[string]any
	if m, ok := toolBlock.Input.(map[string]any); ok {
		inputMap = m
	}

	// Create a normalized representation of the tool call
	data := struct {
		Name  string         `json:"name"`
		Input map[string]any `json:"input"`
	}{
		Name:  toolBlock.Name,
		Input: inputMap,
	}

	// Sort keys for consistent serialization
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		// Fallback to simple string representation
		return fmt.Sprintf("%s:%v", toolBlock.Name, toolBlock.Input)
	}

	// Hash for compact representation
	hash := sha256.Sum256(jsonBytes)
	return fmt.Sprintf("%x", hash[:8]) // Use first 8 bytes of hash
}

// GetLoopMessage returns a message to send when a loop is detected
func (l *LoopDetector) GetLoopMessage(toolBlock *message.ToolUseBlock) string {
	return fmt.Sprintf(
		"Warning: Detected repeated calls to '%s' with the same parameters. "+
			"The agent appears to be in a loop. Consider providing a final summary "+
			"or using the 'finish' tool to complete the task.",
		toolBlock.Name,
	)
}
