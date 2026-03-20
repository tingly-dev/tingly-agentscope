package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/plan"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// ReActAgentConfig holds the configuration for a ReActAgent
type ReActAgentConfig struct {
	Name          string
	SystemPrompt  string
	Model         model.ChatModel
	Toolkit       tool.ToolProvider
	Memory        Memory
	MaxIterations int
	Temperature   *float64
	MaxTokens     *int
	Compression   *CompressionConfig
	PlanNotebook  *plan.PlanNotebook
	InjectorChain *message.InjectorChain // Message injection hook chain
	Streaming     *StreamingConfig       // Streaming callback configuration
}

// ReActAgent implements the ReAct (Reasoning + Acting) pattern
type ReActAgent struct {
	*AgentBase
	config       *ReActAgentConfig
	messages     []*message.Msg
	lastResponse *model.ChatResponse
}

// NewReActAgent creates a new ReAct agent
func NewReActAgent(config *ReActAgentConfig) *ReActAgent {
	base := NewAgentBase(config.Name, config.SystemPrompt)

	return &ReActAgent{
		AgentBase: base,
		config:    config,
		messages:  make([]*message.Msg, 0),
	}
}

// SetStreamingConfig sets the streaming configuration for real-time message display
func (r *ReActAgent) SetStreamingConfig(streaming *StreamingConfig) {
	r.config.Streaming = streaming
}

// SetCompressionConfig sets the compression configuration
func (r *ReActAgent) SetCompressionConfig(compression *CompressionConfig) {
	r.config.Compression = compression
}

// SetMemory sets the memory for the agent
func (r *ReActAgent) SetMemory(mem Memory) {
	r.config.Memory = mem
}

// Reply generates a response to the given message
func (r *ReActAgent) Reply(ctx context.Context, input *message.Msg) (*message.Msg, error) {
	// Start a new round when receiving a user message
	if f, ok := r.AgentBase.formatter.(interface{ NextRound() }); ok {
		f.NextRound()
	}

	// For persistent mode injectors, apply injection before saving to memory
	// This saves the task snapshot to metadata while keeping original content
	inputToSave := input
	if r.config.InjectorChain != nil {
		// Apply injectors to get metadata (with task snapshots)
		injectedInput := r.config.InjectorChain.ApplyAll(ctx, input)
		// Copy metadata from injected version to original
		// This preserves task snapshots for persistent mode
		if injectedInput != input && injectedInput.Metadata != nil {
			if inputToSave.Metadata == nil {
				inputToSave.Metadata = make(map[string]any)
			}
			for k, v := range injectedInput.Metadata {
				inputToSave.Metadata[k] = v
			}
		}
	}

	// Add input to memory (with metadata from persistent injectors)
	if r.config.Memory != nil {
		if err := r.config.Memory.Add(ctx, inputToSave); err != nil {
			return nil, fmt.Errorf("failed to add message to memory: %w", err)
		}
	}

	// Check if memory compression is needed
	if r.ShouldCompressMemory(ctx) {
		if _, err := r.compressMemory(ctx); err != nil {
			// Log error but continue
			fmt.Printf("Warning: memory compression failed: %v\n", err)
		}
	}

	// Build message history
	messages := r.buildMessageHistory(input)

	// Run pre-reply hooks
	kwargs := map[string]any{"message": input}
	if err := r.runPreHooks(ctx, types.HookTypePreReply, input, kwargs); err != nil {
		return nil, err
	}

	var response *message.Msg
	var err error

	// Check if we have tools and need to do ReAct loop
	if r.config.Toolkit != nil && len(r.config.Toolkit.GetSchemas()) > 0 {
		response, err = r.reactLoop(ctx, messages)
		if err != nil {
			return nil, err
		}
	} else {
		// Simple chat without tools
		resp, err := r.callModel(ctx, messages)
		if err != nil {
			return nil, err
		}
		response = r.createResponseMessage(resp)
	}

	// Add response to memory
	if r.config.Memory != nil {
		if err := r.config.Memory.Add(ctx, response); err != nil {
			return nil, fmt.Errorf("failed to add response to memory: %w", err)
		}
	}

	// Print the response
	if err := r.Print(ctx, response); err != nil {
		return nil, err
	}

	// Run post-reply hooks
	if err := r.runPostHooks(ctx, types.HookTypePostReply, input, kwargs); err != nil {
		return nil, err
	}

	// Broadcast to subscribers
	if err := r.BroadcastToSubscribers(ctx, response); err != nil {
		return nil, err
	}

	return response, nil
}

