package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// DualActAgent implements a dual-act (human-like + reactive) agent pattern
// It orchestrates two ReActAgent instances: a human-like decision agent (H) and a reactive execution agent (R)
type DualActAgent struct {
	*AgentBase
	config *DualActConfig
	mu     sync.RWMutex
}

// NewDualActAgent creates a new dual act agent with the given configuration
func NewDualActAgent(config *DualActConfig) *DualActAgent {
	if err := config.Validate(); err != nil {
		panic(fmt.Sprintf("invalid dual act config: %v", err))
	}

	base := NewAgentBase("dualact", "You are a dual act agent coordinating a human-like planner and a reactive executor.")

	return &DualActAgent{
		AgentBase: base,
		config:    config,
	}
}

// NewDualActAgentWithOptions creates a new dual act agent with functional options
func NewDualActAgentWithOptions(human, reactive *ReActAgent, opts ...DualActOption) *DualActAgent {
	config := DefaultDualActConfig(human, reactive)
	config.ApplyOptions(opts)

	if err := config.Validate(); err != nil {
		panic(fmt.Sprintf("invalid dual act config: %v", err))
	}

	base := NewAgentBase("dualact", "You are a dual act agent coordinating a human-like planner and a reactive executor.")

	return &DualActAgent{
		AgentBase: base,
		config:    config,
	}
}

// Reply generates a response to the given message by orchestrating H and R agents
func (d *DualActAgent) Reply(ctx context.Context, input *message.Msg) (*message.Msg, error) {
	// Run pre-reply hooks
	kwargs := map[string]any{"message": input}
	if err := d.runPreHooks(ctx, types.HookTypePreReply, input, kwargs); err != nil {
		return nil, err
	}

	// Store original input
	originalInput := input

	// Track the current instruction for R
	var currentInstruction *message.Msg = input

	// Run H-R loop
	var finalConclusion *Conclusion
	var lastDecision *HumanDecision

	for loopNum := 0; loopNum < d.config.MaxHRLoops; loopNum++ {
		d.log("=== H-R Loop %d ===", loopNum+1)

		// On first iteration, H evaluates the original input directly
		// On subsequent iterations, H evaluates R's conclusion
		if loopNum > 0 {
			// H evaluates R's conclusion
			decision, err := d.evaluateConclusion(ctx, finalConclusion, originalInput)
			if err != nil {
				return nil, fmt.Errorf("human agent evaluation failed: %w", err)
			}
			lastDecision = decision

			d.log("Human decision: %s", decision.Action)
			if decision.Reasoning != "" {
				d.log("Reasoning: %s", decision.Reasoning)
			}

			// Check if H wants to terminate
			if decision.ShouldTerminate() {
				d.log("Terminating by human decision")
				return d.createFinalResponse(finalConclusion, lastDecision, originalInput), nil
			}

			// H wants to continue or redirect - create new instruction for R
			currentInstruction = d.formatInstruction(ctx, decision)
		}

		// R executes the task
		conclusion, err := d.runReactiveLoop(ctx, currentInstruction)
		if err != nil {
			return nil, fmt.Errorf("reactive agent execution failed: %w", err)
		}
		finalConclusion = conclusion

		d.log("Reactive conclusion: %s (confidence: %.2f)", conclusion.Summary, conclusion.Confidence)
	}

	// Max loops reached - use H's final decision or return current conclusion
	if lastDecision != nil && lastDecision.ShouldTerminate() {
		return d.createFinalResponse(finalConclusion, lastDecision, originalInput), nil
	}

	// Max loops reached without explicit termination - return best result
	d.log("Max loops reached, returning current conclusion")
	return d.createFinalResponse(finalConclusion, &HumanDecision{
		Action:    DecisionActionTerminate,
		Reasoning: "Maximum H-R loops reached",
	}, originalInput), nil
}

// Observe receives a message without generating a response
func (d *DualActAgent) Observe(ctx context.Context, msg *message.Msg) error {
	return d.AgentBase.Observe(ctx, msg)
}

// GetHumanAgent returns the human-like decision agent
func (d *DualActAgent) GetHumanAgent() *ReActAgent {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.config.Human
}

// GetReactiveAgent returns the reactive execution agent
func (d *DualActAgent) GetReactiveAgent() *ReActAgent {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.config.Reactive
}

// GetConfig returns the dual act configuration
func (d *DualActAgent) GetConfig() *DualActConfig {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.config
}

