package agent

import (
	"context"
	"fmt"
	"os"
	"strings"

	"example/tingly-code/config"
	"example/tingly-code/tools"

	"github.com/tingly-dev/tingly-agentscope/pkg/agent"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// DiffAgent creates patch files from git changes
type DiffAgent struct {
	*agent.ReActAgent
	bashTools *tools.BashTools
}

// Default system prompt for DiffAgent
var defaultDiffPrompt = strings.Join([]string{
	`You are Diff, a specialized AI assistant for creating patch files from git changes.`,
	``,
	`Your task is to extract all file changes from the current git repository and create a patch file, excluding test-related content.`,
	``,
	`## Available Tools`,
	``,
	`- execute_bash: Run shell commands for git operations`,
	``,
	`## Task Steps`,
	``,
	`1. Get all changes: Use git status to identify modified, added, and deleted files`,
	``,
	`2. Generate patch: Use git diff for unstaged changes and git diff --cached for staged changes`,
	``,
	`3. Filter test files: Exclude files and directories matching these patterns:`,
	`   - Files with "test" in the name (e.g., test_*.py, *_test.py, test*.java)`,
	`   - Directories named "test", "tests", "__tests__", "testing"`,
	`   - Files in "test_*" or "*_test" directories`,
	`   - Test fixtures and mocks directories`,
	``,
	`4. Create patch file: Write the filtered diff to a file named changes.patch`,
	``,
	`5. Report summary: List included and excluded files`,
	``,
	`## Filtering Guidelines`,
	``,
	`Exclude files matching any of these patterns:`,
	`- **/test*.py`,
	`- **/*test*.py`,
	`- **/test/**/*.py`,
	`- **/tests/**/*.py`,
	`- **/__tests__/**/*.py`,
	`- **/fixtures/**/*.py`,
	`- **/mocks/**/*.py`,
	`- **/conftest.py`,
	`- **/*.test.ts`,
	`- **/*.test.js`,
	`- **/*.spec.ts`,
	`- **/*.spec.js`,
	``,
	`## Output Format`,
	``,
	`The patch file should be a standard unified diff format that can be applied with:`,
	`git apply changes.patch`,
	`# or`,
	`patch -p1 < changes.patch`,
	``,
	`Always respond in English.`,
	`Always respond with exactly one tool call.`,
}, "\n")

// NewDiffAgent creates a new DiffAgent
func NewDiffAgent(cfg *config.AgentConfig) (*DiffAgent, error) {
	// Create model
	factory := NewModelFactory()
	chatModel, err := factory.CreateModel(&cfg.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to create model: %w", err)
	}

	// Create type-safe toolkit with only bash tools
	tt := tools.NewTypedToolkit()
	bashSession := tools.GetGlobalBashSession()
	bashTools := tools.NewBashTools(bashSession)
	tt.RegisterAll(bashTools, map[string]string{
		"ExecuteBash": tools.ToolDescExecuteBash,
		"JobDone":     tools.ToolDescJobDone,
	})

	// Get system prompt
	systemPrompt := cfg.Prompt.System
	if systemPrompt == "" {
		systemPrompt = defaultDiffPrompt
	}

	// Create memory
	memory := agent.NewSimpleMemory(50)

	// Create ReAct agent
	reactAgent := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name:          cfg.Name,
		SystemPrompt:  systemPrompt,
		Model:         chatModel,
		Toolkit:       &TypedToolkitAdapter{tt: tt},
		Memory:        memory,
		MaxIterations: 10,
		Temperature:   &cfg.Model.Temperature,
		MaxTokens:     &cfg.Model.MaxTokens,
	})

	return &DiffAgent{
		ReActAgent: reactAgent,
		bashTools:  bashTools,
	}, nil
}

// NewDiffAgentFromConfigFile creates a DiffAgent from a config file
func NewDiffAgentFromConfigFile(configPath string) (*DiffAgent, error) {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return NewDiffAgent(&cfg.Agent)
}

// CreatePatch creates a patch file from git changes
func (da *DiffAgent) CreatePatch(ctx context.Context, outputFilename string) error {
	if outputFilename == "" {
		outputFilename = "changes.patch"
	}

	// Check if we're in a git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		return fmt.Errorf("not in a git repository")
	}

	// Get git status
	prompt := fmt.Sprintf("Create a patch file named '%s' from all current git changes, excluding test files. After creating the patch, call job_done.", outputFilename)

	msg := message.NewMsg(
		"user",
		[]message.ContentBlock{message.Text(prompt)},
		types.RoleUser,
	)

	_, err := da.ReActAgent.Reply(ctx, msg)
	return err
}

// IsJobDone checks if the task is complete
func (da *DiffAgent) IsJobDone(msg *message.Msg) bool {
	if msg == nil {
		return true
	}

	// Check for job_done in content
	blocks, ok := msg.Content.([]message.ContentBlock)
	if !ok {
		return false
	}

	for _, block := range blocks {
		if textBlock, ok := block.(*message.TextBlock); ok {
			if strings.Contains(strings.ToLower(textBlock.Text), "job_done") ||
				strings.Contains(strings.ToLower(textBlock.Text), "patch file has been created") {
				return true
			}
		}
	}

	// Check metadata for interrupted flag
	if msg.Metadata != nil {
		if interrupted, ok := msg.Metadata["_is_interrupted"].(bool); ok && interrupted {
			return true
		}
	}

	return false
}

// GetPatchFiles returns the list of files included in a patch
func GetPatchFiles(patchContent string) ([]string, error) {
	var files []string
	lines := strings.Split(patchContent, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- ") {
			// Extract file path
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				filePath := strings.TrimPrefix(parts[1], "a/")
				filePath = strings.TrimPrefix(filePath, "b/")
				files = append(files, filePath)
			}
		}
	}

	return files, nil
}

// FilterTestFiles filters out test files from a list
func FilterTestFiles(files []string) []string {
	var filtered []string
	testPatterns := []string{
		"test", "Test", "TEST",
		"_test.go", "_test.py", "_test.js", "_test.ts",
		"test_.go", "test_.py", "test_.js", "test_.ts",
		".test.", ".spec.",
	}

	for _, file := range files {
		isTest := false
		for _, pattern := range testPatterns {
			if strings.Contains(file, pattern) {
				isTest = true
				break
			}
		}
		if !isTest {
			filtered = append(filtered, file)
		}
	}

	return filtered
}
