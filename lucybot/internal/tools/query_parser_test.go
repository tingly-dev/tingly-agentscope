package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseQuery(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		symbolType     string
		wantType       QueryType
		wantSymbol     string
		wantFile       string
		wantLineStart  int
		wantLineEnd    int
		wantWildcard   string
		wantErr        bool
	}{
		// 1. Simple names
		{
			name:       "simple name",
			input:      "MyClass",
			wantType:   QuerySimpleName,
			wantSymbol: "MyClass",
		},
		{
			name:       "simple function name",
			input:      "myFunction",
			wantType:   QuerySimpleName,
			wantSymbol: "myFunction",
		},
		// 2. Qualified names
		{
			name:       "qualified name",
			input:      "module.Class.method",
			wantType:   QueryQualifiedName,
			wantSymbol: "module.Class.method",
		},
		{
			name:       "package.function",
			input:      "fmt.Println",
			wantType:   QueryQualifiedName,
			wantSymbol: "fmt.Println",
		},
		// 3. File paths
		{
			name:     "file path",
			input:    "path/to/file.go",
			wantType: QueryFilePath,
			wantFile: "path/to/file.go",
		},
		{
			name:     "file with extension",
			input:    "main.py",
			wantType: QueryFilePath,
			wantFile: "main.py",
		},
		{
			name:     "absolute path",
			input:    "/home/user/project/main.go",
			wantType: QueryFilePath,
			wantFile: "/home/user/project/main.go",
		},
		// 4. File+Symbol
		{
			name:       "file with symbol",
			input:      "file.go:SymbolName",
			wantType:   QueryFileSymbol,
			wantFile:   "file.go",
			wantSymbol: "SymbolName",
		},
		{
			name:       "file with method",
			input:      "main.go:main",
			wantType:   QueryFileSymbol,
			wantFile:   "main.go",
			wantSymbol: "main",
		},
		// 5. File+Line
		{
			name:          "file with line number",
			input:         "file.go:42",
			wantType:      QueryFileLine,
			wantFile:      "file.go",
			wantLineStart: 42,
		},
		{
			name:          "file with line 1",
			input:         "main.py:1",
			wantType:      QueryFileLine,
			wantFile:      "main.py",
			wantLineStart: 1,
		},
		// 6. File+Range
		{
			name:          "file with range",
			input:         "file.go:10-50",
			wantType:      QueryFileRange,
			wantFile:      "file.go",
			wantLineStart: 10,
			wantLineEnd:   50,
		},
		{
			name:          "file with single line range",
			input:         "main.go:5-5",
			wantType:      QueryFileRange,
			wantFile:      "main.go",
			wantLineStart: 5,
			wantLineEnd:   5,
		},
		// 7. File+Start
		{
			name:          "file with start only",
			input:         "file.go:700-",
			wantType:      QueryFileStart,
			wantFile:      "file.go",
			wantLineStart: 700,
		},
		// 8. File+End
		{
			name:        "file with end only",
			input:       "file.go:-100",
			wantType:    QueryFileEnd,
			wantFile:    "file.go",
			wantLineEnd: 100,
		},
		// 9. Wildcards
		{
			name:         "wildcard with star",
			input:        "test.func*",
			wantType:     QueryWildcard,
			wantWildcard: "test.func*",
		},
		{
			name:         "wildcard with question",
			input:        "file?.go",
			wantType:     QueryWildcard,
			wantWildcard: "file?.go",
		},
		{
			name:         "wildcard with both",
			input:        "*.test_?",
			wantType:     QueryWildcard,
			wantWildcard: "*.test_?",
		},
		// 10. Edge cases
		{
			name:    "empty query",
			input:   "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			input:   "   ",
			wantErr: true,
		},
		{
			name:     "common files - makefile",
			input:    "Makefile",
			wantType: QueryFilePath,
			wantFile: "Makefile",
		},
		{
			name:     "common files - dockerfile",
			input:    "Dockerfile",
			wantType: QueryFilePath,
			wantFile: "Dockerfile",
		},
		{
			name:     "go.mod file",
			input:    "go.mod",
			wantType: QueryFilePath,
			wantFile: "go.mod",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseQuery(tt.input, tt.symbolType)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantType, got.Type, "query type mismatch")
			assert.Equal(t, tt.wantSymbol, got.SymbolName, "symbol name mismatch")
			assert.Equal(t, tt.wantFile, got.FilePath, "file path mismatch")
			assert.Equal(t, tt.wantLineStart, got.LineStart, "line start mismatch")
			assert.Equal(t, tt.wantLineEnd, got.LineEnd, "line end mismatch")
			assert.Equal(t, tt.wantWildcard, got.WildcardPattern, "wildcard pattern mismatch")
			assert.Equal(t, tt.symbolType, got.SymbolType, "symbol type mismatch")
		})
	}
}

