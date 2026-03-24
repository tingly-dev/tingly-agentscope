package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// getBundledSkillsPathFunc is a variable that holds the function to get the bundled skills path.
// This allows mocking in tests.
var getBundledSkillsPathFunc func() string

// getBundledSkillsPath returns the path to the bundled skills directory.
// It detects whether we're in development or production mode.
func getBundledSkillsPath() string {
	// If a mock function is set, use it
	if getBundledSkillsPathFunc != nil {
		return getBundledSkillsPathFunc()
	}

	// Get the executable path
	execPath, err := os.Executable()
	if err != nil {
		// Fallback to current directory
		execPath = os.Args[0]
	}

	// Get the directory containing the executable
	execDir := filepath.Dir(execPath)

	// Check if we're in development mode (running from cmd/lucybot)
	// In development, the skills directory is at ../../skills relative to the executable
	devPath := filepath.Join(execDir, "..", "..", "skills")
	if _, err := os.Stat(devPath); err == nil {
		absPath, err := filepath.Abs(devPath)
		if err == nil {
			return absPath
		}
	}

	// Check if we're running from the module root (e.g., lucybot/lucybot binary)
	// In this case, skills are in ./skills relative to the executable
	modulePath := filepath.Join(execDir, "skills")
	if _, err := os.Stat(modulePath); err == nil {
		absPath, err := filepath.Abs(modulePath)
		if err == nil {
			return absPath
		}
	}

	// In production, the skills directory is at ../skills relative to the installed binary
	prodPath := filepath.Join(execDir, "..", "skills")
	absPath, err := filepath.Abs(prodPath)
	if err != nil {
		// Last resort: return the prod path even if we can't make it absolute
		return prodPath
	}

	return absPath
}

// installSkills copies the bundled skills to the target directory.
func installSkills(targetDir string) error {
	srcDir := getBundledSkillsPath()

	// Check if source directory exists
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		return fmt.Errorf("bundled skills directory not found: %s", srcDir)
	}

	// Create target directory if it doesn't exist
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Copy skills from source to target
	if err := copyDir(srcDir, targetDir); err != nil {
		return fmt.Errorf("failed to copy skills: %w", err)
	}

	return nil
}

// copyDir recursively copies a directory tree from src to dst.
func copyDir(src, dst string) error {
	// Get source directory info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectory
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Get source file info
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	// Create destination file
	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy content
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return nil
}

// hasSkills checks if the target directory has skills installed.
func hasSkills(targetDir string) bool {
	// Check if directory exists
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		return false
	}

	// Check if directory has any skill subdirectories
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return false
	}

	// A directory with skills should have at least one subdirectory
	for _, entry := range entries {
		if entry.IsDir() {
			// Check if it's a skill directory (contains skill.json or .gitkeep)
			skillPath := filepath.Join(targetDir, entry.Name())
			if isSkillDirectory(skillPath) {
				return true
			}
		}
	}

	return false
}

// isSkillDirectory checks if a directory is a valid skill directory.
func isSkillDirectory(dir string) bool {
	// Check for skill.json
	skillJsonPath := filepath.Join(dir, "skill.json")
	if _, err := os.Stat(skillJsonPath); err == nil {
		return true
	}

	// Check for .gitkeep (for empty skills)
	gitkeepPath := filepath.Join(dir, ".gitkeep")
	if _, err := os.Stat(gitkeepPath); err == nil {
		return true
	}

	return false
}
