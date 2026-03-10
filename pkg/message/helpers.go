package message

import "github.com/tingly-dev/tingly-agentscope/pkg/types"

// GetToolUseBlocks returns all tool use blocks from the message
func (m *Msg) GetToolUseBlocks() []*ToolUseBlock {
	var blocks []*ToolUseBlock

	for _, block := range m.GetContentBlocks(types.BlockTypeToolUse) {
		if tb, ok := block.(*ToolUseBlock); ok {
			blocks = append(blocks, tb)
		}
	}

	return blocks
}

// GetToolResultBlocks returns all tool result blocks from the message
func (m *Msg) GetToolResultBlocks() []*ToolResultBlock {
	var blocks []*ToolResultBlock

	for _, block := range m.GetContentBlocks(types.BlockTypeToolResult) {
		if trb, ok := block.(*ToolResultBlock); ok {
			blocks = append(blocks, trb)
		}
	}

	return blocks
}

// GetTextBlocks returns all text blocks from the message
func (m *Msg) GetTextBlocks() []*TextBlock {
	var blocks []*TextBlock

	for _, block := range m.GetContentBlocks(types.BlockTypeText) {
		if tb, ok := block.(*TextBlock); ok {
			blocks = append(blocks, tb)
		}
	}

	return blocks
}

// GetThinkingBlocks returns all thinking blocks from the message
func (m *Msg) GetThinkingBlocks() []*ThinkingBlock {
	var blocks []*ThinkingBlock

	for _, block := range m.GetContentBlocks(types.BlockTypeThinking) {
		if tb, ok := block.(*ThinkingBlock); ok {
			blocks = append(blocks, tb)
		}
	}

	return blocks
}

// GetImageBlocks returns all image blocks from the message
func (m *Msg) GetImageBlocks() []*ImageBlock {
	var blocks []*ImageBlock

	for _, block := range m.GetContentBlocks(types.BlockTypeImage) {
		if ib, ok := block.(*ImageBlock); ok {
			blocks = append(blocks, ib)
		}
	}

	return blocks
}

// GetAudioBlocks returns all audio blocks from the message
func (m *Msg) GetAudioBlocks() []*AudioBlock {
	var blocks []*AudioBlock

	for _, block := range m.GetContentBlocks(types.BlockTypeAudio) {
		if ab, ok := block.(*AudioBlock); ok {
			blocks = append(blocks, ab)
		}
	}

	return blocks
}

// GetVideoBlocks returns all video blocks from the message
func (m *Msg) GetVideoBlocks() []*VideoBlock {
	var blocks []*VideoBlock

	for _, block := range m.GetContentBlocks(types.BlockTypeVideo) {
		if vb, ok := block.(*VideoBlock); ok {
			blocks = append(blocks, vb)
		}
	}

	return blocks
}