func TestParseQuery_WithSymbolType(t *testing.T) {
	// Test that symbol_type parameter is preserved
	query, err := ParseQuery("MyClass", "class")
	require.NoError(t, err)
	assert.Equal(t, "class", query.SymbolType)

	query, err = ParseQuery("file.go:42", "function")
	require.NoError(t, err)
	assert.Equal(t, "function", query.SymbolType)

	query, err = ParseQuery("test*", "")
	require.NoError(t, err)
	assert.Equal(t, "", query.SymbolType)
}

func TestParsedQuery_Validate(t *testing.T) {
	tests := []struct {
		name    string
		query   *ParsedQuery
		wantErr bool
	}{
		{
			name: "valid simple name",
			query: &ParsedQuery{
				Type:       QuerySimpleName,
				SymbolName: "MyFunc",
			},
			wantErr: false,
		},
		{
			name: "invalid simple name - empty",
			query: &ParsedQuery{
				Type:       QuerySimpleName,
				SymbolName: "",
			},
			wantErr: true,
		},
		{
			name: "valid file path",
			query: &ParsedQuery{
				Type:     QueryFilePath,
				FilePath: "main.go",
			},
			wantErr: false,
		},
		{
			name: "invalid file path - empty",
			query: &ParsedQuery{
				Type:     QueryFilePath,
				FilePath: "",
			},
			wantErr: true,
		},
		{
			name: "valid file range",
			query: &ParsedQuery{
				Type:      QueryFileRange,
				FilePath:  "main.go",
				LineStart: 10,
				LineEnd:   20,
			},
			wantErr: false,
		},
		{
			name: "invalid file range - start > end",
			query: &ParsedQuery{
				Type:      QueryFileRange,
				FilePath:  "main.go",
				LineStart: 20,
				LineEnd:   10,
			},
			wantErr: true,
		},
		{
			name: "valid wildcard",
			query: &ParsedQuery{
				Type:            QueryWildcard,
				WildcardPattern: "test*",
			},
			wantErr: false,
		},
		{
			name: "invalid wildcard - empty",
			query: &ParsedQuery{
				Type:            QueryWildcard,
				WildcardPattern: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.query.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMatchWildcard(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		want    bool
	}{
		{"test*", "testFunc", true},
		{"test*", "test", true},
		{"test*", "testing", true},
		{"test*", "mytest", false},
		{"file?.go", "file1.go", true},
		{"file?.go", "fileA.go", true},
		{"file?.go", "file12.go", false},
		{"*.go", "main.go", true},
		{"*.go", "test.go", true},
		{"*.go", "main.py", false},
		{"*test*", "mytestfunc", true},
		{"*test*", "testing", true},
		{"*test*", "hello", false},
		{"exact", "exact", true},
		{"exact", "exactly", false},
		{"?", "a", true},
		{"?", "ab", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.input, func(t *testing.T) {
			got, err := MatchWildcard(tt.pattern, tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestQueryType_String(t *testing.T) {
	tests := []struct {
		qt   QueryType
		want string
	}{
		{QuerySimpleName, "SimpleName"},
		{QueryQualifiedName, "QualifiedName"},
		{QueryFilePath, "FilePath"},
		{QueryFileSymbol, "FileSymbol"},
		{QueryFileLine, "FileLine"},
		{QueryFileRange, "FileRange"},
		{QueryFileStart, "FileStart"},
		{QueryFileEnd, "FileEnd"},
		{QueryWildcard, "Wildcard"},
		{QueryType(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.qt.String())
		})
	}
}
