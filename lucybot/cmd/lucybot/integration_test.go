package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tingly-dev/lucybot/internal/config"
)

func TestSkillsInstallationIntegration(t *testing.T) {
	// This test verifies the full skills installation workflow

	// Set up mock to find skills from the test's working directory
	// When running tests, we need to override the path since the test binary
	// is in a temp directory
	originalFunc := getBundledSkillsPathFunc
	defer func() { getBundledSkillsPathFunc = originalFunc }()

	// Find skills relative to current working directory (should be lucybot/cmd/lucybot)
	getBundledSkillsPathFunc = func() string {
		// Try ../../skills from current directory
		if cwd, err := os.Getwd(); err == nil {
			testPath := filepath.Join(cwd, "..", "..", "skills")
			if _, err := os.Stat(testPath); err == nil {
				absPath, _ := filepath.Abs(testPath)
				return absPath
			}
		}
		// Fallback to hardcoded path for this project
		return "/home/xiao/program/tingly-agentscope/lucybot/skills"
	}

	// 1. Verify bundled skills exist
	bundledPath := getBundledSkillsPath()
	if _, err := os.Stat(bundledPath); os.IsNotExist(err) {
		t.Skipf("Bundled skills not found at %s - run tests from project root", bundledPath)
	}

	// 2. Verify expected skills are present
	expectedSkills := []string{
		"code-analysis",
		"specification-generation",
		"verification",
	}

	for _, skill := range expectedSkills {
		skillPath := filepath.Join(bundledPath, skill, "SKILL.md")
		if _, err := os.Stat(skillPath); os.IsNotExist(err) {
			t.Errorf("Expected skill not found: %s at %s", skill, skillPath)
		}
	}

	t.Logf("All %d bundled skills verified", len(expectedSkills))
}

func TestConfigEnablesSkills(t *testing.T) {
	// Verify that config can enable skills

	cfg := config.GetDefaultConfig()

	// Initially should have default settings
	if cfg.Skills.Enabled != false {
		t.Errorf("Expected skills to be disabled by default, got: %v", cfg.Skills.Enabled)
	}

	// Enable skills
	cfg.Skills.Enabled = true
	cfg.Skills.Paths = []string{"~/.lucybot/skills"}

	// Verify
	if !cfg.Skills.Enabled {
		t.Errorf("Failed to enable skills")
	}

	if len(cfg.Skills.Paths) != 1 {
		t.Errorf("Expected 1 path, got %d", len(cfg.Skills.Paths))
	}
}

func TestSkillsCommandAvailability(t *testing.T) {
	// Verify that skills are registered as commands

	// Set up mock to find skills from the test's working directory
	originalFunc := getBundledSkillsPathFunc
	defer func() { getBundledSkillsPathFunc = originalFunc }()

	getBundledSkillsPathFunc = func() string {
		if cwd, err := os.Getwd(); err == nil {
			testPath := filepath.Join(cwd, "..", "..", "skills")
			if _, err := os.Stat(testPath); err == nil {
				absPath, _ := filepath.Abs(testPath)
				return absPath
			}
		}
		return "/home/xiao/program/tingly-agentscope/lucybot/skills"
	}

	// Load skills from bundled directory
	bundledPath := getBundledSkillsPath()
	if _, err := os.Stat(bundledPath); os.IsNotExist(err) {
		t.Skipf("Bundled skills not found at %s", bundledPath)
	}

	// This is a basic sanity check - in a real scenario we'd
	// initialize the agent and check command registry
	// For now, just verify the skills directory structure

	entries, err := os.ReadDir(bundledPath)
	if err != nil {
		t.Fatalf("Failed to read bundled skills: %v", err)
	}

	skillCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			skillPath := filepath.Join(bundledPath, entry.Name(), "SKILL.md")
			if _, err := os.Stat(skillPath); err == nil {
				skillCount++
			}
		}
	}

	if skillCount < 3 {
		t.Errorf("Expected at least 3 skills, found %d", skillCount)
	}

	t.Logf("Found %d skills in bundled directory", skillCount)
}
