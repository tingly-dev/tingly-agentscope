package languages

import (
	"context"
	"regexp"
	"strings"

	"github.com/tingly-dev/lucybot/internal/index"
)

// GoParser parses Go source files
type GoParser struct {
	// Tree-sitter parser would go here when fully implemented
	// For now, use regex-based parsing
}

// NewGoParser creates a new Go parser
func NewGoParser() *GoParser {
	return &GoParser{}
}

// GetLanguage returns the language identifier
func (p *GoParser) GetLanguage() index.Language {
	return index.LanguageGo
}

// GetFileExtensions returns the file extensions this parser handles
func (p *GoParser) GetFileExtensions() []string {
	return []string{".go"}
}

// CanParse returns true if this parser can handle the given file
func (p *GoParser) CanParse(filePath string) bool {
	return strings.HasSuffix(filePath, ".go")
}

// Parse parses Go source code and extracts symbols
func (p *GoParser) Parse(ctx context.Context, content []byte, filePath string) (*index.ParseResult, error) {
	result := &index.ParseResult{
		Symbols:    make([]*index.Symbol, 0),
		References: make([]*index.SymbolReference, 0),
		Scopes:     make([]*index.Scope, 0),
		FileInfo: &index.FileInfo{
			Path:     filePath,
			Language: index.LanguageGo,
			Size:     int64(len(content)),
		},
	}

	// Get package name
	packageName := p.extractPackageName(content)

	// Parse line by line for now (Tree-sitter would be more accurate)
	lines := strings.Split(string(content), "\n")

	var commentBlock []string

	for lineNum, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Handle comments
		if strings.HasPrefix(line, "//") {
			commentBlock = append(commentBlock, strings.TrimPrefix(line, "//"))
			continue
		}

		// Parse function declarations
		if symbol := p.parseFunction(line, lineNum+1, packageName, filePath, commentBlock); symbol != nil {
			result.Symbols = append(result.Symbols, symbol)
			commentBlock = nil
			continue
		}

		// Parse method declarations
		if symbol := p.parseMethod(line, lineNum+1, packageName, filePath, commentBlock); symbol != nil {
			result.Symbols = append(result.Symbols, symbol)
			commentBlock = nil
			continue
		}

		// Parse type declarations (structs, interfaces)
		if symbol := p.parseType(line, lineNum+1, packageName, filePath, commentBlock); symbol != nil {
			result.Symbols = append(result.Symbols, symbol)
			commentBlock = nil
			continue
		}

		// Parse variable declarations
		if symbols := p.parseVars(line, lineNum+1, packageName, filePath); len(symbols) > 0 {
			result.Symbols = append(result.Symbols, symbols...)
			commentBlock = nil
			continue
		}

		// Parse constant declarations
		if symbols := p.parseConsts(line, lineNum+1, packageName, filePath); len(symbols) > 0 {
			result.Symbols = append(result.Symbols, symbols...)
			commentBlock = nil
			continue
		}

		// Parse imports (as references)
		if refs := p.parseImports(line, lineNum+1, filePath); len(refs) > 0 {
			result.References = append(result.References, refs...)
			continue
		}

		// Clear comment block if line is not a comment
		if !strings.HasPrefix(line, "//") {
			commentBlock = nil
		}
	}

	result.FileInfo.SymbolCount = len(result.Symbols)
	return result, nil
}

func (p *GoParser) extractPackageName(content []byte) string {
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "package "))
		}
	}
	return ""
}

func (p *GoParser) parseFunction(line string, lineNum int, packageName, filePath string, comments []string) *index.Symbol {
	// Match: func Name(...) or func Name[...](...)
	re := regexp.MustCompile(`^func\s+(\w+)\s*(?:\[.*?\])?\s*\(`)
	matches := re.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}

	name := matches[1]
	doc := strings.Join(comments, "\n")
	sig := p.extractSignature(line, "func")

	return &index.Symbol{
		ID:            index.GenerateSymbolID(filePath, lineNum, 0),
		Name:          name,
		QualifiedName: p.qualifiedName(packageName, "", name),
		Kind:          index.SymbolKindFunction,
		FilePath:      filePath,
		StartLine:     lineNum,
		EndLine:       lineNum, // Approximate - would need full parsing
		Language:      index.LanguageGo,
		Documentation: doc,
		Signature:     sig,
	}
}

func (p *GoParser) parseMethod(line string, lineNum int, packageName, filePath string, comments []string) *index.Symbol {
	// Match: func (r *Receiver) Name(...) or func (r Receiver) Name(...)
	re := regexp.MustCompile(`^func\s+\(\s*(?:\w+\s+)?\*?(\w+)\s*\)\s*(\w+)\s*(?:\[.*?\])?\s*\(`)
	matches := re.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}

	receiver := matches[1]
	name := matches[2]
	doc := strings.Join(comments, "\n")
	sig := p.extractSignature(line, "func")

	return &index.Symbol{
		ID:            index.GenerateSymbolID(filePath, lineNum, 0),
		Name:          name,
		QualifiedName: p.qualifiedName(packageName, receiver, name),
		Kind:          index.SymbolKindMethod,
		FilePath:      filePath,
		StartLine:     lineNum,
		EndLine:       lineNum,
		Language:      index.LanguageGo,
		Documentation: doc,
		Signature:     sig,
	}
}

