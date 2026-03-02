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

// SetOnConnect registers a callback that fires after a successful sandbox connection.
func (s *SandboxService) SetOnConnect(fn func(mgr *sandbox.SandboxManager)) {
	s.onConnect = fn
}

// SetOnContainerBound registers a callback that fires after a container is connected,
// passing the registry ID and workspace path so they can be bound to the active session.
func (s *SandboxService) SetOnContainerBound(fn func(containerRegID, workspacePath string)) {
	s.onContainerBound = fn
}

// Connect creates a new container and connects to it.
// Registers the new container in the container store.
func (s *SandboxService) Connect() error {
	// Disconnect existing connection (keep container alive)
	if s.manager != nil {
		wailsruntime.EventsEmit(s.ctx, "sandbox:progress", SandboxProgressEvent{
			Step:    "Cleaning up previous connection...",
			Percent: 0,
		})
		_ = s.manager.Disconnect(s.ctx)
		s.manager = nil
	}

	cfg := s.store.Get()
	s.manager = sandbox.NewSandboxManager(*cfg)

	// Get registered Docker IDs to exclude from cleanup
	excludeIDs := s.containerStore.RegisteredDockerIDs()

	progressCallback := func(step string, percent int) {
		wailsruntime.EventsEmit(s.ctx, "sandbox:progress", SandboxProgressEvent{
			Step:    step,
			Percent: percent,
		})
	}

	if err := s.manager.ConnectWithExclusions(s.ctx, progressCallback, excludeIDs); err != nil {
		s.manager = nil
		return fmt.Errorf("sandbox connection failed: %w", err)
	}

	// Register the new container
	docker := s.manager.Docker()
	if docker != nil {
		regID := uuid.New().String()[:8]
		now := time.Now().UnixMilli()

		// Determine owning session ID
		sessionID := ""
		if s.sessionService != nil {
			if active := s.sessionService.GetActiveSession(); active != nil {
				sessionID = active.ID
			}
		}

		container := &model.Container{
			ID:            regID,
			DockerID:      docker.ContainerID(),
			Name:          docker.ContainerName(),
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
	}

	s.setupOutputForwarding()

	if s.onConnect != nil {
		s.onConnect(s.manager)
	}

	// Notify session binding
	if s.onContainerBound != nil && s.activeContainerRegID != "" {
		s.onContainerBound(s.activeContainerRegID, "/workspace")
	}

	wailsruntime.EventsEmit(s.ctx, "sandbox:ready", nil)
	return nil
}

// ConnectExisting reconnects to a previously registered container.
func (s *SandboxService) ConnectExisting(containerRegID string) error {
	container, err := s.containerStore.Get(containerRegID)
	if err != nil {
		return fmt.Errorf("container not found: %w", err)
	}

	// Disconnect existing connection (keep container alive)
	if s.manager != nil {
		_ = s.manager.Disconnect(s.ctx)
		s.manager = nil
	}

	cfg := s.store.Get()
	// Override SSH config with the container's stored connection info
	// to avoid using potentially changed global settings
	cfg.SSH.Host = container.SSHHost
	cfg.SSH.Port = container.SSHPort
	s.manager = sandbox.NewSandboxManager(*cfg)

	progressCallback := func(step string, percent int) {
		wailsruntime.EventsEmit(s.ctx, "sandbox:progress", SandboxProgressEvent{
			Step:    step,
			Percent: percent,
		})
	}

	if err := s.manager.Reconnect(s.ctx, container.DockerID, container.Name, progressCallback); err != nil {
		s.manager = nil
		return fmt.Errorf("reconnect failed: %w", err)
	}

	// Update container status
	container.Status = model.ContainerRunning
	container.LastUsedAt = time.Now().UnixMilli()
	_ = s.containerStore.Update(container)
	s.activeContainerRegID = containerRegID

	s.setupOutputForwarding()

	if s.onConnect != nil {
		s.onConnect(s.manager)
	}

	// Notify session binding
	if s.onContainerBound != nil && s.activeContainerRegID != "" {
		s.onContainerBound(s.activeContainerRegID, "/workspace")
	}

	wailsruntime.EventsEmit(s.ctx, "sandbox:ready", nil)
	return nil
}

// Disconnect closes SSH but keeps the container alive for future reconnection.
func (s *SandboxService) Disconnect() error {
	if s.manager == nil {
		return nil
	}

	err := s.manager.Disconnect(s.ctx)
	s.manager = nil
	return err
}

// DisconnectAndDestroy stops and removes the container, then closes SSH.
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
		SSHConnected:  s.manager.IsConnected(),
		DockerRunning: false,
		ContainerID:   "",
	}

	docker := s.manager.Docker()
	if docker != nil {
		status.DockerRunning = docker.IsRunning()
		status.ContainerID = docker.ContainerID()
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
// whether the connected sandbox is still alive. If it detects a disconnect it
// emits a "sandbox:disconnected" event so the frontend can react.
func (s *SandboxService) StartHealthMonitor(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if s.manager == nil || !s.manager.IsConnected() {
					continue
				}
				// Quick health check: run a trivial command
				op := s.manager.Operator()
				if op == nil {
					continue
				}
				_, err := op.RunCommand(ctx, []string{"echo", "ping"})
				if err != nil {
					// Connection lost
					wailsruntime.EventsEmit(s.ctx, "sandbox:disconnected", nil)
					s.manager = nil
				}
			}
		}
	}()
}
