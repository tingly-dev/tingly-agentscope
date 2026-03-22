package index

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/tingly-dev/lucybot/internal/index/types"
)

// Re-export ParseResult and LanguageParser from types package
type ParseResult = types.ParseResult
type LanguageParser = types.LanguageParser

// ParserRegistry manages language parsers
type ParserRegistry struct {
	parsers map[types.Language]types.LanguageParser
	byExt   map[string]types.LanguageParser
}

// NewParserRegistry creates a new parser registry
func NewParserRegistry() *ParserRegistry {
	return &ParserRegistry{
		parsers: make(map[types.Language]types.LanguageParser),
		byExt:   make(map[string]types.LanguageParser),
	}
}

// Register registers a language parser
func (r *ParserRegistry) Register(parser types.LanguageParser) {
	r.parsers[parser.GetLanguage()] = parser
	for _, ext := range parser.GetFileExtensions() {
		r.byExt[ext] = parser
	}
}

// GetParser returns the parser for a given language
func (r *ParserRegistry) GetParser(lang types.Language) types.LanguageParser {
	return r.parsers[lang]
}

// GetParserForFile returns the parser for a given file path
func (r *ParserRegistry) GetParserForFile(filePath string) types.LanguageParser {
	ext := strings.ToLower(filepath.Ext(filePath))
	return r.byExt[ext]
}

// GetParserForLanguage returns the parser for a language string
func (r *ParserRegistry) GetParserForLanguage(lang string) types.LanguageParser {
	return r.parsers[types.Language(lang)]
}

// CanParse returns true if a parser is available for the file
func (r *ParserRegistry) CanParse(filePath string) bool {
	return r.GetParserForFile(filePath) != nil
}

// GetSupportedLanguages returns all supported languages
func (r *ParserRegistry) GetSupportedLanguages() []types.Language {
	langs := make([]types.Language, 0, len(r.parsers))
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
	language   types.Language
	extensions []string
}

// NewSimpleParser creates a new simple parser
func NewSimpleParser(language types.Language, extensions []string) *SimpleParser {
	return &SimpleParser{
		language:   language,
		extensions: extensions,
	}
}

// GetLanguage returns the language identifier
func (p *SimpleParser) GetLanguage() types.Language {
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
func (p *SimpleParser) Parse(ctx context.Context, content []byte, filePath string) (*types.ParseResult, error) {
	// Default implementation just creates file info
	return &types.ParseResult{
		FileInfo: &types.FileInfo{
			Path:     filePath,
			Language: p.language,
			Size:     int64(len(content)),
		},
	}, nil
}

// DefaultRegistry is deprecated - use registry.DefaultRegistry instead
// Kept for backward compatibility
var DefaultRegistry = NewParserRegistry()

// Register is deprecated - use registry.Register instead
// Kept for backward compatibility
func Register(parser types.LanguageParser) {
	DefaultRegistry.Register(parser)
}

// GetParserForFile returns the parser for a file from the default registry
func GetParserForFile(filePath string) types.LanguageParser {
	return DefaultRegistry.GetParserForFile(filePath)
}

// CanParse returns true if the default registry has a parser for the file
func CanParse(filePath string) bool {
	return DefaultRegistry.CanParse(filePath)
}
