package index

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// ParseResult contains the results of parsing a file
type ParseResult struct {
	Symbols    []*Symbol
	References []*SymbolReference
	Scopes     []*Scope
	FileInfo   *FileInfo
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

// ParserRegistry manages language parsers
type ParserRegistry struct {
	parsers map[Language]LanguageParser
	byExt   map[string]LanguageParser
}

// NewParserRegistry creates a new parser registry
func NewParserRegistry() *ParserRegistry {
	return &ParserRegistry{
		parsers: make(map[Language]LanguageParser),
		byExt:   make(map[string]LanguageParser),
	}
}

// Register registers a language parser
func (r *ParserRegistry) Register(parser LanguageParser) {
	r.parsers[parser.GetLanguage()] = parser
	for _, ext := range parser.GetFileExtensions() {
		r.byExt[ext] = parser
	}
}

// GetParser returns the parser for a given language
func (r *ParserRegistry) GetParser(lang Language) LanguageParser {
	return r.parsers[lang]
}

// GetParserForFile returns the parser for a given file path
func (r *ParserRegistry) GetParserForFile(filePath string) LanguageParser {
	ext := strings.ToLower(filepath.Ext(filePath))
	return r.byExt[ext]
}

// GetParserForLanguage returns the parser for a language string
func (r *ParserRegistry) GetParserForLanguage(lang string) LanguageParser {
	return r.parsers[Language(lang)]
}

// CanParse returns true if a parser is available for the file
func (r *ParserRegistry) CanParse(filePath string) bool {
	return r.GetParserForFile(filePath) != nil
}

// GetSupportedLanguages returns all supported languages
func (r *ParserRegistry) GetSupportedLanguages() []Language {
	langs := make([]Language, 0, len(r.parsers))
	for lang := range r.parsers {
		langs = append(langs, lang)
	}
	return langs
}

// GetSupportedExtensions returns all supported file extensions
func (r *ParserRegistry) GetSupportedExtensions() []string {
	 exts := make([]string, 0, len(r.byExt))
	for ext := range r.byExt {
		exts = append(exts, ext)
	}
	return exts
}

// SimpleParser provides a basic parser implementation using regex/heuristics
// for languages without Tree-sitter support
type SimpleParser struct {
	language    Language
	extensions []string
}

// NewSimpleParser creates a new simple parser
func NewSimpleParser(language Language, extensions []string) *SimpleParser {
	return &SimpleParser{
		language:    language,
		extensions: extensions,
	}
}

// GetLanguage returns the language identifier
func (p *SimpleParser) GetLanguage() Language {
	return p.language
}

// GetFileExtensions returns the file extensions this parser handles
func (p *SimpleParser) GetFileExtensions() []string {
	return p.extensions
}

// CanParse returns true if this parser can handle the given file
func (p *SimpleParser) CanParse(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	for _, e := range p.extensions {
		if e == ext {
			return true
		}
	}
	return false
}

// Parse implements a basic parser that extracts file info only
// Subclasses should override this for actual symbol extraction
func (p *SimpleParser) Parse(ctx context.Context, content []byte, filePath string) (*ParseResult, error) {
	// Default implementation just creates file info
	return &ParseResult{
		FileInfo: &FileInfo{
			Path:     filePath,
			Language: p.language,
			Size:     int64(len(content)),
		},
	}, nil
}

// DefaultRegistry is the global parser registry
var DefaultRegistry = NewParserRegistry()

// Register registers a parser with the default registry
func Register(parser LanguageParser) {
	DefaultRegistry.Register(parser)
}

// GetParserForFile returns the parser for a file from the default registry
func GetParserForFile(filePath string) LanguageParser {
	return DefaultRegistry.GetParserForFile(filePath)
}

// CanParse returns true if the default registry has a parser for the file
func CanParse(filePath string) bool {
	return DefaultRegistry.CanParse(filePath)
}

// extractQualifiedName builds a qualified name from package/module and symbol name
func extractQualifiedName(packageName, symbolName string, parents []string) string {
	parts := []string{}
	if packageName != "" {
		parts = append(parts, packageName)
	}
	parts = append(parts, parents...)
	parts = append(parts, symbolName)
	return strings.Join(parts, ".")
}

// extractPackageName extracts the package/module name from content
// This is a helper that can be used by language-specific parsers
func extractPackageName(content []byte, lang Language) string {
	switch lang {
	case LanguageGo:
		return extractGoPackageName(content)
	case LanguagePython:
		return extractPythonModuleName(content)
	case LanguageJava:
		return extractJavaPackageName(content)
	default:
		return ""
	}
}

func extractGoPackageName(content []byte) string {
	// Look for "package xxx" at the start
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "package "))
		}
	}
	return ""
}

func extractPythonModuleName(content []byte) string {
	// Look for module docstring or try to infer from imports
	// For now, return empty as Python doesn't have explicit module declarations
	return ""
}

func extractJavaPackageName(content []byte) string {
	// Look for "package xxx;"
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package ") {
			pkg := strings.TrimPrefix(line, "package ")
			pkg = strings.TrimSuffix(pkg, ";")
			return strings.TrimSpace(pkg)
		}
	}
	return ""
}

// ParseError represents a parsing error
type ParseError struct {
	FilePath string
	Language Language
	Message  string
	Cause    error
}

func (e *ParseError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("parse error in %s (%s): %s: %v", e.FilePath, e.Language, e.Message, e.Cause)
	}
	return fmt.Sprintf("parse error in %s (%s): %s", e.FilePath, e.Language, e.Message)
}

func (e *ParseError) Unwrap() error {
	return e.Cause
}
