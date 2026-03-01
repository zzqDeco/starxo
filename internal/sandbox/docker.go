package sandbox

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"

	"starxo/internal/config"
)

// RemoteDockerManager manages Docker containers over SSH.
type RemoteDockerManager struct {
	ssh           *SSHClient
	cfg           config.DockerConfig
	containerID   string
	containerName string
	useSudo       bool // whether docker commands need sudo
	mu            sync.Mutex
}

// NewRemoteDockerManager creates a new RemoteDockerManager.
func NewRemoteDockerManager(ssh *SSHClient, cfg config.DockerConfig) *RemoteDockerManager {
	return &RemoteDockerManager{
		ssh: ssh,
		cfg: cfg,
	}
}

// dockerCmd returns "docker" or "sudo docker" depending on whether sudo is needed.
func (m *RemoteDockerManager) dockerCmd() string {
	if m.useSudo {
		return "sudo docker"
	}
	return "docker"
}

// DetectSudo checks if docker commands need sudo and sets the flag.
func (m *RemoteDockerManager) DetectSudo(ctx context.Context) {
	_, _, exitCode, err := m.ssh.RunCommand(ctx, "docker info > /dev/null 2>&1")
	if err == nil && exitCode == 0 {
		m.useSudo = false
		return
	}
	// Try with sudo
	_, _, exitCode, err = m.ssh.RunCommand(ctx, "sudo docker info > /dev/null 2>&1")
	if err == nil && exitCode == 0 {
		m.useSudo = true
		return
	}
	// Default to sudo
	m.useSudo = true
}

// EnsureImageExists pulls the Docker image if it is not already present on the remote host.
func (m *RemoteDockerManager) EnsureImageExists(ctx context.Context) error {
	// Check if image exists locally
	checkCmd := fmt.Sprintf("%s image inspect %s > /dev/null 2>&1 && echo EXISTS || echo MISSING", m.dockerCmd(), m.cfg.Image)
	stdout, _, _, err := m.ssh.RunCommand(ctx, checkCmd)
	if err != nil {
		return fmt.Errorf("failed to check image existence: %w", err)
	}

	if strings.TrimSpace(stdout) == "EXISTS" {
		return nil
	}

	// Pull the image
	pullCmd := fmt.Sprintf("%s pull %s", m.dockerCmd(), m.cfg.Image)
	_, stderr, exitCode, err := m.ssh.RunCommand(ctx, pullCmd)
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", m.cfg.Image, err)
	}
	if exitCode != 0 {
		return fmt.Errorf("docker pull failed (exit %d): %s", exitCode, stderr)
	}

	return nil
}

// CreateContainer creates and starts a new Docker container on the remote host.
func (m *RemoteDockerManager) CreateContainer(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := fmt.Sprintf("eino-sandbox-%s", uuid.New().String()[:8])

	workDir := m.cfg.WorkDir
	if workDir == "" {
		workDir = "/workspace"
	}

	// Build docker run command
	cmd := fmt.Sprintf("%s run -d --name %s", m.dockerCmd(), name)

	if m.cfg.MemoryLimit > 0 {
		cmd += fmt.Sprintf(" -m %dm", m.cfg.MemoryLimit)
	}

	if m.cfg.CPULimit > 0 {
		cmd += fmt.Sprintf(" --cpus %.2f", m.cfg.CPULimit)
	}

	cmd += fmt.Sprintf(" -w %s", workDir)

	if !m.cfg.Network {
		cmd += " --network none"
	}

	cmd += fmt.Sprintf(" %s sleep infinity", m.cfg.Image)

	stdout, stderr, exitCode, err := m.ssh.RunCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("docker run failed (exit %d): %s", exitCode, stderr)
	}

	m.containerID = strings.TrimSpace(stdout)
	m.containerName = name
	return nil
}

// ExecInContainer executes a command inside the running container.
func (m *RemoteDockerManager) ExecInContainer(ctx context.Context, cmd []string) (stdout, stderr string, exitCode int, err error) {
	m.mu.Lock()
	cid := m.containerID
	m.mu.Unlock()

	if cid == "" {
		return "", "", -1, fmt.Errorf("no container is running")
	}

	// Escape command arguments for shell
	escapedArgs := make([]string, len(cmd))
	for i, arg := range cmd {
		escapedArgs[i] = shellQuote(arg)
	}

	execCmd := fmt.Sprintf("%s exec %s %s", m.dockerCmd(), cid, strings.Join(escapedArgs, " "))
	return m.ssh.RunCommand(ctx, execCmd)
}

