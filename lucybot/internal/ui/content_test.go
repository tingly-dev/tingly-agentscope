package ui

import (
	"testing"
)

func TestContentType_Detect_Diff(t *testing.T) {
	diff := `diff --git a/file.go b/file.go
--- a/file.go
+++ b/file.go
@@ -1,5 +1,5 @@
- old line
+ new line`

	contentType := DetectContentType(diff)
	if contentType != ContentTypeDiff {
		t.Errorf("Expected ContentTypeDiff, got %s", contentType)
	}
}

func TestContentType_Detect_Code(t *testing.T) {
	code := `package main

import "fmt"

func main() {
    fmt.Println("hello")
}`

	contentType := DetectContentType(code)
	if contentType != ContentTypeCode {
		t.Errorf("Expected ContentTypeCode, got %s", contentType)
	}
}

func TestContentType_Detect_Markdown(t *testing.T) {
	md := `# Heading

Some text with **bold** and *italic*.

` + "```go\nfmt.Println()\n```"

	contentType := DetectContentType(md)
	if contentType != ContentTypeMarkdown {
		t.Errorf("Expected ContentTypeMarkdown, got %s", contentType)
	}
}

func TestContentType_DetectLanguage(t *testing.T) {
	tests := []struct {
		content  string
		expected string
	}{
		{"package main\nfunc main() {}", "go"},
		{"def hello():\n    pass", "python"},
		{"const x = 1;", "javascript"},
		{"function test() {}", "javascript"},
		{"No code here", ""},
	}

	for _, test := range tests {
		lang := DetectLanguage(test.content)
		if lang != test.expected {
			t.Errorf("DetectLanguage(%q) = %q, expected %q", test.content, lang, test.expected)
		}
	}
}

func TestContentType_ExtractCodeBlocks(t *testing.T) {
	content := "Some text\n```go\nfmt.Println()\n```\nMore text"

	blocks := ExtractCodeBlocks(content)
	if len(blocks) != 1 {
		t.Errorf("Expected 1 code block, got %d", len(blocks))
	}

	if blocks[0].Language != "go" {
		t.Errorf("Expected language 'go', got %s", blocks[0].Language)
	}
}
