# Anthropic Agent Tests

This directory contains integration and end-to-end tests for the Anthropic-style agent using the official SDK adapter.

## Test Configuration

The tests use the following constants for real API testing:

```go
const (
    REAL_APIKey  = "tingly-box-..."
    REAL_BaseURL = "http://localhost:12580/tingly/anthropic"
    REAL_Model   = "tingly-box"
)
```

## Running Tests

### Unit Tests (No API Calls)

Run unit tests that don't require the API server:

```bash
go test ./tests/anthropic/... -run TestSDKAdapter_CreateClient -v
go test ./tests/anthropic/... -run TestSDKAdapter_Convert -v
```

### Integration Tests (Requires API Server)

To run integration tests that make real API calls:

1. **Start your Tingly Box server** (or the API endpoint you're testing):
   ```bash
   # Make sure your server is running at http://localhost:12580/tingly/anthropic
   ```

2. **Run tests with INTEGRATION_TEST=1**:
   ```bash
   INTEGRATION_TEST=1 go test ./tests/anthropic/... -v
   ```

3. **Run specific test**:
   ```bash
   INTEGRATION_TEST=1 go test ./tests/anthropic/... -run TestSDKAdapter_SimpleChat_Integration -v
   ```

4. **Run all tests except integration**:
   ```bash
   go test ./tests/anthropic/... -short -v
   ```

### Test Coverage

```bash
go test ./tests/anthropic/... -cover -v
```

## Test Files

### adapter_test.go

Tests for the SDK adapter implementation:

- **Unit Tests:**
  - `TestSDKAdapter_CreateClient` - Client creation
  - `TestSDKAdapter_CreateClientWithBaseURL` - Custom base URL support
  - `TestSDKAdapter_ConvertMessages` - Message conversion
  - `TestSDKAdapter_ConvertTools` - Tool schema conversion

- **Integration Tests:**
  - `TestSDKAdapter_SimpleChat_Integration` - Basic chat
  - `TestSDKAdapter_ChatWithSystemMessage_Integration` - System messages
  - `TestSDKAdapter_ChatWithTemperature_Integration` - Temperature control
  - `TestSDKAdapter_ChatWithMaxTokens_Integration` - Token limits
  - `TestSDKAdapter_MultiTurnConversation_Integration` - Conversation history
  - `TestSDKAdapter_StreamingChat_Integration` - Streaming responses
  - `TestSDKAdapter_ToolCalling_Integration` - Tool calling
  - `TestSDKAdapter_ToolResultResponse_Integration` - Tool result handling
  - `TestSDKAdapter_ErrorHandling_Integration` - Error scenarios

### e2e_test.go

End-to-end tests for complete agent workflows:

- `TestE2E_SimpleChat` - Basic agent interaction
- `TestE2E_MultiTurnConversation` - Memory and context
- `TestE2E_WithTemperature` - Temperature variations
- `TestE2E_MemoryCompression` - Memory management
- `TestE2E_StreamingResponse` - Streaming mode
- `TestE2E_ErrorRecovery` - Error handling
- `TestE2E_ContextCancellation` - Timeout handling
- `TestE2E_LongRunningTask` - Extended operations
- `TestE2E_StatePersistence` - State management
- `TestE2E_ConcurrentRequests` - Concurrent usage
- `TestE2E_MaxIterations` - Iteration limits
- `TestE2E_CustomSystemPrompt` - Custom prompts

## Test Scenarios

### User Feedback Issues Being Tested

1. **API Connectivity**: Tests verify the agent can connect to `localhost:12580`
2. **Authentication**: Tests use the provided API key format
3. **Model Selection**: Tests verify the "tingly-box" model works correctly
4. **Tool Calling**: Tests cover tool use scenarios
5. **Streaming**: Tests verify streaming responses work
6. **Error Handling**: Tests cover various error conditions

### E2E Test Coverage

The E2E tests verify:
- ✅ Agent creation and initialization
- ✅ Single-turn and multi-turn conversations
- ✅ Memory persistence across turns
- ✅ System prompt application
- ✅ Temperature and other parameters
- ✅ Streaming vs non-streaming modes
- ✅ Tool calling and result handling
- ✅ Error scenarios and recovery
- ✅ Context cancellation and timeouts
- ✅ Concurrent request handling
- ✅ State serialization

## Expected Test Results

When running with `INTEGRATION_TEST=1` and a working server:

```
=== RUN   TestSDKAdapter_SimpleChat_Integration
--- PASS: TestSDKAdapter_SimpleChat_Integration (2.34s)
    adapter_test.go:89: Response: Hello, World!
    adapter_test.go:97: Usage - Input: 15, Output: 8, Total: 23
=== RUN   TestE2E_SimpleChat
--- PASS: TestE2E_SimpleChat (3.12s)
    e2e_test.go:45: Agent response: The sum of 2 + 2 is 4.
PASS
ok      github.com/tingly-dev/tingly-agentscope/tests/anthropic    5.467s
```

## Troubleshooting

### Tests Fail with "Connection Refused"

**Problem**: Cannot connect to the API server.

**Solution**:
1. Ensure your Tingly Box server is running: `curl http://localhost:12580/tingly/anthropic`
2. Check the port matches `REAL_BaseURL` in the test constants
3. Verify firewall settings aren't blocking localhost connections

### Tests Fail with Authentication Error

**Problem**: API key is invalid or expired.

**Solution**:
1. Update `REAL_APIKey` constant with a valid key
2. Check the key format matches your server's expectations
3. Verify the key hasn't expired

### Tests Timeout

**Problem**: Tests take too long and timeout.

**Solution**:
1. Check network connectivity to the API server
2. Increase timeout in test context if needed
3. Verify server isn't overloaded

### Streaming Tests Fail

**Problem**: Streaming responses don't work correctly.

**Solution**:
1. Ensure your server supports SSE (Server-Sent Events)
2. Check for proxy issues that might buffer streaming responses
3. Verify the streaming implementation in SDKAdapter

## Adding New Tests

When adding new tests:

1. **Use descriptive names**: `Test<Feature>_<Scenario>_Integration`
2. **Skip by default**: Use `if os.Getenv("INTEGRATION_TEST") != "1" { t.Skip() }`
3. **Set appropriate timeouts**: Use `context.WithTimeout()` for API calls
4. **Log useful information**: Use `t.Logf()` for debugging
5. **Clean up resources**: Use `defer cancel()` for contexts

Example:

```go
func TestSDKAdapter_NewFeature_Integration(t *testing.T) {
    if os.Getenv("INTEGRATION_TEST") != "1" {
        t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 to enable.")
    }

    cfg := getRealTestConfig(t)
    client, err := NewSDKAdapter(cfg)
    if err != nil {
        t.Fatalf("Failed to create client: %v", err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Test implementation...
}
```

## Continuous Integration

To run these tests in CI:

```yaml
# Example GitHub Actions
- name: Run unit tests
  run: go test ./tests/anthropic/... -short -v

- name: Run integration tests
  env:
    INTEGRATION_TEST: 1
  run: |
    # Start your test server
    ./start-test-server.sh &
    # Run tests
    go test ./tests/anthropic/... -v
```

## Related Documentation

- [Architecture Documentation](../../docs/arch/main/anthropic-agent-arch.md)
- [SDK Adapter](../../pkg/model/anthropic/adapter_sdk.go)
- [ReAct Agent](../../pkg/agent/react_agent.go)