// reactLoop implements the ReAct loop of thought, action, observation
func (r *ReActAgent) reactLoop(ctx context.Context, initialMessages []*message.Msg) (*message.Msg, error) {
	messages := make([]*message.Msg, len(initialMessages))
	copy(messages, initialMessages)

	// Initialize loop detector to prevent infinite loops
	loopDetector := NewLoopDetector(3)

	for i := 0; i < r.config.MaxIterations; i++ {
		// Get tools schema
		tools := r.config.Toolkit.GetSchemas()

		// Call model with tools
		resp, err := r.callModelWithTools(ctx, messages, tools)
		if err != nil {
			return nil, fmt.Errorf("iteration %d: %w", i, err)
		}

		r.lastResponse = resp

		// Check for tool use blocks
		toolBlocks := resp.GetToolUseBlocks()

		if len(toolBlocks) == 0 {
			// No more tools to use, return the final response
			finalMsg := r.createResponseMessage(resp)
			return finalMsg, nil
		}

		// Increment step counter for assistant response
		if f, ok := r.AgentBase.formatter.(interface{ NextStep() }); ok {
			f.NextStep()
		}

		// Create and print assistant message with tool uses for streaming output
		asstMsg := message.NewMsg(
			r.Name(),
			resp.Content,
			types.RoleAssistant,
		)
		if err := r.Print(ctx, asstMsg); err != nil {
			return nil, fmt.Errorf("failed to print assistant message: %w", err)
		}

		// Stream the assistant message via callback for TUI/real-time display
		if r.config.Streaming != nil {
			r.config.Streaming.SafeInvoke(asstMsg)
		}

		// Execute each tool
		for _, toolBlock := range toolBlocks {
			// Check for loop detection
			if loopDetector.DetectLoop(toolBlock) {
				loopMsg := message.NewMsg(
					r.Name(),
					[]message.ContentBlock{message.Text(loopDetector.GetLoopMessage(toolBlock))},
					types.RoleAssistant,
				)
				// Stream the warning
				if r.config.Streaming != nil {
					r.config.Streaming.SafeInvoke(loopMsg)
				}
				// Return with the loop warning
				return loopMsg, nil
			}

			// toolBlock is already *message.ToolUseBlock, no conversion needed
			// Add tool use to messages
			toolMsg := message.NewMsg(
				r.Name(),
				[]message.ContentBlock{toolBlock},
				types.RoleAssistant,
			)
			messages = append(messages, toolMsg)

			// Add tool message to memory for session persistence
			if r.config.Memory != nil {
				if err := r.config.Memory.Add(ctx, toolMsg); err != nil {
					return nil, fmt.Errorf("failed to add tool message to memory: %w", err)
				}
			}

			// Print tool use for streaming output
			if err := r.Print(ctx, toolMsg); err != nil {
				return nil, fmt.Errorf("failed to print tool use: %w", err)
			}

			// Stream the tool use via callback for TUI/real-time display
			if r.config.Streaming != nil {
				r.config.Streaming.SafeInvoke(toolMsg)
			}

			// Execute tool
			toolResp, err := r.config.Toolkit.Call(ctx, toolBlock)
			if err != nil {
				// Tool execution failed, create error result message
				errorResultMsg := message.NewMsg(
					toolBlock.Name,
					[]message.ContentBlock{
						&message.ToolResultBlock{
							ID:     toolBlock.ID,
							Name:   toolBlock.Name,
							Output: []message.ContentBlock{message.Text(fmt.Sprintf("Error: %v", err))},
						},
					},
					types.RoleUser,
				)
				// Increment step counter for tool result
				if f, ok := r.AgentBase.formatter.(interface{ NextStep() }); ok {
					f.NextStep()
				}
				// Print error result for streaming output
				if err := r.Print(ctx, errorResultMsg); err != nil {
					return nil, fmt.Errorf("failed to print tool error: %w", err)
				}

				// Stream the error result via callback
				if r.config.Streaming != nil {
					r.config.Streaming.SafeInvoke(errorResultMsg)
				}

				messages = append(messages, errorResultMsg)

				// Add error result to memory for session persistence
				if r.config.Memory != nil {
					if err := r.config.Memory.Add(ctx, errorResultMsg); err != nil {
						return nil, fmt.Errorf("failed to add error result to memory: %w", err)
					}
				}
				continue
			}

			// Add tool result to messages (wrapping in ToolResultBlock with tool call ID)
			resultMsg := message.NewMsg(
				toolBlock.Name,
				[]message.ContentBlock{
					&message.ToolResultBlock{
						ID:     toolBlock.ID,
						Name:   toolBlock.Name,
						Output: toolResp.Content,
					},
				},
				types.RoleUser,
			)
			messages = append(messages, resultMsg)

			// Add tool result to memory for session persistence
			if r.config.Memory != nil {
				if err := r.config.Memory.Add(ctx, resultMsg); err != nil {
					return nil, fmt.Errorf("failed to add tool result to memory: %w", err)
				}
			}

			// Increment step counter for tool result
			if f, ok := r.AgentBase.formatter.(interface{ NextStep() }); ok {
				f.NextStep()
			}

			// Print tool result for streaming output
			if err := r.Print(ctx, resultMsg); err != nil {
				return nil, fmt.Errorf("failed to print tool result: %w", err)
			}

			// Stream the tool result via callback for TUI/real-time display
			if r.config.Streaming != nil {
				r.config.Streaming.SafeInvoke(resultMsg)
			}
		}
	}

	// Max iterations reached - make one final call without tools to get summary
	resp, err := r.callModel(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("final summary call failed: %w", err)
	}

	return r.createResponseMessage(resp), nil
}

