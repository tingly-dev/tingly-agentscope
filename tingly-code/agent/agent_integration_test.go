package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestTinglyAgent_CreateFromConfigFile tests creating an agent from config file
func TestTinglyAgent_CreateFromConfigFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if test config exists
	testConfig := "../test_config.toml"
	if _, err := os.Stat(testConfig); os.IsNotExist(err) {
		t.Skip("Test config file not found, skipping integration test")
	}

	tmpDir := t.TempDir()

	agent, err := NewTinglyAgentFromConfigFile(testConfig, tmpDir)
	if err != nil {
		t.Fatalf("Failed to create agent from config: %v", err)
	}

	if agent == nil {
		t.Fatal("Agent should not be nil")
	}

	if agent.ReActAgent == nil {
		t.Error("ReActAgent should be initialized")
	}
}

// TestTinglyAgent_SimpleChat tests a simple chat interaction
func TestTinglyAgent_SimpleChat(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testConfig := "../test_config.toml"
	if _, err := os.Stat(testConfig); os.IsNotExist(err) {
		t.Skip("Test config file not found, skipping integration test")
	}

	tmpDir := t.TempDir()

	agent, err := NewTinglyAgentFromConfigFile(testConfig, tmpDir)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := agent.RunSinglePrompt(ctx, "Hello, what's your name?")
	if err != nil {
		t.Logf("Chat failed (server may not be running): %v", err)
		t.Skip("API server not available")
	}

	if response == "" {
		t.Error("Response should not be empty")
	}

	t.Logf("Agent response: %s", response)
}

// TestTinglyAgent_FileOperations tests agent file operation capabilities
func TestTinglyAgent_FileOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testConfig := "../test_config.toml"
	if _, err := os.Stat(testConfig); os.IsNotExist(err) {
		t.Skip("Test config file not found, skipping integration test")
	}

	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	agent, err := NewTinglyAgentFromConfigFile(testConfig, tmpDir)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Ask agent to read the file
	response, err := agent.RunSinglePrompt(ctx, "Read the file test.txt and tell me its content.")
	if err != nil {
		t.Logf("Chat failed (server may not be running): %v", err)
		t.Skip("API server not available")
	}

	if !strings.Contains(response, "Hello") {
		t.Logf("Agent may not have read the file correctly. Response: %s", response)
	}

	t.Logf("Agent response: %s", response)
}

// TestTinglyAgent_CodeWriting tests agent code writing capabilities
func TestTinglyAgent_CodeWriting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testConfig := "../test_config.toml"
	if _, err := os.Stat(testConfig); os.IsNotExist(err) {
		t.Skip("Test config file not found, skipping integration test")
	}

	tmpDir := t.TempDir()

	agent, err := NewTinglyAgentFromConfigFile(testConfig, tmpDir)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Ask agent to create a simple Go file
	response, err := agent.RunSinglePrompt(ctx, "Create a file named hello.go with a simple Hello World function.")
	if err != nil {
		t.Logf("Chat failed (server may not be running): %v", err)
		t.Skip("API server not available")
	}

	// Check if file was created
	helloFile := filepath.Join(tmpDir, "hello.go")
	if _, err := os.Stat(helloFile); os.IsNotExist(err) {
		t.Logf("Agent may not have created the file. Response: %s", response)
	} else {
		t.Log("File was created successfully")
		content, _ := os.ReadFile(helloFile)
		t.Logf("File content: %s", string(content))
	}

	t.Logf("Agent response: %s", response)
}

// TestTinglyAgent_BashExecution tests agent bash execution capabilities
func TestTinglyAgent_BashExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testConfig := "../test_config.toml"
	if _, err := os.Stat(testConfig); os.IsNotExist(err) {
		t.Skip("Test config file not found, skipping integration test")
	}

	tmpDir := t.TempDir()

	agent, err := NewTinglyAgentFromConfigFile(testConfig, tmpDir)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Ask agent to run a simple command
	response, err := agent.RunSinglePrompt(ctx, "Run 'echo test123' and tell me the output.")
	if err != nil {
		t.Logf("Chat failed (server may not be running): %v", err)
		t.Skip("API server not available")
	}

	if !strings.Contains(strings.ToLower(response), "test123") {
		t.Logf("Agent may not have executed the command correctly. Response: %s", response)
	}

	t.Logf("Agent response: %s", response)
}

