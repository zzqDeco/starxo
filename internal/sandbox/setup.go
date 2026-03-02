package sandbox

import (
	"context"
	"fmt"
	"strings"
)

// EnvironmentSetup handles automatic environment initialization on first connection.
type EnvironmentSetup struct {
	ssh        *SSHClient
	docker     *RemoteDockerManager
	onProgress func(step string, pct int)
}

// NewEnvironmentSetup creates a new EnvironmentSetup.
func NewEnvironmentSetup(ssh *SSHClient, docker *RemoteDockerManager, onProgress func(string, int)) *EnvironmentSetup {
	if onProgress == nil {
		onProgress = func(string, int) {}
	}
	return &EnvironmentSetup{
		ssh:        ssh,
		docker:     docker,
		onProgress: onProgress,
	}
}

// EnsureDockerAvailable ensures Docker is installed, the daemon is running,
// and detects whether sudo is needed. This is called after SSH connects,
// before any container operations.
func (s *EnvironmentSetup) EnsureDockerAvailable(ctx context.Context) error {
	s.onProgress("Checking Docker installation", 0)
	if err := s.ensureDockerInstalled(ctx); err != nil {
		return fmt.Errorf("docker installation check failed: %w", err)
	}

	s.onProgress("Starting Docker daemon", 50)
	if err := s.ensureDockerRunning(ctx); err != nil {
		return fmt.Errorf("failed to start Docker daemon: %w", err)
	}

	s.docker.DetectSudo(ctx)

	s.onProgress("Docker ready", 100)
	return nil
}

// SetupNewContainer performs container creation and setup:
// 1. Clean up unregistered old containers
// 2. Pull image if missing
// 3. Create container
// 4. Install Python packages
// 5. Create workspace directory
// 6. Health check
func (s *EnvironmentSetup) SetupNewContainer(ctx context.Context, excludeDockerIDs []string) error {
	s.onProgress("Cleaning up old containers", 0)
	s.cleanupOldContainers(ctx, excludeDockerIDs)

	s.onProgress("Pulling Docker image", 10)
	if err := s.docker.EnsureImageExists(ctx); err != nil {
		return fmt.Errorf("failed to ensure image exists: %w", err)
	}

	s.onProgress("Creating container", 30)
	if err := s.docker.CreateContainer(ctx); err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	s.onProgress("Installing Python packages", 50)
	if err := s.installPythonPackages(ctx); err != nil {
		return fmt.Errorf("failed to install Python packages: %w", err)
	}

	s.onProgress("Creating workspace directory", 80)
	if err := s.createWorkspaceDir(ctx); err != nil {
		return fmt.Errorf("failed to create workspace directory: %w", err)
	}

	s.onProgress("Running health check", 90)
	if err := s.healthCheck(ctx); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	s.onProgress("Container ready", 100)
	return nil
}

// InitializeFresh performs the full environment setup sequence for a new container:
// 1. Check/install Docker
// 2. Ensure Docker daemon is running
// 3. Clean up unregistered old containers
// 4. Pull image if missing
// 5. Create container
// 6. Install Python packages
// 7. Create workspace directory
// 8. Health check
func (s *EnvironmentSetup) InitializeFresh(ctx context.Context, excludeDockerIDs []string) error {
	if err := s.EnsureDockerAvailable(ctx); err != nil {
		return err
	}

	// Remap progress for container setup phase (25% - 100%)
	origProgress := s.onProgress
	s.onProgress = func(step string, pct int) {
		// Map 0-100 of container setup to 25-100 of overall
		origProgress(step, 25+pct*75/100)
	}
	defer func() { s.onProgress = origProgress }()

	return s.SetupNewContainer(ctx, excludeDockerIDs)
}

// InitializeExisting reconnects to an existing container.
// It checks Docker availability, inspects the container state, starts it if stopped,
// and runs a health check. Skips package installation since the container already has them.
func (s *EnvironmentSetup) InitializeExisting(ctx context.Context, dockerID string) error {
	// Step 1: Check Docker installed
	s.onProgress("Checking Docker installation", 0)
	if err := s.ensureDockerInstalled(ctx); err != nil {
		return fmt.Errorf("docker installation check failed: %w", err)
	}

	// Step 2: Ensure Docker daemon is running
	s.onProgress("Starting Docker daemon", 15)
	if err := s.ensureDockerRunning(ctx); err != nil {
		return fmt.Errorf("failed to start Docker daemon: %w", err)
	}

	// Detect whether docker commands need sudo
	s.docker.DetectSudo(ctx)

	// Step 3: Inspect container
	s.onProgress("Checking container state", 30)
	exists, running, err := s.docker.InspectContainer(ctx, dockerID)
	if err != nil {
		return fmt.Errorf("failed to inspect container: %w", err)
	}
	if !exists {
		return fmt.Errorf("container %s no longer exists", dockerID)
	}

	// Step 4: Start if stopped
	if !running {
		s.onProgress("Starting existing container", 50)
		if err := s.docker.StartContainer(ctx, dockerID); err != nil {
			return fmt.Errorf("failed to start container: %w", err)
		}
	} else {
		s.onProgress("Container is running", 50)
	}

	// Step 5: Health check
	s.onProgress("Running health check", 80)
	if err := s.healthCheck(ctx); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	s.onProgress("Reconnected to container", 100)
	return nil
}

// Initialize is kept for backward compatibility; delegates to InitializeFresh with no exclusions.
func (s *EnvironmentSetup) Initialize(ctx context.Context) error {
	return s.InitializeFresh(ctx, nil)
}