// runReactiveLoop runs the reactive agent in a continuous loop until it reaches a conclusion
func (d *DualActAgent) runReactiveLoop(ctx context.Context, instruction *message.Msg) (*Conclusion, error) {
	d.log("Starting reactive execution...")

	response, err := d.config.Reactive.Reply(ctx, instruction)
	if err != nil {
		return nil, err
	}

	// Extract conclusion from R's response
	conclusion := d.extractConclusion(response)
	conclusion.OriginalInput = instruction.GetTextContent()

	// Get iteration count from the last response
	if lastResp := d.config.Reactive.GetLastResponse(); lastResp != nil {
		// Estimate iterations based on tool uses in response
		conclusion.Iterations = d.countToolUses(lastResp.Content)
	}

	d.log("Reactive completed with %d iterations", conclusion.Iterations)

	return conclusion, nil
}

// evaluateConclusion has the human agent evaluate R's conclusion and decide next action
func (d *DualActAgent) evaluateConclusion(ctx context.Context, conclusion *Conclusion, originalInput *message.Msg) (*HumanDecision, error) {
	d.log("Human agent evaluating conclusion...")

	// Format the conclusion for H to evaluate
	evalMsg := d.formatConclusionForHuman(conclusion, originalInput)

	response, err := d.config.Human.Reply(ctx, evalMsg)
	if err != nil {
		return nil, err
	}

	// Parse H's decision from response
	decision := d.parseDecision(response.GetTextContent())

	return decision, nil
}

// formatInstruction formats an instruction message for the reactive agent
func (d *DualActAgent) formatInstruction(ctx context.Context, decision *HumanDecision) *message.Msg {
	var content string

	if d.config.ReactiveTaskPrompt != "" {
		// Use custom prompt template
		content = fmt.Sprintf(d.config.ReactiveTaskPrompt, decision.NewInstruction)
	} else {
		// Default formatting
		if decision.IsRedirect() {
			content = fmt.Sprintf("NEW APPROACH: %s\n\n%s", decision.ModifiedApproach, decision.NewInstruction)
		} else {
			content = decision.NewInstruction
		}
	}

	return message.NewMsg(
		"dualact",
		content,
		types.RoleUser,
	)
}

// formatConclusionForHuman formats R's conclusion for H's evaluation
func (d *DualActAgent) formatConclusionForHuman(conclusion *Conclusion, originalInput *message.Msg) *message.Msg {
	var content string

	if d.config.HumanDecisionPrompt != "" {
		// Use custom prompt template
		content = fmt.Sprintf(d.config.HumanDecisionPrompt,
			originalInput.GetTextContent(),
			conclusion.Summary,
			strings.Join(conclusion.Steps, "\n"),
			conclusion.Confidence,
		)
	} else {
		// Default formatting
		content = fmt.Sprintf(`## Execution Review

**Original Task:** %s

**Work Summary:** %s

**Steps Taken:**
%s

**Confidence:** %.2f

**Suggested Next Action:** %s

---
Please evaluate this work and decide:
- TERMINATE: If the task is complete and satisfactory
- CONTINUE: If more work is needed (provide next instruction)
- REDIRECT: If the approach needs to change (explain new approach)

Respond with your decision and reasoning.`,
			originalInput.GetTextContent(),
			conclusion.Summary,
			formatSteps(conclusion.Steps),
			conclusion.Confidence,
			conclusion.SuggestedNextAction,
		)
	}

	return message.NewMsg(
		"dualact",
		content,
		types.RoleUser,
	)
}

// extractConclusion extracts a conclusion from the reactive agent's response
func (d *DualActAgent) extractConclusion(response *message.Msg) *Conclusion {
	conclusion := NewConclusion()
	conclusion.Summary = response.GetTextContent()

	// Try to extract structured information from the response
	content := response.GetTextContent()

	// Look for confidence indicators
	if strings.Contains(strings.ToLower(content), "done") ||
		strings.Contains(strings.ToLower(content), "complete") ||
		strings.Contains(strings.ToLower(content), "finished") {
		conclusion.Confidence = 0.9
	} else if strings.Contains(strings.ToLower(content), "error") ||
		strings.Contains(strings.ToLower(content), "failed") {
		conclusion.Confidence = 0.2
	} else {
		conclusion.Confidence = 0.5
	}

	// Extract steps if response contains numbered lists
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 0 && (trimmed[0] == '-' || (len(trimmed) > 2 && trimmed[0:2] == "1.")) {
			conclusion.AddStep(trimmed)
		}
	}

	return conclusion
}