// callModel calls the model with the given messages
func (r *ReActAgent) callModel(ctx context.Context, messages []*message.Msg) (*model.ChatResponse, error) {
	options := &model.CallOptions{
		Temperature: r.config.Temperature,
		MaxTokens:   r.config.MaxTokens,
	}

	if r.config.Model.IsStreaming() {
		// For tool calls, use non-streaming mode to avoid complexity
		// Streaming tool calls require careful handling of partial tool use blocks
		return r.config.Model.Call(ctx, messages, options)
	}

	return r.config.Model.Call(ctx, messages, options)
}

// callModelWithTools calls the model with tools enabled
func (r *ReActAgent) callModelWithTools(ctx context.Context, messages []*message.Msg, tools []model.ToolDefinition) (*model.ChatResponse, error) {
	options := &model.CallOptions{
		ToolChoice:  types.ToolChoiceAuto,
		Tools:       tools,
		Temperature: r.config.Temperature,
		MaxTokens:   r.config.MaxTokens,
	}

	if r.config.Model.IsStreaming() {
		// For tool calls, use non-streaming mode to avoid complexity
		// Streaming tool calls require careful handling of partial tool use blocks
		return r.config.Model.Call(ctx, messages, options)
	}

	return r.config.Model.Call(ctx, messages, options)
}

// buildMessageHistory builds the message history for the model call
func (r *ReActAgent) buildMessageHistory(input *message.Msg) []*message.Msg {
	messages := make([]*message.Msg, 0, len(r.messages)+2)

	// Add system prompt
	if r.config.SystemPrompt != "" {
		sysMsg := message.NewMsg(
			"system",
			r.buildSystemPrompt(),
			types.RoleSystem,
		)
		messages = append(messages, sysMsg)
	}

	// Add memory messages
	// For persistent injectors, restore from metadata snapshots
	if r.config.Memory != nil {
		memMessages := r.config.Memory.GetMessages()
		for _, msg := range memMessages {
			// Check if message has task snapshot metadata
			if msg.Metadata != nil {
				if _, hasSnapshot := msg.Metadata["_task_snapshot"]; hasSnapshot {
					// Restore injection from snapshot
					msg = r.restoreFromSnapshot(msg)
				}
			}
			messages = append(messages, msg)
		}
	}

	// Apply injection chain to input message
	inputMsg := input
	if r.config.InjectorChain != nil {
		inputMsg = r.config.InjectorChain.ApplyAll(context.Background(), input)
	}
	messages = append(messages, inputMsg)

	return messages
}

