package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"starxo/internal/config"
	"starxo/internal/model"
	"starxo/internal/sandbox"
	"starxo/internal/storage"
)

// SandboxService manages sandbox lifecycle for the frontend.
type SandboxService struct {
	mu                     sync.RWMutex
	ctx                    context.Context
	manager                *sandbox.SandboxManager
	store                  *config.Store
	containerStore         *storage.ContainerStore
	sessionService         *SessionService
	onConnect              func(mgr *sandbox.SandboxManager)
	onContainerBound       func(containerRegID, workspacePath string)
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
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionService = ss
}

// SetContext stores the Wails application context. Called from app.go startup.
func (s *SandboxService) SetContext(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ctx = ctx
}

// SetOnConnect registers a callback that fires after a container is activated.
func (s *SandboxService) SetOnConnect(fn func(mgr *sandbox.SandboxManager)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onConnect = fn
}

// SetOnContainerBound registers a callback that fires after a container is connected,
// passing the registry ID and workspace path so they can be bound to the active session.
func (s *SandboxService) SetOnContainerBound(fn func(containerRegID, workspacePath string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onContainerBound = fn
}

// SetOnContainerDeactivated registers a callback that fires when the active container
// is deactivated (e.g. user deactivates or session switches to one with no container).
func (s *SandboxService) SetOnContainerDeactivated(fn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onContainerDeactivated = fn
}

// --- New SSH-independent methods ---

// ConnectSSH establishes SSH connection and ensures the lightweight sandbox
// runtime is available on the remote host.
func (s *SandboxService) ConnectSSH() error {
	s.mu.Lock()
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
	mgr := sandbox.NewSandboxManager(*cfg)
	s.manager = mgr
	appCtx := s.ctx
	s.mu.Unlock()

	// Step 1: SSH connect (long-running, outside lock)
	if err := mgr.ConnectSSH(appCtx, func(step string, percent int) {
		wailsruntime.EventsEmit(appCtx, "ssh:progress", SandboxProgressEvent{
			Step:    step,
			Percent: percent / 2, // 0-50%
		})
	}); err != nil {
		s.mu.Lock()
		s.manager = nil
		s.mu.Unlock()
		return fmt.Errorf("SSH connection failed: %w", err)
	}

	// Step 2: Ensure sandbox runtime (long-running, outside lock)
	if err := mgr.EnsureRuntime(appCtx, func(step string, percent int) {
		wailsruntime.EventsEmit(appCtx, "ssh:progress", SandboxProgressEvent{
			Step:    step,
			Percent: 50 + percent/2, // 50-100%
		})
	}); err != nil {
		_ = mgr.Disconnect(appCtx)
		s.mu.Lock()
		s.manager = nil
		s.mu.Unlock()
		return fmt.Errorf("sandbox runtime setup failed: %w", err)
	}

	// Start health monitor in SSH-only mode
	s.StartHealthMonitor(appCtx)

	wailsruntime.EventsEmit(appCtx, "ssh:connected", nil)
	return nil
}

// DisconnectSSH closes the SSH connection. Detaches any active container first.
func (s *SandboxService) DisconnectSSH() error {
	s.mu.Lock()
	if s.manager == nil {
		s.mu.Unlock()
		return nil
	}

	// Deactivate container if active (without emitting events since we're disconnecting entirely)
	var deactivatedCb func()
	if s.activeContainerRegID != "" {
		s.manager.DetachContainer()
		s.activeContainerRegID = ""
		deactivatedCb = s.onContainerDeactivated
	}

	err := s.manager.Disconnect(s.ctx)
	s.manager = nil
	appCtx := s.ctx
	s.mu.Unlock()

	// Call callback outside lock to prevent deadlocks
	if deactivatedCb != nil {
		deactivatedCb()
	}

	wailsruntime.EventsEmit(appCtx, "ssh:disconnected", nil)
	return err
}

// CreateAndActivateContainer creates a new sandbox on the connected SSH host,
// registers it, and activates it for agent use.
func (s *SandboxService) CreateAndActivateContainer() error {
	s.mu.Lock()
	if s.manager == nil || !s.manager.SSHConnected() {
		s.mu.Unlock()
		return fmt.Errorf("SSH not connected")
	}

	// Detach current container if any
	if s.activeContainerRegID != "" {
		s.manager.DetachContainer()
		s.activeContainerRegID = ""
	}

	mgr := s.manager
	appCtx := s.ctx
	s.mu.Unlock()

	cfg := s.store.Get()
	excludeIDs := s.containerStore.RegisteredDockerIDs()

	// Long-running operation outside lock
	inst, err := mgr.CreateNewSandbox(appCtx, excludeIDs, func(step string, percent int) {
		wailsruntime.EventsEmit(appCtx, "container:progress", SandboxProgressEvent{
			Step:    step,
			Percent: percent,
		})
	})
	if err != nil {
		return fmt.Errorf("sandbox creation failed: %w", err)
	}

	// Register the new sandbox. The model name remains Container for Wails compatibility.
	regID := inst.ID
	now := time.Now().UnixMilli()

	s.mu.RLock()
	sessionSvc := s.sessionService
	s.mu.RUnlock()

	sessionID := ""
	if sessionSvc != nil {
		if active := sessionSvc.GetActiveSession(); active != nil {
			sessionID = active.ID
		}
	}

	container := &model.Container{
		ID:            regID,
		RuntimeID:     inst.ID,
		Runtime:       inst.Runtime,
		WorkspacePath: inst.WorkspacePath,
		DockerID:      inst.ID,
		Name:          inst.Name,
		Image:         inst.Runtime,
		SSHHost:       cfg.SSH.Host,
		SSHPort:       cfg.SSH.Port,
		Status:        model.ContainerRunning,
		SetupComplete: true,
		SessionID:     sessionID,
		CreatedAt:     now,
		LastUsedAt:    now,
	}
	_ = s.containerStore.Add(container)

	s.mu.Lock()
	s.activeContainerRegID = regID
	connectCb := s.onConnect
	boundCb := s.onContainerBound
	s.mu.Unlock()

	s.setupOutputForwarding()

	// Call callbacks outside lock to prevent deadlocks
	if connectCb != nil {
		connectCb(mgr)
	}

	if boundCb != nil {
		boundCb(regID, inst.WorkspacePath)
	}

	wailsruntime.EventsEmit(appCtx, "container:ready", map[string]string{
		"containerID": regID,
	})
	return nil
}

// ActivateContainer switches the active container to a previously registered one.
// The container must be on the same SSH host as the current connection.
func (s *SandboxService) ActivateContainer(containerRegID string) error {
	s.mu.Lock()
	if s.manager == nil || !s.manager.SSHConnected() {
		s.mu.Unlock()
		return fmt.Errorf("SSH not connected")
	}
	mgr := s.manager
	appCtx := s.ctx
	s.mu.Unlock()

	container, err := s.containerStore.Get(containerRegID)
	if err != nil {
		return fmt.Errorf("sandbox not found: %w", err)
	}
	if container.Status == model.ContainerUnavailable || container.Runtime == sandbox.RuntimeDocker {
		return fmt.Errorf("sandbox %s is a legacy Docker record and cannot be activated by the dockerless runtime", containerRegID)
	}

	// Validate SSH host matches
	cfg := s.store.Get()
	if container.SSHHost != cfg.SSH.Host || container.SSHPort != cfg.SSH.Port {
		return fmt.Errorf("sandbox is on %s:%d but SSH is connected to %s:%d; disconnect and reconnect SSH to the correct host first",
			container.SSHHost, container.SSHPort, cfg.SSH.Host, cfg.SSH.Port)
	}

	// Detach current container if any
	s.mu.Lock()
	if s.activeContainerRegID != "" {
		s.manager.DetachContainer()
		s.activeContainerRegID = ""
	}
	s.mu.Unlock()

	runtimeID := container.RuntimeID
	if runtimeID == "" {
		runtimeID = container.DockerID
	}

	// Attach to the target sandbox (long-running, outside lock)
	if err := mgr.AttachToSandbox(appCtx, runtimeID, container.Name, container.WorkspacePath, func(step string, percent int) {
		wailsruntime.EventsEmit(appCtx, "container:progress", SandboxProgressEvent{
			Step:    step,
			Percent: percent,
		})
	}); err != nil {
		return fmt.Errorf("failed to activate sandbox: %w", err)
	}

	// Update registry
	container.Status = model.ContainerRunning
	container.LastUsedAt = time.Now().UnixMilli()
	_ = s.containerStore.Update(container)

	s.mu.Lock()
	s.activeContainerRegID = containerRegID
	connectCb := s.onConnect
	boundCb := s.onContainerBound
	s.mu.Unlock()

	s.setupOutputForwarding()

	// Call callbacks outside lock to prevent deadlocks
	if connectCb != nil {
		connectCb(mgr)
	}

	if boundCb != nil {
		boundCb(containerRegID, container.WorkspacePath)
	}

	wailsruntime.EventsEmit(appCtx, "container:activated", map[string]string{
		"containerID": containerRegID,
	})
	return nil
}

// DeactivateContainer detaches the active container without stopping it.
// SSH remains connected.
func (s *SandboxService) DeactivateContainer() error {
	s.mu.Lock()
	if s.manager == nil {
		s.mu.Unlock()
		return nil
	}

	if s.activeContainerRegID == "" {
		s.mu.Unlock()
		return nil
	}

	s.manager.DetachContainer()
	s.activeContainerRegID = ""
	deactivatedCb := s.onContainerDeactivated
	appCtx := s.ctx
	s.mu.Unlock()

	// Call callback outside lock to prevent deadlocks
	if deactivatedCb != nil {
		deactivatedCb()
	}

	wailsruntime.EventsEmit(appCtx, "container:deactivated", nil)
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
	s.mu.Lock()
	needsSSH := s.manager == nil || !s.manager.SSHConnected()
	if needsSSH {
		// Disconnect existing if any
		if s.manager != nil {
			_ = s.manager.Disconnect(s.ctx)
			s.manager = nil
		}

		cfg := s.store.Get()
		cfg.SSH.Host = container.SSHHost
		cfg.SSH.Port = container.SSHPort
		mgr := sandbox.NewSandboxManager(*cfg)
		s.manager = mgr
		appCtx := s.ctx
		s.mu.Unlock()

		// Long-running operations outside lock
		if err := mgr.ConnectSSH(appCtx, func(step string, percent int) {
			wailsruntime.EventsEmit(appCtx, "ssh:progress", SandboxProgressEvent{
				Step:    step,
				Percent: percent / 2,
			})
		}); err != nil {
			s.mu.Lock()
			s.manager = nil
			s.mu.Unlock()
			return fmt.Errorf("SSH connection failed: %w", err)
		}

		if err := mgr.EnsureRuntime(appCtx, func(step string, percent int) {
			wailsruntime.EventsEmit(appCtx, "ssh:progress", SandboxProgressEvent{
				Step:    step,
				Percent: 50 + percent/2,
			})
		}); err != nil {
			_ = mgr.Disconnect(appCtx)
			s.mu.Lock()
			s.manager = nil
			s.mu.Unlock()
			return fmt.Errorf("sandbox runtime setup failed: %w", err)
		}

		wailsruntime.EventsEmit(appCtx, "ssh:connected", nil)
	} else {
		s.mu.Unlock()
	}

	return s.ActivateContainer(containerRegID)
}

// Disconnect closes SSH but keeps the container alive for future reconnection.
func (s *SandboxService) Disconnect() error {
	return s.DisconnectSSH()
}

// DisconnectAndDestroy stops and removes the active container, then closes SSH.
func (s *SandboxService) DisconnectAndDestroy() error {
	s.mu.Lock()
	if s.manager == nil {
		s.mu.Unlock()
		return nil
	}

	mgr := s.manager
	activeRegID := s.activeContainerRegID
	appCtx := s.ctx
	s.manager = nil
	s.activeContainerRegID = ""
	s.mu.Unlock()

	err := mgr.DisconnectAndDestroy(appCtx)

	// Remove from registry
	if activeRegID != "" {
		_ = s.containerStore.Remove(activeRegID)
	}

	return err
}

// GetStatus returns the current sandbox connection status.
func (s *SandboxService) GetStatus() SandboxStatusDTO {
	s.mu.RLock()
	mgr := s.manager
	activeRegID := s.activeContainerRegID
	s.mu.RUnlock()

	if mgr == nil {
		return SandboxStatusDTO{}
	}

	status := SandboxStatusDTO{
		SSHConnected:      mgr.SSHConnected(),
		DockerRunning:     false,
		ContainerID:       "",
		DockerAvailable:   mgr.Docker() != nil,
		ActiveContainerID: activeRegID,
		RuntimeAvailable:  mgr.Runtime() != nil,
		ActiveSandboxID:   activeRegID,
	}

	runtime := mgr.Runtime()
	if runtime != nil {
		status.DockerRunning = runtime.IsRunning()
		status.ContainerID = runtime.ContainerID()
		status.ActiveContainerName = runtime.ContainerName()
		status.SandboxActive = runtime.IsActive()
		status.ActiveSandboxName = runtime.RuntimeName()
	}

	return status
}

// Manager returns the underlying SandboxManager for internal use by other services.
func (s *SandboxService) Manager() *sandbox.SandboxManager {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.manager
}

// ActiveContainerRegID returns the registry ID of the currently connected container.
func (s *SandboxService) ActiveContainerRegID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activeContainerRegID
}

