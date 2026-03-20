package index

import (
	"github.com/tingly-dev/lucybot/internal/index/types"
)

// Re-export types from the types package for backward compatibility
type Symbol = types.Symbol
type SymbolReference = types.SymbolReference
type Scope = types.Scope
type Relationship = types.Relationship
type FileInfo = types.FileInfo
type SymbolKind = types.SymbolKind
type ReferenceKind = types.ReferenceKind
type Language = types.Language

// Re-export constants
const (
	SymbolKindFunction   = types.SymbolKindFunction
	SymbolKindClass      = types.SymbolKindClass
	SymbolKindMethod     = types.SymbolKindMethod
	SymbolKindVariable   = types.SymbolKindVariable
	SymbolKindParameter  = types.SymbolKindParameter
	SymbolKindModule     = types.SymbolKindModule
	SymbolKindInterface  = types.SymbolKindInterface
	SymbolKindTypeAlias  = types.SymbolKindTypeAlias
	SymbolKindConstant   = types.SymbolKindConstant
	SymbolKindProperty   = types.SymbolKindProperty
	SymbolKindEnum       = types.SymbolKindEnum
	SymbolKindEnumMember = types.SymbolKindEnumMember
	SymbolKindUnknown    = types.SymbolKindUnknown

	ReferenceKindRead            = types.ReferenceKindRead
	ReferenceKindWrite           = types.ReferenceKindWrite
	ReferenceKindCall            = types.ReferenceKindCall
	ReferenceKindImport          = types.ReferenceKindImport
	ReferenceKindInstantiation   = types.ReferenceKindInstantiation
	ReferenceKindAttributeAccess = types.ReferenceKindAttributeAccess
	ReferenceKindDefinition      = types.ReferenceKindDefinition

	LanguageGo         = types.LanguageGo
	LanguagePython     = types.LanguagePython
	LanguageJavaScript = types.LanguageJavaScript
	LanguageTypeScript = types.LanguageTypeScript
	LanguageJava       = types.LanguageJava
	LanguageRust       = types.LanguageRust
	LanguageCpp        = types.LanguageCpp
	LanguageC          = types.LanguageC
	LanguageUnknown    = types.LanguageUnknown
)

// Re-export functions
var GenerateSymbolID = types.GenerateSymbolID
var GenerateReferenceID = types.GenerateReferenceID
var GenerateScopeID = types.GenerateScopeID
var DetectLanguage = types.DetectLanguage

// IndexMetadata stores metadata about the index itself
type IndexMetadata struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// IndexVersion is the current schema version
const IndexVersion = 1

