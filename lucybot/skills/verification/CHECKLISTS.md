# Verification Checklists

This document provides comprehensive checklists for verifying Go code correctness.

## Table of Contents

- [General Verification Checklist](#general-verification-checklist)
- [Function-Specific Checklist](#function-specific-checklist)
- [Common Issues to Look For](#common-issues-to-look-for)

---

## General Verification Checklist

### Correctness

- [ ] **Algorithm Correctness**: Does the algorithm produce the correct output for all valid inputs?
- [ ] **Logic Soundness**: Is the control flow logic correct and free of contradictions?
- [ ] **Mathematical Operations**: Are mathematical operations correct (division by zero, overflow, underflow)?
- [ ] **Data Type Usage**: Are appropriate data types used for the operations performed?

```go
// ❌ Incorrect: Doesn't handle negative numbers
func abs(n int) int {
    return n // Always returns positive, even for negative input
}

// ✅ Correct: Properly handles negative numbers
func abs(n int) int {
    if n < 0 {
        return -n
    }
    return n
}
```

### Input Validation

- [ ] **Nil Checks**: Are nil/pointer checks performed before dereferencing?
- [ ] **Empty/Zero Values**: Are empty strings, zero-length slices, and zero values handled?
- [ ] **Range Validation**: Are numeric inputs validated against expected ranges?
- [ ] **Format Validation**: Are string inputs validated for expected formats?
- [ ] **Type Assertions**: Are type assertions checked with the comma-ok pattern?

```go
// ❌ Missing nil check
func processUser(u *User) error {
    return u.Validate() // Panics if u is nil
}

// ✅ Proper nil check
func processUser(u *User) error {
    if u == nil {
        return errors.New("user cannot be nil")
    }
    return u.Validate()
}

// ❌ Unsafe type assertion
func getValue(m map[string]interface{}, key string) string {
    return m[key].(string) // Panics if key missing or not a string
}

// ✅ Safe type assertion
func getValue(m map[string]interface{}, key string) (string, bool) {
    val, ok := m[key].(string)
    return val, ok
}
```

### Error Handling

- [ ] **Error Checking**: Are all errors checked after operations that can fail?
- [ ] **Error Wrapping**: Are errors wrapped with context using `fmt.Errorf` or `errors.Wrap`?
- [ ] **Error Propagation**: Are errors properly propagated up the call stack?
- [ ] **Error Messages**: Are error messages informative and actionable?
- [ ] **Resource Cleanup**: Are resources cleaned up in error paths (defer statements)?

```go
// ❌ Not checking error
func readFile(path string) string {
    data, _ := os.ReadFile(path) // Silently ignores error
    return string(data)
}

// ✅ Proper error handling
func readFile(path string) (string, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return "", fmt.Errorf("failed to read file %s: %w", path, err)
    }
    return string(data), nil
}

// ❌ Resource not cleaned up on error
func processFile(path string) error {
    f, err := os.Open(path)
    if err != nil {
        return err
    }
    // If next line fails, file remains open
    _, err = io.ReadAll(f)
    f.Close()
    return err
}

// ✅ Proper cleanup with defer
func processFile(path string) error {
    f, err := os.Open(path)
    if err != nil {
        return err
    }
    defer f.Close()

    _, err = io.ReadAll(f)
    return err
}
```

### Edge Cases

- [ ] **Empty Inputs**: Are empty strings, nil slices, and nil maps handled?
- [ ] **Single Element**: Does the code work correctly with single-element collections?
- [ ] **Boundary Values**: Are minimum and maximum boundary values tested?
- [ ] **Duplicate Values**: Are duplicate values handled correctly?
- [ ] **Concurrent Access**: Is the code safe for concurrent use?

```go
// ❌ Doesn't handle empty slice
func first(items []string) string {
    return items[0] // Panics on empty slice
}

// ✅ Handles empty slice
func first(items []string) (string, bool) {
    if len(items) == 0 {
        return "", false
    }
    return items[0], true
}

// ❌ Off-by-one error in boundary
func sumInRange(numbers []int, max int) int {
    sum := 0
    for i := 0; i <= max; i++ { // Should be i < max
        if i < len(numbers) {
            sum += numbers[i]
        }
    }
    return sum
}

// ✅ Correct boundary handling
func sumInRange(numbers []int, max int) int {
    sum := 0
    for i := 0; i < max && i < len(numbers); i++ {
        sum += numbers[i]
    }
    return sum
}
```

### Resource Management

- [ ] **File Handles**: Are files always closed (use defer)?
- [ ] **Goroutines**: Are goroutines properly managed and cleaned up?
- [ ] **Channels**: Are channels properly closed and buffered appropriately?
- [ ] **Locks**: Are mutex locks unlocked in all code paths?
- [ ] **Connections**: Are network connections always closed?
- [ ] **Memory**: Are large resources released when no longer needed?

```go
// ❌ Lock not released on panic
func safeUpdate() {
    mu.Lock()
    // If this panics, mutex remains locked
    doSomething()
    mu.Unlock()
}

// ✅ Lock released even on panic
func safeUpdate() {
    mu.Lock()
    defer mu.Unlock()
    doSomething()
}

// ❌ Channel never closed
func producer() <-chan int {
    ch := make(chan int)
    go func() {
        for i := 0; i < 10; i++ {
            ch <- i
        }
        // Channel never closed, consumers will hang
    }()
    return ch
}

// ✅ Channel properly closed
func producer() <-chan int {
    ch := make(chan int)
    go func() {
        defer close(ch)
        for i := 0; i < 10; i++ {
            ch <- i
        }
    }()
    return ch
}
```

### Thread Safety

- [ ] **Shared State**: Is shared mutable state protected by locks?
- [ ] **Race Conditions**: Are there potential race conditions in concurrent code?
- [ ] **Deadlocks**: Is the code free from potential deadlocks?
- [ ] **Data Races**: Are there potential data races?
- [ ] **Sync Packages**: Are appropriate synchronization primitives used?

```go
// ❌ Race condition: multiple goroutines updating counter
type Counter struct {
    value int
}

func (c *Counter) Increment() {
    c.value++ // Not thread-safe
}

// ✅ Thread-safe with mutex
type SafeCounter struct {
    mu    sync.Mutex
    value int
}

func (c *SafeCounter) Increment() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.value++
}

// ❌ Potential deadlock: locking in inconsistent order
func transfer(a, b *Account, amount int) error {
    a.mu.Lock()
    b.mu.Lock() // Can deadlock if another goroutine does b.Lock(); a.Lock()
    defer a.mu.Unlock()
    defer b.mu.Unlock()
    // ... transfer logic
    return nil
}

// ✅ Deadlock-free: always lock in consistent order
func transfer(a, b *Account, amount int) error {
    // Always lock the account with lower ID first
    if a.id < b.id {
        a.mu.Lock()
        b.mu.Lock()
    } else {
        b.mu.Lock()
        a.mu.Lock()
    }
    defer a.mu.Unlock()
    defer b.mu.Unlock()
    // ... transfer logic
    return nil
}
```

---

## Function-Specific Checklist

### For Functions That Modify State

- [ ] **State Consistency**: Is the state left consistent after modification?
- [ ] **Atomicity**: Are related modifications done atomically?
- [ ] **Revertibility**: Can failed modifications be rolled back?
- [ ] **Notifications**: Are interested parties notified of changes?
- [ ] **Validation**: Is the new state validated before applying?

```go
// ❌ State inconsistency if validation fails
type Account struct {
    balance int
}

func (a *Account) Withdraw(amount int) error {
    if amount <= 0 {
        return errors.New("invalid amount")
    }
    if a.balance < amount {
        return errors.New("insufficient funds")
    }
    // What if balance changes between check and deduction?
    a.balance -= amount
    return nil
}

// ✅ Atomic state modification with locking
type SafeAccount struct {
    mu      sync.Mutex
    balance int
}

func (a *SafeAccount) Withdraw(amount int) error {
    if amount <= 0 {
        return errors.New("invalid amount")
    }
    a.mu.Lock()
    defer a.mu.Unlock()
    if a.balance < amount {
        return errors.New("insufficient funds")
    }
    a.balance -= amount
    return nil
}
```

### For Functions That Return Values

- [ ] **Return Type Consistency**: Is the return type consistent across all code paths?
- [ ] **Error Values**: Are error values properly set alongside return values?
- [ ] **Zero Values**: Are zero/nil values returned for error cases?
- [ ] **Multiple Returns**: Are multiple return values used correctly?
- [ ] **Pointer vs Value**: Is the correct return type used (pointer vs value)?

```go
// ❌ Inconsistent return types
func findUser(id int) *User {
    if id < 0 {
        return nil
    }
    // Missing return in some code path
}

// ✅ Consistent returns with error
func findUser(id int) (*User, error) {
    if id < 0 {
        return nil, errors.New("invalid id")
    }
    user, ok := userMap[id]
    if !ok {
        return nil, fmt.Errorf("user %d not found", id)
    }
    return user, nil
}

// ❌ Returning pointer to local variable
func createUserData() *UserData {
    data := UserData{Name: "test"}
    return &data // Safe in Go, but consider implications
}

// ✅ Clear ownership transfer
func createUserData() *UserData {
    return &UserData{Name: "test"}
}
```

### For Functions That Process Collections

- [ ] **Empty Collections**: Are empty collections handled correctly?
- [ ] **Nil Collections**: Are nil slices/maps handled differently from empty ones?
- [ ] **Index Bounds**: Are all array/slice accesses bounds-checked?
- [ ] **Iteration Safety**: Is the collection not modified during iteration?
- [ ] **Memory Efficiency**: Is memory usage appropriate for large collections?

```go
// ❌ Doesn't distinguish nil from empty
func processItems(items []string) []string {
    if len(items) == 0 {
        return items // Returns nil even if input was empty
    }
    // ... processing
    return items
}

// ✅ Preserves nil vs empty distinction
func processItems(items []string) []string {
    if items == nil {
        return nil
    }
    if len(items) == 0 {
        return items
    }
    // ... processing
    return items
}

// ❌ Modifying collection during iteration
func removeDuplicates(items []string) []string {
    seen := make(map[string]bool)
    for i, item := range items {
        if seen[item] {
            items = append(items[:i], items[i+1:]...) // Unsafe iteration
        }
        seen[item] = true
    }
    return items
}

// ✅ Safe iteration with new slice
func removeDuplicates(items []string) []string {
    if len(items) == 0 {
        return items
    }
    seen := make(map[string]bool)
    result := make([]string, 0, len(items))
    for _, item := range items {
        if !seen[item] {
            seen[item] = true
            result = append(result, item)
        }
    }
    return result
}
```

### For Functions That Do I/O

- [ ] **Timeout Handling**: Are I/O operations protected with timeouts?
- [ ] **Context Usage**: Is context used for cancellation?
- [ ] **Resource Cleanup**: Are resources cleaned up in all cases?
- [ ] **Retry Logic**: Is transient failure handled with retry?
- [ ] **Buffer Management**: Are buffers sized appropriately?

```go
// ❌ No timeout, can hang forever
func fetchURL(url string) ([]byte, error) {
    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    return io.ReadAll(resp.Body)
}

// ✅ Proper timeout and context
func fetchURL(ctx context.Context, url string) ([]byte, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    return io.ReadAll(resp.Body)
}

// ❌ Not checking read count
func readData(r io.Reader, buf []byte) {
    io.ReadFull(r, buf) // Ignores error
}

// ✅ Proper error checking
func readData(r io.Reader, buf []byte) error {
    n, err := io.ReadFull(r, buf)
    if err != nil {
        return fmt.Errorf("read %d/%d bytes: %w", n, len(buf), err)
    }
    return nil
}
```

---

## Common Issues to Look For

### Missing Checks

#### Missing Nil Checks
```go
// ❌ Missing nil check
func (s *Server) HandleRequest(req *Request) {
    s.process(req.Data) // Panics if req or req.Data is nil
}

// ✅ Proper nil checks
func (s *Server) HandleRequest(req *Request) error {
    if req == nil {
        return errors.New("request cannot be nil")
    }
    if req.Data == nil {
        return errors.New("request data cannot be nil")
    }
    return s.process(req.Data)
}
```

#### Missing Error Checks
```go
// ❌ Not checking error
func writeConfig(data []byte) {
    os.WriteFile("config.json", data, 0644) // Error ignored
}

// ✅ Checking and handling error
func writeConfig(data []byte) error {
    err := os.WriteFile("config.json", data, 0644)
    if err != nil {
        return fmt.Errorf("failed to write config: %w", err)
    }
    return nil
}
```

#### Missing Boundary Checks
```go
// ❌ No boundary check
func getNth(items []int, n int) int {
    return items[n] // Panics if n >= len(items)
}

// ✅ Proper boundary check
func getNth(items []int, n int) (int, error) {
    if n < 0 || n >= len(items) {
        return 0, fmt.Errorf("index %d out of bounds [0, %d)", n, len(items))
    }
    return items[n], nil
}
```

### Uncaught Exceptions

#### In Go: Unhandled Errors
```go
// ❌ Unhandled error in goroutine
func processAsync(data []string) {
    for _, d := range data {
        go func(s string) {
            result := processString(s) // Error ignored
            results <- result
        }(d)
    }
}

// ✅ Proper error handling in goroutine
func processAsync(data []string) <-chan error {
    errChan := make(chan error, len(data))
    for _, d := range data {
        go func(s string) {
            err := processString(s)
            errChan <- err
        }(d)
    }
    return errChan
}
```

#### File Operations Without Error Handling
```go
// ❌ No error handling
func readConfig() *Config {
    data, _ := os.ReadFile("config.json")
    cfg, _ := parseConfig(data)
    return cfg
}

// ✅ Proper error handling
func readConfig() (*Config, error) {
    data, err := os.ReadFile("config.json")
    if err != nil {
        return nil, fmt.Errorf("failed to read config: %w", err)
    }
    cfg, err := parseConfig(data)
    if err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }
    return cfg, nil
}
```

### Inconsistent Returns

#### Inconsistent Error Handling
```go
// ❌ Sometimes returns error, sometimes panics
func parseValue(s string) (int, error) {
    if s == "" {
        panic("empty string") // Should return error
    }
    val, err := strconv.Atoi(s)
    if err != nil {
        return 0, err
    }
    return val, nil
}

// ✅ Consistent error handling
func parseValue(s string) (int, error) {
    if s == "" {
        return 0, errors.New("empty string")
    }
    val, err := strconv.Atoi(s)
    if err != nil {
        return 0, fmt.Errorf("invalid integer %q: %w", s, err)
    }
    return val, nil
}
```

#### Inconsistent Return Types
```go
// ❌ Sometimes returns slice, sometimes nil
func filterItems(items []int, pred func(int) bool) []int {
    var result []int
    for _, item := range items {
        if pred(item) {
            result = append(result, item)
        }
    }
    if len(result) == 0 {
        return nil // Inconsistent with empty case
    }
    return result
}

// ✅ Consistent returns
func filterItems(items []int, pred func(int) bool) []int {
    result := make([]int, 0, len(items))
    for _, item := range items {
        if pred(item) {
            result = append(result, item)
        }
    }
    return result // Always returns slice, possibly empty
}
```

### Resource Leaks

#### Unclosed Files
```go
// ❌ File not closed on error path
func processFile(path string) error {
    f, err := os.Open(path)
    if err != nil {
        return err
    }
    // If this errors, file is not closed
    data := make([]byte, 1024)
    _, err = f.Read(data)
    f.Close()
    return err
}

// ✅ Always closed with defer
func processFile(path string) error {
    f, err := os.Open(path)
    if err != nil {
        return err
    }
    defer f.Close()

    data := make([]byte, 1024)
    _, err = f.Read(data)
    return err
}
```

#### Goroutine Leaks
```go
// ❌ Goroutine leaks if context cancelled
func monitor(ctx context.Context) {
    go func() {
        for {
            select {
            case <-ctx.Done():
                return
            case <-time.After(time.Second):
                // If this blocks, goroutine leaks
                doWork()
            }
        }
    }()
}

// ✅ Proper goroutine cleanup
func monitor(ctx context.Context) {
    go func() {
        ticker := time.NewTicker(time.Second)
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                doWork()
            }
        }
    }()
}
```

---

## Usage Tips

1. **Start with the general checklist** for every review
2. **Use function-specific checklists** based on what the function does
3. **Look for common issues** as a quick sanity check
4. **Provide code examples** when reporting issues
5. **Be specific** about what's wrong and how to fix it
6. **Consider the context** - some "issues" might be intentional design choices
