package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Store struct {
	path   string
	config *AppConfig
	mu     sync.RWMutex
}

func NewStore() (*Store, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(homeDir, ".starxo")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	s := &Store{
		path: filepath.Join(dir, "config.json"),
	}
	if err := s.Load(); err != nil {
		s.config = DefaultConfig()
	}
	return s, nil
}

func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return err
	}
	s.config = cfg
	return nil
}

func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, err := json.MarshalIndent(s.config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

func (s *Store) Get() *AppConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg := *s.config
	return &cfg
}

func (s *Store) Update(fn func(*AppConfig)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(s.config)
	data, err := json.MarshalIndent(s.config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}