// cleanupOldContainers removes leftover eino-sandbox containers, skipping those in the exclude list.
func (s *EnvironmentSetup) cleanupOldContainers(ctx context.Context, excludeDockerIDs []string) {
	excludeSet := make(map[string]bool, len(excludeDockerIDs))
	for _, id := range excludeDockerIDs {
		excludeSet[id] = true
	}

	// List all containers with "eino-sandbox" in the name
	listCmd := fmt.Sprintf("%s ps -a --filter name=eino-sandbox --format '{{.ID}}'", s.docker.dockerCmd())
	stdout, _, exitCode, err := s.ssh.RunCommand(ctx, listCmd)
	if err != nil || exitCode != 0 {
		return // best-effort cleanup
	}
	ids := strings.TrimSpace(stdout)
	if ids == "" {
		return
	}
	// Remove non-excluded containers
	for _, id := range strings.Split(ids, "\n") {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if excludeSet[id] {
			continue // skip registered container
		}
		// Also check if the full docker ID starts with an excluded ID (short vs long ID matching)
		skip := false
		for exID := range excludeSet {
			if strings.HasPrefix(id, exID) || strings.HasPrefix(exID, id) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		rmCmd := fmt.Sprintf("%s rm -f %s", s.docker.dockerCmd(), id)
		s.ssh.RunCommand(ctx, rmCmd)
	}
}

// ensureDockerInstalled checks if Docker is installed and installs it if not.
func (s *EnvironmentSetup) ensureDockerInstalled(ctx context.Context) error {
	stdout, _, exitCode, err := s.ssh.RunCommand(ctx, "docker --version")
	if err != nil {
		return fmt.Errorf("failed to check Docker: %w", err)
	}

	if exitCode == 0 && strings.Contains(stdout, "Docker") {
		return nil
	}

	// Auto-install Docker
	s.onProgress("Installing Docker", 5)
	_, stderr, exitCode, err := s.ssh.RunCommand(ctx, "curl -fsSL https://get.docker.com | sudo sh")
	if err != nil {
		return fmt.Errorf("failed to install Docker: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("Docker installation failed (exit %d): %s", exitCode, stderr)
	}

	return nil
}

// ensureDockerRunning ensures the Docker daemon is running.
func (s *EnvironmentSetup) ensureDockerRunning(ctx context.Context) error {
	// Check if Docker daemon is responsive
	_, _, exitCode, err := s.ssh.RunCommand(ctx, "docker info > /dev/null 2>&1")
	if err != nil {
		return fmt.Errorf("failed to check Docker daemon: %w", err)
	}

	if exitCode == 0 {
		return nil
	}

	// Try to start Docker with sudo
	_, stderr, exitCode, err := s.ssh.RunCommand(ctx, "sudo systemctl start docker")
	if err != nil {
		return fmt.Errorf("failed to start Docker: %w", err)
	}
	if exitCode != 0 {
		// Fallback: try `sudo service docker start` for systems without systemctl
		_, stderr2, exitCode2, err2 := s.ssh.RunCommand(ctx, "sudo service docker start")
		if err2 != nil || exitCode2 != 0 {
			return fmt.Errorf("failed to start Docker daemon (exit %d): %s / %s", exitCode, stderr, stderr2)
		}
	}

	// Ensure current user is in docker group
	s.ssh.RunCommand(ctx, "sudo usermod -aG docker $(whoami)")

	// Verify Docker is now running
	_, _, exitCode, err = s.ssh.RunCommand(ctx, "docker info > /dev/null 2>&1")
	if err != nil {
		return fmt.Errorf("failed to verify Docker daemon: %w", err)
	}
	if exitCode != 0 {
		_, _, exitCode, err = s.ssh.RunCommand(ctx, "sudo docker info > /dev/null 2>&1")
		if err != nil || exitCode != 0 {
			return fmt.Errorf("Docker daemon failed to start")
		}
	}

	return nil
}

// installPythonPackages installs common Python packages inside the container.
func (s *EnvironmentSetup) installPythonPackages(ctx context.Context) error {
	packages := "pandas numpy matplotlib openpyxl"
	cmd := []string{"pip", "install", "--no-cache-dir", "pandas", "numpy", "matplotlib", "openpyxl"}

	stdout, stderr, exitCode, err := s.docker.ExecInContainer(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to install packages (%s): %w", packages, err)
	}
	if exitCode != 0 {
		return fmt.Errorf("pip install failed (exit %d): stdout=%s stderr=%s", exitCode, stdout, stderr)
	}

	return nil
}

// createWorkspaceDir creates the workspace directory inside the container.
func (s *EnvironmentSetup) createWorkspaceDir(ctx context.Context) error {
	workDir := s.docker.cfg.WorkDir
	if workDir == "" {
		workDir = "/workspace"
	}

	_, stderr, exitCode, err := s.docker.ExecInContainer(ctx, []string{"mkdir", "-p", workDir})
	if err != nil {
		return fmt.Errorf("failed to create workspace dir: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("mkdir failed (exit %d): %s", exitCode, stderr)
	}

	return nil
}

// healthCheck verifies the container is healthy by running basic commands.
func (s *EnvironmentSetup) healthCheck(ctx context.Context) error {
	// Check Python is available
	stdout, _, exitCode, err := s.docker.ExecInContainer(ctx, []string{"python3", "--version"})
	if err != nil {
		return fmt.Errorf("python3 check failed: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("python3 is not available in the container")
	}
	if !strings.Contains(stdout, "Python") {
		return fmt.Errorf("unexpected python3 output: %s", stdout)
	}

	// Check workspace directory exists
	_, _, exitCode, err = s.docker.ExecInContainer(ctx, []string{"test", "-d", s.docker.cfg.WorkDir})
	if err != nil {
		return fmt.Errorf("workspace check failed: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("workspace directory %s does not exist", s.docker.cfg.WorkDir)
	}

	return nil
}