// CopyToContainer copies a file from the remote host into the container.
func (m *RemoteDockerManager) CopyToContainer(ctx context.Context, hostPath, containerPath string) error {
	m.mu.Lock()
	cid := m.containerID
	m.mu.Unlock()

	if cid == "" {
		return fmt.Errorf("no container is running")
	}

	cmd := fmt.Sprintf("%s cp %s %s:%s", m.dockerCmd(), shellQuote(hostPath), cid, shellQuote(containerPath))
	_, stderr, exitCode, err := m.ssh.RunCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to copy to container: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("docker cp to container failed (exit %d): %s", exitCode, stderr)
	}

	return nil
}

// CopyFromContainer copies a file from the container to the remote host.
func (m *RemoteDockerManager) CopyFromContainer(ctx context.Context, containerPath, hostPath string) error {
	m.mu.Lock()
	cid := m.containerID
	m.mu.Unlock()

	if cid == "" {
		return fmt.Errorf("no container is running")
	}

	cmd := fmt.Sprintf("%s cp %s:%s %s", m.dockerCmd(), cid, shellQuote(containerPath), shellQuote(hostPath))
	_, stderr, exitCode, err := m.ssh.RunCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to copy from container: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("docker cp from container failed (exit %d): %s", exitCode, stderr)
	}

	return nil
}

// StopAndRemove stops and removes the container.
func (m *RemoteDockerManager) StopAndRemove(ctx context.Context) error {
	m.mu.Lock()
	cid := m.containerID
	m.mu.Unlock()

	if cid == "" {
		return nil
	}

	// Stop the container with a timeout
	stopCmd := fmt.Sprintf("%s stop -t 5 %s", m.dockerCmd(), cid)
	_, _, _, _ = m.ssh.RunCommand(ctx, stopCmd)

	// Remove the container
	rmCmd := fmt.Sprintf("%s rm -f %s", m.dockerCmd(), cid)
	_, stderr, exitCode, err := m.ssh.RunCommand(ctx, rmCmd)
	if err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("docker rm failed (exit %d): %s", exitCode, stderr)
	}

	m.mu.Lock()
	m.containerID = ""
	m.mu.Unlock()

	return nil
}

// ContainerID returns the current container ID.
func (m *RemoteDockerManager) ContainerID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.containerID
}

// ContainerName returns the current container name.
func (m *RemoteDockerManager) ContainerName() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.containerName
}

// IsRunning returns true if a container is currently running.
func (m *RemoteDockerManager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.containerID != ""
}

// SetContainerID sets the container ID and name directly (used for reconnection).
func (m *RemoteDockerManager) SetContainerID(id, name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.containerID = id
	m.containerName = name
}

// InspectContainer checks if a container exists and whether it is running.
func (m *RemoteDockerManager) InspectContainer(ctx context.Context, dockerID string) (exists bool, running bool, err error) {
	cmd := fmt.Sprintf("%s inspect --format '{{.State.Running}}' %s 2>/dev/null", m.dockerCmd(), dockerID)
	stdout, _, exitCode, err := m.ssh.RunCommand(ctx, cmd)
	if err != nil {
		return false, false, fmt.Errorf("failed to inspect container: %w", err)
	}
	if exitCode != 0 {
		return false, false, nil // container does not exist
	}
	state := strings.TrimSpace(stdout)
	return true, state == "true", nil
}

// StartContainer starts a stopped container.
func (m *RemoteDockerManager) StartContainer(ctx context.Context, dockerID string) error {
	cmd := fmt.Sprintf("%s start %s", m.dockerCmd(), dockerID)
	_, stderr, exitCode, err := m.ssh.RunCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("docker start failed (exit %d): %s", exitCode, stderr)
	}
	return nil
}

// StopContainer stops the current container without removing it.
func (m *RemoteDockerManager) StopContainer(ctx context.Context) error {
	m.mu.Lock()
	cid := m.containerID
	m.mu.Unlock()

	if cid == "" {
		return nil
	}

	cmd := fmt.Sprintf("%s stop -t 5 %s", m.dockerCmd(), cid)
	_, stderr, exitCode, err := m.ssh.RunCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("docker stop failed (exit %d): %s", exitCode, stderr)
	}

	return nil
}

// shellQuote wraps a string in single quotes for safe shell usage.
func shellQuote(s string) string {
	// Replace single quotes with '\'' and wrap in single quotes
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
