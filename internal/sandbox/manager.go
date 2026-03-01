package sandbox

import (
	"context"
	"fmt"
	"sync"

	"starxo/internal/config"
)

// SandboxManager is the top-level orchestrator for the sandbox lifecycle.
// It coordinates SSH, Docker, operator, transfer, and setup subsystems.
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

// Connect establishes the full sandbox environment with a new container:
// 1. Creates SSH client and connects
// 2. Creates Docker manager
// 3. Runs environment setup (Docker install, image pull, container creation, package install)
// 4. Creates operator and transfer subsystems
func (m *SandboxManager) Connect(ctx context.Context, onProgress func(string, int)) error {
	return m.ConnectWithExclusions(ctx, onProgress, nil)
}

// ConnectWithExclusions is like Connect but accepts a list of Docker IDs to exclude from cleanup.
func (m *SandboxManager) ConnectWithExclusions(ctx context.Context, onProgress func(string, int), excludeDockerIDs []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if onProgress == nil {
		onProgress = func(string, int) {}
	}

	// Step 1: Create SSH client and connect
	onProgress("Connecting via SSH", 0)
	m.ssh = NewSSHClient(m.config.SSH)
	if err := m.ssh.Connect(ctx); err != nil {
		return fmt.Errorf("SSH connection failed: %w", err)
	}

	// Step 2: Create Docker manager
	onProgress("Initializing Docker manager", 10)
	m.docker = NewRemoteDockerManager(m.ssh, m.config.Docker)

	// Step 3: Run environment setup (fresh container)
	m.setup = NewEnvironmentSetup(m.ssh, m.docker, onProgress)
	if err := m.setup.InitializeFresh(ctx, excludeDockerIDs); err != nil {
		_ = m.ssh.Close()
		m.ssh = nil
		m.docker = nil
		m.setup = nil
		return fmt.Errorf("environment setup failed: %w", err)
	}

	// Step 4: Create operator and transfer subsystems
	m.operator = NewRemoteOperator(m.docker)
	m.transfer = NewFileTransfer(m.ssh)

	return nil
}

// Reconnect establishes connection to an existing container:
// 1. Creates SSH client and connects
// 2. Creates Docker manager
// 3. Inspects existing container, starts if stopped
// 4. Creates operator and transfer subsystems
func (m *SandboxManager) Reconnect(ctx context.Context, dockerID, containerName string, onProgress func(string, int)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if onProgress == nil {
		onProgress = func(string, int) {}
	}

	// Step 1: Create SSH client and connect
	onProgress("Connecting via SSH", 0)
	m.ssh = NewSSHClient(m.config.SSH)
	if err := m.ssh.Connect(ctx); err != nil {
		return fmt.Errorf("SSH connection failed: %w", err)
	}

	// Step 2: Create Docker manager and set existing container ID
	onProgress("Initializing Docker manager", 10)
	m.docker = NewRemoteDockerManager(m.ssh, m.config.Docker)
	m.docker.SetContainerID(dockerID, containerName)

	// Step 3: Run existing-container setup path
	m.setup = NewEnvironmentSetup(m.ssh, m.docker, onProgress)
	if err := m.setup.InitializeExisting(ctx, dockerID); err != nil {
		_ = m.ssh.Close()
		m.ssh = nil
		m.docker = nil
		m.setup = nil
		return fmt.Errorf("reconnect setup failed: %w", err)
	}

	// Step 4: Create operator and transfer subsystems
	m.operator = NewRemoteOperator(m.docker)
	m.transfer = NewFileTransfer(m.ssh)

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
// This is the old Disconnect behavior.
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

// IsConnected returns true if the sandbox is fully connected and operational.
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
