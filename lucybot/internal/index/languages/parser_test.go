package languages

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tingly-dev/lucybot/internal/index"
)

func TestGoParser_ParseFunction(t *testing.T) {
	parser := NewGoParser()
	code := []byte(`package main

// TestFunc is a test function
func TestFunc(arg string) error {
	return nil
}
`)

	result, err := parser.Parse(context.Background(), code, "test.go")
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Should find the function
	require.Len(t, result.Symbols, 1)
	assert.Equal(t, "TestFunc", result.Symbols[0].Name)
	assert.Equal(t, index.SymbolKindFunction, result.Symbols[0].Kind)
	assert.Equal(t, "main.TestFunc", result.Symbols[0].QualifiedName)
	assert.Contains(t, result.Symbols[0].Documentation, "test function")
}

func TestGoParser_ParseMethod(t *testing.T) {
	parser := NewGoParser()
	code := []byte(`package main

type MyStruct struct{}

// Method comment
func (m *MyStruct) MyMethod() string {
	return "hello"
}
`)

	result, err := parser.Parse(context.Background(), code, "test.go")
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Should find the struct and method
	require.Len(t, result.Symbols, 2)

	var method *index.Symbol
	for _, s := range result.Symbols {
		if s.Name == "MyMethod" {
			method = s
			break
		}
	}
	require.NotNil(t, method)
	assert.Equal(t, index.SymbolKindMethod, method.Kind)
	assert.Equal(t, "main.MyStruct.MyMethod", method.QualifiedName)
}

func TestGoParser_ParseTypes(t *testing.T) {
	parser := NewGoParser()
	code := []byte(`package main

type MyInterface interface {
	Method() string
}

type MyStruct struct {
	Field string
}
`)

	result, err := parser.Parse(context.Background(), code, "test.go")
	require.NoError(t, err)

	require.Len(t, result.Symbols, 2)

	var iface, strct *index.Symbol
	for _, s := range result.Symbols {
		if s.Name == "MyInterface" {
			iface = s
		} else if s.Name == "MyStruct" {
			strct = s
		}
	}

	require.NotNil(t, iface)
	require.NotNil(t, strct)
	assert.Equal(t, index.SymbolKindInterface, iface.Kind)
	assert.Equal(t, index.SymbolKindClass, strct.Kind)
}

func TestGoParser_ParseImports(t *testing.T) {
	parser := NewGoParser()
	code := []byte(`package main

import "fmt"
`)

	result, err := parser.Parse(context.Background(), code, "test.go")
	require.NoError(t, err)

	// Single line import should be found
	assert.GreaterOrEqual(t, len(result.References), 1)

	var hasFmt bool
	for _, ref := range result.References {
		if ref.ReferenceName == "fmt" {
			hasFmt = true
			break
		}
	}

	assert.True(t, hasFmt, "should find fmt import")
}

func TestGoParser_ExtractsRelationships(t *testing.T) {
	parser := NewGoParser()
	code := []byte(`package main

func callee() {}

func caller1() {
	callee()
}

func caller2() {
	callee()
}
`)

	result, err := parser.Parse(context.Background(), code, "test.go")
	require.NoError(t, err)

	// Should find symbols
	require.GreaterOrEqual(t, len(result.Symbols), 3)

	// Should find call references
	var callRefs []*index.SymbolReference
	for _, ref := range result.References {
		if ref.ReferenceKind == index.ReferenceKindCall {
			callRefs = append(callRefs, ref)
		}
	}
	require.GreaterOrEqual(t, len(callRefs), 2)

	// Should build relationships
	require.GreaterOrEqual(t, len(result.Relationships), 2)

	// Verify relationships are of type "calls"
	for _, rel := range result.Relationships {
		assert.Equal(t, "calls", rel.RelationshipType)
	}
}

func TestPythonParser_ParseClass(t *testing.T) {
	parser := NewPythonParser()
	code := []byte(`"""Module docstring"""

class MyClass:
    """Class docstring"""

    def method(self):
        pass
`)

	result, err := parser.Parse(context.Background(), code, "test.py")
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Should find at least the class
	require.GreaterOrEqual(t, len(result.Symbols), 1)

	var class *index.Symbol
	for _, s := range result.Symbols {
		if s.Name == "MyClass" {
			class = s
			break
		}
	}
	require.NotNil(t, class)
	assert.Equal(t, index.SymbolKindClass, class.Kind)
	assert.Equal(t, "MyClass", class.QualifiedName)
}

func TestPythonParser_ParseFunction(t *testing.T) {
	parser := NewPythonParser()
	code := []byte(`def standalone_func():
    """Function docstring"""
    return 42

async def async_func():
    return await something()
`)

	result, err := parser.Parse(context.Background(), code, "test.py")
	require.NoError(t, err)

	// Should find at least one function
	require.GreaterOrEqual(t, len(result.Symbols), 1)

	var funcFound bool
	for _, s := range result.Symbols {
		if s.Name == "standalone_func" {
			funcFound = true
			assert.Equal(t, index.SymbolKindFunction, s.Kind)
			break
		}
	}

	assert.True(t, funcFound, "should find standalone_func")
}

func TestPythonParser_ParseImports(t *testing.T) {
	parser := NewPythonParser()
	code := []byte(`import os
import sys as system
from collections import defaultdict
from typing import List, Dict
`)

	result, err := parser.Parse(context.Background(), code, "test.py")
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(result.References), 3)

	var hasOs, hasCollections, hasTyping bool
	for _, ref := range result.References {
		switch ref.ReferenceName {
		case "os":
			hasOs = true
		case "collections":
			hasCollections = true
		case "typing":
			hasTyping = true
		}
	}

	assert.True(t, hasOs, "should find os import")
	assert.True(t, hasCollections, "should find collections import")
	assert.True(t, hasTyping, "should find typing import")
}

func TestParserRegistry(t *testing.T) {
	registry := index.NewParserRegistry()

	registry.Register(NewGoParser())
	registry.Register(NewPythonParser())

	// Test Go parser lookup
	goParser := registry.GetParserForFile("test.go")
	assert.NotNil(t, goParser)
	assert.Equal(t, index.LanguageGo, goParser.GetLanguage())

	// Test Python parser lookup
	pyParser := registry.GetParserForFile("test.py")
	assert.NotNil(t, pyParser)
	assert.Equal(t, index.LanguagePython, pyParser.GetLanguage())

	// Test unsupported file
	unknownParser := registry.GetParserForFile("test.unknown")
	assert.Nil(t, unknownParser)

	// Test supported languages
	langs := registry.GetSupportedLanguages()
	assert.Len(t, langs, 2)
}
