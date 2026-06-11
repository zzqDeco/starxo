package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"starxo/internal/config"
	"starxo/internal/logger"
	"starxo/internal/model"
	"starxo/internal/sandbox"
	"starxo/internal/storage"
)

const (
	sandboxHealthInterval          = 30 * time.Second
	sandboxHealthProbeTimeout      = 5 * time.Second
	sandboxHealthMaxSSHFailures    = 2
	sandboxConnectionLostAgentText = "Sandbox connection was lost; the agent run was stopped."
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
	beforeSandboxActivate  func(containerRegID string) error
	// activeContainerRegID tracks the registry ID of the currently connected container
	activeContainerRegID string
	healthCancel         context.CancelFunc
	healthGeneration     uint64
	healthSSHFailures    int
	healthSSHProbe       func(context.Context, *sandbox.SandboxManager) error
	healthSandboxProbe   func(context.Context, *sandbox.SandboxManager) (bool, error)
	destroySandboxRemote func(context.Context, *sandbox.SandboxManager, string, string) error
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

// SetBeforeSandboxActivation registers a guard that runs before any operation
// rebinds the shared sandbox manager to a different active sandbox.
func (s *SandboxService) SetBeforeSandboxActivation(fn func(containerRegID string) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.beforeSandboxActivate = fn
}

func (s *SandboxService) runBeforeSandboxActivation(containerRegID string) error {
	s.mu.RLock()
	fn := s.beforeSandboxActivate
	s.mu.RUnlock()
	if fn == nil {
		return nil
	}
	return fn(containerRegID)
}

// --- New SSH-independent methods ---

// ConnectSSH establishes SSH connection and ensures the lightweight sandbox
// runtime is available on the remote host.
func (s *SandboxService) ConnectSSH() error {
	if s.Manager() != nil {
		wailsruntime.EventsEmit(s.ctx, "ssh:progress", SandboxProgressEvent{
			Step:    "Cleaning up previous connection...",
			Percent: 0,
		})
		_ = s.disconnectCurrent("reconnecting", func(ctx context.Context, mgr *sandbox.SandboxManager) error {
			return mgr.Disconnect(ctx)
		}, false)
	}

	s.mu.Lock()
	cfg := s.store.Get()
	mgr := sandbox.NewSandboxManager(*cfg)
	s.manager = mgr
	s.healthGeneration++
	generation := s.healthGeneration
	appCtx := s.ctx
	s.mu.Unlock()

	// Step 1: SSH connect (long-running, outside lock)
	if err := mgr.ConnectSSH(appCtx, func(step string, percent int) {
		wailsruntime.EventsEmit(appCtx, "ssh:progress", SandboxProgressEvent{
			Step:    step,
			Percent: percent / 2, // 0-50%
		})
	}); err != nil {
		s.clearManagerIfCurrent(mgr, generation)
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
		s.clearManagerIfCurrent(mgr, generation)
		return fmt.Errorf("sandbox runtime setup failed: %w", err)
	}

	s.startHealthMonitor(generation, appCtx)

	wailsruntime.EventsEmit(appCtx, "ssh:connected", nil)
	return nil
}

// DisconnectSSH closes the SSH connection. Detaches any active container first.
func (s *SandboxService) DisconnectSSH() error {
	return s.disconnectCurrent("manual disconnect", func(ctx context.Context, mgr *sandbox.SandboxManager) error {
		return mgr.Disconnect(ctx)
	}, false)
}

// CreateAndActivateContainer creates a new sandbox on the connected SSH host,
// registers it, and activates it for agent use.
func (s *SandboxService) CreateAndActivateContainer() error {
	s.mu.RLock()
	if s.manager == nil || !s.manager.SSHConnected() {
		s.mu.RUnlock()
		return fmt.Errorf("SSH not connected")
	}
	s.mu.RUnlock()

	if err := s.runBeforeSandboxActivation(""); err != nil {
		return fmt.Errorf("sandbox activation blocked: %w", err)
	}

	s.mu.Lock()
	if s.manager == nil || !s.manager.SSHConnected() {
		s.mu.Unlock()
		return fmt.Errorf("SSH not connected")
	}
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
	container, err := s.containerStore.Get(containerRegID)
	if err != nil {
		return fmt.Errorf("sandbox not found: %w", err)
	}
	if container.Status == model.ContainerUnavailable || container.Runtime == sandbox.RuntimeDocker {
		return fmt.Errorf("sandbox %s is a legacy Docker record and cannot be activated by the dockerless runtime", containerRegID)
	}

	s.mu.RLock()
	sessionSvc := s.sessionService
	s.mu.RUnlock()
	activeSessionID := ""
	if sessionSvc != nil {
		if active := sessionSvc.GetActiveSession(); active != nil {
			activeSessionID = active.ID
		}
	}
	if container.SessionID != "" && container.SessionID != activeSessionID {
		return fmt.Errorf("sandbox %s belongs to another session; switch to that session before activating it", containerRegID)
	}

	// Validate SSH host matches
	cfg := s.store.Get()
	if container.SSHHost != cfg.SSH.Host || container.SSHPort != cfg.SSH.Port {
		return fmt.Errorf("sandbox is on %s:%d but SSH is connected to %s:%d; disconnect and reconnect SSH to the correct host first",
			container.SSHHost, container.SSHPort, cfg.SSH.Host, cfg.SSH.Port)
	}

	s.mu.RLock()
	if s.activeContainerRegID == containerRegID && s.manager != nil && s.manager.SSHConnected() && s.manager.HasActiveContainer() {
		appCtx := s.ctx
		boundCb := s.onContainerBound
		s.mu.RUnlock()
		if boundCb != nil {
			boundCb(containerRegID, container.WorkspacePath)
		}
		wailsruntime.EventsEmit(appCtx, "container:activated", map[string]string{
			"containerID": containerRegID,
		})
		return nil
	}
	s.mu.RUnlock()

	if err := s.runBeforeSandboxActivation(containerRegID); err != nil {
		return fmt.Errorf("sandbox activation blocked: %w", err)
	}

	s.mu.Lock()
	if s.manager == nil || !s.manager.SSHConnected() {
		s.mu.Unlock()
		return fmt.Errorf("SSH not connected")
	}
	mgr := s.manager
	appCtx := s.ctx
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
	s.markActiveSandboxUnavailable("sandbox deactivated")
	return nil
}

func (s *SandboxService) destroyActiveSandbox(containerRegID, runtimeID, workspacePath string) error {
	s.mu.RLock()
	mgr := s.manager
	activeRegID := s.activeContainerRegID
	destroyRemote := s.destroySandboxRemote
	appCtx := s.ctx
	s.mu.RUnlock()

	if mgr == nil {
		return fmt.Errorf("SSH not connected")
	}
	if activeRegID != containerRegID {
		return fmt.Errorf("sandbox %s is not active", containerRegID)
	}
	if runtimeID == "" {
		return fmt.Errorf("sandbox runtime id is empty")
	}
	if destroyRemote == nil {
		if !mgr.SSHConnected() {
			return fmt.Errorf("SSH not connected")
		}
		destroyRemote = func(ctx context.Context, mgr *sandbox.SandboxManager, id, path string) error {
			return mgr.DestroySandbox(ctx, id, path)
		}
	}

	ctx := appCtx
	if ctx == nil {
		ctx = context.Background()
	}
	if err := destroyRemote(ctx, mgr, runtimeID, workspacePath); err != nil {
		return err
	}

	s.mu.Lock()
	if s.manager != mgr || s.activeContainerRegID != containerRegID {
		s.mu.Unlock()
		return nil
	}
	deactivatedCb := s.onContainerDeactivated
	s.activeContainerRegID = ""
	s.healthSSHFailures = 0
	mgr.DetachContainer()
	s.mu.Unlock()

	if deactivatedCb != nil {
		deactivatedCb()
	}
	wailsEmit(appCtx, "container:deactivated", map[string]string{
		"containerID": containerRegID,
		"reason":      "sandbox destroyed",
	})
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
	s.mu.RLock()
	needsSSH := s.manager == nil || !s.manager.SSHConnected()
	s.mu.RUnlock()
	if needsSSH {
		if s.Manager() != nil {
			_ = s.disconnectCurrent("reconnecting", func(ctx context.Context, mgr *sandbox.SandboxManager) error {
				return mgr.Disconnect(ctx)
			}, false)
		}

		cfg := s.store.Get()
		cfg.SSH.Host = container.SSHHost
		cfg.SSH.Port = container.SSHPort
		mgr := sandbox.NewSandboxManager(*cfg)
		s.mu.Lock()
		s.manager = mgr
		s.healthGeneration++
		generation := s.healthGeneration
		appCtx := s.ctx
		s.mu.Unlock()

		// Long-running operations outside lock
		if err := mgr.ConnectSSH(appCtx, func(step string, percent int) {
			wailsruntime.EventsEmit(appCtx, "ssh:progress", SandboxProgressEvent{
				Step:    step,
				Percent: percent / 2,
			})
		}); err != nil {
			s.clearManagerIfCurrent(mgr, generation)
			return fmt.Errorf("SSH connection failed: %w", err)
		}

		if err := mgr.EnsureRuntime(appCtx, func(step string, percent int) {
			wailsruntime.EventsEmit(appCtx, "ssh:progress", SandboxProgressEvent{
				Step:    step,
				Percent: 50 + percent/2,
			})
		}); err != nil {
			_ = mgr.Disconnect(appCtx)
			s.clearManagerIfCurrent(mgr, generation)
			return fmt.Errorf("sandbox runtime setup failed: %w", err)
		}

		s.startHealthMonitor(generation, appCtx)
		wailsruntime.EventsEmit(appCtx, "ssh:connected", nil)
	}

	return s.ActivateContainer(containerRegID)
}

// Disconnect closes SSH but keeps the container alive for future reconnection.
func (s *SandboxService) Disconnect() error {
	return s.DisconnectSSH()
}

// DisconnectAndDestroy stops and removes the active container, then closes SSH.
func (s *SandboxService) DisconnectAndDestroy() error {
	return s.disconnectCurrent("disconnect and destroy", func(ctx context.Context, mgr *sandbox.SandboxManager) error {
		return mgr.DisconnectAndDestroy(ctx)
	}, true)
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

// RunTerminalCommand executes a user-submitted command in the active sandbox workspace.
func (s *SandboxService) RunTerminalCommand(command string) (TerminalCommandResult, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return TerminalCommandResult{}, fmt.Errorf("terminal command is empty")
	}

	s.mu.RLock()
	mgr := s.manager
	appCtx := s.ctx
	eventCtx := s.ctx
	activeContainerID := s.activeContainerRegID
	sessionService := s.sessionService
	s.mu.RUnlock()
	if appCtx == nil {
		appCtx = context.Background()
	}

	if mgr == nil || !mgr.SSHConnected() {
		return TerminalCommandResult{}, fmt.Errorf("SSH not connected")
	}
	if !mgr.HasActiveContainer() {
		return TerminalCommandResult{}, fmt.Errorf("no sandbox is active")
	}
	if sessionService != nil {
		boundContainerID := sessionService.GetBoundContainerID()
		if boundContainerID == "" {
			return TerminalCommandResult{}, fmt.Errorf("please activate a sandbox for this session")
		}
		if activeContainerID != boundContainerID {
			return TerminalCommandResult{}, fmt.Errorf("active sandbox %s does not match session sandbox %s", activeContainerID, boundContainerID)
		}
	}
	op := mgr.Operator()
	if op == nil {
		return TerminalCommandResult{}, fmt.Errorf("sandbox operator is not available")
	}

	output, err := op.RunCommand(appCtx, []string{"sh", "-lc", command})
	if err != nil {
		result := TerminalCommandResult{Command: command, Stderr: err.Error(), ExitCode: -1}
		return result, err
	}
	if output.ExitCode == 0 {
		wailsEmit(eventCtx, "workspace:changed", WorkspaceChangedEvent{
			ContainerID: activeContainerID,
			Source:      "terminal",
			Action:      "command",
			CreatedAt:   time.Now().UnixMilli(),
		})
	}
	return TerminalCommandResult{
		Command:  command,
		Stdout:   output.Stdout,
		Stderr:   output.Stderr,
		ExitCode: output.ExitCode,
	}, nil
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
// whether SSH is still alive. It is kept for compatibility with older callers;
// ConnectSSH starts the generation-scoped monitor used by normal runtime flow.
func (s *SandboxService) StartHealthMonitor(ctx context.Context) {
	s.mu.Lock()
	generation := s.healthGeneration
	s.mu.Unlock()
	s.startHealthMonitor(generation, ctx)
}

func (s *SandboxService) startHealthMonitor(generation uint64, parent context.Context) {
	if parent == nil {
		parent = context.Background()
	}
	s.mu.Lock()
	if generation != s.healthGeneration {
		s.mu.Unlock()
		return
	}
	s.stopHealthMonitorLocked()
	s.healthGeneration = generation
	healthCtx, cancel := context.WithCancel(parent)
	s.healthCancel = cancel
	s.mu.Unlock()

	go func() {
		ticker := time.NewTicker(sandboxHealthInterval)
		defer ticker.Stop()
		for {
			select {
			case <-healthCtx.Done():
				return
			case <-ticker.C:
				s.healthCheck(generation)
			}
		}
	}()
}

func (s *SandboxService) stopHealthMonitorLocked() {
	if s.healthCancel != nil {
		s.healthCancel()
		s.healthCancel = nil
	}
	s.healthSSHFailures = 0
}

// healthCheck performs a single health check iteration with short-lived locks.
func (s *SandboxService) healthCheck(generation uint64) {
	s.mu.RLock()
	mgr := s.manager
	currentGeneration := s.healthGeneration
	activeRegID := s.activeContainerRegID
	sshProbe := s.healthSSHProbe
	sandboxProbe := s.healthSandboxProbe
	s.mu.RUnlock()

	if mgr == nil || generation != currentGeneration {
		return
	}
	if sshProbe == nil {
		sshProbe = defaultHealthSSHProbe
	}
	if sandboxProbe == nil {
		sandboxProbe = defaultHealthSandboxProbe
	}

	probeCtx, cancel := context.WithTimeout(context.Background(), sandboxHealthProbeTimeout)
	err := sshProbe(probeCtx, mgr)
	cancel()
	if err != nil {
		failures := s.recordHealthSSHFailure(generation, mgr, err)
		if failures >= sandboxHealthMaxSSHFailures {
			s.markDisconnectedIfCurrent(generation, mgr, "SSH health check failed")
		}
		return
	}
	s.resetHealthSSHFailures(generation, mgr)

	if activeRegID == "" {
		return
	}
	sandboxCtx, sandboxCancel := context.WithTimeout(context.Background(), sandboxHealthProbeTimeout)
	exists, err := sandboxProbe(sandboxCtx, mgr)
	sandboxCancel()
	if err != nil {
		logger.Warn("[SANDBOX] Active sandbox health probe failed without dropping SSH", "error", err)
		return
	}
	if !exists {
		s.markActiveSandboxUnavailableIfCurrent(generation, mgr, activeRegID, "active sandbox workspace is unavailable")
	}
}

func (s *SandboxService) recordHealthSSHFailure(generation uint64, mgr *sandbox.SandboxManager, err error) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.manager != mgr || s.healthGeneration != generation {
		return 0
	}
	s.healthSSHFailures++
	logger.Warn("[SANDBOX] SSH health probe failed",
		"failures", s.healthSSHFailures,
		"maxFailures", sandboxHealthMaxSSHFailures,
		"error", err,
	)
	return s.healthSSHFailures
}

func (s *SandboxService) resetHealthSSHFailures(generation uint64, mgr *sandbox.SandboxManager) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.manager == mgr && s.healthGeneration == generation {
		s.healthSSHFailures = 0
	}
}

