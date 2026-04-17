package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"starxo/internal/logger"
	"starxo/internal/model"

	"github.com/google/uuid"
)

// SessionStore manages session persistence on disk.
// Sessions are stored as individual directories under ~/.starxo/sessions/{id}/
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
	dir := filepath.Join(homeDir, ".starxo", "sessions")
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
// It handles migration from the old single-containerID format to the new
// containers array format.
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

	// Migrate old format: if raw JSON has "containerID" but session.Containers is empty,
	// migrate the single value into the new array format.
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) == nil {
		if oldID, ok := raw["containerID"]; ok {
			var cid string
			if json.Unmarshal(oldID, &cid) == nil && cid != "" && len(sess.Containers) == 0 {
				sess.Containers = []string{cid}
				sess.ActiveContainerID = cid
				// Persist the migrated format
				_ = s.saveSession(&sess)
			}
		}
	}

	// Ensure Containers is never nil for JSON serialization
	if sess.Containers == nil {
		sess.Containers = []string{}
	}

	return &sess, nil
}

// SaveSessionData atomically writes the unified session data (messages + display + streaming state).
// Uses write-to-tmp + rename for crash safety.
func (s *SessionStore) SaveSessionData(sessionID string, data *model.SessionData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := filepath.Join(s.baseDir, sessionID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("ensure session dir: %w", err)
	}

	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session data: %w", err)
	}

	tmpFile := filepath.Join(dir, "session_data.tmp")
	finalFile := filepath.Join(dir, "session_data.json")

	if err := os.WriteFile(tmpFile, bytes, 0644); err != nil {
		return fmt.Errorf("write tmp file: %w", err)
	}
	if err := os.Rename(tmpFile, finalFile); err != nil {
		// On Windows, Rename may fail if target exists; remove first then retry
		_ = os.Remove(finalFile)
		if err := os.Rename(tmpFile, finalFile); err != nil {
			return fmt.Errorf("atomic rename: %w", err)
		}
	}

	// Also write messages.json for backward compatibility
	msgBytes, err := json.MarshalIndent(data.Messages, "", "  ")
	if err == nil {
		msgPath := filepath.Join(dir, "messages.json")
		_ = os.WriteFile(msgPath, msgBytes, 0644)
	}

	return nil
}

// LoadSessionData reads the unified session data file.
// Falls back to loading messages.json + display.json separately if session_data.json doesn't exist.
func (s *SessionStore) LoadSessionData(sessionID string) (*model.SessionData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Try unified file first
	dataPath := filepath.Join(s.baseDir, sessionID, "session_data.json")
	data, err := os.ReadFile(dataPath)
	if err == nil {
		var sd model.SessionData
		if err := json.Unmarshal(data, &sd); err != nil {
			return nil, fmt.Errorf("unmarshal session data: %w", err)
		}
		return normalizeLoadedSessionData(sessionID, &sd), nil
	}

	// Fallback: reconstruct from legacy files
	sd := &model.SessionData{Version: model.SessionDataVersion}

	// Load messages.json
	msgPath := filepath.Join(s.baseDir, sessionID, "messages.json")
	if msgData, err := os.ReadFile(msgPath); err == nil {
		var msgs []model.PersistedMessage
		if json.Unmarshal(msgData, &msgs) == nil {
			sd.Messages = msgs
		}
	}

	// Load display.json
	dispPath := filepath.Join(s.baseDir, sessionID, "display.json")
	if dispData, err := os.ReadFile(dispPath); err == nil {
		var turns []model.DisplayTurn
		if json.Unmarshal(dispData, &turns) == nil {
			sd.Display = turns
		}
	}

	if sd.Messages == nil && sd.Display == nil {
		return nil, nil
	}
	return normalizeLoadedSessionData(sessionID, sd), nil
}

func normalizeLoadedSessionData(sessionID string, data *model.SessionData) *model.SessionData {
	normalized, warnings := model.NormalizeSessionData(data)
	for _, warning := range warnings {
		logger.Warn("[SESSION_STORE] Normalized session data on load",
			"session", sessionID,
			"warning", warning,
		)
	}
	return normalized
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
