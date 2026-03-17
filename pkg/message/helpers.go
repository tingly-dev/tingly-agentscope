package message

import "github.com/tingly-dev/tingly-agentscope/pkg/types"

// getBlocksOfType is a generic helper that extracts blocks of a specific type
func getBlocksOfType[T any](m *Msg, blockType types.ContentBlockType) []T {
	var result []T
	for _, block := range m.GetContentBlocks(blockType) {
		if typed, ok := block.(T); ok {
			result = append(result, typed)
		}
	}
	return result
}

// GetToolUseBlocks returns all tool use blocks from the message
func (m *Msg) GetToolUseBlocks() []*ToolUseBlock {
	return getBlocksOfType[*ToolUseBlock](m, types.BlockTypeToolUse)
}

// GetToolResultBlocks returns all tool result blocks from the message
func (m *Msg) GetToolResultBlocks() []*ToolResultBlock {
	return getBlocksOfType[*ToolResultBlock](m, types.BlockTypeToolResult)
}

// GetTextBlocks returns all text blocks from the message
func (m *Msg) GetTextBlocks() []*TextBlock {
	return getBlocksOfType[*TextBlock](m, types.BlockTypeText)
}

// GetThinkingBlocks returns all thinking blocks from the message
func (m *Msg) GetThinkingBlocks() []*ThinkingBlock {
	return getBlocksOfType[*ThinkingBlock](m, types.BlockTypeThinking)
}

// GetImageBlocks returns all image blocks from the message
func (m *Msg) GetImageBlocks() []*ImageBlock {
	return getBlocksOfType[*ImageBlock](m, types.BlockTypeImage)
}

// GetAudioBlocks returns all audio blocks from the message
func (m *Msg) GetAudioBlocks() []*AudioBlock {
	return getBlocksOfType[*AudioBlock](m, types.BlockTypeAudio)
}

// GetVideoBlocks returns all video blocks from the message
func (m *Msg) GetVideoBlocks() []*VideoBlock {
	return getBlocksOfType[*VideoBlock](m, types.BlockTypeVideo)
}
