package tools

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// QueryType represents the type of query
type QueryType int

const (
	QuerySimpleName QueryType = iota
	QueryQualifiedName
	QueryFilePath
	QueryFileSymbol
	QueryFileLine
	QueryFileRange
	QueryFileStart
	QueryFileEnd
	QueryWildcard
)

func (q QueryType) String() string {
	switch q {
	case QuerySimpleName:
		return "SimpleName"
	case QueryQualifiedName:
		return "QualifiedName"
	case QueryFilePath:
		return "FilePath"
	case QueryFileSymbol:
		return "FileSymbol"
	case QueryFileLine:
		return "FileLine"
	case QueryFileRange:
		return "FileRange"
	case QueryFileStart:
		return "FileStart"
	case QueryFileEnd:
		return "FileEnd"
	case QueryWildcard:
		return "Wildcard"
	default:
		return "Unknown"
	}
}

// ParsedQuery represents a parsed query
type ParsedQuery struct {
	Type            QueryType
	SymbolName      string
	FilePath        string
	LineStart       int
	LineEnd         int
	SymbolType      string
	WildcardPattern string
}

// ParseQuery parses a query string into a structured query
func ParseQuery(query string, symbolType string) (*ParsedQuery, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("query cannot be empty")
	}

	// Check for : separator (file-based queries)
	if strings.Contains(query, ":") {
		parts := strings.SplitN(query, ":", 2)
		return parseFileQuery(parts[0], parts[1], symbolType)
	}

	// Check for wildcards
	if strings.ContainsAny(query, "*?") {
		return &ParsedQuery{
			Type:            QueryWildcard,
			WildcardPattern: query,
			SymbolType:      symbolType,
		}, nil
	}

	// Check for qualified name (contains . but looks like a symbol path, not file path)
	if strings.Contains(query, ".") && !looksLikeFilePath(query) {
		return &ParsedQuery{
			Type:       QueryQualifiedName,
			SymbolName: query,
			SymbolType: symbolType,
		}, nil
	}

	// Check for file path
	if looksLikeFilePath(query) {
		return &ParsedQuery{
			Type:       QueryFilePath,
			FilePath:   query,
			SymbolType: symbolType,
		}, nil
	}

	// Default: simple name
	return &ParsedQuery{
		Type:       QuerySimpleName,
		SymbolName: query,
		SymbolType: symbolType,
	}, nil
}

// parseFileQuery parses file-based queries (file.go:...)
func parseFileQuery(filePath, afterColon, symbolType string) (*ParsedQuery, error) {
	filePath = strings.TrimSpace(filePath)
	afterColon = strings.TrimSpace(afterColon)

	// Check for line range: file.go:10-50
	if strings.Contains(afterColon, "-") {
		parts := strings.SplitN(afterColon, "-", 2)
		start, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		end, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))

		if err1 == nil && err2 == nil {
			// Both are numbers: file.go:10-50
			return &ParsedQuery{
				Type:       QueryFileRange,
				FilePath:   filePath,
				LineStart:  start,
				LineEnd:    end,
				SymbolType: symbolType,
			}, nil
		}

		if err1 == nil && parts[1] == "" {
			// file.go:700-
			return &ParsedQuery{
				Type:       QueryFileStart,
				FilePath:   filePath,
				LineStart:  start,
				SymbolType: symbolType,
			}, nil
		}

		if err2 == nil && parts[0] == "" {
			// file.go:-100
			return &ParsedQuery{
				Type:       QueryFileEnd,
				FilePath:   filePath,
				LineEnd:    end,
				SymbolType: symbolType,
			}, nil
		}
	}

	// Check if after colon is a number (line number)
	if lineNum, err := strconv.Atoi(afterColon); err == nil {
		return &ParsedQuery{
			Type:       QueryFileLine,
			FilePath:   filePath,
			LineStart:  lineNum,
			SymbolType: symbolType,
		}, nil
	}

	// Otherwise, it's a symbol name in a file: file.go:SymbolName
	return &ParsedQuery{
		Type:       QueryFileSymbol,
		FilePath:   filePath,
		SymbolName: afterColon,
		SymbolType: symbolType,
	}, nil
}

// knownExtensions is a list of known file extensions
var knownExtensions = map[string]bool{
	".go": true, ".py": true, ".js": true, ".ts": true, ".java": true,
	".c": true, ".cpp": true, ".h": true, ".hpp": true, ".rs": true,
	".rb": true, ".php": true, ".swift": true, ".kt": true, ".scala": true,
	".json": true, ".yaml": true, ".yml": true, ".xml": true, ".toml": true,
	".md": true, ".txt": true, ".html": true, ".css": true, ".sql": true,
	".sh": true, ".bash": true, ".zsh": true, ".fish": true,
	".mod": true, ".sum": true, ".work": true,
}

// looksLikeFilePath checks if a string looks like a file path
func looksLikeFilePath(s string) bool {
	// If it has path separators, it's a file path
	if strings.Contains(s, "/") || strings.Contains(s, string(filepath.Separator)) {
		return true
	}

	// Check for common file extensions (but not if it looks like a qualified name with dots)
	ext := filepath.Ext(s)
	if ext != "" {
		// Check if it's a known file extension
		if knownExtensions[strings.ToLower(ext)] {
			// Known extension and no other dots before it (e.g., "main.go" not "module.Class.go")
			base := s[:len(s)-len(ext)]
			if !strings.Contains(base, ".") {
				return true
			}
		}
	}

	// Check for common file patterns (e.g., Makefile, Dockerfile)
	commonFiles := []string{"makefile", "dockerfile", "readme", "license", "go.mod", "go.sum"}
	lower := strings.ToLower(s)
	for _, f := range commonFiles {
		if lower == f || strings.HasPrefix(lower, f+".") {
			return true
		}
	}

	return false
}

// Validate validates a parsed query
func (q *ParsedQuery) Validate() error {
	switch q.Type {
	case QuerySimpleName, QueryQualifiedName:
		if q.SymbolName == "" {
			return errors.New("symbol name is required")
		}
	case QueryFilePath, QueryFileLine, QueryFileRange, QueryFileStart, QueryFileEnd, QueryFileSymbol:
		if q.FilePath == "" {
			return errors.New("file path is required")
		}
		if q.Type == QueryFileRange && q.LineStart > q.LineEnd {
			return fmt.Errorf("invalid range: start line %d is greater than end line %d", q.LineStart, q.LineEnd)
		}
	case QueryWildcard:
		if q.WildcardPattern == "" {
			return errors.New("wildcard pattern is required")
		}
		// Validate wildcard pattern
		if _, err := wildcardToRegexPattern(q.WildcardPattern); err != nil {
			return fmt.Errorf("invalid wildcard pattern: %w", err)
		}
	}
	return nil
}

// wildcardToRegexPattern converts a wildcard pattern to a regex pattern
func wildcardToRegexPattern(pattern string) (*regexp.Regexp, error) {
	// Escape special regex characters except * and ?
	regex := regexp.QuoteMeta(pattern)
	// Convert * to .*
	regex = strings.ReplaceAll(regex, `\*`, `.*`)
	// Convert ? to .
	regex = strings.ReplaceAll(regex, `\?`, `.`)
	// Anchor the pattern
	regex = "^" + regex + "$"
	return regexp.Compile(regex)
}

// MatchWildcard checks if a string matches a wildcard pattern
func MatchWildcard(pattern, s string) (bool, error) {
	re, err := wildcardToRegexPattern(pattern)
	if err != nil {
		return false, err
	}
	return re.MatchString(s), nil
}
