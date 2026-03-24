---
name: verification
description: Verify code correctness and check properties. Use when the user asks to verify, validate, check correctness, or test code.
version: 1.0.0
---

# Verification Skill

## Overview

This skill verifies code correctness, checks properties, and validates implementations against specifications.

## When to Use

Activate this skill when the user asks to:
- Verify code is correct
- Check if code meets requirements
- Validate implementations
- "Is this correct?"
- "Check that..."
- "Verify the [X] implementation"

## Verification Workflow

### 1. Understand the Claim
Identify what property or behavior is being verified.

### 2. Examine the Code
Use `view_source()` to read the implementation.

### 3. Check Key Aspects
- **Correctness**: Does the code do what it claims?
- **Edge cases**: Are edge cases handled?
- **Error handling**: Are errors properly handled?
- **Consistency**: Is the behavior consistent?

### 4. Report Findings
Provide clear verification results with evidence.

## Verification Report Format

```markdown
## Verification: [Target]

### Claim
[What is being verified]

### Examination
[Code analysis findings]

### Findings
- ✅ [Correct aspects]
- ⚠️ [Potential issues]
- ❌ [Definite problems]

### Recommendations
[Suggestions for improvement]
```

## Verification Checklist

### Correctness
- [ ] Does the code implement the stated behavior?
- [ ] Are algorithms correct?
- [ ] Is the logic sound?

### Edge Cases
- [ ] Empty inputs handled?
- [ ] Null/None checks present?
- [ ] Boundary conditions handled?
- [ ] Invalid input handling?

### Error Handling
- [ ] Exceptions caught appropriately?
- [ ] Error messages informative?
- [ ] Failures handled gracefully?

### Consistency
- [ ] Similar operations handled consistently?
- [ ] Naming conventions followed?
- [ ] Return types consistent?

## Tips

1. **Be thorough**: Check multiple aspects (correctness, edge cases, errors)
2. **Provide evidence**: Reference specific code locations
3. **Be constructive**: Offer specific improvement suggestions
4. **Stay objective**: Base findings on actual code analysis
5. **Use symbols**: ✅ for correct, ⚠️ for warnings, ❌ for problems

## Common Issues to Look For

### Missing Checks
```python
# ❌ Missing None check
def process(value):
    return value.upper()

# ✅ Proper handling
def process(value):
    if value is None:
        return None
    return value.upper()
```

### Uncaught Exceptions
```python
# ❌ No error handling
def read_file(path):
    return Path(path).read_text()

# ✅ Proper error handling
def read_file(path):
    try:
        return Path(path).read_text()
    except (OSError, UnicodeDecodeError) as e:
        return f"Error reading file: {e}"
```

### Inconsistent Returns
```python
# ❌ Inconsistent return types
def get_value(key):
    if key in cache:
        return cache[key]
    else:
        return None  # Sometimes returns value, sometimes None

# ✅ Consistent behavior
def get_value(key):
    return cache.get(key)  # Always returns value or None
```

## Example

**User**: "Verify the authenticate function is correct"

**Process**:
1. `view_source("authenticate")`
2. Check correctness: Does it actually authenticate?
3. Check edge cases: Empty password? None username?
4. Check error handling: Invalid credentials handled?
5. Generate verification report with findings
