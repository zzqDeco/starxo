package sandbox

import (
	"context"
	"fmt"
	"sync"

	"starxo/internal/config"
)

// SandboxManager coordinates SSH, lightweight runtime execution, file transfer,
// and the commandline operator used by agent tools.
type SandboxManager struct {
	ssh      *SSHClient
	runtime  *RemoteRuntimeManager
	operator *RemoteOperator
	transfer *FileTransfer
	config   config.AppConfig
	mu       sync.Mutex
}

func NewSandboxManager(cfg config.AppConfig) *SandboxManager {
	config.MigrateLegacyDockerConfig(&cfg)
	config.NormalizeAppConfig(&cfg)
	return &SandboxManager{config: cfg}
}

func (m *SandboxManager) ConnectSSH(ctx context.Context, onProgress func(string, int)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if onProgress == nil {
		onProgress = func(string, int) {}
	}

	onProgress("Connecting via SSH", 0)
	m.ssh = NewSSHClient(m.config.SSH)
	if err := m.ssh.Connect(ctx); err != nil {
		return fmt.Errorf("SSH connection failed: %w", err)
	}
	m.transfer = NewFileTransfer(m.ssh)
	onProgress("SSH connected", 100)
	return nil
}

func (m *SandboxManager) EnsureRuntime(ctx context.Context, onProgress func(string, int)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ssh == nil || !m.ssh.IsConnected() {
		return fmt.Errorf("SSH not connected")
	}
	if onProgress == nil {
		onProgress = func(string, int) {}
	}

	onProgress("Checking sandbox runtime", 0)
	rt := NewRemoteRuntimeManager(m.ssh, m.config.Sandbox)
	result, err := rt.Detect(ctx)
	if err != nil {
		return fmt.Errorf("sandbox runtime check failed: %w", err)
	}
	if !result.Available {
		return fmt.Errorf("sandbox runtime unavailable: %s", result.Message)
	}
	m.runtime = rt
	onProgress("Sandbox runtime ready", 100)
	return nil
}

// EnsureDocker is kept as a compatibility shim for callers not yet renamed.
func (m *SandboxManager) EnsureDocker(ctx context.Context, onProgress func(string, int)) error {
	return m.EnsureRuntime(ctx, onProgress)
}

func (m *SandboxManager) CheckRuntime(ctx context.Context) (RuntimeCheckResult, error) {
	m.mu.Lock()
	if m.ssh == nil || !m.ssh.IsConnected() {
		m.mu.Unlock()
		return RuntimeCheckResult{}, fmt.Errorf("SSH not connected")
	}
	rt := m.runtime
	if rt == nil {
		rt = NewRemoteRuntimeManager(m.ssh, m.config.Sandbox)
	}
	m.mu.Unlock()
	return rt.Detect(ctx)
}

func (m *SandboxManager) InstallRuntime(ctx context.Context) (RuntimeInstallResult, error) {
	m.mu.Lock()
	if m.ssh == nil || !m.ssh.IsConnected() {
		m.mu.Unlock()
		return RuntimeInstallResult{}, fmt.Errorf("SSH not connected")
	}
	rt := m.runtime
	if rt == nil {
		rt = NewRemoteRuntimeManager(m.ssh, m.config.Sandbox)
		m.runtime = rt
	}
	m.mu.Unlock()
	return rt.Install(ctx)
}

func (m *SandboxManager) CreateNewSandbox(ctx context.Context, excludeIDs []string, onProgress func(string, int)) (*SandboxInstance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ssh == nil || !m.ssh.IsConnected() {
		return nil, fmt.Errorf("SSH not connected")
	}
	if m.runtime == nil {
		return nil, fmt.Errorf("sandbox runtime not initialized, call EnsureRuntime first")
	}
	if onProgress == nil {
		onProgress = func(string, int) {}
	}

	onProgress("Creating sandbox workspace", 10)
	inst, err := m.runtime.CreateSandbox(ctx, excludeIDs)
	if err != nil {
		return nil, fmt.Errorf("sandbox setup failed: %w", err)
	}
	m.operator = NewRemoteOperator(m.runtime)
	onProgress("Sandbox ready", 100)
	return inst, nil
}

func (m *SandboxManager) CreateNewContainer(ctx context.Context, excludeIDs []string, onProgress func(string, int)) (string, string, error) {
	inst, err := m.CreateNewSandbox(ctx, excludeIDs, onProgress)
	if err != nil {
		return "", "", err
	}
	return inst.ID, inst.Name, nil
}

