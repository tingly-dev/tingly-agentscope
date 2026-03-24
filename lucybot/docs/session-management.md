# Session Management

## Session ID Format

Session IDs are 32-character hexadecimal strings generated using MD5 hashing.

### Generation Algorithm

- **Components:** timestamp + agent_name + first 128 chars of first query
- **Format:** `MD5("{timestamp}:{agent_name}:{query}")`
- **Length:** 32 hexadecimal characters
- **Generation:** Lazy (triggered on first user message)

### Example

```
Input:  "1705315845.123456:lucybot:Help me understand authentication"
Output: "a3f2b8c9d4e5f6a7b8c9d0e1f2a3b4c5"
```

### File Naming

Session files use the format: `{agent_name}_{session_id}.jsonl`

Example: `lucybot_a3f2b8c9d4e5f6a7b8c9d0e1f2a3b4c5.jsonl`

### Backward Compatibility

Old format files (without agent prefix) are still supported:
- Old: `{session_id}.jsonl`
- New: `{agent_name}_{session_id}.jsonl`

The system automatically detects and handles both formats.

## Implementation Details

### Lazy Generation

Session IDs are generated lazily on the first user message. This allows the system to:
- Defer session file creation until actual conversation starts
- Include the first query content in the session ID generation
- Avoid creating empty session files

### Session ID Generation Function

The `GenerateSessionID` function in `internal/session/generate.go` implements the generation algorithm:

```go
func GenerateSessionID(agentName string, firstQuery string) string
```

Parameters:
- `agentName`: The name of the agent (e.g., "lucybot")
- `firstQuery`: The first user message content (truncated to 128 characters)

Returns: A 32-character hexadecimal string

### File Storage

Session files are stored in the `.agentscope/sessions/` directory by default, with filenames following the pattern `{agent_name}_{session_id}.jsonl`.

Each session file contains:
1. A JSONL header with session metadata
2. JSONL message entries for each conversation turn