// setupOutputForwarding sets up terminal output forwarding to the frontend.
func (s *SandboxService) setupOutputForwarding() {
	s.mu.RLock()
	mgr := s.manager
	appCtx := s.ctx
	s.mu.RUnlock()

	if mgr == nil {
		return
	}
	if op := mgr.Operator(); op != nil {
		op.SetOnOutput(func(stdout, stderr string, exitCode int) {
			wailsruntime.EventsEmit(appCtx, "terminal:output", TerminalOutputEvent{
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
				s.healthCheck(ctx)
			}
		}
	}()
}

// healthCheck performs a single health check iteration with short-lived locks.
func (s *SandboxService) healthCheck(ctx context.Context) {
	s.mu.RLock()
	mgr := s.manager
	appCtx := s.ctx
	s.mu.RUnlock()

	if mgr == nil || !mgr.SSHConnected() {
		return
	}

	// If there's an active container with an operator, ping through it
	if mgr.HasActiveContainer() {
		op := mgr.Operator()
		if op == nil {
			return
		}
		_, err := op.RunCommand(ctx, []string{"echo", "ping"})
		if err != nil {
			// Connection lost
			s.mu.Lock()
			s.manager = nil
			s.activeContainerRegID = ""
			s.mu.Unlock()
			wailsruntime.EventsEmit(appCtx, "ssh:disconnected", nil)
		}
	} else {
		// SSH-only mode: check SSH is still alive via the SSH client
		ssh := mgr.SSH()
		if ssh == nil {
			return
		}
		_, _, _, err := ssh.RunCommand(ctx, "echo ping")
		if err != nil {
			s.mu.Lock()
			s.manager = nil
			s.mu.Unlock()
			wailsruntime.EventsEmit(appCtx, "ssh:disconnected", nil)
		}
	}
}