func (m *SandboxManager) AttachToSandbox(ctx context.Context, id, name, workspacePath string, onProgress func(string, int)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ssh == nil || !m.ssh.IsConnected() {
		return fmt.Errorf("SSH not connected")
	}
	if m.runtime == nil {
		return fmt.Errorf("sandbox runtime not initialized, call EnsureRuntime first")
	}
	if onProgress == nil {
		onProgress = func(string, int) {}
	}

	onProgress("Checking sandbox workspace", 20)
	if _, err := m.runtime.AttachSandbox(ctx, id, name, workspacePath); err != nil {
		m.runtime.Deactivate()
		return fmt.Errorf("attach to sandbox failed: %w", err)
	}
	m.operator = NewRemoteOperator(m.runtime)
	onProgress("Sandbox activated", 100)
	return nil
}

func (m *SandboxManager) AttachToContainer(ctx context.Context, id, name string, onProgress func(string, int)) error {
	return m.AttachToSandbox(ctx, id, name, "", onProgress)
}

func (m *SandboxManager) DetachContainer() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.operator = nil
	if m.runtime != nil {
		m.runtime.Deactivate()
	}
}

func (m *SandboxManager) SSHConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ssh != nil && m.ssh.IsConnected()
}

func (m *SandboxManager) HasActiveContainer() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.operator != nil && m.runtime != nil && m.runtime.IsActive()
}

func (m *SandboxManager) ConnectWithExclusions(ctx context.Context, onProgress func(string, int), excludeIDs []string) error {
	if onProgress == nil {
		onProgress = func(string, int) {}
	}
	if err := m.ConnectSSH(ctx, func(step string, pct int) { onProgress(step, pct*10/100) }); err != nil {
		return err
	}
	if err := m.EnsureRuntime(ctx, func(step string, pct int) { onProgress(step, 10+pct*30/100) }); err != nil {
		_ = m.Disconnect(ctx)
		return err
	}
	if _, err := m.CreateNewSandbox(ctx, excludeIDs, func(step string, pct int) { onProgress(step, 40+pct*60/100) }); err != nil {
		_ = m.Disconnect(ctx)
		return err
	}
	return nil
}

func (m *SandboxManager) Connect(ctx context.Context, onProgress func(string, int)) error {
	return m.ConnectWithExclusions(ctx, onProgress, nil)
}

func (m *SandboxManager) Reconnect(ctx context.Context, id, name string, onProgress func(string, int)) error {
	if onProgress == nil {
		onProgress = func(string, int) {}
	}
	if err := m.ConnectSSH(ctx, func(step string, pct int) { onProgress(step, pct*10/100) }); err != nil {
		return err
	}
	if err := m.EnsureRuntime(ctx, func(step string, pct int) { onProgress(step, 10+pct*30/100) }); err != nil {
		_ = m.Disconnect(ctx)
		return err
	}
	if err := m.AttachToSandbox(ctx, id, name, "", func(step string, pct int) { onProgress(step, 40+pct*60/100) }); err != nil {
		_ = m.Disconnect(ctx)
		return err
	}
	return nil
}

func (m *SandboxManager) Disconnect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var firstErr error
	if m.ssh != nil {
		if err := m.ssh.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to close SSH connection: %w", err)
		}
	}
	m.operator = nil
	m.transfer = nil
	m.runtime = nil
	m.ssh = nil
	return firstErr
}

func (m *SandboxManager) DisconnectAndDestroy(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var firstErr error
	if m.runtime != nil && m.runtime.IsActive() {
		if err := m.runtime.Destroy(ctx); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to destroy sandbox: %w", err)
		}
	}
	if m.ssh != nil {
		if err := m.ssh.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to close SSH connection: %w", err)
		}
	}
	m.operator = nil
	m.transfer = nil
	m.runtime = nil
	m.ssh = nil
	return firstErr
}

func (m *SandboxManager) StopContainer(ctx context.Context) error {
	m.DetachContainer()
	return nil
}

func (m *SandboxManager) DestroySandbox(ctx context.Context, id, workspacePath string) error {
	m.mu.Lock()
	rt := m.runtime
	m.mu.Unlock()
	if rt == nil {
		return fmt.Errorf("sandbox runtime not initialized")
	}
	return rt.DestroySandbox(ctx, id, workspacePath)
}

func (m *SandboxManager) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ssh != nil && m.ssh.IsConnected() && m.runtime != nil && m.runtime.IsActive()
}

func (m *SandboxManager) Operator() *RemoteOperator {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.operator
}

func (m *SandboxManager) Transfer() *FileTransfer {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.transfer
}

func (m *SandboxManager) Runtime() *RemoteRuntimeManager {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.runtime
}

// Docker is a compatibility accessor for older service code. It returns the
// lightweight runtime manager, not a Docker-backed manager.
func (m *SandboxManager) Docker() *RemoteRuntimeManager {
	return m.Runtime()
}

func (m *SandboxManager) SSH() *SSHClient {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ssh
}

func (m *SandboxManager) SSHHostPort() (string, int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.config.SSH.Host, m.config.SSH.Port
}

func (m *SandboxManager) WorkspacePath() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.runtime == nil {
		return ""
	}
	return m.runtime.WorkspacePath()
}
