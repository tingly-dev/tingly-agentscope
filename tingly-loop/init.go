package main

import (
	"bufio"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/urfave/cli/v2"
)

//go:embed prompts/generate_tasks.md
//go:embed prompts/create_spec.md
var generateTasksPromptFS embed.FS

var initCommand = &cli.Command{
	Name:  "init",
	Usage: "Interactively create a tasks.json template",
	Description: `Creates a basic tasks.json template through interactive prompts.
After creation, you can edit the file to add more stories or details.`,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "workdir",
			Aliases: []string{"w"},
			Usage:   "Working directory",
		},
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Output file path",
			Value:   "docs/loop/tasks.json",
		},
	},
	Action: func(c *cli.Context) error {
		workDir := c.String("workdir")
		if workDir == "" {
			var err error
			workDir, err = os.Getwd()
			if err != nil {
				return err
			}
		}

		outputPath := c.String("output")

		scanner := bufio.NewScanner(os.Stdin)

		fmt.Println("🚀 Tingly-Loop Tasks Generator")
		fmt.Println("This will create a tasks.json template for your project.")
		fmt.Println()

		// Project name
		fmt.Print("Project name: ")
		scanner.Scan()
		project := scanner.Text()

		// Branch name
		defaultBranch := "feature/" + strings.ToLower(strings.ReplaceAll(project, " ", "-"))
		fmt.Printf("Branch name [%s]: ", defaultBranch)
		scanner.Scan()
		branch := scanner.Text()
		if branch == "" {
			branch = defaultBranch
		}

		// Description
		fmt.Print("Feature description (one line): ")
		scanner.Scan()
		description := scanner.Text()

		// Collect user stories
		var stories []UserStory
		fmt.Println("\n📝 Enter user stories (press Enter with empty input to finish):")
		fmt.Println("   Format: <title> | <description>")
		fmt.Println("   Example: Add login button | As a user, I want to see a login button")

		storyNum := 1
		for {
			fmt.Printf("\nStory %d (or press Enter to finish): ", storyNum)
			scanner.Scan()
			input := scanner.Text()

			if input == "" {
				break
			}

			// Parse input
			parts := strings.SplitN(input, "|", 2)
			title := strings.TrimSpace(parts[0])
			desc := ""
			if len(parts) > 1 {
				desc = strings.TrimSpace(parts[1])
			} else {
				desc = "As a user, I want " + strings.ToLower(title)
			}

			stories = append(stories, UserStory{
				ID:                 fmt.Sprintf("US-%03d", storyNum),
				Title:              title,
				Description:        desc,
				AcceptanceCriteria: []string{"Specific criterion 1", "Specific criterion 2", "Typecheck passes", "Tests pass"},
				Priority:           storyNum,
				Passes:             false,
				Notes:              "",
			})
			storyNum++
		}

		if len(stories) == 0 {
			// Add a default story if none provided
			stories = append(stories, UserStory{
				ID:                 "US-001",
				Title:              "Example story - replace this",
				Description:        "As a user, I want [feature] so that [benefit]",
				AcceptanceCriteria: []string{"Specific verifiable criterion", "Typecheck passes", "Tests pass"},
				Priority:           1,
				Passes:             false,
				Notes:              "",
			})
		}

		// Create tasks
		tasks := &Tasks{
			Project:     project,
			BranchName:  branch,
			Description: description,
			UserStories: stories,
		}

		// Save
		if err := SaveTasks(outputPath, tasks); err != nil {
			return fmt.Errorf("failed to save tasks: %w", err)
		}

		fmt.Printf("\n✅ Created %s with %d stories\n", outputPath, len(stories))
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Edit the file to refine acceptance criteria")
		fmt.Println("  2. Run 'tingly-loop run' to start the loop")

		return nil
	},
}