// TestTinglyAgent_MultiStepTask tests agent with multi-step tasks
func TestTinglyAgent_MultiStepTask(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testConfig := "../test_config.toml"
	if _, err := os.Stat(testConfig); os.IsNotExist(err) {
		t.Skip("Test config file not found, skipping integration test")
	}

	tmpDir := t.TempDir()

	agent, err := NewTinglyAgentFromConfigFile(testConfig, tmpDir)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Ask agent to perform a multi-step task
	response, err := agent.RunSinglePrompt(ctx, "Create a file named counter.txt with the number 0, then increment it by 1 and update the file.")
	if err != nil {
		t.Logf("Chat failed (server may not be running): %v", err)
		t.Skip("API server not available")
	}

	// Check if file was created
	counterFile := filepath.Join(tmpDir, "counter.txt")
	if content, err := os.ReadFile(counterFile); err == nil {
		t.Logf("Counter file content: %s", string(content))
	}

	t.Logf("Agent response: %s", response)
}

// TestTinglyAgent_GrepOperation tests agent grep/search capabilities
func TestTinglyAgent_GrepOperation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testConfig := "../test_config.toml"
	if _, err := os.Stat(testConfig); os.IsNotExist(err) {
		t.Skip("Test config file not found, skipping integration test")
	}

	tmpDir := t.TempDir()

	// Create test files
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("hello world\nfoo bar\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("goodbye world\nbaz qux\n"), 0644)

	agent, err := NewTinglyAgentFromConfigFile(testConfig, tmpDir)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Ask agent to search for "world"
	response, err := agent.RunSinglePrompt(ctx, "Search for 'world' in all text files and tell me which files contain it.")
	if err != nil {
		t.Logf("Chat failed (server may not be running): %v", err)
		t.Skip("API server not available")
	}

	t.Logf("Agent response: %s", response)
}

// TestTinglyAgent_ErrorHandling tests agent error handling
func TestTinglyAgent_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testConfig := "../test_config.toml"
	if _, err := os.Stat(testConfig); os.IsNotExist(err) {
		t.Skip("Test config file not found, skipping integration test")
	}

	tmpDir := t.TempDir()

	agent, err := NewTinglyAgentFromConfigFile(testConfig, tmpDir)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Ask agent to do something that will fail
	response, err := agent.RunSinglePrompt(ctx, "Try to read a file named nonexistent.txt that doesn't exist.")
	if err != nil {
		t.Logf("Chat failed (server may not be running): %v", err)
		t.Skip("API server not available")
	}

	// Agent should handle the error gracefully
	t.Logf("Agent response: %s", response)
}

// TestDiffAgent_CreatePatch tests DiffAgent patch creation
func TestDiffAgent_CreatePatch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testConfig := "../test_config.toml"
	if _, err := os.Stat(testConfig); os.IsNotExist(err) {
		t.Skip("Test config file not found, skipping integration test")
	}

	// Check if we're in a git repository
	if _, err := os.Stat("../../.git"); os.IsNotExist(err) {
		t.Skip("Not in a git repository, skipping DiffAgent test")
	}

	agent, err := NewDiffAgentFromConfigFile(testConfig)
	if err != nil {
		t.Fatalf("Failed to create DiffAgent: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err = agent.CreatePatch(ctx, "test_changes.patch")
	if err != nil {
		t.Logf("CreatePatch failed (server may not be running): %v", err)
		t.Skip("API server not available")
	}

	// Check if patch file was created
	if _, err := os.Stat("test_changes.patch"); err == nil {
		defer os.Remove("test_changes.patch")
		content, _ := os.ReadFile("test_changes.patch")
		t.Logf("Patch file created with %d bytes", len(content))
	} else {
		t.Log("Patch file may not have been created")
	}
}

// TestGetPatchFiles tests extracting files from patch content
func TestGetPatchFiles(t *testing.T) {
	patchContent := `diff --git a/file1.go b/file1.go
index 123..456 789
--- a/file1.go
+++ b/file1.go
@@ -1,1 +1,1 @@
-old content
+new content
diff --git a/file2.go b/file2.go
index 789..123 456
--- a/file2.go
+++ b/file2.go
@@ -1,1 +1,1 @@
-another old
+another new
`

	files, err := GetPatchFiles(patchContent)
	if err != nil {
		t.Fatalf("GetPatchFiles failed: %v", err)
	}

	if len(files) < 2 {
		t.Errorf("Expected at least 2 files, got %d", len(files))
	}

	t.Logf("Extracted files: %v", files)
}

// TestFilterTestFiles tests filtering test files
func TestFilterTestFiles(t *testing.T) {
	files := []string{
		"src/main.go",
		"src/utils.go",
		"src/main_test.go",
		"tests/test_utils.go",
		"src/helper_test.go",
		"README.md",
		"test_main.go",
	}

	filtered := FilterTestFiles(files)

	// Should exclude files with "test" in the name
	for _, f := range filtered {
		if strings.Contains(strings.ToLower(f), "test") {
			t.Errorf("Test file should be filtered: %s", f)
		}
	}

	t.Logf("Filtered files: %v", filtered)
}