func (s *SandboxService) disconnectCurrent(reason string, disconnect func(context.Context, *sandbox.SandboxManager) error, removeContainer bool) error {
	return s.disconnectCurrentIfCurrent(reason, nil, 0, disconnect, removeContainer)
}

func (s *SandboxService) disconnectCurrentIfCurrent(reason string, expected *sandbox.SandboxManager, generation uint64, disconnect func(context.Context, *sandbox.SandboxManager) error, removeContainer bool) error {
	s.mu.Lock()
	mgr := s.manager
	if expected != nil && (mgr != expected || s.healthGeneration != generation) {
		s.mu.Unlock()
		return nil
	}
	if mgr == nil {
		s.stopHealthMonitorLocked()
		s.healthGeneration++
		s.mu.Unlock()
		return nil
	}
	activeRegID := s.activeContainerRegID
	deactivatedCb := s.onContainerDeactivated
	appCtx := s.ctx
	s.stopHealthMonitorLocked()
	s.healthGeneration++
	s.manager = nil
	s.activeContainerRegID = ""
	s.mu.Unlock()

	ctx := appCtx
	if ctx == nil {
		ctx = context.Background()
	}
	var err error
	if disconnect != nil {
		err = disconnect(ctx, mgr)
	}
	if activeRegID != "" {
		if removeContainer && s.containerStore != nil {
			_ = s.containerStore.Remove(activeRegID)
		}
		if deactivatedCb != nil {
			deactivatedCb()
		}
		wailsEmit(appCtx, "container:deactivated", map[string]string{"reason": reason})
	}
	wailsEmit(appCtx, "ssh:disconnected", map[string]string{"reason": reason})
	return err
}

