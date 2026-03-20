package registry

import (
	"github.com/tingly-dev/lucybot/internal/index/types"
)

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
	ext := ""
	for i := len(filePath) - 1; i >= 0; i-- {
		if filePath[i] == '.' {
			ext = filePath[i:]
			break
		}
	}
	return r.byExt[ext]
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

// DefaultRegistry is the global parser registry
var DefaultRegistry = NewParserRegistry()

// Register registers a parser with the default registry
func Register(parser types.LanguageParser) {
	DefaultRegistry.Register(parser)
}
