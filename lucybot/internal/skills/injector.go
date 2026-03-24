package skills

import (
	"context"
	"fmt"
	"strings"

	"github.com/tingly-dev/tingly-agentscope/pkg/memory"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
)

const (
	// SystemPromptMark is the metadata key marking messages as system prompt content
	SystemPromptMark = "system_prompt_mark"

	// SkillNameMark is the metadata key storing the skill name
	SkillNameMark = "skill_name"
)

// SkillInjector injects skill content into user messages
type SkillInjector struct {
	skill *Skill
}

// NewSkillInjector creates a new skill injector
func NewSkillInjector(skill *Skill) *SkillInjector {
	return &SkillInjector{
		skill: skill,
	}
}

// Inject adds the skill content to the beginning of the user message
// The skill content is marked to prevent it from being compressed
func (i *SkillInjector) Inject(ctx context.Context, msg *message.Msg) *message.Msg {
	// Get original content blocks
	blocks := msg.GetContentBlocks()

	// Create skill content block
	skillBlock := message.Text(i.formatSkillContent())

	// Prepend skill content
	newBlocks := make([]message.ContentBlock, 0, len(blocks)+1)
	newBlocks = append(newBlocks, skillBlock)
	newBlocks = append(newBlocks, blocks...)

	// Create new message with injected content
	newMsg := message.NewMsgWithTimestamp(
		msg.Name,
		newBlocks,
		msg.Role,
		msg.Timestamp,
	)

	// Copy existing metadata
	if msg.Metadata != nil {
		newMsg.Metadata = make(map[string]any)
		for k, v := range msg.Metadata {
			newMsg.Metadata[k] = v
		}
	} else {
		newMsg.Metadata = make(map[string]any)
	}

	// Mark as system prompt content (prevents compression)
	newMsg.Metadata[SystemPromptMark] = true
	newMsg.Metadata[SkillNameMark] = i.skill.Name

	return newMsg
}

// formatSkillContent formats the skill content for injection
func (i *SkillInjector) formatSkillContent() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# Skill: %s\n\n", i.skill.Name))
	b.WriteString(fmt.Sprintf("**Description:** %s\n\n", i.skill.Description))
	b.WriteString("**Instructions:**\n")
	b.WriteString(i.skill.Content)
	b.WriteString("\n\n---\n\n")

	return b.String()
}

// IsSkillLoaded checks if the skill is already loaded in memory
// It looks for messages with system_prompt_mark metadata and matching skill name
func (i *SkillInjector) IsSkillLoaded(mem memory.Memory) bool {
	if mem == nil {
		return false
	}

	messages := mem.GetMessages()
	for _, msg := range messages {
		if msg == nil || msg.Metadata == nil {
			continue
		}

		// Check if this message has the system_prompt_mark
		if hasMark, ok := msg.Metadata[SystemPromptMark].(bool); ok && hasMark {
			// Check if the skill name matches
			if skillName, ok := msg.Metadata[SkillNameMark].(string); ok && skillName == i.skill.Name {
				return true
			}
		}
	}

	return false
}