var specCommand = &cli.Command{
	Name:  "spec",
	Usage: "Create a spec document from a feature description",
	Description: `Creates a spec document in docs/spec/ from a natural language description.

The spec document will include:
- Problem statement
- Proposed solution
- Open questions for discussion

After creating the spec, you can:
1. Edit it manually to refine details
2. Run 'tingly-loop run --spec <path>' for discussion phase
3. Run 'tingly-loop generate' to create tasks.json from the spec

Example:
  tingly-loop spec "Add user authentication with email and password"
  tingly-loop spec "Create a dashboard showing sales metrics"`,
	ArgsUsage: "<feature description>",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "workdir",
			Aliases: []string{"w"},
			Usage:   "Working directory",
		},
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Output spec file path",
		},
		&cli.StringFlag{
			Name:    "project",
			Aliases: []string{"p"},
			Usage:   "Project name (default: directory name)",
		},
		&cli.StringFlag{
			Name:  "agent",
			Usage: "Agent to use for generation",
			Value: "claude",
		},
		&cli.BoolFlag{
			Name:  "force",
			Usage: "Overwrite existing spec file",
		},
	},
	Action: func(c *cli.Context) error {
		if c.Args().Len() < 1 {
			return fmt.Errorf("usage: tingly-loop spec <feature description>")
		}

		featureDesc := c.Args().First()

		workDir := c.String("workdir")
		if workDir == "" {
			var err error
			workDir, err = os.Getwd()
			if err != nil {
				return err
			}
		}

		projectName := c.String("project")
		if projectName == "" {
			projectName = filepath.Base(workDir)
		}

		// Generate spec path if not provided
		specPath := c.String("output")
		if specPath == "" {
			dateStr := time.Now().Format("20060102")
			slug := strings.ToLower(strings.ReplaceAll(featureDesc, " ", "-"))
			// Remove non-alphanumeric chars except hyphen
			slug = strings.Map(func(r rune) rune {
				if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
					return r
				}
				return -1
			}, slug)
			// Limit slug length
			if len(slug) > 30 {
				slug = slug[:30]
			}
			specPath = filepath.Join(workDir, "docs", "spec", fmt.Sprintf("%s-%s.md", dateStr, slug))
		}

		// Check if file exists
		if _, err := os.Stat(specPath); err == nil && !c.Bool("force") {
			return fmt.Errorf("spec file already exists: %s\nUse --force to overwrite", specPath)
		}

		// Build the generation prompt
		prompt := buildCreateSpecPrompt(featureDesc, projectName, specPath)

		// Create agent
		cfg := &Config{
			WorkDir:      workDir,
			AgentType:    c.String("agent"),
			Instructions: "",
		}

		agent, err := CreateAgent(cfg)
		if err != nil {
			return fmt.Errorf("failed to create agent: %w", err)
		}

		fmt.Printf("📝 Creating spec using %s agent...\n", agent.Name())
		fmt.Printf("Feature: %s\n\n", featureDesc)

		// Call agent
		output, err := agent.Execute(c.Context, prompt)
		if err != nil {
			return fmt.Errorf("spec creation failed: %w", err)
		}

		// Ensure directory exists
		specDir := filepath.Dir(specPath)
		if err := os.MkdirAll(specDir, 0755); err != nil {
			return fmt.Errorf("failed to create spec directory: %w", err)
		}

		// Save spec
		if err := os.WriteFile(specPath, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to save spec: %w", err)
		}

		fmt.Printf("\n✅ Created spec: %s\n", specPath)
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Review and edit the spec to refine details")
		fmt.Printf("  2. Run 'tingly-loop run --spec %s' to discuss requirements\n", specPath)
		fmt.Println("  3. Run 'tingly-loop generate' to create tasks.json")

		return nil
	},
}

