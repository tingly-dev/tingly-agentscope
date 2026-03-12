package boot

// EnvVar represents an environment variable
type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// AbstractInstallConfig is the base configuration for install backends
type AbstractInstallConfig interface {
	// isInstallConfig marks this as an install config
	isInstallConfig()
}

// LocalInstallConfig is configuration for local execution backend
type LocalInstallConfig struct{}

func (LocalInstallConfig) isInstallConfig() {}

// DockerInstallConfig is configuration for docker backend
type DockerInstallConfig struct {
	Command       string            `json:"command"`
	Image         string            `json:"image"`
	ContainerName string            `json:"container_name,omitempty"`
	Platform      string            `json:"platform"`
	Detach        bool              `json:"detach"`
	Envs          map[string]string `json:"envs,omitempty"`
}

func (DockerInstallConfig) isInstallConfig() {}

// DockerMountInstallConfig is configuration for docker mount backend
type DockerMountInstallConfig struct {
	DockerInstallConfig
	Volumes map[string]string `json:"volumes,omitempty"`
}

func (DockerMountInstallConfig) isInstallConfig() {}

// DefaultDockerInstallConfig returns default docker install config
func DefaultDockerInstallConfig() *DockerInstallConfig {
	return &DockerInstallConfig{
		Image:    "python:3.11",
		Platform: "linux/amd64",
		Detach:   true,
		Envs:     make(map[string]string),
	}
}

// DefaultDockerMountInstallConfig returns default docker mount install config
func DefaultDockerMountInstallConfig() *DockerMountInstallConfig {
	return &DockerMountInstallConfig{
		DockerInstallConfig: DockerInstallConfig{
			Image:    "python:3.11",
			Platform: "linux/amd64",
			Detach:   true,
			Envs:     make(map[string]string),
		},
		Volumes: make(map[string]string),
	}
}

// AgentBootConfig holds the configuration for AgentBoot
type AgentBootConfig struct {
	RootPath      string                `json:"root_path"`
	InstallConfig AbstractInstallConfig `json:"install_config"`
	Env           []EnvVar              `json:"env,omitempty"`
	Shell         string                `json:"shell"`
}

// DefaultAgentBootConfig returns default agent boot config
func DefaultAgentBootConfig() *AgentBootConfig {
	return &AgentBootConfig{
		RootPath:      ".",
		InstallConfig: &LocalInstallConfig{},
		Env:           []EnvVar{},
		Shell:         "bash",
	}
}