func (p *GoParser) parseType(line string, lineNum int, packageName, filePath string, comments []string) *index.Symbol {
	// Match: type Name ...
	re := regexp.MustCompile(`^type\s+(\w+)\s+(.+)$`)
	matches := re.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}

	name := matches[1]
	typeDef := strings.TrimSpace(matches[2])
	doc := strings.Join(comments, "\n")

	var kind index.SymbolKind
	if strings.HasPrefix(typeDef, "struct") {
		kind = index.SymbolKindClass // Go uses struct, map to class
	} else if strings.HasPrefix(typeDef, "interface") {
		kind = index.SymbolKindInterface
	} else {
		kind = index.SymbolKindTypeAlias
	}

	return &index.Symbol{
		ID:            index.GenerateSymbolID(filePath, lineNum, 0),
		Name:          name,
		QualifiedName: p.qualifiedName(packageName, "", name),
		Kind:          kind,
		FilePath:      filePath,
		StartLine:     lineNum,
		EndLine:       lineNum,
		Language:      index.LanguageGo,
		Documentation: doc,
		Signature:     "type " + name + " " + typeDef,
	}
}

func (p *GoParser) parseVars(line string, lineNum int, packageName, filePath string) []*index.Symbol {
	// Match: var Name Type or var Name = value or var Name Type = value
	re := regexp.MustCompile(`^var\s+(\w+)`)
	matches := re.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}

	name := matches[1]
	symbol := &index.Symbol{
		ID:            index.GenerateSymbolID(filePath, lineNum, 0),
		Name:          name,
		QualifiedName: p.qualifiedName(packageName, "", name),
		Kind:          index.SymbolKindVariable,
		FilePath:      filePath,
		StartLine:     lineNum,
		EndLine:       lineNum,
		Language:      index.LanguageGo,
	}

	return []*index.Symbol{symbol}
}

func (p *GoParser) parseConsts(line string, lineNum int, packageName, filePath string) []*index.Symbol {
	// Match: const Name or const ( ... )
	re := regexp.MustCompile(`^const\s+(\w+)`)
	matches := re.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}

	name := matches[1]
	symbol := &index.Symbol{
		ID:            index.GenerateSymbolID(filePath, lineNum, 0),
		Name:          name,
		QualifiedName: p.qualifiedName(packageName, "", name),
		Kind:          index.SymbolKindConstant,
		FilePath:      filePath,
		StartLine:     lineNum,
		EndLine:       lineNum,
		Language:      index.LanguageGo,
	}

	return []*index.Symbol{symbol}
}

func (p *GoParser) parseImports(line string, lineNum int, filePath string) []*index.SymbolReference {
	var refs []*index.SymbolReference

	// Match: import "path"
	if strings.HasPrefix(line, "import ") && !strings.Contains(line, "(") {
		re := regexp.MustCompile(`import\s+["']([^"']+)["']`)
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			refs = append(refs, &index.SymbolReference{
				ID:            index.GenerateReferenceID(filePath, lineNum, 0),
				ReferenceName: matches[1],
				FilePath:      filePath,
				LineNumber:    lineNum,
				ReferenceKind: index.ReferenceKindImport,
			})
		}
		return refs
	}

	// Match imports within import ( ... ) block
	// Lines like: "fmt" or "os" or alias "strings"
	if strings.Contains(line, `"`) || strings.Contains(line, `'`) {
		re := regexp.MustCompile(`["']([^"']+)["']`)
		matches := re.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			if len(m) > 1 {
				refs = append(refs, &index.SymbolReference{
					ID:            index.GenerateReferenceID(filePath, lineNum, 0),
					ReferenceName: m[1],
					FilePath:      filePath,
					LineNumber:    lineNum,
					ReferenceKind: index.ReferenceKindImport,
				})
			}
		}
	}

	return refs
}

func (p *GoParser) qualifiedName(packageName, receiver, name string) string {
	parts := []string{}
	if packageName != "" {
		parts = append(parts, packageName)
	}
	if receiver != "" {
		parts = append(parts, receiver)
	}
	parts = append(parts, name)
	return strings.Join(parts, ".")
}

func (p *GoParser) extractSignature(line, prefix string) string {
	// Simple signature extraction - finds matching parens
	if idx := strings.Index(line, prefix); idx != -1 {
		return strings.TrimSpace(line[idx:])
	}
	return ""
}

// init registers the Go parser
func init() {
	index.Register(NewGoParser())
}
