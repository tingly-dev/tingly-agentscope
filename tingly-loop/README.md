# Tingly Loop

An autonomous AI agent **loop controller** based on the [Ralph pattern](https://ghuntley.com/ralph/). Tingly-loop manages the iteration loop and calls **agents** to do the actual work.

## Install
```
go install github.com/tingly-dev/tingly-agentscope/tingly-loop@latest
```

## Architecture

```
tingly-loop (Loop Controller)
    ├── Manages tasks state (docs/loop/tasks.json)
    ├── Tracks progress (docs/loop/progress.md)
    ├── Handles git branch switching
    └── Calls Agent each iteration
            │
            ├── claude CLI (default, like ralph)
            ├── tingly-code
            └── custom subprocess
                    │
                    └── Has full tool access (file, bash, etc.)
```

Unlike ralph which only supports external CLI tools, tingly-loop supports multiple agent types while providing the same loop control pattern.

## Installation

```bash
cd example/tingly-loop
go build -o tingly-loop .
```

## Usage

### Spec-Driven Workflow (Recommended)

```bash
# 1. Create a spec from your feature description
tingly-loop spec "Add user authentication with email and password"

# 2. (Optional) Run discussion to refine requirements
tingly-loop run --spec docs/spec/20260220-add-user-authentication.md

# 3. Generate tasks.json from the spec
tingly-loop generate

# 4. Run the loop to implement tasks
tingly-loop run
```

### Basic Usage (direct implementation)

```bash
# In a project directory with docs/loop/tasks.json
cd /path/to/project
tingly-loop run

# The claude CLI will be called with --dangerously-skip-permissions --print
```

### Using tingly-code as agent

```bash
tingly-loop run --agent tingly-code

# Or specify binary path
tingly-loop run --agent tingly-code --agent-binary /path/to/tingly-code
```

### Using custom subprocess

```bash
tingly-loop run --agent subprocess --agent-binary ./my-agent --agent-arg "--flag"
```

### CLI Commands

```bash
# Create a spec document from a feature description
tingly-loop spec "<feature description>" [options]

# Generate tasks.json from a spec document
tingly-loop generate [options]

# Run the loop
tingly-loop run [options]

# Show status without running
tingly-loop status [options]

# Interactively create tasks.json (manual)
tingly-loop init [options]
```

### Spec Workflow Commands

```bash
# Step 1: Create spec
tingly-loop spec "Add user authentication"
# → Creates docs/spec/20260220-add-user-authentication.md

# Step 2: (Optional) Discuss requirements
tingly-loop run --spec docs/spec/20260220-add-user-authentication.md
# → Agent asks questions, you can edit the spec between iterations

# Step 3: Generate tasks
tingly-loop generate
# → Uses most recent spec to create docs/loop/tasks.json

# Step 4: Implement
tingly-loop run
```

### Options

| Flag                   | Default                 | Description                                    |
| ---------------------- | ----------------------- | ---------------------------------------------- |
| `--tasks, -t`          | `docs/loop/tasks.json`  | Path to tasks JSON file                        |
| `--progress`           | `docs/loop/progress.md` | Path to progress log                           |
| `--spec`               | (auto-detect)           | Path to spec file (for discussion or generate) |
| `--skip-spec`          | `false`                 | Skip spec phase, go directly to implementation |
| `--max-iterations, -n` | `10`                    | Maximum loop iterations                        |
| `--agent`              | `claude`                | Agent type: claude, tingly-code, subprocess    |
| `--agent-binary`       | (auto-detect)           | Path to agent binary                           |
| `--agent-arg`          | (none)                  | Additional args for subprocess (repeatable)    |
| `--config, -c`         | (none)                  | Config file for agent                          |
| `--instructions, -i`   | (embedded)              | Custom instructions for claude agent           |
| `--workdir, -w`        | (current dir)           | Working directory                              |

## Tasks Format

Create a `docs/loop/tasks.json` file in your project:

```json
{
  "project": "MyProject",
  "branchName": "feature/my-feature",
  "description": "Feature description",
  "userStories": [
    {
      "id": "US-001",
      "title": "Story title",
      "description": "As a user, I want X so that Y",
      "acceptanceCriteria": [
        "Specific criterion 1",
        "Typecheck passes"
      ],
      "priority": 1,
      "passes": false,
      "notes": ""
    }
  ]
}
```

### Tasks Fields

- `project`: Project name
- `branchName`: Git branch to work on (created if doesn't exist)
- `description`: Overall feature description
- `userStories`: List of stories to implement
  - `id`: Unique identifier (e.g., US-001)
  - `title`: Short title
  - `description`: Full story description
  - `acceptanceCriteria`: List of verifiable criteria
  - `priority`: Execution order (lower = higher priority)
  - `passes`: Whether the story is complete (agent sets to true)
  - `notes`: Optional notes

## Progress Tracking

The `docs/loop/progress.md` file tracks iterations. The agent appends to this file after completing each story.

## Completion

The loop terminates when:

1. **Success**: Agent outputs `<promise>COMPLETE</promise>` (all stories pass)
2. **Max iterations**: Reached the maximum iteration limit

## Agent Types

### claude (Default)

Calls the claude CLI directly, similar to ralph. The instructions are passed via stdin.

```bash
tingly-loop run --agent claude
```

Requirements:
- `claude` CLI must be installed and in PATH

### tingly-code

Calls tingly-code in `auto` mode with the iteration prompt.

```bash
tingly-loop run --agent tingly-code --config /path/to/tingly-config.toml
```

Requirements:
- tingly-code binary built or available

### subprocess

Calls any custom binary that accepts the prompt via stdin.

```bash
tingly-loop run --agent subprocess \
  --agent-binary ./my-agent \
  --agent-arg "--verbose"
```

## Example Workflow

### Spec-Driven Workflow (Recommended)

```bash
# 1. Create a spec from your feature idea
tingly-loop spec "Add user authentication with email and password"
# → Creates docs/spec/20260220-add-user-authentication.md
# → Agent writes initial spec with problem statement and open questions

# 2. Review and optionally discuss the spec
vim docs/spec/20260220-add-user-authentication.md  # Edit manually if needed
tingly-loop run --spec docs/spec/20260220-add-user-authentication.md
# → Agent asks clarifying questions
# → When ready, agent outputs <discussion-complete/> and generates tasks

# 3. Generate tasks from the spec
tingly-loop generate
# → Reads spec and creates docs/loop/tasks.json

# 4. Run the implementation loop
tingly-loop run
# → Each iteration implements one story
# → Agent commits changes and updates tasks
# → Loop exits when all stories pass
```

### Quick Workflow (Skip Spec)

```bash
# 1. Create tasks manually or use init
tingly-loop init
# → Interactive prompt to create tasks.json

# 2. Run the loop
tingly-loop run
```

## Comparison with Ralph

| Feature          | Ralph                      | Tingly Loop                                     |
| ---------------- | -------------------------- | ----------------------------------------------- |
| Implementation   | Bash script                | Go program                                      |
| Agent Types      | claude, amp                | claude, tingly-code, subprocess                 |
| State Management | File I/O                   | File I/O + Go structs                           |
| Error Handling   | Basic                      | Structured with Go errors                       |
| Loop Control     | for loop in bash           | Go loop controller                              |
| Extensibility    | Limited                    | Pluggable Agent interface                       |
| Default Paths    | `prd.json`, `progress.txt` | `docs/loop/tasks.json`, `docs/loop/progress.md` |

## License

MIT
