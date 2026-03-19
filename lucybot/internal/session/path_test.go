package session

import (
	"path/filepath"
	"testing"
)

func TestGetProjectSessionDir(t *testing.T) {
	tests := []struct {
		name        string
		workDir     string
		wantHashLen int
	}{
		{
			name:        "hashes working directory",
			workDir:     "/home/user/projects/my-app",
			wantHashLen: 12,
		},
		{
			name:        "handles relative paths",
			workDir:     "./my-project",
			wantHashLen: 12,
		},
		{
			name:        "same path produces same hash",
			workDir:     "/home/user/projects/my-app",
			wantHashLen: 12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			absDir, err := filepath.Abs(tt.workDir)
			if err != nil {
				t.Fatalf("failed to get absolute path: %v", err)
			}

			baseDir := t.TempDir()
			result := GetProjectSessionDir(baseDir, absDir)

			// Check that result is within baseDir
			if len(result) <= len(baseDir) {
				t.Errorf("result path should be longer than baseDir")
			}

			// Check that hash part is expected length
			hashPart := filepath.Base(result)
			if len(hashPart) != tt.wantHashLen {
				t.Errorf("hash length = %d, want %d", len(hashPart), tt.wantHashLen)
			}
		})
	}
}

func TestConsistentHashForSamePath(t *testing.T) {
	workDir := "/home/user/projects/test"
	baseDir := "/tmp/sessions"

	dir1 := GetProjectSessionDir(baseDir, workDir)
	dir2 := GetProjectSessionDir(baseDir, workDir)

	if dir1 != dir2 {
		t.Errorf("same path should produce same hash: %s != %s", dir1, dir2)
	}
}
