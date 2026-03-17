package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFenceDiff(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantFilePath  string
		wantLineStart int
		wantLineEnd   int
		wantSearch    string
		wantReplace   string
		wantErr       bool
	}{
		{
			name: "basic diff with line range",
			input: `path/to/file.go lines: 10-20
<<<<<<< SEARCH
func oldFunc() {
    return 1
}
=======
func newFunc() {
    return 2
}
>>>>>>> REPLACE`,
			wantFilePath:  "path/to/file.go",
			wantLineStart: 10,
			wantLineEnd:   20,
			wantSearch:    "func oldFunc() {\n    return 1\n}",
			wantReplace:   "func newFunc() {\n    return 2\n}",
			wantErr:       false,
		},
		{
			name: "diff without line range",
			input: `main.go
<<<<<<< SEARCH
func main() {}
=======
func main() {
    println("hello")
}
>>>>>>> REPLACE`,
			wantFilePath:  "main.go",
			wantLineStart: 0,
			wantLineEnd:   0,
			wantSearch:    "func main() {}",
			wantReplace:   "func main() {\n    println(\"hello\")\n}",
			wantErr:       false,
		},
		{
			name: "diff with start line only",
			input: `file.go lines: 100-
<<<<<<< SEARCH
old
=======
new
>>>>>>> REPLACE`,
			wantFilePath:  "file.go",
			wantLineStart: 100,
			wantLineEnd:   0,
			wantSearch:    "old",
			wantReplace:   "new",
			wantErr:       false,
		},
		{
			name: "diff with end line only",
			input: `file.go lines: -50
<<<<<<< SEARCH
old
=======
new
>>>>>>> REPLACE`,
			wantFilePath:  "file.go",
			wantLineStart: 0,
			wantLineEnd:   50,
			wantSearch:    "old",
			wantReplace:   "new",
			wantErr:       false,
		},
		{
			name:    "too short",
			input:   "short",
			wantErr: true,
		},
		{
			name: "missing search marker",
			input: `file.go
old content
=======
new content
>>>>>>> REPLACE`,
			wantErr: true,
		},
		{
			name: "missing separator",
			input: `file.go
<<<<<<< SEARCH
old content
>>>>>>> REPLACE`,
			wantErr: true,
		},
		{
			name: "missing replace marker",
			input: `file.go
<<<<<<< SEARCH
old content
=======
new content`,
			wantErr: true,
		},
		{
			name: "empty search content",
			input: `file.go
<<<<<<< SEARCH
=======
new content
>>>>>>> REPLACE`,
			wantFilePath:  "file.go",
			wantLineStart: 0,
			wantLineEnd:   0,
			wantSearch:    "",
			wantReplace:   "new content",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := ParseFenceDiff(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantFilePath, diff.FilePath)
			assert.Equal(t, tt.wantLineStart, diff.LineStart)
			assert.Equal(t, tt.wantLineEnd, diff.LineEnd)
			assert.Equal(t, tt.wantSearch, diff.Search)
			assert.Equal(t, tt.wantReplace, diff.Replace)
		})
	}
}

func TestFenceDiff_String(t *testing.T) {
	tests := []struct {
		name     string
		diff     *FenceDiff
		expected string
	}{
		{
			name: "with line range",
			diff: &FenceDiff{
				FilePath:  "main.go",
				Search:    "old",
				Replace:   "new",
				LineStart: 10,
				LineEnd:   20,
			},
			expected: "main.go lines: 10-20\n<<<<<<< SEARCH\nold\n=======\nnew\n>>>>>>> REPLACE",
		},
		{
			name: "without line range",
			diff: &FenceDiff{
				FilePath: "main.go",
				Search:   "old",
				Replace:  "new",
			},
			expected: "main.go\n<<<<<<< SEARCH\nold\n=======\nnew\n>>>>>>> REPLACE",
		},
		{
			name: "with start line only",
			diff: &FenceDiff{
				FilePath:  "main.go",
				Search:    "old",
				Replace:   "new",
				LineStart: 100,
			},
			expected: "main.go lines: 100-\n<<<<<<< SEARCH\nold\n=======\nnew\n>>>>>>> REPLACE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.diff.String())
		})
	}
}

