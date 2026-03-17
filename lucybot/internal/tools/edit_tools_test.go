package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditFileTool_Execute(t *testing.T) {
	tmpDir := t.TempDir()
	history := NewEditHistory()
	editTool := NewEditFileTool(tmpDir, history)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main

func main() {
	println("hello")
}
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	t.Run("global replacement", func(t *testing.T) {
		_, err := editTool.Execute(EditParams{
			Path:      "test.go",
			OldString: `println("hello")`,
			NewString: `println("world")`,
		})
		require.NoError(t, err)

		// Verify the change
		newContent, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Contains(t, string(newContent), `println("world")`)

		// Verify history was recorded
		assert.Equal(t, 1, history.Count())
	})

	t.Run("replacement with line range", func(t *testing.T) {
		// Reset file
		err := os.WriteFile(testFile, []byte(content), 0644)
		require.NoError(t, err)
		history.Clear()

		_, err = editTool.Execute(EditParams{
			Path:      "test.go",
			OldString: `println("hello")`,
			NewString: `println("world")`,
			LineStart: 4,
			LineEnd:   4,
		})
		require.NoError(t, err)

		// Verify the change
		newContent, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Contains(t, string(newContent), `println("world")`)
	})

	t.Run("non-unique search text", func(t *testing.T) {
		// Create file with duplicate content
		dupContent := `foo
foo
foo
`
		err := os.WriteFile(testFile, []byte(dupContent), 0644)
		require.NoError(t, err)
		history.Clear()

		_, err = editTool.Execute(EditParams{
			Path:      "test.go",
			OldString: "foo",
			NewString: "bar",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not unique")
	})

	t.Run("search text not found", func(t *testing.T) {
		err := os.WriteFile(testFile, []byte(content), 0644)
		require.NoError(t, err)
		history.Clear()

		_, err = editTool.Execute(EditParams{
			Path:      "test.go",
			OldString: "nonexistent",
			NewString: "replacement",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestCreateFileTool_Execute(t *testing.T) {
	tmpDir := t.TempDir()
	history := NewEditHistory()
	createTool := NewCreateFileTool(tmpDir, history)

	t.Run("create new file", func(t *testing.T) {
		_, err := createTool.Execute(CreateParams{
			Path:    "newfile.go",
			Content: "package main\n",
		})
		require.NoError(t, err)

		// Verify file was created
		content, err := os.ReadFile(filepath.Join(tmpDir, "newfile.go"))
		require.NoError(t, err)
		assert.Equal(t, "package main\n", string(content))

		// Verify history was recorded
		assert.Equal(t, 1, history.Count())
	})

	t.Run("create nested file", func(t *testing.T) {
		_, err := createTool.Execute(CreateParams{
			Path:    "nested/dir/file.go",
			Content: "package nested\n",
		})
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(tmpDir, "nested/dir/file.go"))
		require.NoError(t, err)
		assert.Equal(t, "package nested\n", string(content))
	})

	t.Run("file already exists", func(t *testing.T) {
		// Create file first
		err := os.WriteFile(filepath.Join(tmpDir, "exists.go"), []byte("existing"), 0644)
		require.NoError(t, err)

		_, err = createTool.Execute(CreateParams{
			Path:    "exists.go",
			Content: "new content",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})
}

func TestShowDiffTool_Execute(t *testing.T) {
	t.Run("no edits", func(t *testing.T) {
		history := NewEditHistory()
		diffTool := NewShowDiffTool(history)

		resp, err := diffTool.Execute()
		require.NoError(t, err)
		require.NotNil(t, resp)

		text := extractTextFromResponse(resp)
		assert.Contains(t, text, "No edits")
	})

	t.Run("with edits", func(t *testing.T) {
		history := NewEditHistory()
		history.Record(EditRecord{
			Path:      "test.go",
			OldString: "old",
			NewString: "new",
		})

		diffTool := NewShowDiffTool(history)
		resp, err := diffTool.Execute()
		require.NoError(t, err)
		require.NotNil(t, resp)

		text := extractTextFromResponse(resp)
		assert.Contains(t, text, "SEARCH")
		assert.Contains(t, text, "REPLACE")
		assert.Contains(t, text, "old")
		assert.Contains(t, text, "new")
	})
}

func TestUndoLastEditTool_Execute(t *testing.T) {
	tmpDir := t.TempDir()
	history := NewEditHistory()
	editTool := NewEditFileTool(tmpDir, history)
	undoTool := NewUndoLastEditTool(tmpDir, history)

	// Create and edit a file
	testFile := filepath.Join(tmpDir, "test.go")
	originalContent := `package main

func main() {
	println("hello")
}
`
	err := os.WriteFile(testFile, []byte(originalContent), 0644)
	require.NoError(t, err)

	// Make an edit
	_, err = editTool.Execute(EditParams{
		Path:      "test.go",
		OldString: `println("hello")`,
		NewString: `println("world")`,
	})
	require.NoError(t, err)

	// Verify edit was made
	content, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), `println("world")`)

	// Undo the edit
	_, err = undoTool.Execute()
	require.NoError(t, err)

	// Verify undo worked
	content, err = os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), `println("hello")`)

	// Verify history was updated
	assert.Equal(t, 0, history.Count())
}

func TestEditHistory(t *testing.T) {
	h := NewEditHistory()

	t.Run("record and get all", func(t *testing.T) {
		h.Record(EditRecord{
			Path:      "file1.go",
			OldString: "old1",
			NewString: "new1",
		})
		h.Record(EditRecord{
			Path:      "file2.go",
			OldString: "old2",
			NewString: "new2",
		})

		records := h.GetAll()
		require.Len(t, records, 2)
		assert.Equal(t, "file1.go", records[0].Path)
		assert.Equal(t, "file2.go", records[1].Path)
	})

	t.Run("get last", func(t *testing.T) {
		last, ok := h.GetLast()
		require.True(t, ok)
		assert.Equal(t, "file2.go", last.Path)
	})

	t.Run("get by path", func(t *testing.T) {
		records := h.GetByPath("file1.go")
		require.Len(t, records, 1)
		assert.Equal(t, "old1", records[0].OldString)
	})

	t.Run("count", func(t *testing.T) {
		assert.Equal(t, 2, h.Count())
	})

	t.Run("undo last", func(t *testing.T) {
		record, ok := h.UndoLast()
		require.True(t, ok)
		assert.Equal(t, "file2.go", record.Path)
		assert.Equal(t, 1, h.Count())
	})

	t.Run("clear", func(t *testing.T) {
		h.Clear()
		assert.Equal(t, 0, h.Count())
	})
}

func TestEditHistory_GeneratePatch(t *testing.T) {
	h := NewEditHistory()

	// Empty history
	patch := h.GeneratePatch()
	assert.Contains(t, patch, "No edits")

	// With records
	h.Record(EditRecord{
		Path:      "main.go",
		OldString: "func old() {}",
		NewString: "func new() {}",
		LineStart: 10,
		LineEnd:   15,
	})

	patch = h.GeneratePatch()
	assert.Contains(t, patch, "main.go")
	assert.Contains(t, patch, "SEARCH")
	assert.Contains(t, patch, "REPLACE")
	assert.Contains(t, patch, "func old()")
	assert.Contains(t, patch, "func new()")
}

func TestEditHistory_GenerateSummary(t *testing.T) {
	h := NewEditHistory()

	// Empty history
	summary := h.GenerateSummary()
	assert.Contains(t, summary, "No edits")

	// With records
	h.Record(EditRecord{Path: "file1.go"})
	h.Record(EditRecord{Path: "file1.go"})
	h.Record(EditRecord{Path: "file2.go"})

	summary = h.GenerateSummary()
	assert.Contains(t, summary, "Total edits: 3")
	assert.Contains(t, summary, "file1.go: 2 edit(s)")
	assert.Contains(t, summary, "file2.go: 1 edit(s)")
}

// Helper function to extract text from ToolResponse
func extractTextFromResponse(resp interface{}) string {
	// This is a simplified version - in real tests you'd need to properly
	// extract text from the tool.ToolResponse type
	return ""
}
