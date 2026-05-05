package service

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/schema"

	"starxo/internal/config"
	"starxo/internal/llm"
	"starxo/internal/sandbox"
)

// SettingsService manages application settings for the frontend.
type SettingsService struct {
	ctx            context.Context
	store          *config.Store
	onSettingsSave func() // called after settings are saved to invalidate cached state
}

// NewSettingsService creates a new SettingsService.
func NewSettingsService(store *config.Store) *SettingsService {
	return &SettingsService{
		store: store,
	}
}

// SetOnSettingsSave registers a callback that fires after settings are saved.
func (s *SettingsService) SetOnSettingsSave(fn func()) {
	s.onSettingsSave = fn
}

// SetContext stores the Wails application context. Called from app.go startup.
func (s *SettingsService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// GetSettings returns the current application configuration.
func (s *SettingsService) GetSettings() *config.AppConfig {
	return s.store.Get()
}

// SaveSettings saves the provided configuration.
func (s *SettingsService) SaveSettings(cfg config.AppConfig) error {
	if err := s.store.Update(func(current *config.AppConfig) {
		*current = cfg
	}); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}
	// Invalidate cached runner so next message uses new settings (headers, model, etc.)
	if s.onSettingsSave != nil {
		s.onSettingsSave()
	}
	return nil
}

// TestSSHConnection tests whether an SSH connection can be established
// with the provided SSH configuration.
func (s *SettingsService) TestSSHConnection(sshCfg config.SSHConfig) error {
	client := sandbox.NewSSHClient(sshCfg)
	if err := client.Connect(s.ctx); err != nil {
		return fmt.Errorf("SSH connection test failed: %w", err)
	}
	_ = client.Close()
	return nil
}

func (s *SettingsService) CheckSandboxRuntime(cfg config.AppConfig) (sandbox.RuntimeCheckResult, error) {
	config.MigrateLegacyDockerConfig(&cfg)
	config.NormalizeAppConfig(&cfg)
	client := sandbox.NewSSHClient(cfg.SSH)
	if err := client.Connect(s.ctx); err != nil {
		return sandbox.RuntimeCheckResult{}, fmt.Errorf("SSH connection failed: %w", err)
	}
	defer client.Close()

	runtime := sandbox.NewRemoteRuntimeManager(client, cfg.Sandbox)
	result, err := runtime.Detect(s.ctx)
	if err != nil {
		return sandbox.RuntimeCheckResult{}, fmt.Errorf("sandbox runtime check failed: %w", err)
	}
	return result, nil
}

func (s *SettingsService) InstallSandboxRuntime(cfg config.AppConfig) (sandbox.RuntimeInstallResult, error) {
	config.MigrateLegacyDockerConfig(&cfg)
	config.NormalizeAppConfig(&cfg)
	client := sandbox.NewSSHClient(cfg.SSH)
	if err := client.Connect(s.ctx); err != nil {
		return sandbox.RuntimeInstallResult{}, fmt.Errorf("SSH connection failed: %w", err)
	}
	defer client.Close()

	runtime := sandbox.NewRemoteRuntimeManager(client, cfg.Sandbox)
	result, err := runtime.Install(s.ctx)
	if err != nil {
		return result, fmt.Errorf("sandbox runtime install failed: %w", err)
	}
	return result, nil
}

// TestLLMConnection tests whether an LLM connection can be established
// with the provided LLM configuration. It sends a minimal request to verify
// the API endpoint is reachable and credentials are valid.
func (s *SettingsService) TestLLMConnection(llmCfg config.LLMConfig) error {
	mdl, err := llm.NewChatModel(s.ctx, llmCfg)
	if err != nil {
		return fmt.Errorf("LLM connection test failed: %w", err)
	}

	// Send a minimal request to verify the API is reachable
	ctx, cancel := context.WithTimeout(s.ctx, 15*time.Second)
	defer cancel()

	testMsg := []*schema.Message{
		{Role: schema.User, Content: "hi"},
	}
	_, err = mdl.Generate(ctx, testMsg)
	if err != nil {
		return fmt.Errorf("LLM API test failed: %w", err)
	}
	return nil
}