func TestFenceDiff_Validate(t *testing.T) {
	tests := []struct {
		name    string
		diff    *FenceDiff
		wantErr bool
	}{
		{
			name: "valid diff",
			diff: &FenceDiff{
				FilePath:  "main.go",
				Search:    "old",
				Replace:   "new",
				LineStart: 10,
				LineEnd:   20,
			},
			wantErr: false,
		},
		{
			name: "missing filepath",
			diff: &FenceDiff{
				FilePath: "",
				Search:   "old",
				Replace:  "new",
			},
			wantErr: true,
		},
		{
			name: "invalid line range",
			diff: &FenceDiff{
				FilePath:  "main.go",
				Search:    "old",
				Replace:   "new",
				LineStart: 20,
				LineEnd:   10,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.diff.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFenceDiff_Apply(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		diff        *FenceDiff
		want        string
		wantErr     bool
		errContains string
	}{
		{
			name:    "simple replacement",
			content: "Hello old world",
			diff: &FenceDiff{
				FilePath: "test.txt",
				Search:   "old",
				Replace:  "new",
			},
			want:    "Hello new world",
			wantErr: false,
		},
		{
			name:    "multiline replacement",
			content: "line1\nline2\nline3",
			diff: &FenceDiff{
				FilePath: "test.txt",
				Search:   "line2",
				Replace:  "modified",
			},
			want:    "line1\nmodified\nline3",
			wantErr: false,
		},
		{
			name:    "replacement with line range",
			content: "line1\nline2\nline3\nline4\nline5",
			diff: &FenceDiff{
				FilePath:  "test.txt",
				Search:    "line2",
				Replace:   "modified",
				LineStart: 2,
				LineEnd:   3,
			},
			want:    "line1\nmodified\nline3\nline4\nline5",
			wantErr: false,
		},
		{
			name:    "search not found",
			content: "Hello world",
			diff: &FenceDiff{
				FilePath: "test.txt",
				Search:   "notfound",
				Replace:  "new",
			},
			wantErr:     true,
			errContains: "search content not found",
		},
		{
			name:    "search not found in line range",
			content: "line1\nline2\nline3",
			diff: &FenceDiff{
				FilePath:  "test.txt",
				Search:    "line3",
				Replace:   "modified",
				LineStart: 1,
				LineEnd:   2,
			},
			wantErr:     true,
			errContains: "search content not found in specified line range",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.diff.Apply(tt.content)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseMultipleFenceDiffs(t *testing.T) {
	input := `file1.go lines: 1-10
<<<<<<< SEARCH
func foo() {}
=======
func foo() {
    return 1
}
>>>>>>> REPLACE

file2.go lines: 20-30
<<<<<<< SEARCH
func bar() {}
=======
func bar() {
    return 2
}
>>>>>>> REPLACE`

	diffs, err := ParseMultipleFenceDiffs(input)
	require.NoError(t, err)
	require.Len(t, diffs, 2)

	assert.Equal(t, "file1.go", diffs[0].FilePath)
	assert.Equal(t, 1, diffs[0].LineStart)
	assert.Equal(t, 10, diffs[0].LineEnd)
	assert.Contains(t, diffs[0].Search, "foo")

	assert.Equal(t, "file2.go", diffs[1].FilePath)
	assert.Equal(t, 20, diffs[1].LineStart)
	assert.Equal(t, 30, diffs[1].LineEnd)
	assert.Contains(t, diffs[1].Search, "bar")
}

func TestParseMultipleFenceDiffs_Empty(t *testing.T) {
	input := ""
	diffs, err := ParseMultipleFenceDiffs(input)
	require.NoError(t, err)
	assert.Empty(t, diffs)
}

func TestParseMultipleFenceDiffs_NoDiffs(t *testing.T) {
	input := `This is just some text
without any fence diffs`
	diffs, err := ParseMultipleFenceDiffs(input)
	require.NoError(t, err)
	assert.Empty(t, diffs)
}