// restoreFromSnapshot restores message content from task snapshot metadata
// This is used for persistent mode to reconstruct the injected content
func (r *ReActAgent) restoreFromSnapshot(msg *message.Msg) *message.Msg {
	if msg.Metadata == nil {
		return msg
	}

	snapshotRaw, ok := msg.Metadata["_task_snapshot"]
	if !ok {
		return msg
	}

	snapshot, ok := snapshotRaw.(map[string]any)
	if !ok {
		return msg
	}

	// Format the task summary from snapshot
	summary := r.formatSnapshotSummary(snapshot)

	// Get original content blocks
	blocks := msg.GetContentBlocks()

	// Create new blocks with summary prepended
	newBlocks := make([]message.ContentBlock, 0, len(blocks)+1)
	newBlocks = append(newBlocks, message.Text(summary))
	newBlocks = append(newBlocks, blocks...)

	// Create a new message with the injected content
	injectedMsg := message.NewMsgWithTimestamp(
		msg.Name,
		newBlocks,
		msg.Role,
		msg.Timestamp,
	)
	injectedMsg.ID = msg.ID
	injectedMsg.Metadata = msg.Metadata
	injectedMsg.InvocationID = msg.InvocationID

	return injectedMsg
}

// formatSnapshotSummary formats task summary from snapshot metadata
func (r *ReActAgent) formatSnapshotSummary(snapshot map[string]any) string {
	var parts []string
	parts = append(parts, "<system-reminder>")
	parts = append(parts, "# Task Progress")

	total := 0
	if v, ok := snapshot["total"].(int); ok {
		total = v
	}
	completed := 0
	if c, ok := snapshot["completed"].([]string); ok {
		completed = len(c)
	}

	// Summary line
	percent := 0
	if total > 0 {
		percent = (completed * 100) / total
	}
	parts = append(parts, fmt.Sprintf("**Progress:** %d/%d completed (%d%%)", completed, total, percent))

	// Current in-progress task
	if inProgress, ok := snapshot["in_progress"].([]string); ok && len(inProgress) > 0 {
		parts = append(parts, "\n## Current Task")
		parts = append(parts, fmt.Sprintf("🔄 **%s**: (in progress)", inProgress[0]))
	}

	// Pending tasks
	if pending, ok := snapshot["pending"].([]string); ok && len(pending) > 0 {
		parts = append(parts, "\n## Pending Tasks")
		for _, id := range pending {
			parts = append(parts, fmt.Sprintf("⏳ **%s**: (pending)", id))
		}
	}

	// Completed tasks (show last 3)
	if completedList, ok := snapshot["completed"].([]string); ok && len(completedList) > 0 {
		parts = append(parts, "\n## Recently Completed")
		count := 0
		for i := len(completedList) - 1; i >= 0 && count < 3; i++ {
			parts = append(parts, fmt.Sprintf("✅ **%s**: (completed)", completedList[i]))
			count++
		}
		if len(completedList) > 3 {
			parts = append(parts, fmt.Sprintf("   ... and %d more", len(completedList)-3))
		}
	}

	parts = append(parts, "</system-reminder>")

	return strings.Join(parts, "\n")
}

// buildSystemPrompt builds the full system prompt including tool descriptions
func (r *ReActAgent) buildSystemPrompt() string {
	prompt := r.config.SystemPrompt

	// Add plan hint if plan notebook is configured
	if r.config.PlanNotebook != nil {
		planHint := r.config.PlanNotebook.GenerateHint()
		if planHint != "" {
			prompt += "\n\n" + planHint + "\n"
		}
	}

	if r.config.Toolkit != nil && len(r.config.Toolkit.GetSchemas()) > 0 {
		prompt += "\n\n# Tools\n\nYou have access to the following tools:\n\n"

		for _, schema := range r.config.Toolkit.GetSchemas() {
			fn := schema.Function
			prompt += fmt.Sprintf("## %s\n", fn.Name)
			if fn.Description != "" {
				prompt += fn.Description + "\n"
			}

			// Add parameters info
			if fn.Parameters != nil {
				if params, ok := fn.Parameters["properties"].(map[string]any); ok {
					for name, param := range params {
						if pm, ok := param.(map[string]any); ok {
							prompt += fmt.Sprintf("- %s", name)
							if desc, ok := pm["description"].(string); ok && desc != "" {
								prompt += fmt.Sprintf(": %s", desc)
							}
							if t, ok := pm["type"].(string); ok {
								prompt += fmt.Sprintf(" (%s)", t)
							}
							prompt += "\n"
						}
					}
				}
			}
			prompt += "\n"
		}

		prompt += "To use a tool, respond with a tool_use block containing the tool name and parameters.\n"
	}

	return prompt
}

