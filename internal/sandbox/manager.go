package sandbox

import (
	"context"
	"fmt"
	"sync"

	"starxo/internal/config"
)

// SandboxManager is the top-level orchestrator for the sandbox lifecycle.
// It coordinates SSH, Docker, operator, transfer, and setup subsystems.
//
// Lifecycle: ConnectSSH → EnsureDocker → CreateNewContainer/AttachToContainer → DetachContainer → Disconnect
type SandboxManager struct {
	ssh      *SSHClient
	docker   *RemoteDockerManager
	operator *RemoteOperator
	transfer *FileTransfer
	setup    *EnvironmentSetup
	config   config.AppConfig
	mu       sync.Mutex
}

// NewSandboxManager creates a new SandboxManager from the application config.
func NewSandboxManager(cfg config.AppConfig) *SandboxManager {
	return &SandboxManager{
		config: cfg,
	}
}

// ConnectSSH establishes SSH connection and creates the FileTransfer subsystem.
// Does NOT create any Docker container. Call EnsureDocker + CreateNewContainer separately.
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

	// FileTransfer depends only on SSH, available immediately
	m.transfer = NewFileTransfer(m.ssh)

	onProgress("SSH connected", 100)
	return nil
}

// EnsureDocker creates the Docker manager and ensures Docker is installed and running on the remote host.
// Must be called after ConnectSSH.
func (m *SandboxManager) EnsureDocker(ctx context.Context, onProgress func(string, int)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ssh == nil || !m.ssh.IsConnected() {
		return fmt.Errorf("SSH not connected")
	}

	if onProgress == nil {
		onProgress = func(string, int) {}
	}

	m.docker = NewRemoteDockerManager(m.ssh, m.config.Docker)
	m.setup = NewEnvironmentSetup(m.ssh, m.docker, onProgress)

	if err := m.setup.EnsureDockerAvailable(ctx); err != nil {
		m.docker = nil
		m.setup = nil
		return fmt.Errorf("Docker setup failed: %w", err)
	}

	return nil
}

// CreateNewContainer creates a fresh container: cleanup old → pull image → create → setup → operator.
// Must be called after EnsureDocker. Returns the Docker container ID and name.
func (m *SandboxManager) CreateNewContainer(ctx context.Context, excludeDockerIDs []string, onProgress func(string, int)) (dockerID string, containerName string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ssh == nil || !m.ssh.IsConnected() {
		return "", "", fmt.Errorf("SSH not connected")
	}
	if m.docker == nil {
		return "", "", fmt.Errorf("Docker not initialized, call EnsureDocker first")
	}

	if onProgress == nil {
		onProgress = func(string, int) {}
	}

	// Use a temporary setup with the provided progress callback
	setup := NewEnvironmentSetup(m.ssh, m.docker, onProgress)
	if err := setup.SetupNewContainer(ctx, excludeDockerIDs); err != nil {
		return "", "", fmt.Errorf("container setup failed: %w", err)
	}

	// Create operator for the new container
	m.operator = NewRemoteOperator(m.docker)

	return m.docker.ContainerID(), m.docker.ContainerName(), nil
}

// AttachToContainer connects to an existing container: inspect → start if stopped → health check → operator.
// Must be called after EnsureDocker.
func (m *SandboxManager) AttachToContainer(ctx context.Context, dockerID, containerName string, onProgress func(string, int)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ssh == nil || !m.ssh.IsConnected() {
		return fmt.Errorf("SSH not connected")
	}
	if m.docker == nil {
		return fmt.Errorf("Docker not initialized, call EnsureDocker first")
	}

	if onProgress == nil {
		onProgress = func(string, int) {}
	}

	// Set the target container
	m.docker.SetContainerID(dockerID, containerName)

	// Use InitializeExisting which inspects, starts if stopped, and health-checks
	setup := NewEnvironmentSetup(m.ssh, m.docker, onProgress)
	if err := setup.InitializeExisting(ctx, dockerID); err != nil {
		m.docker.ClearContainer()
		return fmt.Errorf("attach to container failed: %w", err)
	}

	// Create operator for the attached container
	m.operator = NewRemoteOperator(m.docker)

	return nil
}

// DetachContainer clears the active container operator and docker reference
// without closing SSH. The container keeps running on the remote host.
func (m *SandboxManager) DetachContainer() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.operator = nil
	if m.docker != nil {
		m.docker.ClearContainer()
	}
}

// SSHConnected returns true if SSH is connected (regardless of container state).
func (m *SandboxManager) SSHConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ssh != nil && m.ssh.IsConnected()
}

// HasActiveContainer returns true if a container is attached and has an operator.
func (m *SandboxManager) HasActiveContainer() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.operator != nil && m.docker != nil && m.docker.IsRunning()
}

// --- Legacy methods kept for backward compatibility and existing callers ---

