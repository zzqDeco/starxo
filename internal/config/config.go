package config

// AppConfig is the root configuration.
type AppConfig struct {
	SSH     SSHConfig     `json:"ssh"`
	Sandbox SandboxConfig `json:"sandbox"`
	// Docker is kept only for one-version JSON compatibility with existing
	// config files. Runtime code must use Sandbox instead.
	Docker *DockerConfig `json:"docker,omitempty"`
	LLM    LLMConfig     `json:"llm"`
	MCP    MCPConfig     `json:"mcp"`
	Agent  AgentConfig   `json:"agent"`
}

type SSHConfig struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	User       string `json:"user"`
	Password   string `json:"password,omitempty"`
	PrivateKey string `json:"privateKey,omitempty"`
}

type DockerConfig struct {
	Image       string  `json:"image"`
	MemoryLimit int64   `json:"memoryLimit"`
	CPULimit    float64 `json:"cpuLimit"`
	WorkDir     string  `json:"workDir"`
	Network     bool    `json:"network"`
}

type SandboxConfig struct {
	Runtime           string   `json:"runtime"`
	RootDir           string   `json:"rootDir"`
	WorkDirName       string   `json:"workDirName"`
	Network           bool     `json:"network"`
	MemoryLimitMB     int64    `json:"memoryLimitMB"`
	CommandTimeoutSec int      `json:"commandTimeoutSec"`
	BootstrapPython   bool     `json:"bootstrapPython"`
	PythonPackages    []string `json:"pythonPackages"`
}

type LLMConfig struct {
	Type    string            `json:"type"`
	BaseURL string            `json:"baseURL"`
	APIKey  string            `json:"apiKey"`
	Model   string            `json:"model"`
	Headers map[string]string `json:"headers,omitempty"`
}

type MCPConfig struct {
	Servers []MCPServerConfig `json:"servers"`
}

type MCPServerConfig struct {
	Name      string            `json:"name"`
	Transport string            `json:"transport"`
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	URL       string            `json:"url,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	Enabled   bool              `json:"enabled"`
}

type AgentConfig struct {
	MaxIterations int `json:"maxIterations"`
}

func DefaultConfig() *AppConfig {
	return &AppConfig{
		SSH: SSHConfig{Port: 22, User: "root"},
		Sandbox: SandboxConfig{
			Runtime:           "auto",
			RootDir:           "~/.starxo/sandboxes",
			WorkDirName:       "workspace",
			Network:           true,
			MemoryLimitMB:     2048,
			CommandTimeoutSec: 120,
			BootstrapPython:   true,
			PythonPackages:    []string{"pandas", "numpy", "matplotlib", "openpyxl"},
		},
		LLM:   LLMConfig{Type: "openai", Model: "gpt-4o"},
		Agent: AgentConfig{MaxIterations: 30},
	}
}

func NormalizeAppConfig(cfg *AppConfig) {
	if cfg == nil {
		return
	}
	defaults := DefaultConfig()

	if cfg.SSH.Port == 0 {
		cfg.SSH.Port = defaults.SSH.Port
	}
	if cfg.SSH.User == "" {
		cfg.SSH.User = defaults.SSH.User
	}

	if cfg.Sandbox.Runtime == "" {
		cfg.Sandbox.Runtime = defaults.Sandbox.Runtime
	}
	if cfg.Sandbox.RootDir == "" {
		cfg.Sandbox.RootDir = defaults.Sandbox.RootDir
	}
	if cfg.Sandbox.WorkDirName == "" {
		cfg.Sandbox.WorkDirName = defaults.Sandbox.WorkDirName
	}
	if cfg.Sandbox.MemoryLimitMB == 0 {
		cfg.Sandbox.MemoryLimitMB = defaults.Sandbox.MemoryLimitMB
	}
	if cfg.Sandbox.CommandTimeoutSec == 0 {
		cfg.Sandbox.CommandTimeoutSec = defaults.Sandbox.CommandTimeoutSec
	}
	if len(cfg.Sandbox.PythonPackages) == 0 {
		cfg.Sandbox.PythonPackages = append([]string(nil), defaults.Sandbox.PythonPackages...)
	}

	if cfg.LLM.Type == "" {
		cfg.LLM.Type = defaults.LLM.Type
	}
	if cfg.LLM.Model == "" {
		cfg.LLM.Model = defaults.LLM.Model
	}
	if cfg.Agent.MaxIterations == 0 {
		cfg.Agent.MaxIterations = defaults.Agent.MaxIterations
	}

	cfg.Docker = nil
}

func MigrateLegacyDockerConfig(cfg *AppConfig) {
	if cfg == nil || cfg.Docker == nil {
		return
	}
	if cfg.Docker.WorkDir != "" {
		cfg.Sandbox.WorkDirName = "workspace"
	}
	if cfg.Docker.MemoryLimit > 0 {
		cfg.Sandbox.MemoryLimitMB = cfg.Docker.MemoryLimit
	}
	cfg.Sandbox.Network = cfg.Docker.Network
}
