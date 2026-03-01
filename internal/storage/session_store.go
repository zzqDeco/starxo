package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"starxo/internal/model"

	"github.com/google/uuid"
)

// SessionStore manages session persistence on disk.
// Sessions are stored as individual directories under ~/.eino-agent/sessions/{id}/
type SessionStore struct {
	baseDir string
	mu      sync.RWMutex
}

// NewSessionStore creates a new SessionStore, ensuring the base directory exists.
func NewSessionStore() (*SessionStore, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(homeDir, ".eino-agent", "sessions")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &SessionStore{baseDir: dir}, nil
}

// List returns all sessions sorted by UpdatedAt descending (most recent first).
func (s *SessionStore) List() ([]model.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, err
	}

	var sessions []model.Session
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sess, err := s.loadSession(entry.Name())
		if err != nil {
			continue // skip corrupt sessions
		}
		sessions = append(sessions, *sess)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt > sessions[j].UpdatedAt
	})
	return sessions, nil
}

// Get returns a session by ID.
func (s *SessionStore) Get(id string) (*model.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loadSession(id)
}

// Create creates a new session with the given title and returns it.
func (s *SessionStore) Create(title string) (*model.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixMilli()
	sess := &model.Session{
		ID:           uuid.New().String()[:8],
		Title:        title,
		CreatedAt:    now,
		UpdatedAt:    now,
		MessageCount: 0,
	}

	dir := filepath.Join(s.baseDir, sess.ID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create session dir: %w", err)
	}

	if err := s.saveSession(sess); err != nil {
		return nil, err
	}

	// Create empty messages file
	msgPath := filepath.Join(dir, "messages.json")
	if err := os.WriteFile(msgPath, []byte("[]"), 0644); err != nil {
		return nil, fmt.Errorf("create messages file: %w", err)
	}

	return sess, nil
}

// Update persists changes to an existing session's metadata.
func (s *SessionStore) Update(sess *model.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess.UpdatedAt = time.Now().UnixMilli()
	return s.saveSession(sess)
}

// Delete removes a session and all its data.
func (s *SessionStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir := filepath.Join(s.baseDir, id)
	return os.RemoveAll(dir)
}

// SaveMessages writes the conversation messages for a session.
func (s *SessionStore) SaveMessages(sessionID string, messages []model.PersistedMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	msgPath := filepath.Join(s.baseDir, sessionID, "messages.json")
	data, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal messages: %w", err)
	}
	return os.WriteFile(msgPath, data, 0644)
}

// LoadMessages reads the conversation messages for a session.
func (s *SessionStore) LoadMessages(sessionID string) ([]model.PersistedMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msgPath := filepath.Join(s.baseDir, sessionID, "messages.json")
	data, err := os.ReadFile(msgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var messages []model.PersistedMessage
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, fmt.Errorf("unmarshal messages: %w", err)
	}
	return messages, nil
}

// SaveDisplayData saves the frontend's rich message display data (with timeline events).
func (s *SessionStore) SaveDisplayData(sessionID string, data string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := filepath.Join(s.baseDir, sessionID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	path := filepath.Join(dir, "display.json")
	return os.WriteFile(path, []byte(data), 0644)
}

// LoadDisplayData loads the frontend's rich message display data.
func (s *SessionStore) LoadDisplayData(sessionID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.baseDir, sessionID, "display.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// loadSession reads a session.json from disk (caller must hold lock).
func (s *SessionStore) loadSession(id string) (*model.Session, error) {
	path := filepath.Join(s.baseDir, id, "session.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var sess model.Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

// saveSession writes a session.json to disk (caller must hold lock).
func (s *SessionStore) saveSession(sess *model.Session) error {
	path := filepath.Join(s.baseDir, sess.ID, "session.json")
	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
