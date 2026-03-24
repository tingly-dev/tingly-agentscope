# Code Analysis Workflows

## Deep Analysis Workflow

For comprehensive code understanding:

1. **Read the primary file**
   - Start with the main implementation
   - Note imports and dependencies
   - Identify key structures

2. **Trace execution flow**
   - Follow function calls
   - Understand the call chain
   - Note side effects

3. **Examine related code**
   - Look for parent classes/interfaces
   - Check sibling implementations
   - Find usage examples

4. **Search for patterns**
   - Find similar implementations
   - Identify design patterns used
   - Note architectural decisions

## Quick Reference Workflow

For answering specific questions:

1. **Identify the question type**
   - "What does X do?" → Read implementation
   - "How does X relate to Y?" → Find relationships
   - "Where is X used?" → Search for references

2. **Use targeted tools**
   - Read for direct inspection
   - Search for finding references
   - Grep for pattern matching

3. **Provide concise answer**
   - Direct response to question
   - Include relevant code snippet
   - Note file location

## Dependency Analysis Workflow

To understand code dependencies:

1. **Start with the target**
   - Read the implementation
   - List all imports
   - Note external dependencies

2. **Trace up the dependency tree**
   - What does this depend on?
   - What do those depend on?
   - Continue to root dependencies

3. **Analyze coupling**
   - Direct dependencies (imports)
   - Indirect dependencies (through calls)
   - Optional dependencies (interfaces)

4. **Document findings**
   - Dependency tree diagram
   - Critical dependencies
   - Potential refactoring opportunities
