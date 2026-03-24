---
name: code-analysis
description: Analyze code patterns, find relationships, and provide behavioral insights. Use when the user asks about code architecture, patterns, dependencies, or behavior.
version: 1.0.0
---

# Code Analysis Skill

## Overview

This skill provides systematic code analysis capabilities for understanding code patterns, relationships, and behaviors in a codebase.

## When to Use

Activate this skill when the user asks about:
- Code architecture and design patterns
- Function/class relationships and dependencies
- Code behavior and execution flow
- Implementation details
- "How does X work?"
- "What is the architecture of...?"
- "Analyze the [X] code"

## Analysis Workflow

Follow this systematic approach:

### 1. Identify the Target
Extract the primary symbol names from the user's query:
- Look for capitalized words (class names)
- Look for snake_case words (function names)
- Identify the main subject of the question

### 2. Examine the Code
Use `view_source(symbol)` to read the actual implementation:

```python
view_source("MyClass")
view_source("my_function")
```

### 3. Find Relationships
Use `traverse_code()` to understand relationships:

```python
traverse_code("MyClass", "callees", 1)     # What this calls
traverse_code("MyClass", "callers", 1)     # What calls this
traverse_code("MyClass", "children", 1)    # Subclasses
```

### 4. Search for Patterns
Use `grep()` to find similar patterns:

```python
grep("class.*Base", "**/*.py")  # Find all classes with "Base"
```

### 5. Synthesize Findings
Provide comprehensive analysis with:
- Clear structure using markdown headers
- Specific file locations and line numbers
- Code snippets as evidence
- Explanation of patterns and relationships

## Output Format

Structure your analysis as:

```markdown
## Analysis: [Symbol Name]

### Purpose
[Brief description of what this code does]

### Implementation
[Key implementation details with code snippets]

### Relationships
- **Calls**: [list of functions this calls]
- **Called by**: [list of functions that call this]
- **Related**: [related classes/functions]

### Patterns
[Architectural patterns, design patterns, code patterns identified]

### File Locations
- `path/to/file.py:123` - [description]
```

## Tips

1. **Start with view_source**: Always read the actual code first
2. **Follow the relationships**: Use traverse_code to understand connections
3. **Be specific**: Include exact file paths and line numbers
4. **Show evidence**: Include code snippets to support your analysis
5. **Think hierarchically**: Understand the broader architecture before details

## Example

**User**: "Analyze the authentication flow"

**Process**:
1. Identify symbols: `authenticate`, `AuthHandler`, `login`
2. `view_source("AuthHandler")` - Read the handler
3. `traverse_code("AuthHandler", "callees", 1)` - Find what it calls
4. `traverse_code("AuthHandler", "callers", 1)` - Find what calls it
5. Synthesize into comprehensive analysis
