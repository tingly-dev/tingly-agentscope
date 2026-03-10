package boot

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// AbstractInstall is the interface for install backends
type AbstractInstall interface {
	// Start starts the backend execution environment
	Start(ctx context.Context) error

	// Execute runs a command in the backend
	Execute(ctx context.Context, command string, args ...string) ([]byte, error)

	// Close cleans up the backend
	Close(ctx context.Context) error

	// Write writes content to a file in the backend
	Write(ctx context.Context, path string, content []byte) error
}

// LocalInstall runs commands directly on the host system
type LocalInstall struct {
	rootPath string
	mu       sync.RWMutex
}

// NewLocalInstall creates a new local install backend
func NewLocalInstall(rootPath string) *LocalInstall {
	return &LocalInstall{
		rootPath: rootPath,
	}
}

// Start starts the local environment
func (li *LocalInstall) Start(ctx context.Context) error {
	li.mu.Lock()
	defer li.mu.Unlock()

	// Create root directory if it doesn't exist
	if err := os.MkdirAll(li.rootPath, 0755); err != nil {
		return fmt.Errorf("failed to create root path: %w", err)
	}

	return nil
}

// Execute runs a command locally
func (li *LocalInstall) Execute(ctx context.Context, command string, args ...string) ([]byte, error) {
	li.mu.RLock()
	defer li.mu.RUnlock()

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = li.rootPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("command failed: %w", err)
	}

	return output, nil
}

// Close cleans up local environment
func (li *LocalInstall) Close(ctx context.Context) error {
	// Local install doesn't need cleanup
	return nil
}

// Write writes content to a file
func (li *LocalInstall) Write(ctx context.Context, path string, content []byte) error {
	fullPath := filepath.Join(li.rootPath, path)

	// Create directory if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// DockerInstall runs commands in a Docker container
type DockerInstall struct {
	rootPath    string
	config      *DockerInstallConfig
	containerID string
	mu          sync.RWMutex
}

// NewDockerInstall creates a new docker install backend
func NewDockerInstall(rootPath string, config *DockerInstallConfig) *DockerInstall {
	if config == nil {
		config = &DockerInstallConfig{}
	}
	return &DockerInstall{
		rootPath: rootPath,
		config:   config,
	}
}

// Start starts the docker container
func (di *DockerInstall) Start(ctx context.Context) error {
	di.mu.Lock()
	defer di.mu.Unlock()

	image := di.config.Image
	if image == "" {
		image = "python:3.11"
	}

	// Pull docker image
	pullCmd := exec.CommandContext(ctx, "docker", "pull", image)
	pullCmd.Stdout = os.Stdout
	pullCmd.Stderr = os.Stderr
	if err := pullCmd.Run(); err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}

	// Create container
	args := []string{"create", image}

	if di.config.ContainerName != "" {
		args = append(args, "--name", di.config.ContainerName)
	}

	args = append(args, "--tty", "--interactive")

	// Add environment variables
	for k, v := range di.config.Envs {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	args = append(args, "--detach")

	// Run container
	createCmd := exec.CommandContext(ctx, "docker", args...)
	output, err := createCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create container: %w\nOutput: %s", err, string(output))
	}

	di.containerID = string(output)[:12] // Get first 12 chars (container ID)

	return nil
}

// Execute runs a command in the docker container
func (di *DockerInstall) Execute(ctx context.Context, command string, args ...string) ([]byte, error) {
	di.mu.RLock()
	containerID := di.containerID
	di.mu.RUnlock()

	if containerID == "" {
		return nil, fmt.Errorf("container not started")
	}

	execArgs := append([]string{"exec", containerID, command}, args...)
	cmd := exec.CommandContext(ctx, "docker", execArgs...)

	return cmd.CombinedOutput()
}

// Close stops and removes the docker container
func (di *DockerInstall) Close(ctx context.Context) error {
	di.mu.Lock()
	defer di.mu.Unlock()

	if di.containerID == "" {
		return nil
	}

	containerID := di.containerID
	di.containerID = ""

	// Kill container
	killCmd := exec.CommandContext(ctx, "docker", "kill", containerID)
	_ = killCmd.Run() // Ignore errors if already stopped

	// Remove container
	rmCmd := exec.CommandContext(ctx, "docker", "rm", "-f", containerID)
	_ = rmCmd.Run() // Ignore errors if already removed

	return nil
}