var generateCommand = &cli.Command{
	Name:  "generate",
	Usage: "Generate tasks.json from a spec document",
	Description: `Uses an AI worker to generate a structured tasks.json from an existing spec document.

If no spec is specified, uses the most recent spec in docs/spec/.

After creating a spec with 'tingly-loop spec', use this command to generate tasks.json.

Examples:
  tingly-loop generate                     # Use most recent spec
  tingly-loop generate --spec path/to.md   # Use specific spec`,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "workdir",
			Aliases: []string{"w"},
			Usage:   "Working directory",
		},
		&cli.StringFlag{
			Name:    "spec",
			Aliases: []string{"s"},
			Usage:   "Path to spec file",
		},
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Output file path",
			Value:   "docs/loop/tasks.json",
		},
		&cli.StringFlag{
			Name:  "agent",
			Usage: "Agent to use for generation",
			Value: "claude",
		},
	},
	Action: func(c *cli.Context) error {
		workDir := c.String("workdir")
		if workDir == "" {
			var err error
			workDir, err = os.Getwd()
			if err != nil {
				return err
			}
		}

		outputPath := c.String("output")

		// Find spec file
		specPath := c.String("spec")
		if specPath == "" {
			var err error
			specPath, err = FindSpecFile(workDir)
			if err != nil {
				return fmt.Errorf("no spec file found. Run 'tingly-loop spec <description>' first, or use --spec to specify a file: %w", err)
			}
		} else {
			// Validate spec file exists
			if _, err := os.Stat(specPath); os.IsNotExist(err) {
				return fmt.Errorf("spec file not found: %s", specPath)
			}
		}

		// Build the generation prompt
		prompt := buildSpecToTasksPrompt(specPath)

		// Create agent
		cfg := &Config{
			WorkDir:      workDir,
			AgentType:    c.String("agent"),
			Instructions: "",
		}

		agent, err := CreateAgent(cfg)
		if err != nil {
			return fmt.Errorf("failed to create agent: %w", err)
		}

		fmt.Printf("🤖 Generating tasks using %s agent...\n", agent.Name())
		fmt.Printf("Spec: %s\n\n", specPath)

		// Archive existing tasks if present
		if _, err := os.Stat(outputPath); err == nil {
			if err := archiveTasks(outputPath); err != nil {
				fmt.Printf("Warning: failed to archive tasks: %v\n", err)
			}
		}

		// Call agent
		output, err := agent.Execute(c.Context, prompt)
		if err != nil {
			return fmt.Errorf("generation failed: %w", err)
		}

		fmt.Println(output)

		// Try to extract and save JSON from output
		if err := extractAndSaveTasks(output, outputPath); err != nil {
			fmt.Printf("\n⚠️  Could not automatically extract tasks.json from output.\n")
			fmt.Printf("Please review the output above and create tasks.json manually.\n")
			return err
		}

		fmt.Printf("\n✅ Created %s\n", outputPath)
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Review and edit the generated tasks")
		fmt.Println("  2. Run 'tingly-loop run' to start the loop")

		return nil
	},
}

func buildGeneratePrompt(featureDesc, projectName string) string {
	tmplData, err := generateTasksPromptFS.ReadFile("prompts/generate_tasks.md")
	if err != nil {
		panic(err)
	}

	tmpl, err := template.New("generate_tasks").Parse(string(tmplData))
	if err != nil {
		panic(err)
	}

	data := struct {
		Project string
		Feature string
	}{
		Project: projectName,
		Feature: featureDesc,
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, data); err != nil {
		panic(err)
	}

	return result.String()
}

func buildCreateSpecPrompt(featureDesc, projectName, specPath string) string {
	tmplData, err := generateTasksPromptFS.ReadFile("prompts/create_spec.md")
	if err != nil {
		panic(err)
	}

	tmpl, err := template.New("create_spec").Parse(string(tmplData))
	if err != nil {
		panic(err)
	}

	data := struct {
		Project  string
		Feature  string
		SpecPath string
	}{
		Project:  projectName,
		Feature:  featureDesc,
		SpecPath: specPath,
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, data); err != nil {
		panic(err)
	}

	return result.String()
}

func buildSpecToTasksPrompt(specPath string) string {
	// Use the embedded specToTasksPrompt template
	return strings.ReplaceAll(specToTasksPrompt, "{{.SpecPath}}", specPath)
}

func extractAndSaveTasks(output, outputPath string) error {
	// Find JSON in output
	start := strings.Index(output, "{")
	end := strings.LastIndex(output, "}")
	if start == -1 || end == -1 || end < start {
		return fmt.Errorf("no valid JSON found in output")
	}

	jsonStr := output[start : end+1]

	// Validate it's valid tasks
	var tasks Tasks
	if err := json.Unmarshal([]byte(jsonStr), &tasks); err != nil {
		return fmt.Errorf("invalid tasks JSON: %w", err)
	}

	// Save
	return SaveTasks(outputPath, &tasks)
}
