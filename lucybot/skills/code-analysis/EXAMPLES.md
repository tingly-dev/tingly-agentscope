# Code Analysis Examples

## Example 1: Understanding a Compression Function

**User Query**: "How does the compression work in agentscope?"

**Analysis Result**:

```markdown
## Analysis: Compression Functionality

### Purpose
Agentscope provides compression utilities to reduce message size and token count, particularly useful for long conversation histories.

### Implementation

**File**: `agentscope/utils/compress.py`

The compression system provides two main compression strategies:

1. **Token-based compression** - Uses tiktoken to compress text
2. **Semantic compression** - Summarizes content while preserving key information

Key functions:
- `compress_msg()` - Main entry point for message compression
- `compress_by_token()` - Token-based compression using tiktoken
- `compress_by_semantic()` - Semantic summarization

### Relationships

**CompressionManager** class orchestrates the compression:
- Calls `compress_by_token()` for token-based strategies
- Calls `compress_by_semantic()` for semantic strategies
- Used by `Memory` class when storing conversations

### Patterns

- **Strategy Pattern**: Different compression algorithms can be selected
- **Decorator Pattern**: Compression wraps message storage
- **Configuration-based**: Compression level is configurable

### File Locations
- `agentscope/utils/compress.py:45` - `compress_msg()` function
- `agentscope/memory/memory.py:123` - Integration with memory system
```

---

## Example 2: Analyzing Reply Function Call Graph

**User Query**: "Trace how a reply is generated in agentscope"

**Analysis Result**:

```markdown
## Analysis: Reply Generation Flow

### Purpose
Understand the complete flow from user message to agent response in the agentscope framework.

### Execution Flow

```
User Input
    ↓
Agent.reply()
    ↓
ModelAgent.generate_response()
    ↓
ModelAgent._format_prompt()
    ↓
ModelAgent._call_model_api()
    ↓
ModelAgent._parse_response()
    ↓
Return Msg object
```

### Implementation Details

**Step 1: Agent.reply()**
- **File**: `agentscope/agents/agent.py:89`
- Validates input message
- Updates conversation history
- Delegates to subclass implementation

**Step 2: ModelAgent.generate_response()**
- **File**: `agentscope/agents/model_agent.py:156`
- Retrieves model configuration
- Formats prompt with system message and history
- Calls model API

**Step 3: ModelAgent._call_model_api()**
- **File**: `agentscope/agents/model_agent.py:203`
- Handles API communication
- Manages retries and rate limiting
- Returns raw response

**Step 4: ModelAgent._parse_response()**
- **File**: `agentscope/agents/model_agent.py:234`
- Extracts content from API response
- Creates Msg object
- Handles parsing errors

### Relationships

**Calls**:
- `Agent.reply()` → `ModelAgent.generate_response()`
- `ModelAgent.generate_response()` → `_format_prompt()`
- `ModelAgent.generate_response()` → `_call_model_api()`
- `ModelAgent.generate_response()` → `_parse_response()`

**Called By**:
- User code directly calls `Agent.reply()`
- Framework's orchestration layer uses `reply()`

**Related Classes**:
- `Msg` - Message data structure
- `ModelWrapper` - API communication layer
- `Memory` - Conversation history storage

### Patterns

- **Template Method**: `reply()` defines skeleton, subclasses implement specifics
- **Wrapper Pattern**: ModelWrapper abstracts API details
- **Immutable Messages**: Msg objects are never mutated after creation

### File Locations
- `agentscope/agents/agent.py:89` - Base Agent.reply()
- `agentscope/agents/model_agent.py:156` - ModelAgent.generate_response()
- `agentscope/models/model.py:78` - ModelWrapper API calls
- `agentscope/message.py:45` - Msg class definition
```

---

## Example 3: Architecture Analysis - Skills System

**User Query**: "What is the architecture of the skills system?"

**Analysis Result**:

```markdown
## Analysis: Skills System Architecture

### Purpose
The skills system provides a plugin architecture for extending agent capabilities with reusable, discoverable skills.

### Architecture Overview

The skills system follows a **plugin-based architecture** with four main components:

```
┌─────────────────────────────────────────┐
│          Skill Discovery                │
│  (SKILL.md files in skills/ directories)│
└──────────────┬──────────────────────────┘
               │
               ↓
