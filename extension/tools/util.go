package tools

import (
	"fmt"
	"path/filepath"
	"strings"
)

// validatePath checks if the path is allowed and safe.
// It prevents directory traversal attacks by ensuring the resolved
// path is within one of the allowed directories.
func validatePath(path string, allowedDirs []string) error {
	if len(allowedDirs) == 0 {
		return nil // No restrictions
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Normalize path to resolve any symlinks or relative components
	absPath = filepath.Clean(absPath)

	for _, allowedDir := range allowedDirs {
		absAllowedDir, err := filepath.Abs(allowedDir)
		if err != nil {
			return fmt.Errorf("invalid allowed directory %q: %w", allowedDir, err)
		}

		// Clean the allowed directory path as well
		absAllowedDir = filepath.Clean(absAllowedDir)

		// Check if the path is exactly the allowed directory or within it
		// by checking path prefix with proper path separator handling
		if absPath == absAllowedDir {
			return nil
		}

		// Add path separator to ensure we're checking for prefix, not partial match
		// e.g., /safe should not match /safe2/file
		prefix := absAllowedDir + string(filepath.Separator)
		if strings.HasPrefix(absPath, prefix) {
			return nil
		}
	}

	return fmt.Errorf("path not allowed: %s", path)
}
