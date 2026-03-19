package session

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"path/filepath"
)

// GetProjectSessionDir returns the session directory for a project.
// The directory is organized as: <baseDir>/projects/<pathHash>/
// where pathHash is the first 12 characters of the MD5 hash of the working directory.
func GetProjectSessionDir(baseDir, workingDir string) string {
	// Hash the working directory path
	hash := md5.Sum([]byte(workingDir))
	hashStr := hex.EncodeToString(hash[:])[:12]

	return filepath.Join(baseDir, "projects", hashStr)
}

// GetSessionPath returns the full path to a session file.
// Format: <projectDir>/<agentName>_<sessionId>.jsonl
func GetSessionPath(baseDir, workingDir, agentName, sessionID string) string {
	projectDir := GetProjectSessionDir(baseDir, workingDir)
	filename := fmt.Sprintf("%s_%s.jsonl", agentName, sessionID)
	return filepath.Join(projectDir, filename)
}
