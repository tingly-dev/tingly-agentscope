package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestInitConfigWithSkills(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "lucybot-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a bundled skills directory for testing
	bundledSkills := filepath.Join(tmpDir, "bundled", "skills")
	if err := os.MkdirAll(filepath.Join(bundledSkills, "test-skill"), 0755); err != nil {
		t.Fatal(err)
	}
	skillContent := []byte("---\nname: test-skill\ndescription: Test\n---\n# Test")
	if err := os.WriteFile(filepath.Join(bundledSkills, "test-skill", "SKILL.md"), skillContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Save current getBundledSkillsPathFunc and restore after test
	originalFunc := getBundledSkillsPathFunc
	defer func() { getBundledSkillsPathFunc = originalFunc }()

	// Mock getBundledSkillsPathFunc to return our test directory
	getBundledSkillsPathFunc = func() string {
		abs, _ := filepath.Abs(bundledSkills)
		return abs
	}

	// We can't easily test the full interactive init-config,
	// but we can test the installSkills function directly
	targetDir := filepath.Join(tmpDir, "skills")

	if err := installSkills(targetDir); err != nil {
		t.Fatalf("installSkills failed: %v", err)
	}

	// Check that skills were copied
	skillPath := filepath.Join(targetDir, "test-skill", "SKILL.md")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		t.Errorf("Skill was not copied to target directory")
	}

	// Verify content
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(content, []byte("name: test-skill")) {
		t.Errorf("Skill content is incorrect")
	}

	// Test idempotency - running again should skip existing
	if err := installSkills(targetDir); err != nil {
		t.Fatalf("installSkills failed on second run: %v", err)
	}
}
