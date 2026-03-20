package types

import (
	"context"
	"fmt"
	"time"
)

// SymbolKind represents the type of a code symbol
type SymbolKind string

const (
	SymbolKindFunction   SymbolKind = "function"
	SymbolKindClass      SymbolKind = "class"
	SymbolKindMethod     SymbolKind = "method"
	SymbolKindVariable   SymbolKind = "variable"
	SymbolKindParameter  SymbolKind = "parameter"
	SymbolKindModule     SymbolKind = "module"
	SymbolKindInterface  SymbolKind = "interface"
	SymbolKindTypeAlias  SymbolKind = "type_alias"
	SymbolKindConstant   SymbolKind = "constant"
	SymbolKindProperty   SymbolKind = "property"
	SymbolKindEnum       SymbolKind = "enum"
	SymbolKindEnumMember SymbolKind = "enum_member"
	SymbolKindUnknown    SymbolKind = "unknown"
)

// ReferenceKind represents the type of a symbol reference
type ReferenceKind string

const (
	ReferenceKindRead            ReferenceKind = "read"
	ReferenceKindWrite           ReferenceKind = "write"
	ReferenceKindCall            ReferenceKind = "call"
	ReferenceKindImport          ReferenceKind = "import"
	ReferenceKindInstantiation   ReferenceKind = "instantiation"
	ReferenceKindAttributeAccess ReferenceKind = "attribute_access"
	ReferenceKindDefinition      ReferenceKind = "definition"
)

// Language represents supported programming languages
type Language string

const (
	LanguageGo         Language = "go"
	LanguagePython     Language = "python"
	LanguageJavaScript Language = "javascript"
	LanguageTypeScript Language = "typescript"
	LanguageJava       Language = "java"
	LanguageRust       Language = "rust"
	LanguageCpp        Language = "cpp"
	LanguageC          Language = "c"
	LanguageUnknown    Language = "unknown"
)

// Symbol represents a code symbol definition
type Symbol struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	QualifiedName string     `json:"qualified_name"`
	Kind          SymbolKind `json:"kind"`
	FilePath      string     `json:"file_path"`
	StartLine     int        `json:"start_line"`
	StartColumn   int        `json:"start_column"`
	EndLine       int        `json:"end_line"`
	EndColumn     int        `json:"end_column"`
	Language      Language   `json:"language"`
	ParentID      *string    `json:"parent_id,omitempty"`
	Documentation string     `json:"documentation,omitempty"`
	Signature     string     `json:"signature,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// SymbolReference represents a reference to a symbol
type SymbolReference struct {
	ID            string        `json:"id"`
	SymbolID      *string       `json:"symbol_id,omitempty"` // nil for unresolved references
	ReferenceName string        `json:"reference_name"`
	FilePath      string        `json:"file_path"`
	LineNumber    int           `json:"line_number"`
	ColumnNumber  int           `json:"column_number"`
	ReferenceKind ReferenceKind `json:"reference_kind"`
	ScopeID       *string       `json:"scope_id,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
}

// Scope represents a lexical scope in the code
type Scope struct {
	ID            string    `json:"id"`
	FilePath      string    `json:"file_path"`
	StartLine     int       `json:"start_line"`
	StartColumn   int       `json:"start_column"`
	EndLine       int       `json:"end_line"`
	EndColumn     int       `json:"end_column"`
	ScopeKind     string    `json:"scope_kind"` // module, class, function, method, etc.
	ParentScopeID *string   `json:"parent_scope_id,omitempty"`
	SymbolID      *string   `json:"symbol_id,omitempty"` // Associated symbol (if any)
	CreatedAt     time.Time `json:"created_at"`
}

// Relationship represents a relationship between two symbols
type Relationship struct {
	SourceID         string    `json:"source_id"`
	TargetID         string    `json:"target_id"`
	RelationshipType string    `json:"relationship_type"` // calls, extends, implements, contains, etc.
	CreatedAt        time.Time `json:"created_at"`
}

// FileInfo stores information about indexed files
type FileInfo struct {
	Path        string    `json:"path"`
	Language    Language  `json:"language"`
	Size        int64     `json:"size"`
	ModTime     time.Time `json:"mod_time"`
	Hash        string    `json:"hash"`
	SymbolCount int       `json:"symbol_count"`
	IndexedAt   time.Time `json:"indexed_at"`
}

// ParseResult contains the results of parsing a file
type ParseResult struct {
	Symbols      []*Symbol
	References   []*SymbolReference
	Scopes       []*Scope
	Relationships []*Relationship
	FileInfo     *FileInfo
}

// LanguageParser defines the interface for language-specific parsers
type LanguageParser interface {
	// Parse parses the given content and extracts symbols, references, and scopes
	Parse(ctx context.Context, content []byte, filePath string) (*ParseResult, error)

	// GetLanguage returns the language identifier
	GetLanguage() Language

	// GetFileExtensions returns the file extensions this parser handles
	GetFileExtensions() []string

	// CanParse returns true if this parser can handle the given file
	CanParse(filePath string) bool
}

// GenerateSymbolID creates a unique ID for a symbol based on its location
func GenerateSymbolID(filePath string, startLine, startCol int) string {
	return fmt.Sprintf("%s:%d:%d", filePath, startLine, startCol)
}

// GenerateReferenceID creates a unique ID for a reference
func GenerateReferenceID(filePath string, line, col int) string {
	return fmt.Sprintf("ref:%s:%d:%d", filePath, line, col)
}

// GenerateScopeID creates a unique ID for a scope
func GenerateScopeID(filePath string, startLine, startCol int) string {
	return fmt.Sprintf("scope:%s:%d:%d", filePath, startLine, startCol)
}

// String returns the string representation of SymbolKind
func (k SymbolKind) String() string {
	return string(k)
}

// String returns the string representation of ReferenceKind
func (k ReferenceKind) String() string {
	return string(k)
}

// String returns the string representation of Language
func (l Language) String() string {
	return string(l)
}

// DetectLanguage detects the programming language from a file extension
func DetectLanguage(filePath string) Language {
	ext := ""
	for i := len(filePath) - 1; i >= 0; i-- {
		if filePath[i] == '.' {
			ext = filePath[i:]
			break
		}
	}

	switch ext {
	case ".go":
		return LanguageGo
	case ".py":
		return LanguagePython
	case ".js":
		return LanguageJavaScript
	case ".ts", ".tsx":
		return LanguageTypeScript
	case ".java":
		return LanguageJava
	case ".rs":
		return LanguageRust
	case ".cpp", ".cc", ".cxx":
		return LanguageCpp
	case ".c", ".h":
		return LanguageC
	default:
		return LanguageUnknown
	}
}
