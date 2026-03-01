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
}

// NewContainerService creates a new ContainerService.
func NewContainerService(containerStore *storage.ContainerStore, sandboxService *SandboxService) *ContainerService {
	return &ContainerService{
		containerStore: containerStore,
		sandboxService: sandboxService,
	}
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

// DestroyContainer stops, removes, and unregisters a container.
func (s *ContainerService) DestroyContainer(containerRegID string) error {
	// If this is the active container, use full disconnect+destroy
	if s.sandboxService.ActiveContainerRegID() == containerRegID {
		return s.sandboxService.DisconnectAndDestroy()
	}

	// Otherwise, we need an SSH connection to the same host to remove it
	// For now, just remove from registry
	return s.containerStore.Remove(containerRegID)
}