// ConnectWithExclusions establishes the full sandbox environment with a new container.
// This is a convenience method that calls ConnectSSH + EnsureDocker + CreateNewContainer.
func (m *SandboxManager) ConnectWithExclusions(ctx context.Context, onProgress func(string, int), excludeDockerIDs []string) error {
	if onProgress == nil {
		onProgress = func(string, int) {}
	}

	// Step 1: SSH
	onProgress("Connecting via SSH", 0)
	if err := m.ConnectSSH(ctx, func(step string, pct int) {
		onProgress(step, pct*10/100) // 0-10%
	}); err != nil {
		return err
	}

	// Step 2: Docker
	onProgress("Initializing Docker", 10)
	if err := m.EnsureDocker(ctx, func(step string, pct int) {
		onProgress(step, 10+pct*15/100) // 10-25%
	}); err != nil {
		_ = m.ssh.Close()
		m.mu.Lock()
		m.ssh = nil
		m.transfer = nil
		m.mu.Unlock()
		return err
	}

	// Step 3: Container
	if _, _, err := m.CreateNewContainer(ctx, excludeDockerIDs, func(step string, pct int) {
		onProgress(step, 25+pct*75/100) // 25-100%
	}); err != nil {
		_ = m.ssh.Close()
		m.mu.Lock()
		m.ssh = nil
		m.docker = nil
		m.setup = nil
		m.transfer = nil
		m.mu.Unlock()
		return fmt.Errorf("environment setup failed: %w", err)
	}

	return nil
}

// Connect establishes the full sandbox environment with a new container (no exclusions).
func (m *SandboxManager) Connect(ctx context.Context, onProgress func(string, int)) error {
	return m.ConnectWithExclusions(ctx, onProgress, nil)
}

// Reconnect establishes connection to an existing container.
// This is a convenience method that calls ConnectSSH + EnsureDocker + AttachToContainer.
func (m *SandboxManager) Reconnect(ctx context.Context, dockerID, containerName string, onProgress func(string, int)) error {
	if onProgress == nil {
		onProgress = func(string, int) {}
	}

	// Step 1: SSH
	if err := m.ConnectSSH(ctx, func(step string, pct int) {
		onProgress(step, pct*10/100)
	}); err != nil {
		return err
	}

	// Step 2: Docker
	if err := m.EnsureDocker(ctx, func(step string, pct int) {
		onProgress(step, 10+pct*15/100)
	}); err != nil {
		_ = m.ssh.Close()
		m.mu.Lock()
		m.ssh = nil
		m.transfer = nil
		m.mu.Unlock()
		return err
	}

	// Step 3: Attach
	if err := m.AttachToContainer(ctx, dockerID, containerName, func(step string, pct int) {
		onProgress(step, 25+pct*75/100)
	}); err != nil {
		_ = m.ssh.Close()
		m.mu.Lock()
		m.ssh = nil
		m.docker = nil
		m.setup = nil
		m.transfer = nil
		m.mu.Unlock()
		return fmt.Errorf("reconnect setup failed: %w", err)
	}

	return nil
}

// Disconnect closes the SSH connection but keeps the container alive.
// The container can be reconnected to later.
func (m *SandboxManager) Disconnect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var firstErr error

	// Close SSH connection (container stays running on remote host)
	if m.ssh != nil {
		if err := m.ssh.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to close SSH connection: %w", err)
		}
	}

	// Clear local references
	m.operator = nil
	m.transfer = nil
	m.docker = nil
	m.setup = nil
	m.ssh = nil

	return firstErr
}

// DisconnectAndDestroy stops and removes the container, then closes SSH.
func (m *SandboxManager) DisconnectAndDestroy(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var firstErr error

	// Stop and remove the container
	if m.docker != nil && m.docker.IsRunning() {
		if err := m.docker.StopAndRemove(ctx); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to stop container: %w", err)
		}
	}

	// Close SSH connection
	if m.ssh != nil {
		if err := m.ssh.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to close SSH connection: %w", err)
		}
	}

	// Clear all references
	m.operator = nil
	m.transfer = nil
	m.docker = nil
	m.setup = nil
	m.ssh = nil

	return firstErr
}

// StopContainer stops the container without removing it or closing SSH.
func (m *SandboxManager) StopContainer(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.docker == nil {
		return fmt.Errorf("no docker manager")
	}

	return m.docker.StopContainer(ctx)
}

// IsConnected returns true if SSH is connected AND an active container is attached.
func (m *SandboxManager) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ssh != nil && m.ssh.IsConnected() && m.docker != nil && m.docker.IsRunning()
}

// Operator returns the commandline.Operator implementation for use with eino tools.
func (m *SandboxManager) Operator() *RemoteOperator {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.operator
}

// Transfer returns the FileTransfer subsystem.
func (m *SandboxManager) Transfer() *FileTransfer {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.transfer
}

// Docker returns the RemoteDockerManager subsystem.
func (m *SandboxManager) Docker() *RemoteDockerManager {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.docker
}

// SSH returns the SSHClient subsystem.
func (m *SandboxManager) SSH() *SSHClient {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ssh
}
