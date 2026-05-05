package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	require.NotNil(t, cfg)

	// SSH defaults
	assert.Equal(t, 22, cfg.SSH.Port)
	assert.Equal(t, "root", cfg.SSH.User)
	assert.Empty(t, cfg.SSH.Host)

	// Sandbox defaults
	require.Nil(t, cfg.Docker)
	assert.Equal(t, "auto", cfg.Sandbox.Runtime)
	assert.Equal(t, "~/.starxo/sandboxes", cfg.Sandbox.RootDir)
	assert.Equal(t, "workspace", cfg.Sandbox.WorkDirName)
	assert.Equal(t, int64(2048), cfg.Sandbox.MemoryLimitMB)
	assert.Equal(t, 120, cfg.Sandbox.CommandTimeoutSec)
	assert.True(t, cfg.Sandbox.Network)
	assert.True(t, cfg.Sandbox.BootstrapPython)
	assert.Equal(t, []string{"pandas", "numpy", "matplotlib", "openpyxl"}, cfg.Sandbox.PythonPackages)

	// LLM defaults
	assert.Equal(t, "openai", cfg.LLM.Type)
	assert.Equal(t, "gpt-4o", cfg.LLM.Model)
	assert.Empty(t, cfg.LLM.BaseURL)
	assert.Empty(t, cfg.LLM.APIKey)

	// Agent defaults
	assert.Equal(t, 30, cfg.Agent.MaxIterations)

	// MCP defaults (empty)
	assert.Nil(t, cfg.MCP.Servers)
}

func TestConfigJSONRoundTrip(t *testing.T) {
	original := DefaultConfig()
	original.SSH.Host = "192.168.1.100"
	original.SSH.Password = "secret"
	original.LLM.APIKey = "sk-test"
	original.LLM.BaseURL = "https://api.example.com"
	original.LLM.Headers = map[string]string{"X-Custom": "value"}
	original.MCP.Servers = []MCPServerConfig{
		{Name: "test-server", Transport: "stdio", Command: "node", Args: []string{"index.js"}, Enabled: true},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var restored AppConfig
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	assert.Equal(t, original.SSH, restored.SSH)
	assert.Equal(t, original.Sandbox, restored.Sandbox)
	assert.Nil(t, restored.Docker)
	assert.Equal(t, original.LLM, restored.LLM)
	assert.Equal(t, original.Agent, restored.Agent)
	assert.Equal(t, original.MCP.Servers, restored.MCP.Servers)
}

func TestMigrateLegacyDockerConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Sandbox = SandboxConfig{}
	cfg.Docker = &DockerConfig{
		Image:       "python:3.11-slim",
		MemoryLimit: 1024,
		CPULimit:    1.0,
		WorkDir:     "/workspace",
		Network:     false,
	}

	MigrateLegacyDockerConfig(cfg)
	NormalizeAppConfig(cfg)

	assert.Nil(t, cfg.Docker)
	assert.Equal(t, int64(1024), cfg.Sandbox.MemoryLimitMB)
	assert.False(t, cfg.Sandbox.Network)
	assert.Equal(t, "workspace", cfg.Sandbox.WorkDirName)
}

func TestConfigJSONOmitsEmptyFields(t *testing.T) {
	cfg := DefaultConfig()
	data, err := json.Marshal(cfg)
	require.NoError(t, err)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &raw))

	// SSH password and privateKey should be omitted when empty
	var sshRaw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw["ssh"], &sshRaw))
	assert.NotContains(t, sshRaw, "password")
	assert.NotContains(t, sshRaw, "privateKey")

	// LLM headers should be omitted when nil
	var llmRaw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw["llm"], &llmRaw))
	assert.NotContains(t, llmRaw, "headers")
}

func TestDefaultConfigReturnsNewInstance(t *testing.T) {
	a := DefaultConfig()
	b := DefaultConfig()
	assert.NotSame(t, a, b, "DefaultConfig should return a new pointer each time")

	a.SSH.Port = 2222
	assert.Equal(t, 22, b.SSH.Port, "modifying one instance should not affect another")
}
