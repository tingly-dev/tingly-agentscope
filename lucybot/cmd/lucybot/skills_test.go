package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetBundledSkillsPath(t *testing.T) {
	// Get the path
	path := getBundledSkillsPath()

	// Should be an absolute path
	if !filepath.IsAbs(path) {
		t.Errorf("Expected absolute path, got: %s", path)
	}

	// Should end with "skills"
	if filepath.Base(path) != "skills" {
		t.Errorf("Expected path to end with 'skills', got: %s", path)
	}

	// Should exist (since we're in the lucybot directory)
	if info, err := os.Stat(path); err != nil {
		t.Logf("Warning: Bundled skills path doesn't exist yet: %s", path)
	} else if !info.IsDir() {
		t.Errorf("Bundled skills path is not a directory: %s", path)
	}
}
