package boot

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalInstall(t *testing.T) {
	tmpDir := t.TempDir()
	li := NewLocalInstall(tmpDir)

	ctx := context.Background()

	// Test start
	if err := li.Start(ctx); err != nil {
		t.Fatalf("LocalInstall Start failed: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("Root directory was not created")
	}

	// Test execute
	output, err := li.Execute(ctx, "echo", "test")
	if err != nil {
		t.Fatalf("LocalInstall Execute failed: %v", err)
	}

	if !strings.Contains(string(output), "test") {
		t.Errorf("Expected 'test' in output, got: %s", string(output))
	}

	// Test write
	err = li.Write(ctx, "test.txt", []byte("test content"))
	if err != nil {
		t.Fatalf("LocalInstall Write failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "test.txt"))
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}

	if string(content) != "test content" {
		t.Errorf("Expected 'test content', got: %s", string(content))
	}

	// Test close
	if err := li.Close(ctx); err != nil {
		t.Fatalf("LocalInstall Close failed: %v", err)
	}
}

func TestDockerInstall(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	tmpDir := t.TempDir()
	config := &DockerInstallConfig{
		Image:         "alpine:latest",
		Platform:      "linux/amd64",
		ContainerName: "test-local-install",
		Detach:        true,
	}

	di := NewDockerInstall(tmpDir, config)
	ctx := context.Background()

	// Test start - this will pull the image
	if err := di.Start(ctx); err != nil {
		t.Logf("DockerInstall Start failed (Docker may not be available): %v", err)
		t.Skip("Docker not available")
		return
	}

	if di.containerID == "" {
		t.Error("Container ID should be set after Start")
	}

	// Test execute
	_, err := di.Execute(ctx, "echo", "from-docker")
	if err != nil {
		t.Logf("Execute failed: %v", err)
	}

	// Test close
	if err := di.Close(ctx); err != nil {
		t.Errorf("DockerInstall Close failed: %v", err)
	}
}

func TestDockerMountInstall(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	tmpDir := t.TempDir()
	config := &DockerMountInstallConfig{
		DockerInstallConfig: DockerInstallConfig{
			Image:    "alpine:latest",
			Platform: "linux/amd64",
			Detach:   true,
		},
		Volumes: map[string]string{
			tmpDir: "/workspace",
		},
	}

	dmi := NewDockerMountInstall(tmpDir, config)
	ctx := context.Background()

	if err := dmi.Start(ctx); err != nil {
		t.Logf("DockerMountInstall Start failed (Docker may not be available): %v", err)
		t.Skip("Docker not available")
		return
	}

	// Test write to mounted volume
	err := dmi.Write(ctx, "/workspace/test.txt", []byte("mount test"))
	if err != nil {
		t.Errorf("DockerMountInstall Write failed: %v", err)
	}

	// Verify file was written in host directory
	content, err := os.ReadFile(filepath.Join(tmpDir, "test.txt"))
	if err != nil {
		t.Errorf("Failed to read mounted file: %v", err)
	}

	if string(content) != "mount test" {
		t.Errorf("Expected 'mount test', got: %s", string(content))
	}

	// Clean up
	if err := dmi.Close(ctx); err != nil {
		t.Errorf("DockerMountInstall Close failed: %v", err)
	}
}

func TestAgentBoot_Local(t *testing.T) {
	tmpDir := t.TempDir()

	config := &AgentBootConfig{
		RootPath:      tmpDir,
		InstallConfig: &LocalInstallConfig{},
		Shell:         "bash",
	}

	ab, err := NewAgentBootFromConfig(config)
	if err != nil {
		t.Fatalf("NewAgentBootFromConfig failed: %v", err)
	}

	ctx := context.Background()

	// Test start
	if err := ab.Start(ctx); err != nil {
		t.Fatalf("AgentBoot Start failed: %v", err)
	}

	if !ab.IsStarted() {
		t.Error("Expected IsStarted to be true after Start")
	}

	// Test command execution
	result, err := ab.RunCommand(ctx, "echo 'agent boot test'")
	if err != nil {
		t.Fatalf("RunCommand failed: %v", err)
	}

	if !strings.Contains(result, "agent boot test") {
		t.Errorf("Expected command output, got: %s", result)
	}

	// Test write
	err = ab.Write(ctx, "boot-test.txt", []byte("boot content"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Verify file
	content, err := os.ReadFile(filepath.Join(tmpDir, "boot-test.txt"))
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}

	if string(content) != "boot content" {
		t.Errorf("Expected 'boot content', got: %s", string(content))
	}

	// Test close
	if err := ab.Close(ctx); err != nil {
		t.Fatalf("AgentBoot Close failed: %v", err)
	}
}

func TestAgentBoot_Docker(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	tmpDir := t.TempDir()

	config := &AgentBootConfig{
		RootPath: tmpDir,
		InstallConfig: &DockerInstallConfig{
			Image: "alpine:latest",
		},
		Shell: "sh",
	}

	ab, err := NewAgentBootFromConfig(config)
	if err != nil {
		t.Fatalf("NewAgentBootFromConfig failed: %v", err)
	}

	ctx := context.Background()

	if err := ab.Start(ctx); err != nil {
		t.Logf("AgentBoot Start with Docker failed (may be expected): %v", err)
		// Don't skip if Docker is just not installed
	}

	// Clean up
	_ = ab.Close(ctx)
}

func TestAgentBoot_DoubleStart(t *testing.T) {
	tmpDir := t.TempDir()

	config := &AgentBootConfig{
		RootPath:      tmpDir,
		InstallConfig: &LocalInstallConfig{},
		Shell:         "bash",
	}

	ab, err := NewAgentBootFromConfig(config)
	if err != nil {
		t.Fatalf("NewAgentBootFromConfig failed: %v", err)
	}

	ctx := context.Background()

	// First start
	if err := ab.Start(ctx); err != nil {
		t.Fatalf("First Start failed: %v", err)
	}

	// Second start should be idempotent
	if err := ab.Start(ctx); err != nil {
		t.Errorf("Second Start should not error, got: %v", err)
	}

	_ = ab.Close(ctx)
}

func TestAgentBoot_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()

	config := &AgentBootConfig{
		RootPath:      tmpDir,
		InstallConfig: &LocalInstallConfig{},
		Shell:         "bash",
	}

	ab, err := NewAgentBootFromConfig(config)
	if err != nil {
		t.Fatalf("NewAgentBootFromConfig failed: %v", err)
	}

	ctx := context.Background()

	if err := ab.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Test concurrent access
	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func() {
			_, _ = ab.RunCommand(ctx, "echo test")
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}

	_ = ab.Close(ctx)
}

func TestAgentBoot_Getters(t *testing.T) {
	tmpDir := t.TempDir()

	config := &AgentBootConfig{
		RootPath:      tmpDir,
		InstallConfig: &LocalInstallConfig{},
		Shell:         "zsh",
	}

	ab, err := NewAgentBootFromConfig(config)
	if err != nil {
		t.Fatalf("NewAgentBootFromConfig failed: %v", err)
	}

	if ab.GetRootPath() != tmpDir {
		t.Errorf("Expected rootPath '%s', got '%s'", tmpDir, ab.GetRootPath())
	}

	if ab.GetShell() != "zsh" {
		t.Errorf("Expected shell 'zsh', got '%s'", ab.GetShell())
	}

	if ab.GetTarget() == nil {
		t.Error("Expected target to be set")
	}
}

func TestAgentBoot_NotStarted(t *testing.T) {
	tmpDir := t.TempDir()

	config := &AgentBootConfig{
		RootPath:      tmpDir,
		InstallConfig: &LocalInstallConfig{},
		Shell:         "bash",
	}

	ab, err := NewAgentBootFromConfig(config)
	if err != nil {
		t.Fatalf("NewAgentBootFromConfig failed: %v", err)
	}

	// Don't start

	if ab.IsStarted() {
		t.Error("Expected IsStarted to be false before Start")
	}

	ctx := context.Background()

	// Try to execute without starting
	_, err = ab.RunCommand(ctx, "echo test")
	if err == nil {
		t.Error("Expected error when executing before start")
	}

	// Note: Write works even without start for LocalInstall (writes directly to filesystem)
	_ = ab.Write(ctx, "test.txt", []byte("test"))
}

func TestDefaultConfigs(t *testing.T) {
	config := DefaultAgentBootConfig()

	if config.RootPath != "." {
		t.Errorf("Expected default rootPath '.', got '%s'", config.RootPath)
	}

	if config.Shell != "bash" {
		t.Errorf("Expected default shell 'bash', got '%s'", config.Shell)
	}

	if config.Env == nil {
		t.Error("Expected Env to be initialized")
	}

	dockerConfig := DefaultDockerInstallConfig()
	if dockerConfig.Image != "python:3.11" {
		t.Errorf("Expected default image 'python:3.11', got '%s'", dockerConfig.Image)
	}

	mountConfig := DefaultDockerMountInstallConfig()
	if mountConfig.Image != "python:3.11" {
		t.Errorf("Expected default image 'python:3.11', got '%s'", mountConfig.Image)
	}

	if mountConfig.Volumes == nil {
		t.Error("Expected Volumes to be initialized")
	}
}

func TestAgentBoot_CloseWithoutStart(t *testing.T) {
	tmpDir := t.TempDir()

	config := &AgentBootConfig{
		RootPath:      tmpDir,
		InstallConfig: &LocalInstallConfig{},
		Shell:         "bash",
	}

	ab, err := NewAgentBootFromConfig(config)
	if err != nil {
		t.Fatalf("NewAgentBootFromConfig failed: %v", err)
	}

	ctx := context.Background()

	// Close without starting should not error
	if err := ab.Close(ctx); err != nil {
		t.Errorf("Close without start should not error: %v", err)
	}
}

func TestAgentBoot_TempDirCleanup(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "agentboot-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := &AgentBootConfig{
		RootPath:      tempDir,
		InstallConfig: &LocalInstallConfig{},
		Shell:         "bash",
	}

	ab, err := NewAgentBootFromConfig(config)
	if err != nil {
		t.Fatalf("NewAgentBootFromConfig failed: %v", err)
	}

	ctx := context.Background()

	if err := ab.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Close should cleanup temp directory
	if err := ab.Close(ctx); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Verify temp directory was cleaned up
	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Error("Temp directory was not cleaned up")
	}
}
