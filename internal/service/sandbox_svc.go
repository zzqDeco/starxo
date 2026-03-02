package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"starxo/internal/config"
	"starxo/internal/model"
	"starxo/internal/sandbox"
	"starxo/internal/storage"
)

// SandboxService manages sandbox lifecycle for the frontend.
type SandboxService struct {
	ctx              context.Context
	manager          *sandbox.SandboxManager
	store            *config.Store
	containerStore   *storage.ContainerStore
	sessionService   *SessionService
	onConnect        func(mgr *sandbox.SandboxManager)
	onContainerBound func(containerRegID, workspacePath string)
	onContainerDeactivated func()
	// activeContainerRegID tracks the registry ID of the currently connected container
	activeContainerRegID string
}

// NewSandboxService creates a new SandboxService.
func NewSandboxService(store *config.Store, containerStore *storage.ContainerStore) *SandboxService {
	return &SandboxService{
		store:          store,
		containerStore: containerStore,
	}
}

// SetSessionService sets the session service dependency for container ownership.
func (s *SandboxService) SetSessionService(ss *SessionService) {
	s.sessionService = ss
}

// SetContext stores the Wails application context. Called from app.go startup.
func (s *SandboxService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// SetOnConnect registers a callback that fires after a container is activated.
func (s *SandboxService) SetOnConnect(fn func(mgr *sandbox.SandboxManager)) {
	s.onConnect = fn
}

// SetOnContainerBound registers a callback that fires after a container is connected,
// passing the registry ID and workspace path so they can be bound to the active session.
func (s *SandboxService) SetOnContainerBound(fn func(containerRegID, workspacePath string)) {
	s.onContainerBound = fn
}

// SetOnContainerDeactivated registers a callback that fires when the active container
// is deactivated (e.g. user deactivates or session switches to one with no container).
func (s *SandboxService) SetOnContainerDeactivated(fn func()) {
	s.onContainerDeactivated = fn
}

// --- New SSH-independent methods ---

// ConnectSSH establishes SSH connection and ensures Docker is available on the remote host.
// Does NOT create any container. Use CreateAndActivateContainer separately.
func (s *SandboxService) ConnectSSH() error {
	// Disconnect existing SSH (keep containers alive on remote)
	if s.manager != nil {
		wailsruntime.EventsEmit(s.ctx, "ssh:progress", SandboxProgressEvent{
			Step:    "Cleaning up previous connection...",
			Percent: 0,
		})
		if s.activeContainerRegID != "" {
			s.manager.DetachContainer()
			s.activeContainerRegID = ""
		}
		_ = s.manager.Disconnect(s.ctx)
		s.manager = nil
	}

	cfg := s.store.Get()
	s.manager = sandbox.NewSandboxManager(*cfg)

	// Step 1: SSH connect
	if err := s.manager.ConnectSSH(s.ctx, func(step string, percent int) {
		wailsruntime.EventsEmit(s.ctx, "ssh:progress", SandboxProgressEvent{
			Step:    step,
			Percent: percent / 2, // 0-50%
		})
	}); err != nil {
		s.manager = nil
		return fmt.Errorf("SSH connection failed: %w", err)
	}

	// Step 2: Ensure Docker
	if err := s.manager.EnsureDocker(s.ctx, func(step string, percent int) {
		wailsruntime.EventsEmit(s.ctx, "ssh:progress", SandboxProgressEvent{
			Step:    step,
			Percent: 50 + percent/2, // 50-100%
		})
	}); err != nil {
		_ = s.manager.Disconnect(s.ctx)
		s.manager = nil
		return fmt.Errorf("Docker setup failed: %w", err)
	}

	// Start health monitor in SSH-only mode
	s.StartHealthMonitor(s.ctx)

	wailsruntime.EventsEmit(s.ctx, "ssh:connected", nil)
	return nil
}

// DisconnectSSH closes the SSH connection. Detaches any active container first.
func (s *SandboxService) DisconnectSSH() error {
	if s.manager == nil {
		return nil
	}

	// Deactivate container if active (without emitting events since we're disconnecting entirely)
	if s.activeContainerRegID != "" {
		s.manager.DetachContainer()
		s.activeContainerRegID = ""
		if s.onContainerDeactivated != nil {
			s.onContainerDeactivated()
		}
	}

	err := s.manager.Disconnect(s.ctx)
	s.manager = nil

	wailsruntime.EventsEmit(s.ctx, "ssh:disconnected", nil)
	return err
}

// CreateAndActivateContainer creates a new container on the connected SSH host,
// registers it, and activates it for agent use.
func (s *SandboxService) CreateAndActivateContainer() error {
	if s.manager == nil || !s.manager.SSHConnected() {
		return fmt.Errorf("SSH not connected")
	}

	// Detach current container if any
	if s.activeContainerRegID != "" {
		s.manager.DetachContainer()
		s.activeContainerRegID = ""
	}

	cfg := s.store.Get()
	excludeIDs := s.containerStore.RegisteredDockerIDs()

	dockerID, containerName, err := s.manager.CreateNewContainer(s.ctx, excludeIDs, func(step string, percent int) {
		wailsruntime.EventsEmit(s.ctx, "container:progress", SandboxProgressEvent{
			Step:    step,
			Percent: percent,
		})
	})
	if err != nil {
		return fmt.Errorf("container creation failed: %w", err)
	}

	// Register the new container
	regID := uuid.New().String()[:8]
	now := time.Now().UnixMilli()

	sessionID := ""
	if s.sessionService != nil {
		if active := s.sessionService.GetActiveSession(); active != nil {
			sessionID = active.ID
		}
	}

	container := &model.Container{
		ID:            regID,
		DockerID:      dockerID,
		Name:          containerName,
		Image:         cfg.Docker.Image,
		SSHHost:       cfg.SSH.Host,
		SSHPort:       cfg.SSH.Port,
		Status:        model.ContainerRunning,
		SetupComplete: true,
		SessionID:     sessionID,
		CreatedAt:     now,
		LastUsedAt:    now,
	}
	_ = s.containerStore.Add(container)
	s.activeContainerRegID = regID

	s.setupOutputForwarding()

	if s.onConnect != nil {
		s.onConnect(s.manager)
	}

	if s.onContainerBound != nil {
		s.onContainerBound(regID, "/workspace")
	}

	wailsruntime.EventsEmit(s.ctx, "container:ready", map[string]string{
		"containerID": regID,
	})
	return nil
}

// ActivateContainer switches the active container to a previously registered one.
// The container must be on the same SSH host as the current connection.
func (s *SandboxService) ActivateContainer(containerRegID string) error {
	if s.manager == nil || !s.manager.SSHConnected() {
		return fmt.Errorf("SSH not connected")
	}

	container, err := s.containerStore.Get(containerRegID)
	if err != nil {
		return fmt.Errorf("container not found: %w", err)
	}

	// Validate SSH host matches
	cfg := s.store.Get()
	if container.SSHHost != cfg.SSH.Host || container.SSHPort != cfg.SSH.Port {
		return fmt.Errorf("container is on %s:%d but SSH is connected to %s:%d — disconnect and reconnect SSH to the correct host first",
			container.SSHHost, container.SSHPort, cfg.SSH.Host, cfg.SSH.Port)
	}

	// Detach current container if any
	if s.activeContainerRegID != "" {
		s.manager.DetachContainer()
		s.activeContainerRegID = ""
	}

	// Attach to the target container
	if err := s.manager.AttachToContainer(s.ctx, container.DockerID, container.Name, func(step string, percent int) {
		wailsruntime.EventsEmit(s.ctx, "container:progress", SandboxProgressEvent{
			Step:    step,
			Percent: percent,
		})
	}); err != nil {
		return fmt.Errorf("failed to activate container: %w", err)
	}

	// Update registry
	container.Status = model.ContainerRunning
	container.LastUsedAt = time.Now().UnixMilli()
	_ = s.containerStore.Update(container)
	s.activeContainerRegID = containerRegID

	s.setupOutputForwarding()

	if s.onConnect != nil {
		s.onConnect(s.manager)
	}

	if s.onContainerBound != nil {
		s.onContainerBound(containerRegID, "/workspace")
	}

	wailsruntime.EventsEmit(s.ctx, "container:activated", map[string]string{
		"containerID": containerRegID,
	})
	return nil
}

// DeactivateContainer detaches the active container without stopping it.
// SSH remains connected.
func (s *SandboxService) DeactivateContainer() error {
	if s.manager == nil {
		return nil
	}

	if s.activeContainerRegID == "" {
		return nil
	}

	s.manager.DetachContainer()
	s.activeContainerRegID = ""

	if s.onContainerDeactivated != nil {
		s.onContainerDeactivated()
	}

	wailsruntime.EventsEmit(s.ctx, "container:deactivated", nil)
	return nil
}

// --- Legacy methods (kept for backward compatibility, internally use new methods) ---

// Connect creates a new container and connects to it.
// This is a convenience method that calls ConnectSSH + CreateAndActivateContainer.
func (s *SandboxService) Connect() error {
	if err := s.ConnectSSH(); err != nil {
		return err
	}
	return s.CreateAndActivateContainer()
}

// ConnectExisting reconnects to a previously registered container.
// If SSH is not connected, connects SSH first using the container's stored host.
func (s *SandboxService) ConnectExisting(containerRegID string) error {
	container, err := s.containerStore.Get(containerRegID)
	if err != nil {
		return fmt.Errorf("container not found: %w", err)
	}

	// If SSH is not connected or connected to a different host, reconnect
	if s.manager == nil || !s.manager.SSHConnected() {
		// Disconnect existing if any
		if s.manager != nil {
			_ = s.manager.Disconnect(s.ctx)
			s.manager = nil
		}

		cfg := s.store.Get()
		cfg.SSH.Host = container.SSHHost
		cfg.SSH.Port = container.SSHPort
		s.manager = sandbox.NewSandboxManager(*cfg)

		if err := s.manager.ConnectSSH(s.ctx, func(step string, percent int) {
			wailsruntime.EventsEmit(s.ctx, "ssh:progress", SandboxProgressEvent{
				Step:    step,
				Percent: percent / 2,
			})
		}); err != nil {
			s.manager = nil
			return fmt.Errorf("SSH connection failed: %w", err)
		}

		if err := s.manager.EnsureDocker(s.ctx, func(step string, percent int) {
			wailsruntime.EventsEmit(s.ctx, "ssh:progress", SandboxProgressEvent{
				Step:    step,
				Percent: 50 + percent/2,
			})
		}); err != nil {
			_ = s.manager.Disconnect(s.ctx)
			s.manager = nil
			return fmt.Errorf("Docker setup failed: %w", err)
		}

		wailsruntime.EventsEmit(s.ctx, "ssh:connected", nil)
	}

	return s.ActivateContainer(containerRegID)
}

// Disconnect closes SSH but keeps the container alive for future reconnection.
func (s *SandboxService) Disconnect() error {
	return s.DisconnectSSH()
}

// DisconnectAndDestroy stops and removes the active container, then closes SSH.
func (s *SandboxService) DisconnectAndDestroy() error {
	if s.manager == nil {
		return nil
	}

	err := s.manager.DisconnectAndDestroy(s.ctx)
	s.manager = nil

	// Remove from registry
	if s.activeContainerRegID != "" {
		_ = s.containerStore.Remove(s.activeContainerRegID)
		s.activeContainerRegID = ""
	}

	return err
}

// GetStatus returns the current sandbox connection status.
func (s *SandboxService) GetStatus() SandboxStatusDTO {
	if s.manager == nil {
		return SandboxStatusDTO{}
	}

	status := SandboxStatusDTO{
		SSHConnected:      s.manager.SSHConnected(),
		DockerRunning:     false,
		ContainerID:       "",
		DockerAvailable:   s.manager.Docker() != nil,
		ActiveContainerID: s.activeContainerRegID,
	}

	docker := s.manager.Docker()
	if docker != nil {
		status.DockerRunning = docker.IsRunning()
		status.ContainerID = docker.ContainerID()
		status.ActiveContainerName = docker.ContainerName()
	}

	return status
}

// Manager returns the underlying SandboxManager for internal use by other services.
func (s *SandboxService) Manager() *sandbox.SandboxManager {
	return s.manager
}

// ActiveContainerRegID returns the registry ID of the currently connected container.
func (s *SandboxService) ActiveContainerRegID() string {
	return s.activeContainerRegID
}

// setupOutputForwarding sets up terminal output forwarding to the frontend.
func (s *SandboxService) setupOutputForwarding() {
	if s.manager == nil {
		return
	}
	if op := s.manager.Operator(); op != nil {
		wailsCtx := s.ctx
		op.SetOnOutput(func(stdout, stderr string, exitCode int) {
			wailsruntime.EventsEmit(wailsCtx, "terminal:output", TerminalOutputEvent{
				Stdout:   stdout,
				Stderr:   stderr,
				ExitCode: exitCode,
			})
		})
	}
}

// StartHealthMonitor launches a background goroutine that periodically checks
// whether the connected sandbox is still alive. Supports two modes:
// - SSH-only: pings SSH when no container is active
// - Full: pings through the operator when a container is active
func (s *SandboxService) StartHealthMonitor(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if s.manager == nil {
					continue
				}

				if !s.manager.SSHConnected() {
					continue
				}

				// If there's an active container with an operator, ping through it
				if s.manager.HasActiveContainer() {
					op := s.manager.Operator()
					if op == nil {
						continue
					}
					_, err := op.RunCommand(ctx, []string{"echo", "ping"})
					if err != nil {
						// Connection lost
						wailsruntime.EventsEmit(s.ctx, "ssh:disconnected", nil)
						s.manager = nil
						s.activeContainerRegID = ""
					}
				} else {
					// SSH-only mode: check SSH is still alive via the SSH client
					ssh := s.manager.SSH()
					if ssh == nil {
						continue
					}
					_, _, _, err := ssh.RunCommand(ctx, "echo ping")
					if err != nil {
						wailsruntime.EventsEmit(s.ctx, "ssh:disconnected", nil)
						s.manager = nil
					}
				}
			}
		}
	}()
}
