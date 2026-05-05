package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"starxo/internal/model"
)

// ContainerStore manages the container registry on disk.
// All containers are stored in a single ~/.starxo/containers.json file.
type ContainerStore struct {
	path       string
	legacyPath string
	containers []model.Container
	mu         sync.RWMutex
}

// NewContainerStore creates a new ContainerStore, loading existing data if present.
func NewContainerStore() (*ContainerStore, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(homeDir, ".starxo")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	s := &ContainerStore{
		path:       filepath.Join(dir, "sandboxes.json"),
		legacyPath: filepath.Join(dir, "containers.json"),
	}
	if err := s.load(); err != nil {
		s.containers = []model.Container{}
	}
	return s, nil
}

// List returns all registered containers.
func (s *ContainerStore) List() []model.Container {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]model.Container, len(s.containers))
	copy(result, s.containers)
	return result
}

// Get returns a container by its registry ID.
func (s *ContainerStore) Get(id string) (*model.Container, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.containers {
		if s.containers[i].ID == id {
			c := s.containers[i]
			return &c, nil
		}
	}
	return nil, fmt.Errorf("container %s not found", id)
}

// Add registers a new container and persists to disk.
func (s *ContainerStore) Add(container *model.Container) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.containers = append(s.containers, *container)
	return s.save()
}

// Update modifies an existing container entry and persists to disk.
func (s *ContainerStore) Update(container *model.Container) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.containers {
		if s.containers[i].ID == container.ID {
			s.containers[i] = *container
			return s.save()
		}
	}
	return fmt.Errorf("container %s not found", container.ID)
}

// Remove deletes a container from the registry.
func (s *ContainerStore) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.containers {
		if s.containers[i].ID == id {
			s.containers = append(s.containers[:i], s.containers[i+1:]...)
			return s.save()
		}
	}
	return fmt.Errorf("container %s not found", id)
}

// FindBySSH returns all containers associated with a given SSH host:port.
func (s *ContainerStore) FindBySSH(host string, port int) []model.Container {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []model.Container
	for _, c := range s.containers {
		if c.SSHHost == host && c.SSHPort == port {
			result = append(result, c)
		}
	}
	return result
}

// FindBySessionID returns all containers owned by the given session.
func (s *ContainerStore) FindBySessionID(sessionID string) []model.Container {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []model.Container
	for _, c := range s.containers {
		if c.SessionID == sessionID {
			result = append(result, c)
		}
	}
	return result
}

// RegisteredDockerIDs returns all Docker container IDs in the registry.
// Used by setup.go to avoid cleaning up registered containers.
func (s *ContainerStore) RegisteredDockerIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.containers))
	for _, c := range s.containers {
		if c.RuntimeID != "" && c.Status != model.ContainerDestroyed {
			ids = append(ids, c.RuntimeID)
		} else if c.DockerID != "" && c.Status != model.ContainerDestroyed {
			ids = append(ids, c.DockerID)
		}
	}
	return ids
}

// load reads containers.json from disk (caller must not hold lock).
func (s *ContainerStore) load() error {
	data, err := os.ReadFile(s.path)
	if err == nil {
		if err := json.Unmarshal(data, &s.containers); err != nil {
			return err
		}
		s.normalize()
		return nil
	}

	legacyData, legacyErr := os.ReadFile(s.legacyPath)
	if legacyErr != nil {
		return err
	}
	if err := json.Unmarshal(legacyData, &s.containers); err != nil {
		return err
	}
	for i := range s.containers {
		if s.containers[i].RuntimeID == "" {
			s.containers[i].RuntimeID = s.containers[i].DockerID
		}
		if s.containers[i].Runtime == "" {
			s.containers[i].Runtime = "docker"
		}
		if s.containers[i].Status != model.ContainerDestroyed {
			s.containers[i].Status = model.ContainerUnavailable
		}
	}
	s.normalize()
	return s.save()
}

// save writes containers.json to disk (caller must hold write lock).
func (s *ContainerStore) save() error {
	data, err := json.MarshalIndent(s.containers, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

func (s *ContainerStore) normalize() {
	for i := range s.containers {
		c := &s.containers[i]
		if c.RuntimeID == "" {
			c.RuntimeID = c.DockerID
		}
		if c.DockerID == "" {
			c.DockerID = c.RuntimeID
		}
		if c.Runtime == "" {
			c.Runtime = "bwrap"
		}
	}
}
