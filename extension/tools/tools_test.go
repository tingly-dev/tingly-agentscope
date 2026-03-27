package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
)

func TestReadTool(t *testing.T) {
	// Create temp directory for tests
	tempDir := t.TempDir()

	t.Run("read existing file", func(t *testing.T) {
		// Create test file
		testFile := filepath.Join(tempDir, "test.txt")
		content := "line1\nline2\nline3\nline4\nline5"
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		rt := NewReadTool()
		resp, err := rt.Read(context.Background(), ReadParams{Path: testFile})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(resp.Content) == 0 {
			t.Fatal("expected content in response")
		}

		textBlock, ok := resp.Content[0].(*message.TextBlock)
		if !ok {
			t.Fatalf("expected text block in response, got %T", resp.Content[0])
		}

		if textBlock.Text != content {
			t.Errorf("expected %q, got %q", content, textBlock.Text)
		}
	})

	t.Run("read with offset and limit", func(t *testing.T) {
		// Create test file
		testFile := filepath.Join(tempDir, "test2.txt")
		content := "line1\nline2\nline3\nline4\nline5"
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		rt := NewReadTool()
		resp, err := rt.Read(context.Background(), ReadParams{
			Path:   testFile,
			Offset: 2,
			Limit:  2,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		textBlock, ok := resp.Content[0].(*message.TextBlock)
		if !ok {
			t.Fatalf("expected text block in response, got %T", resp.Content[0])
		}

		expected := "line2\nline3"
		if textBlock.Text != expected {
			t.Errorf("expected %q, got %q", expected, textBlock.Text)
		}
	})

	t.Run("read non-existent file", func(t *testing.T) {
		rt := NewReadTool()
		resp, err := rt.Read(context.Background(), ReadParams{Path: "/nonexistent/file.txt"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		textBlock, ok := resp.Content[0].(*message.TextBlock)
		if !ok {
			t.Fatalf("expected text block in response, got %T", resp.Content[0])
		}

		if !strings.Contains(textBlock.Text, "file not found") {
			t.Errorf("expected 'file not found' error, got %q", textBlock.Text)
		}
	})

	t.Run("read with allowed dirs restriction", func(t *testing.T) {
		rt := NewReadTool(ReadOptions([]string{"/allowed"}, 0))
		resp, err := rt.Read(context.Background(), ReadParams{Path: "/other/file.txt"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		textBlock, ok := resp.Content[0].(*message.TextBlock)
		if !ok {
			t.Fatalf("expected text block in response, got %T", resp.Content[0])
		}

		if !strings.Contains(textBlock.Text, "path not allowed") {
			t.Errorf("expected 'path not allowed' error, got %q", textBlock.Text)
		}
	})

	t.Run("read with path traversal protection", func(t *testing.T) {
		// Create temp directory with a file
		allowedDir := filepath.Join(tempDir, "allowed")
		if err := os.MkdirAll(allowedDir, 0755); err != nil {
			t.Fatalf("failed to create allowed dir: %v", err)
		}
		allowedFile := filepath.Join(allowedDir, "safe.txt")
		if err := os.WriteFile(allowedFile, []byte("safe content"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		rt := NewReadTool(ReadOptions([]string{allowedDir}, 0))

		// Test: ../ traversal should be blocked
		resp, err := rt.Read(context.Background(), ReadParams{Path: allowedDir + "/../other.txt"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		textBlock, ok := resp.Content[0].(*message.TextBlock)
		if !ok {
			t.Fatalf("expected text block in response, got %T", resp.Content[0])
		}

		if !strings.Contains(textBlock.Text, "path not allowed") {
			t.Errorf("expected 'path not allowed' error for path traversal, got %q", textBlock.Text)
		}

		// Test: similar prefix path should be blocked
		allowedDir2 := allowedDir + "2"
		resp2, err := rt.Read(context.Background(), ReadParams{Path: filepath.Join(allowedDir2, "file.txt")})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		textBlock2, ok := resp2.Content[0].(*message.TextBlock)
		if !ok {
			t.Fatalf("expected text block in response, got %T", resp2.Content[0])
		}

		if !strings.Contains(textBlock2.Text, "path not allowed") {
			t.Errorf("expected 'path not allowed' error for similar prefix path, got %q", textBlock2.Text)
		}
	})

	t.Run("read with negative limit", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "negative.txt")
		if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		rt := NewReadTool()
		resp, err := rt.Read(context.Background(), ReadParams{
			Path:  testFile,
			Limit: -1,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		textBlock, ok := resp.Content[0].(*message.TextBlock)
		if !ok {
			t.Fatalf("expected text block in response, got %T", resp.Content[0])
		}

		if !strings.Contains(textBlock.Text, "limit must be non-negative") {
			t.Errorf("expected limit validation error, got %q", textBlock.Text)
		}
	})
}

func TestWriteTool(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("write new file", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "newfile.txt")
		wt := NewWriteTool()
		resp, err := wt.Write(context.Background(), WriteParams{
			Path:    testFile,
			Content: "hello world",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		textBlock, ok := resp.Content[0].(*message.TextBlock)
		if !ok {
			t.Fatalf("expected text block in response, got %T", resp.Content[0])
		}

		if !strings.Contains(textBlock.Text, "created") {
			t.Errorf("expected 'created' in response, got %q", textBlock.Text)
		}

		// Verify file content
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("failed to read created file: %v", err)
		}
		if string(content) != "hello world" {
			t.Errorf("expected 'hello world', got %q", string(content))
		}
	})

	t.Run("overwrite existing file", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "overwrite.txt")
		if err := os.WriteFile(testFile, []byte("old content"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		wt := NewWriteTool()
		resp, err := wt.Write(context.Background(), WriteParams{
			Path:    testFile,
			Content: "new content",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		textBlock, ok := resp.Content[0].(*message.TextBlock)
		if !ok {
			t.Fatalf("expected text block in response, got %T", resp.Content[0])
		}

		if !strings.Contains(textBlock.Text, "overwritten") {
			t.Errorf("expected 'overwritten' in response, got %q", textBlock.Text)
		}

		// Verify file content
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		if string(content) != "new content" {
			t.Errorf("expected 'new content', got %q", string(content))
		}
	})

	t.Run("create nested directories", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "nested", "dirs", "file.txt")
		wt := NewWriteTool()
		resp, err := wt.Write(context.Background(), WriteParams{
			Path:    testFile,
			Content: "nested content",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		textBlock, ok := resp.Content[0].(*message.TextBlock)
		if !ok {
			t.Fatalf("expected text block in response, got %T", resp.Content[0])
		}

		if !strings.Contains(textBlock.Text, "created") {
			t.Errorf("expected 'created' in response, got %q", textBlock.Text)
		}

		// Verify file exists
		if _, err := os.Stat(testFile); err != nil {
			t.Errorf("file should exist: %v", err)
		}
	})

	t.Run("disallow overwrite", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "nooverwrite.txt")
		if err := os.WriteFile(testFile, []byte("existing"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		wt := NewWriteTool(WriteOptions(nil, false))
		resp, err := wt.Write(context.Background(), WriteParams{
			Path:    testFile,
			Content: "new content",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		textBlock, ok := resp.Content[0].(*message.TextBlock)
		if !ok {
			t.Fatalf("expected text block in response, got %T", resp.Content[0])
		}

		if !strings.Contains(textBlock.Text, "overwrite is not allowed") {
			t.Errorf("expected 'overwrite is not allowed' error, got %q", textBlock.Text)
		}
	})

	t.Run("max write size limit", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "maxsize.txt")
		wt := NewWriteTool(WriteMaxSize(100))

		largeContent := strings.Repeat("x", 200)
		resp, err := wt.Write(context.Background(), WriteParams{
			Path:    testFile,
			Content: largeContent,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		textBlock, ok := resp.Content[0].(*message.TextBlock)
		if !ok {
			t.Fatalf("expected text block in response, got %T", resp.Content[0])
		}

		if !strings.Contains(textBlock.Text, "content too large") {
			t.Errorf("expected 'content too large' error, got %q", textBlock.Text)
		}
	})
}

func TestEditTool(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("edit existing text", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "edit.txt")
		original := "hello world\nfoo bar\ngoodbye"
		if err := os.WriteFile(testFile, []byte(original), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		et := NewEditTool()
		resp, err := et.Edit(context.Background(), EditParams{
			Path:    testFile,
			OldText: "foo bar",
			NewText: "baz qux",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		textBlock, ok := resp.Content[0].(*message.TextBlock)
		if !ok {
			t.Fatalf("expected text block in response, got %T", resp.Content[0])
		}

		if !strings.Contains(textBlock.Text, "Successfully edited") {
			t.Errorf("expected success message, got %q", textBlock.Text)
		}

		// Verify file content
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		expected := "hello world\nbaz qux\ngoodbye"
		if string(content) != expected {
			t.Errorf("expected %q, got %q", expected, string(content))
		}
	})

	t.Run("edit non-existent file", func(t *testing.T) {
		et := NewEditTool()
		resp, err := et.Edit(context.Background(), EditParams{
			Path:    "/nonexistent/file.txt",
			OldText: "old",
			NewText: "new",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		textBlock, ok := resp.Content[0].(*message.TextBlock)
		if !ok {
			t.Fatalf("expected text block in response, got %T", resp.Content[0])
		}

		if !strings.Contains(textBlock.Text, "file not found") {
			t.Errorf("expected 'file not found' error, got %q", textBlock.Text)
		}
	})

	t.Run("edit with non-matching oldText", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "nomatch.txt")
		if err := os.WriteFile(testFile, []byte("hello world"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		et := NewEditTool()
		resp, err := et.Edit(context.Background(), EditParams{
			Path:    testFile,
			OldText: "nonexistent",
			NewText: "replacement",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		textBlock, ok := resp.Content[0].(*message.TextBlock)
		if !ok {
			t.Fatalf("expected text block in response, got %T", resp.Content[0])
		}

		if !strings.Contains(textBlock.Text, "oldText not found") {
			t.Errorf("expected 'oldText not found' error, got %q", textBlock.Text)
		}
	})

	t.Run("edit with ambiguous oldText", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "ambiguous.txt")
		if err := os.WriteFile(testFile, []byte("foo bar foo"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		et := NewEditTool()
		resp, err := et.Edit(context.Background(), EditParams{
			Path:    testFile,
			OldText: "foo",
			NewText: "baz",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		textBlock, ok := resp.Content[0].(*message.TextBlock)
		if !ok {
			t.Fatalf("expected text block in response, got %T", resp.Content[0])
		}

		if !strings.Contains(textBlock.Text, "appears") {
			t.Errorf("expected ambiguous match error, got %q", textBlock.Text)
		}
	})
}

func TestBashTool(t *testing.T) {
	t.Run("execute simple command", func(t *testing.T) {
		bt := NewBashTool()
		resp, err := bt.Bash(context.Background(), BashParams{
			Command: "echo hello",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		textBlock, ok := resp.Content[0].(*message.TextBlock)
		if !ok {
			t.Fatalf("expected text block in response, got %T", resp.Content[0])
		}

		if !strings.Contains(textBlock.Text, "hello") {
			t.Errorf("expected 'hello' in output, got %q", textBlock.Text)
		}
	})

	t.Run("execute command with error", func(t *testing.T) {
		bt := NewBashTool()
		resp, err := bt.Bash(context.Background(), BashParams{
			Command: "exit 1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		textBlock, ok := resp.Content[0].(*message.TextBlock)
		if !ok {
			t.Fatalf("expected text block in response, got %T", resp.Content[0])
		}

		if !strings.Contains(textBlock.Text, "exited with code 1") {
			t.Errorf("expected exit code error, got %q", textBlock.Text)
		}
	})

	t.Run("execute with timeout", func(t *testing.T) {
		bt := NewBashTool()
		resp, err := bt.Bash(context.Background(), BashParams{
			Command: "sleep 10",
			Timeout: 1,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		textBlock, ok := resp.Content[0].(*message.TextBlock)
		if !ok {
			t.Fatalf("expected text block in response, got %T", resp.Content[0])
		}

		if !strings.Contains(textBlock.Text, "timed out") {
			t.Errorf("expected timeout error, got %q", textBlock.Text)
		}
	})

	t.Run("blocked command", func(t *testing.T) {
		bt := NewBashTool()
		resp, err := bt.Bash(context.Background(), BashParams{
			Command: "rm -rf /",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		textBlock, ok := resp.Content[0].(*message.TextBlock)
		if !ok {
			t.Fatalf("expected text block in response, got %T", resp.Content[0])
		}

		if !strings.Contains(textBlock.Text, "blocked pattern") {
			t.Errorf("expected blocked pattern error, got %q", textBlock.Text)
		}
	})

	t.Run("working directory", func(t *testing.T) {
		tempDir := t.TempDir()
		bt := NewBashTool(BashOptions(nil, nil, 30*time.Second, tempDir))
		resp, err := bt.Bash(context.Background(), BashParams{
			Command: "pwd",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		textBlock, ok := resp.Content[0].(*message.TextBlock)
		if !ok {
			t.Fatalf("expected text block in response, got %T", resp.Content[0])
		}

		if !strings.Contains(textBlock.Text, tempDir) {
			t.Errorf("expected working directory %q in output, got %q", tempDir, textBlock.Text)
		}
	})

	t.Run("command chaining blocked by default", func(t *testing.T) {
		bt := NewBashTool()
		resp, err := bt.Bash(context.Background(), BashParams{
			Command: "echo test && rm -rf /",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		textBlock, ok := resp.Content[0].(*message.TextBlock)
		if !ok {
			t.Fatalf("expected text block in response, got %T", resp.Content[0])
		}

		if !strings.Contains(textBlock.Text, "command chaining not allowed") {
			t.Errorf("expected command chaining error, got %q", textBlock.Text)
		}
	})

	t.Run("command chaining allowed when enabled", func(t *testing.T) {
		bt := NewBashTool(BashAllowChaining(true))
		resp, err := bt.Bash(context.Background(), BashParams{
			Command: "echo test && echo success",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		textBlock, ok := resp.Content[0].(*message.TextBlock)
		if !ok {
			t.Fatalf("expected text block in response, got %T", resp.Content[0])
		}

		if !strings.Contains(textBlock.Text, "success") {
			t.Errorf("expected chaining to work when allowed, got %q", textBlock.Text)
		}
	})
}

func TestNewToolkit(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("create toolkit with all tools", func(t *testing.T) {
		tk, err := NewToolkit(nil)
		if err != nil {
			t.Fatalf("failed to create toolkit: %v", err)
		}

		schemas := tk.GetSchemas()
		if len(schemas) != 4 {
			t.Errorf("expected 4 tools, got %d", len(schemas))
		}

		// Check that all tools are registered
		toolNames := make(map[string]bool)
		for _, schema := range schemas {
			toolNames[schema.Function.Name] = true
		}

		expectedTools := []string{"read", "write", "edit", "bash"}
		for _, name := range expectedTools {
			if !toolNames[name] {
				t.Errorf("expected tool %q to be registered", name)
			}
		}
	})

	t.Run("schemas have correct required fields", func(t *testing.T) {
		tk, err := NewToolkit(nil)
		if err != nil {
			t.Fatalf("failed to create toolkit: %v", err)
		}

		schemas := tk.GetSchemas()

		// Build lookup by tool name
		schemaMap := make(map[string]map[string]any)
		for _, s := range schemas {
			schemaMap[s.Function.Name] = s.Function.Parameters
		}

		tests := []struct {
			tool     string
			required []string
		}{
			{"read", []string{"path"}},
			{"write", []string{"path", "content"}},
			{"edit", []string{"path", "oldText", "newText"}},
			{"bash", []string{"command"}},
		}

		for _, tc := range tests {
			params, ok := schemaMap[tc.tool]
			if !ok {
				t.Errorf("tool %q not found in schemas", tc.tool)
				continue
			}

			reqRaw, ok := params["required"]
			if !ok {
				t.Errorf("tool %q: missing required field in schema", tc.tool)
				continue
			}

			reqSlice, ok := reqRaw.([]string)
			if !ok {
				t.Errorf("tool %q: required field is not []string, got %T", tc.tool, reqRaw)
				continue
			}

			reqSet := make(map[string]bool, len(reqSlice))
			for _, r := range reqSlice {
				reqSet[r] = true
			}

			for _, expected := range tc.required {
				if !reqSet[expected] {
					t.Errorf("tool %q: expected %q in required, got %v", tc.tool, expected, reqSlice)
				}
			}

			if len(reqSlice) != len(tc.required) {
				t.Errorf("tool %q: expected %d required fields, got %d: %v", tc.tool, len(tc.required), len(reqSlice), reqSlice)
			}
		}
	})

	t.Run("use tool instances directly", func(t *testing.T) {
		readTool := NewReadTool(ReadOptions([]string{tempDir}, 1024*1024))
		writeTool := NewWriteTool(WriteOptions([]string{tempDir}, true))
		editTool := NewEditTool(EditOptions([]string{tempDir}))
		bashTool := NewBashTool()

		// Test write
		testFile := filepath.Join(tempDir, "direct.txt")
		resp, err := writeTool.Write(context.Background(), WriteParams{
			Path:    testFile,
			Content: "test content",
		})
		if err != nil {
			t.Fatalf("write failed: %v", err)
		}
		if len(resp.Content) == 0 {
			t.Fatal("expected content in write response")
		}

		// Test read
		resp, err = readTool.Read(context.Background(), ReadParams{
			Path: testFile,
		})
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		textBlock, ok := resp.Content[0].(*message.TextBlock)
		if !ok {
			t.Fatalf("expected text block in response, got %T", resp.Content[0])
		}
		if textBlock.Text != "test content" {
			t.Errorf("expected 'test content', got %q", textBlock.Text)
		}

		// Test edit
		_, err = editTool.Edit(context.Background(), EditParams{
			Path:    testFile,
			OldText: "test",
			NewText: "modified",
		})
		if err != nil {
			t.Fatalf("edit failed: %v", err)
		}

		// Test bash
		resp, err = bashTool.Bash(context.Background(), BashParams{
			Command: "echo hello",
		})
		if err != nil {
			t.Fatalf("bash failed: %v", err)
		}
		textBlock, ok = resp.Content[0].(*message.TextBlock)
		if !ok {
			t.Fatalf("expected text block in response, got %T", resp.Content[0])
		}
		if !strings.Contains(textBlock.Text, "hello") {
			t.Errorf("expected 'hello' in output, got %q", textBlock.Text)
		}
	})
}