// Write writes content to a file inside the docker container
func (di *DockerInstall) Write(ctx context.Context, path string, content []byte) error {
	di.mu.RLock()
	containerID := di.containerID
	di.mu.RUnlock()

	if containerID == "" {
		return fmt.Errorf("container not started")
	}

	// For docker, we use docker cp to write files
	// Create a temporary file locally first
	tmpFile := filepath.Join(os.TempDir(), types.GenerateID()+filepath.Base(path))
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile)

	// Copy file into container
	cpCmd := exec.CommandContext(ctx, "docker", "cp", tmpFile, containerID+":"+path)
	if err := cpCmd.Run(); err != nil {
		return fmt.Errorf("failed to copy file to container: %w", err)
	}

	return nil
}

// DockerMountInstall runs commands in a Docker container with volume mounts
type DockerMountInstall struct {
	rootPath    string
	config      *DockerMountInstallConfig
	containerID string
	mu          sync.RWMutex
}

// NewDockerMountInstall creates a new docker mount install backend
func NewDockerMountInstall(rootPath string, config *DockerMountInstallConfig) *DockerMountInstall {
	if config == nil {
		config = &DockerMountInstallConfig{}
	}
	return &DockerMountInstall{
		rootPath: rootPath,
		config:   config,
	}
}

// Start starts the docker container with volume mounts
func (dmi *DockerMountInstall) Start(ctx context.Context) error {
	dmi.mu.Lock()
	defer dmi.mu.Unlock()

	image := dmi.config.Image
	if image == "" {
		image = "python:3.11"
	}

	// Pull docker image
	pullCmd := exec.CommandContext(ctx, "docker", "pull", image)
	pullCmd.Stdout = os.Stdout
	pullCmd.Stderr = os.Stderr
	if err := pullCmd.Run(); err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}

	// Create container with volume mounts
	args := []string{"create", image}

	if dmi.config.ContainerName != "" {
		args = append(args, "--name", dmi.config.ContainerName)
	}

	args = append(args, "--tty", "--interactive")

	// Add environment variables
	for k, v := range dmi.config.Envs {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	// Add volume mounts
	for hostPath, containerPath := range dmi.config.Volumes {
		args = append(args, "-v", fmt.Sprintf("%s:%s:rw", hostPath, containerPath))
	}

	args = append(args, "--detach")

	// Run container
	createCmd := exec.CommandContext(ctx, "docker", args...)
	output, err := createCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create container: %w\nOutput: %s", err, string(output))
	}

	dmi.containerID = string(output)[:12]

	return nil
}

// Execute runs a command in the docker container
func (dmi *DockerMountInstall) Execute(ctx context.Context, command string, args ...string) ([]byte, error) {
	dmi.mu.RLock()
	containerID := dmi.containerID
	dmi.mu.RUnlock()

	if containerID == "" {
		return nil, fmt.Errorf("container not started")
	}

	execArgs := append([]string{"exec", containerID, command}, args...)
	cmd := exec.CommandContext(ctx, "docker", execArgs...)

	return cmd.CombinedOutput()
}

// Close stops and removes the docker container
func (dmi *DockerMountInstall) Close(ctx context.Context) error {
	dmi.mu.Lock()
	defer dmi.mu.Unlock()

	if dmi.containerID == "" {
		return nil
	}

	containerID := dmi.containerID
	dmi.containerID = ""

	// Kill container
	killCmd := exec.CommandContext(ctx, "docker", "kill", containerID)
	_ = killCmd.Run()

	// Remove container
	rmCmd := exec.CommandContext(ctx, "docker", "rm", "-f", containerID)
	_ = rmCmd.Run()

	return nil
}

// Write writes content to a file in the mounted volume
func (dmi *DockerMountInstall) Write(ctx context.Context, path string, content []byte) error {
	// For mount install, write directly to the mounted path
	fullPath := filepath.Join(dmi.rootPath, path)

	// Create directory if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// StreamOutput streams container output to stdout
func StreamOutput(ctx context.Context, containerID string) error {
	cmd := exec.CommandContext(ctx, "docker", "logs", "-f", containerID)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	go func() {
		io.Copy(os.Stdout, stdout)
		cmd.Wait()
	}()

	return nil
}