func (s *SandboxService) markDisconnectedIfCurrent(generation uint64, mgr *sandbox.SandboxManager, reason string) {
	_ = s.disconnectCurrentIfCurrent(reason, mgr, generation, func(ctx context.Context, mgr *sandbox.SandboxManager) error {
		return mgr.Disconnect(ctx)
	}, false)
}

func (s *SandboxService) clearManagerIfCurrent(mgr *sandbox.SandboxManager, generation uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.manager != mgr || s.healthGeneration != generation {
		return
	}
	s.stopHealthMonitorLocked()
	s.manager = nil
	s.activeContainerRegID = ""
	s.healthGeneration++
}

func (s *SandboxService) markActiveSandboxUnavailable(reason string) {
	s.mu.RLock()
	mgr := s.manager
	generation := s.healthGeneration
	activeRegID := s.activeContainerRegID
	s.mu.RUnlock()
	s.markActiveSandboxUnavailableIfCurrent(generation, mgr, activeRegID, reason)
}

func (s *SandboxService) markActiveSandboxUnavailableIfCurrent(generation uint64, mgr *sandbox.SandboxManager, expectedActiveRegID, reason string) {
	s.mu.Lock()
	if mgr == nil || s.manager != mgr || s.healthGeneration != generation || s.activeContainerRegID == "" {
		s.mu.Unlock()
		return
	}
	if expectedActiveRegID != "" && s.activeContainerRegID != expectedActiveRegID {
		s.mu.Unlock()
		return
	}
	activeRegID := s.activeContainerRegID
	deactivatedCb := s.onContainerDeactivated
	appCtx := s.ctx
	s.activeContainerRegID = ""
	s.healthSSHFailures = 0
	mgr.DetachContainer()
	s.mu.Unlock()

	if deactivatedCb != nil {
		deactivatedCb()
	}
	wailsEmit(appCtx, "container:deactivated", map[string]string{
		"containerID": activeRegID,
		"reason":      reason,
	})
}

func defaultHealthSSHProbe(ctx context.Context, mgr *sandbox.SandboxManager) error {
	if mgr == nil {
		return fmt.Errorf("sandbox manager is not available")
	}
	ssh := mgr.SSH()
	if ssh == nil {
		return fmt.Errorf("SSH client is not available")
	}
	_, stderr, exitCode, err := ssh.RunCommand(ctx, "echo ping")
	if err != nil {
		return err
	}
	if exitCode != 0 {
		return fmt.Errorf("SSH health check failed (exit %d): %s", exitCode, strings.TrimSpace(stderr))
	}
	return nil
}

func defaultHealthSandboxProbe(ctx context.Context, mgr *sandbox.SandboxManager) (bool, error) {
	if mgr == nil {
		return false, fmt.Errorf("sandbox manager is not available")
	}
	runtime := mgr.Runtime()
	if runtime == nil {
		return false, nil
	}
	runtimeID := runtime.RuntimeID()
	workspacePath := runtime.WorkspacePath()
	if runtimeID == "" || workspacePath == "" {
		return false, nil
	}
	exists, _, err := runtime.InspectSandbox(ctx, runtimeID, workspacePath)
	return exists, err
}
