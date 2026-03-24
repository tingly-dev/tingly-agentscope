package message

// Text creates a new text block
func Text(text string) *TextBlock {
	return &TextBlock{Text: text}
}

// Thinking creates a new thinking block
func Thinking(thinking string) *ThinkingBlock {
	return &ThinkingBlock{Thinking: thinking}
}

// Base64Image creates a new image block with base64 data
func Base64Image(mediaType string, data string) *ImageBlock {
	return &ImageBlock{
		Source: &MediaSource{
			Type:      "base64",
			MediaType: mediaType,
			Data:      data,
		},
	}
}

// URLImage creates a new image block with URL source
func URLImage(url string) *ImageBlock {
	return &ImageBlock{
		Source: &MediaSource{
			Type: "url",
			URL:  url,
		},
	}
}

// Base64Audio creates a new audio block with base64 data
func Base64Audio(mediaType string, data string) *AudioBlock {
	return &AudioBlock{
		Source: &MediaSource{
			Type:      "base64",
			MediaType: mediaType,
			Data:      data,
		},
	}
}

// URLAudio creates a new audio block with URL source
func URLAudio(url string) *AudioBlock {
	return &AudioBlock{
		Source: &MediaSource{
			Type: "url",
			URL:  url,
		},
	}
}

// Base64Video creates a new video block with base64 data
func Base64Video(mediaType string, data string) *VideoBlock {
	return &VideoBlock{
		Source: &MediaSource{
			Type:      "base64",
			MediaType: mediaType,
			Data:      data,
		},
	}
}

// URLVideo creates a new video block with URL source
func URLVideo(url string) *VideoBlock {
	return &VideoBlock{
		Source: &MediaSource{
			Type: "url",
			URL:  url,
		},
	}
}

// ToolUse creates a new tool use block
func ToolUse(id, name string, input any) *ToolUseBlock {
	return &ToolUseBlock{
		ID:    id,
		Name:  name,
		Input: input,
	}
}

// ToolResult creates a new tool result block
func ToolResult(id, name string, output []ContentBlock) *ToolResultBlock {
	return &ToolResultBlock{
		ID:     id,
		Name:   name,
		Output: output,
	}
}

// Error creates a new error block
func Error(errType ErrorType, message string) *ErrorBlock {
	return &ErrorBlock{
		ErrorType: errType,
		Message:   message,
	}
}
