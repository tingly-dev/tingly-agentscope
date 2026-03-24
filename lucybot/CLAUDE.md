# LucyBot Development Notes

This file contains patterns and conventions specific to LucyBot development that are useful for future development work.

## Session Management

- Session IDs are generated lazily on first user message using MD5 hash
- Session files include agent name prefix: `{agent}_{id}.jsonl`
- Use `GenerateSessionID()` from `internal/session` package for manual ID generation
- Backward compatible with old format files (no agent prefix)

## File Organization

- `internal/session/`: Session storage and management
- `docs/`: Documentation including skills system and session management
- Session files stored in `.agentscope/sessions/` by default

## Important Patterns

- When modifying session handling, ensure backward compatibility with old session file formats
- The session recorder performs lazy initialization - session files are only created on first message
- Agent name is passed through the initialization chain and used for session file naming
