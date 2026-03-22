package ui

import (
	"regexp"
	"strings"
)

// ContentType represents the detected type of content
type ContentType string

const (
	ContentTypeDiff     ContentType = "diff"
	ContentTypeCode     ContentType = "code"
	ContentTypeMarkdown ContentType = "markdown"
	ContentTypePlain    ContentType = "plain"
)

// CodeBlock represents an extracted code block
type CodeBlock struct {
	Language string
	Code     string
}

// Content diff detection patterns
var contentDiffPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?m)^diff --git`),
	regexp.MustCompile(`(?m)^\+\+\+ `),
	regexp.MustCompile(`(?m)^--- `),
	regexp.MustCompile(`(?m)^@@ -\d+,\d+ \+\d+,\d+ @@`),
}

// Code detection patterns by language
var contentLanguagePatterns = map[string]*regexp.Regexp{
	"go":         regexp.MustCompile(`(?m)^\s*package\s+\w+|^\s*func\s+\w+\(|^\s*import\s+"`),
	"python":     regexp.MustCompile(`(?m)^\s*def\s+\w+\(|^\s*import\s+\w+|^\s*class\s+\w+:`),
	"javascript": regexp.MustCompile(`(?m)^\s*const\s+|^\s*let\s+|^\s*var\s+|^\s*function\s+|=\>\s*\{`),
	"typescript": regexp.MustCompile(`(?m)^\s*interface\s+|^\s*type\s+\w+\s*=*|:\s*(string|number|boolean)`),
	"rust":       regexp.MustCompile(`(?m)^\s*fn\s+\w+\(|^\s*let\s+mut\s+|^\s*use\s+\w+::`),
	"c":          regexp.MustCompile(`(?m)^\s*#include|^\s*int\s+main\s*\(`),
	"cpp":        regexp.MustCompile(`(?m)^\s*#include|^\s*std::`),
	"java":       regexp.MustCompile(`(?m)^\s*public\s+class|^\s*import\s+java\.`),
}

// File extension to language mapping
var contentExtensionToLang = map[string]string{
	".go":    "go",
	".py":    "python",
	".js":    "javascript",
	".ts":    "typescript",
	".tsx":   "typescript",
	".rs":    "rust",
	".c":     "c",
	".cpp":   "cpp",
	".cc":    "cpp",
	".h":     "c",
	".hpp":   "cpp",
	".java":  "java",
	".rb":    "ruby",
	".php":   "php",
	".swift": "swift",
	".kt":    "kotlin",
	".scala": "scala",
}

// Markdown patterns
var contentMarkdownPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?m)^#{1,6}\s`),            // Headers
	regexp.MustCompile(`(?m)^\*\s|^\-\s|^\d+\.\s`), // Lists
	regexp.MustCompile(`\*\*.*?\*\*`),              // Bold
	regexp.MustCompile(`\*.*?\*`),                  // Italic
	regexp.MustCompile("`{3}"),                     // Code blocks
	regexp.MustCompile("`[^`]+`"),                  // Inline code
	regexp.MustCompile(`\[.*?\]\(.*?\)`),           // Links
}

// Content code block extraction regex
var contentCodeBlockRegex = regexp.MustCompile("(?s)```(\\w*)\\n(.*?)```")

// DetectContentType determines the type of content
func DetectContentType(content string) ContentType {
	if content == "" {
		return ContentTypePlain
	}

	// Check for diff first (most specific)
	if IsDiffContent(content) {
		return ContentTypeDiff
	}

	// Check for markdown
	if IsMarkdownContent(content) {
		return ContentTypeMarkdown
	}

	// Check for code
	if IsCodeContent(content) {
		return ContentTypeCode
	}

	return ContentTypePlain
}

// IsDiffContent checks if content is a git diff
func IsDiffContent(content string) bool {
	// Must have at least 2 diff indicators
	matchCount := 0
	for _, pattern := range contentDiffPatterns {
		if pattern.MatchString(content) {
			matchCount++
		}
	}

	// Also check for + and - lines (need both for a real diff)
	hasPlus := regexp.MustCompile(`(?m)^\+[^+]`).MatchString(content)
	hasMinus := regexp.MustCompile(`(?m)^-[^-]`).MatchString(content)

	return matchCount >= 2 || (hasPlus && hasMinus && matchCount >= 1)
}

// IsCodeContent checks if content appears to be source code
func IsCodeContent(content string) bool {
	// Check first 20 lines
	lines := strings.Split(content, "\n")
	checkLines := 20
	if len(lines) < checkLines {
		checkLines = len(lines)
	}

	checkContent := strings.Join(lines[:checkLines], "\n")

	// Check for file path patterns (e.g., /path/to/file.go:50)
	if regexp.MustCompile(`[\w/]+\.\w+:\d+`).MatchString(checkContent) {
		return true
	}

	// Check for language-specific patterns
	for _, pattern := range contentLanguagePatterns {
		if pattern.MatchString(checkContent) {
			return true
		}
	}

	return false
}

// IsMarkdownContent checks if content contains markdown formatting
func IsMarkdownContent(content string) bool {
	matchCount := 0
	for _, pattern := range contentMarkdownPatterns {
		if pattern.MatchString(content) {
			matchCount++
			if matchCount >= 2 {
				return true
			}
		}
	}
	return false
}

// DetectLanguage attempts to detect the programming language
func DetectLanguage(content string) string {
	// Check for file extensions in content
	for ext, lang := range contentExtensionToLang {
		if strings.Contains(content, ext) {
			return lang
		}
	}

	// Check for language patterns
	for lang, pattern := range contentLanguagePatterns {
		if pattern.MatchString(content) {
			return lang
		}
	}

	return ""
}

// ExtractCodeBlocks extracts code blocks from markdown content
func ExtractCodeBlocks(content string) []CodeBlock {
	blocks := make([]CodeBlock, 0)
	matches := contentCodeBlockRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			blocks = append(blocks, CodeBlock{
				Language: match[1],
				Code:     match[2],
			})
		}
	}

	return blocks
}

// TruncateContent truncates content to specified lines with indicator
func TruncateContent(content string, maxLines int) (string, bool) {
	lines := strings.Split(content, "\n")

	if len(lines) <= maxLines {
		return content, false
	}

	truncated := strings.Join(lines[:maxLines], "\n")

	return truncated, true
}

// GetLineCount efficiently counts lines in content
func GetLineCount(content string) int {
	return strings.Count(content, "\n") + 1
}