// createResponseMessage creates a response message from a chat response
func (r *ReActAgent) createResponseMessage(resp *model.ChatResponse) *message.Msg {
	if resp == nil {
		return message.NewMsg(
			r.Name(),
			[]message.ContentBlock{message.Text("No response generated")},
			types.RoleAssistant,
		)
	}

	if len(resp.Content) == 0 {
		return message.NewMsg(
			r.Name(),
			[]message.ContentBlock{message.Text("Empty response from model")},
			types.RoleAssistant,
		)
	}

	return message.NewMsg(
		r.Name(),
		resp.Content,
		types.RoleAssistant,
	)
}

// GetLastResponse returns the last response from the model
func (r *ReActAgent) GetLastResponse() *model.ChatResponse {
	return r.lastResponse
}

// ClearMemory clears the agent's message history
func (r *ReActAgent) ClearMemory() {
	r.messages = make([]*message.Msg, 0)
	if r.config.Memory != nil {
		r.config.Memory.Clear()
	}
}

// AddMessage adds a message to the agent's memory
func (r *ReActAgent) AddMessage(ctx context.Context, msg *message.Msg) error {
	if r.config.Memory != nil {
		return r.config.Memory.Add(ctx, msg)
	}
	return nil
}

// GetMemory returns the agent's memory
func (r *ReActAgent) GetMemory() Memory {
	return r.config.Memory
}

// StateDict returns the agent's state for serialization
func (r *ReActAgent) StateDict() map[string]any {
	state := r.StateModuleBase.StateDict()
	state["name"] = r.Name()
	state["system_prompt"] = r.SystemPrompt()
	if r.config.Memory != nil {
		// Check if Memory implements StateDict method (like History)
		if stateMem, ok := r.config.Memory.(interface{ StateDict() map[string]any }); ok {
			state["memory"] = stateMem.StateDict()
		} else {
			state["memory"] = r.config.Memory
		}
	}
	return state
}

// LoadStateDict loads the agent's state
func (r *ReActAgent) LoadStateDict(ctx context.Context, state map[string]any) error {
	if err := r.StateModuleBase.LoadStateDict(ctx, state); err != nil {
		return err
	}

	// Restore memory state if present
	if memoryState, ok := state["memory"]; ok && r.config.Memory != nil {
		// Check if Memory implements LoadStateDict method (like History)
		if stateMem, ok := r.config.Memory.(interface {
			LoadStateDict(ctx context.Context, state map[string]any) error
		}); ok {
			if memoryDict, ok := memoryState.(map[string]any); ok {
				if err := stateMem.LoadStateDict(ctx, memoryDict); err != nil {
					return fmt.Errorf("failed to load memory state: %w", err)
				}
			}
		}
	}

	return nil
}

// GetConfig returns the agent's configuration
func (r *ReActAgent) GetConfig() *ReActAgentConfig {
	return r.config
}

// GetModel returns the agent's model
func (r *ReActAgent) GetModel() model.ChatModel {
	return r.config.Model
}

// GetToolkit returns the agent's toolkit
func (r *ReActAgent) GetToolkit() tool.ToolProvider {
	return r.config.Toolkit
}

// SetSystemPrompt sets a new system prompt
func (r *ReActAgent) SetSystemPrompt(prompt string) {
	r.config.SystemPrompt = prompt
	if r.AgentBase != nil {
		r.AgentBase.SetSystemPrompt(prompt)
	}
}

// GetPlanNotebook returns the agent's plan notebook
func (r *ReActAgent) GetPlanNotebook() *plan.PlanNotebook {
	return r.config.PlanNotebook
}

// SetPlanNotebook sets a new plan notebook
func (r *ReActAgent) SetPlanNotebook(notebook *plan.PlanNotebook) {
	r.config.PlanNotebook = notebook
}

// GetCompressionConfig returns the agent's compression configuration
func (r *ReActAgent) GetCompressionConfig() *CompressionConfig {
	return r.config.Compression
}
