package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// archiveTasks moves existing tasks.json to archive
func archiveTasks(tasksPath string) error {
	archiveDir := filepath.Join(filepath.Dir(tasksPath), "..", "archive")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return fmt.Errorf("failed to create archive directory: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	archiveName := fmt.Sprintf("tasks-%s.json", timestamp)
	archivePath := filepath.Join(archiveDir, archiveName)

	// Read existing file
	data, err := os.ReadFile(tasksPath)
	if err != nil {
		return fmt.Errorf("failed to read existing tasks: %w", err)
	}

	// Write to archive
	if err := os.WriteFile(archivePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write archive: %w", err)
	}

	fmt.Printf("Archived existing tasks: %s\n", archivePath)
	return nil
}

// FindSpecFile finds the most recent spec file in docs/spec/
func FindSpecFile(workDir string) (string, error) {
	specDir := filepath.Join(workDir, "docs", "spec")
	entries, err := os.ReadDir(specDir)
	if err != nil {
		return "", fmt.Errorf("failed to read spec directory: %w", err)
	}

	// Find most recent .md file
	var newest string
	var newestTime time.Time
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(newestTime) {
			newestTime = info.ModTime()
			newest = filepath.Join(specDir, entry.Name())
		}
	}

	if newest == "" {
		return "", fmt.Errorf("no spec files found in %s", specDir)
	}

	return newest, nil
}

// IsSpecReady checks if a spec is ready for implementation
func IsSpecReady(specPath string) (bool, error) {
	content, err := os.ReadFile(specPath)
	if err != nil {
		return false, fmt.Errorf("failed to read spec: %w", err)
	}
	return strings.Contains(string(content), "[x] Ready for Implementation"), nil
}

// SpecExists checks if any spec file exists
func SpecExists(workDir string) bool {
	specDir := filepath.Join(workDir, "docs", "spec")
	entries, err := os.ReadDir(specDir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			return true
		}
	}
	return false
}
