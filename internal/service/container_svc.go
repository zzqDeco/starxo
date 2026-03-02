package service

import (
	"context"
	"fmt"

	"starxo/internal/model"
	"starxo/internal/storage"
)

// ContainerService manages container lifecycle for the frontend.
type ContainerService struct {
	ctx            context.Context
	containerStore *storage.ContainerStore
	sandboxService *SandboxService
	sessionService *SessionService
}

// NewContainerService creates a new ContainerService.
func NewContainerService(containerStore *storage.ContainerStore, sandboxService *SandboxService) *ContainerService {
	return &ContainerService{
		containerStore: containerStore,
		sandboxService: sandboxService,
	}
}

// SetSessionService sets the session service dependency.
func (s *ContainerService) SetSessionService(ss *SessionService) {
	s.sessionService = ss
}

// SetContext stores the Wails application context.
func (s *ContainerService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// ListContainers returns all registered containers.
func (s *ContainerService) ListContainers() ([]model.Container, error) {
	return s.containerStore.List(), nil
}

// RefreshContainerStatus checks the actual status of a container on the remote host
// and updates the registry.
func (s *ContainerService) RefreshContainerStatus(containerRegID string) (*model.Container, error) {
	container, err := s.containerStore.Get(containerRegID)
	if err != nil {
		return nil, fmt.Errorf("container not found: %w", err)
	}

	// We can only check status if there's an active sandbox connection to the same SSH host
	mgr := s.sandboxService.Manager()
	if mgr == nil {
		container.Status = model.ContainerUnknown
		_ = s.containerStore.Update(container)
		return container, nil
	}

	docker := mgr.Docker()
	if docker == nil {
		container.Status = model.ContainerUnknown
		_ = s.containerStore.Update(container)
		return container, nil
	}

	exists, running, err := docker.InspectContainer(s.ctx, container.DockerID)
	if err != nil {
		container.Status = model.ContainerUnknown
	} else if !exists {
		container.Status = model.ContainerDestroyed
	} else if running {
		container.Status = model.ContainerRunning
	} else {
		container.Status = model.ContainerStopped
	}

	_ = s.containerStore.Update(container)
	return container, nil
}

// StopContainer stops a running container without removing it.
func (s *ContainerService) StopContainer(containerRegID string) error {
	container, err := s.containerStore.Get(containerRegID)
	if err != nil {
		return fmt.Errorf("container not found: %w", err)
	}

	// If this is the active container, use the manager to stop it
	if s.sandboxService.ActiveContainerRegID() == containerRegID {
		mgr := s.sandboxService.Manager()
		if mgr != nil {
			if err := mgr.StopContainer(s.ctx); err != nil {
				return err
			}
		}
	}

	container.Status = model.ContainerStopped
	return s.containerStore.Update(container)
}

// StartContainer starts a stopped container via reconnection.
func (s *ContainerService) StartContainer(containerRegID string) error {
	return s.sandboxService.ConnectExisting(containerRegID)
}

// CreateContainer creates a new container on the connected SSH host and activates it.
func (s *ContainerService) CreateContainer() error {
	return s.sandboxService.CreateAndActivateContainer()
}

// ActivateContainer switches the active container to a previously registered one.
func (s *ContainerService) ActivateContainer(containerRegID string) error {
	return s.sandboxService.ActivateContainer(containerRegID)
}

// DeactivateContainer detaches the active container without stopping it.
func (s *ContainerService) DeactivateContainer() error {
	return s.sandboxService.DeactivateContainer()
}

// DestroyContainer stops, removes, and unregisters a container.
// Also removes the container from its owning session's container list.
func (s *ContainerService) DestroyContainer(containerRegID string) error {
	// Look up the container to find its owning session
	container, _ := s.containerStore.Get(containerRegID)

	// If this is the active container, use full disconnect+destroy
	if s.sandboxService.ActiveContainerRegID() == containerRegID {
		if err := s.sandboxService.DisconnectAndDestroy(); err != nil {
			return err
		}
	} else {
		// Otherwise, just remove from registry
		_ = s.containerStore.Remove(containerRegID)
	}

	// Update owning session's container list
	if container != nil && container.SessionID != "" && s.sessionService != nil {
		sess, err := s.sessionService.sessionStore.Get(container.SessionID)
		if err == nil && sess != nil {
			sess.RemoveContainer(containerRegID)
			_ = s.sessionService.sessionStore.Update(sess)
		}
	}

	return nil
}