┌─────────────────────────────────────────┐
│         Skill Registry                  │
│  (Maintains index of available skills)  │
└──────────────┬──────────────────────────┘
               │
               ↓
┌─────────────────────────────────────────┐
│        Skill Executor                   │
│  (Loads and executes skill logic)       │
└──────────────┬──────────────────────────┘
               │
               ↓
┌─────────────────────────────────────────┐
│        Agent Integration                │
│  (Agents invoke skills via interface)   │
└─────────────────────────────────────────┘
```

### Component Details

**1. Skill Discovery**
- **Location**: `skills/*/SKILL.md`
- **Pattern**: Each skill has a SKILL.md metadata file
- **Purpose**: Declarative skill definition with name, description, trigger patterns
- **Example**: `skills/code-analysis/SKILL.md`

**2. Skill Registry**
- **File**: `lucybot/skills/registry.py:45`
- **Class**: `SkillRegistry`
- **Purpose**: Indexes all available skills for fast lookup
- **Methods**:
  - `register(skill)` - Add skill to registry
  - `lookup(query)` - Find skills matching query
  - `list_all()` - Get all registered skills

**3. Skill Executor**
- **File**: `lucybot/skills/executor.py:67`
- **Class**: `SkillExecutor`
- **Purpose**: Dynamically loads and executes skill code
- **Key Methods**:
  - `execute(skill_name, context)` - Run a skill
  - `validate(skill_name)` - Check skill validity
  - `get_dependencies(skill_name)` - Return required dependencies

**4. Agent Integration**
- **File**: `lucybot/agent.py:123`
- **Pattern**: Agents have a `skills` attribute
- **Invocation**: `agent.skills.execute("code-analysis", query)`
- **Purpose**: Clean interface for agents to use skills

### Relationships

```
SkillDiscovery
    ↓ discovers
SkillRegistry
    ↓ provides
SkillExecutor
    ↓ serves
Agent
```

**Dependencies**:
- SkillRegistry depends on SkillDiscovery for indexing
- SkillExecutor depends on SkillRegistry for resolution
- Agent depends on SkillExecutor for execution

### Patterns

**1. Plugin Architecture**
- Skills are dynamically discovered at runtime
- No hard-coded skill references
- Easy to add new skills without modifying core

**2. Registry Pattern**
- Central registry maintains skill index
- Fast lookup by name or pattern
- Singleton instance shared across application

**3. Dependency Injection**
- Skills receive their dependencies via context parameter
- No tight coupling to specific implementations
- Testable with mock dependencies

**4. Convention Over Configuration**
- Skills follow standard directory structure
- SKILL.md provides metadata
- No manual registration required

### File Locations

**Core Components**:
- `lucybot/skills/registry.py:45` - SkillRegistry class
- `lucybot/skills/executor.py:67` - SkillExecutor class
- `lucybot/agent.py:123` - Agent skills integration

**Example Skills**:
- `lucybot/skills/code-analysis/SKILL.md` - Code analysis skill definition
- `lucybot/skills/git-operations/SKILL.md` - Git operations skill definition

**Utilities**:
- `lucybot/skills/utils.py:23` - Helper functions for skill loading
- `lucybot/skills/discovery.py:12` - Directory scanning logic

### Key Design Decisions

1. **File-based Discovery**: Skills are discovered by scanning directories, not by importing
   - Rationale: Allows skills to be added without code changes
   - Trade-off: Slightly slower startup time

2. **Markdown Metadata**: SKILL.md files use markdown for human readability
   - Rationale: Easy to edit and review
   - Trade-off: Requires parsing (frontmatter + content)

3. **Dynamic Execution**: Skills are executed dynamically, not pre-loaded
   - Rationale: Lower memory footprint
   - Trade-off: First execution has loading overhead

4. **Context Passing**: All execution context passed as single parameter
   - Rationale: Clean interface, easy to extend
   - Trade-off: Less type safety
```