// parseDecision parses a human decision from text
func (d *DualActAgent) parseDecision(text string) *HumanDecision {
	decision := NewHumanDecision(DecisionActionContinue)
	lowerText := strings.ToLower(text)

	// Determine action
	if strings.Contains(lowerText, "terminate") ||
		strings.Contains(lowerText, "done") ||
		strings.Contains(lowerText, "complete") ||
		strings.Contains(lowerText, "finished") {
		decision.Action = DecisionActionTerminate
	} else if strings.Contains(lowerText, "redirect") ||
		strings.Contains(lowerText, "change approach") ||
		strings.Contains(lowerText, "different approach") {
		decision.Action = DecisionActionRedirect
	}

	// Extract reasoning - look for reasoning section in the response
	// Try to find section markers like "Reasoning:" or "**Reasoning:**"
	lines := strings.Split(text, "\n")
	inReasoning := false
	var reasoningLines []string

	for _, line := range lines {
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "reasoning") || strings.Contains(lowerLine, "**reasoning**") {
			inReasoning = true
			// Skip the header line itself
			if strings.Contains(line, ":") {
				// Get content after the colon
				if idx := strings.Index(line, ":"); idx >= 0 && idx < len(line)-1 {
					content := strings.TrimSpace(line[idx+1:])
					// Clean up markdown formatting
					content = strings.TrimPrefix(content, "**")
					content = strings.TrimSuffix(content, "**")
					content = strings.TrimSpace(content)
					if content != "" {
						reasoningLines = append(reasoningLines, content)
					}
				}
			}
			continue
		}
		if inReasoning {
			// Stop if we hit another section
			if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "##") ||
				(strings.HasPrefix(line, "**") && strings.Contains(line, ":**")) {
				break
			}
			trimmed := strings.TrimSpace(line)
			// Clean up list item markers
			trimmed = strings.TrimPrefix(trimmed, "*")
			trimmed = strings.TrimPrefix(trimmed, "-")
			trimmed = strings.TrimSpace(trimmed)
			if trimmed != "" {
				reasoningLines = append(reasoningLines, trimmed)
			}
		}
	}

	if len(reasoningLines) > 0 {
		decision.Reasoning = strings.Join(reasoningLines, " ")
	}

	// Extract new instruction for continue/redirect
	if decision.Action != DecisionActionTerminate {
		for i, line := range lines {
			lowerLine := strings.ToLower(line)
			if strings.Contains(lowerLine, "next") ||
				strings.Contains(lowerLine, "instruction") ||
				strings.Contains(lowerLine, "do") {
				if i+1 < len(lines) {
					decision.NewInstruction = strings.TrimSpace(lines[i+1])
				}
				break
			}
		}
	}

	// If no instruction found, use full text as instruction
	if decision.NewInstruction == "" && decision.Action != DecisionActionTerminate {
		decision.NewInstruction = strings.TrimSpace(text)
	}

	return decision
}

// createFinalResponse creates the final response message to return to the user
func (d *DualActAgent) createFinalResponse(conclusion *Conclusion, decision *HumanDecision, originalInput *message.Msg) *message.Msg {
	var content strings.Builder

	content.WriteString(fmt.Sprintf("## Task: %s\n\n", originalInput.GetTextContent()))
	content.WriteString(fmt.Sprintf("**Summary:** %s\n\n", conclusion.Summary))

	if len(conclusion.Steps) > 0 {
		content.WriteString("**Steps Taken:**\n")
		for _, step := range conclusion.Steps {
			content.WriteString(fmt.Sprintf("  %s\n", step))
		}
		content.WriteString("\n")
	}

	if decision.Reasoning != "" {
		content.WriteString(fmt.Sprintf("**Final Decision:** %s\n\n", decision.Reasoning))
	}

	return message.NewMsg(
		d.Name(),
		content.String(),
		types.RoleAssistant,
	)
}

// formatSteps formats steps for display
func formatSteps(steps []string) string {
	if len(steps) == 0 {
		return "No detailed steps recorded."
	}
	var sb strings.Builder
	for i, step := range steps {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
	}
	return sb.String()
}

// countToolUses counts the number of tool use blocks in content
func (d *DualActAgent) countToolUses(content []message.ContentBlock) int {
	count := 0
	for _, block := range content {
		if _, ok := block.(*message.ToolUseBlock); ok {
			count++
		}
	}
	return count
}

// log logs a message if verbose logging is enabled
func (d *DualActAgent) log(format string, args ...any) {
	if d.config.EnableVerboseLogging {
		fmt.Printf("[DualAct] %s\n", fmt.Sprintf(format, args...))
	}
}

// StateDict returns the agent's state for serialization
func (d *DualActAgent) StateDict() map[string]any {
	state := d.AgentBase.StateDict()
	state["config"] = map[string]any{
		"max_hr_loops": d.config.MaxHRLoops,
	}
	return state
}

// LoadStateDict loads the agent's state
func (d *DualActAgent) LoadStateDict(ctx context.Context, state map[string]any) error {
	if err := d.AgentBase.LoadStateDict(state); err != nil {
		return err
	}
	return nil
}
