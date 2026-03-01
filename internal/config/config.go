package config

// AppConfig is the root configuration.
type AppConfig struct {
	SSH    SSHConfig    `json:"ssh"`
	Docker DockerConfig `json:"docker"`
	LLM    LLMConfig    `json:"llm"`
	MCP    MCPConfig    `json:"mcp"`
	Agent  AgentConfig  `json:"agent"`
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
		Docker: DockerConfig{
			Image:       "python:3.11-slim",
			MemoryLimit: 2048,
			CPULimit:    1.0,
			WorkDir:     "/workspace",
			Network:     true,
		},
		LLM:   LLMConfig{Type: "openai", Model: "gpt-4o"},
		Agent: AgentConfig{MaxIterations: 30},
	}
}
