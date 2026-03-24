---
name: specification-generation
description: Generate formal specifications from code. Use when the user asks to generate specs, documentation, formal descriptions, or "spec for X".
version: 1.0.0
---

# Specification Generation Skill

## Overview

This skill analyzes code and generates formal specifications describing behavior, constraints, and interfaces.

## When to Use

Activate this skill when the user asks to:
- Generate a specification for a function/class
- Document code behavior
- Extract formal descriptions
- "What's the spec for...?"
- "Generate documentation for..."
- "Spec out the [X] function"

## Specification Workflow

### 1. Identify the Target
Extract the symbol name from the user's query.

### 2. Examine the Code
Use `view_source(symbol)` to read the implementation.

### 3. Analyze Components
Extract and document:
- Function signature
- Parameters (names, types, purposes)
- Return value
- Behavior
- Preconditions
- Postconditions
- Side effects

### 4. Generate Specification
Use the format below to structure the specification.

## Specification Format

```markdown
## Specification: [Symbol Name]

### Signature
```python
[Function signature from source]
```

### Purpose
[What this function/class does]

### Parameters
- `param1`: [Type] - [Description]
- `param2`: [Type] - [Description]

### Return Value
[Description of what is returned]

### Preconditions
- [Required conditions for valid input]
- [State requirements]

### Postconditions
- [Guarantees after successful execution]
- [State changes]

### Side Effects
- [I/O operations]
- [State mutations]
- [External interactions]

### Usage Example
```python
[Example showing typical usage]
```
```

## Tips

1. **Be precise**: Extract exact parameter names and types from code
2. **Infer behavior**: Analyze the implementation to understand what it does
3. **Document constraints**: Note any validation or error handling
4. **Provide context**: Explain why certain parameters are required
5. **Include examples**: Show typical usage patterns

## Inference Guidelines

When analyzing code:

**Parameters**: Look for function signature after `def`
```python
def view_source(symbol: str, file_path: str | None = None)
# Parameters: symbol (str), file_path (optional str)
```

**Behavior**: Analyze the function body
- If it calls `index.find_symbol()`: "Uses code index for lookup"
- If it reads files: "Reads and processes file contents"
- If it returns formatted output: "Formats and returns results"

**Preconditions**: Look for validation
- `if not index:`: "Requires code index to exist"
- `if not symbol:`: "Requires non-empty symbol name"

**Side Effects**: Look for I/O
- `file.read_text()`: "Reads from filesystem"
- `print()`: "Writes to console"
- Network calls: "Makes network requests"

## Example

**User**: "Generate a spec for view_source"

**Process**:
1. `view_source("view_source")`
2. Extract signature: `def view_source(symbol, file_path=None, context_lines=10)`
3. Analyze body for behavior, preconditions, side effects
4. Generate formatted specification
