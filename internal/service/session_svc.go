package service

import (
	"context"
	"fmt"
	"sync"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	agentctx "starxo/internal/context"
	"starxo/internal/model"
	"starxo/internal/storage"
)

// SessionService manages chat sessions for the frontend.
type SessionService struct {
	ctx             context.Context
	sessionStore    *storage.SessionStore
	containerStore  *storage.ContainerStore
	ctxEngine       *agentctx.Engine
	activeSession   *model.Session
	onSessionSwitch func(containerRegID string)
	mu              sync.Mutex
}

// NewSessionService creates a new SessionService.
func NewSessionService(sessionStore *storage.SessionStore, containerStore *storage.ContainerStore) *SessionService {
	return &SessionService{
		sessionStore:   sessionStore,
		containerStore: containerStore,
	}
}

// SetContext stores the Wails application context.
func (s *SessionService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// SetCtxEngine sets the context engine dependency.
func (s *SessionService) SetCtxEngine(engine *agentctx.Engine) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ctxEngine = engine
}

// SetOnSessionSwitch registers a callback fired when the active session changes.
// The callback receives the target session's ContainerID (may be empty).
func (s *SessionService) SetOnSessionSwitch(fn func(containerRegID string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onSessionSwitch = fn
}

// BindContainer associates the current session with a container registry ID and workspace path.
func (s *SessionService) BindContainer(containerRegID, workspacePath string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.activeSession == nil {
		return
	}

	s.activeSession.ContainerID = containerRegID
	s.activeSession.WorkspacePath = workspacePath
	_ = s.sessionStore.Update(s.activeSession)
}

// GetBoundContainerID returns the container registry ID bound to the active session.
func (s *SessionService) GetBoundContainerID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeSession == nil {
		return ""
	}
	return s.activeSession.ContainerID
}

// GetWorkspacePath returns the workspace path for the active session.
func (s *SessionService) GetWorkspacePath() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeSession == nil {
		return "/workspace"
	}
	if s.activeSession.WorkspacePath == "" {
		return "/workspace"
	}
	return s.activeSession.WorkspacePath
}

// ListSessions returns all sessions sorted by most recent first.
func (s *SessionService) ListSessions() ([]model.Session, error) {
	return s.sessionStore.List()
}

// CreateSession creates a new session, saves the current one first if it exists.
func (s *SessionService) CreateSession(title string) (*model.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Auto-save current session
	if s.activeSession != nil && s.ctxEngine != nil {
		s.saveCurrentLocked()
	}

	if title == "" {
		title = "New Session"
	}

	sess, err := s.sessionStore.Create(title)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Clear context engine for new session
	if s.ctxEngine != nil {
		s.ctxEngine.ClearHistory()
	}

	s.activeSession = sess
	return sess, nil
}

// SwitchSession saves the current session and loads the target one.
func (s *SessionService) SwitchSession(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Don't switch to the same session
	if s.activeSession != nil && s.activeSession.ID == sessionID {
		return nil
	}

	// Save current session
	if s.activeSession != nil && s.ctxEngine != nil {
		s.saveCurrentLocked()
	}

	// Load target session
	sess, err := s.sessionStore.Get(sessionID)
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	// Load messages
	messages, err := s.sessionStore.LoadMessages(sessionID)
	if err != nil {
		return fmt.Errorf("failed to load messages: %w", err)
	}

	// Restore into context engine
	if s.ctxEngine != nil {
		if messages != nil {
			s.ctxEngine.ImportMessages(messages)
		} else {
			s.ctxEngine.ClearHistory()
		}
	}

	s.activeSession = sess

	// Emit event so frontend can update
	if s.ctx != nil {
		wailsruntime.EventsEmit(s.ctx, "session:switched", SessionSwitchedEvent{
			Session:     *sess,
			ContainerID: sess.ContainerID,
		})
	}

	// Notify listeners (e.g. sandbox auto-reconnect)
	if s.onSessionSwitch != nil {
		containerID := sess.ContainerID
		s.mu.Unlock()
		s.onSessionSwitch(containerID)
		s.mu.Lock() // Re-lock so deferred Unlock is balanced
		return nil
	}

	return nil
}

// DeleteSession deletes a session. Cannot delete the active session.
func (s *SessionService) DeleteSession(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.activeSession != nil && s.activeSession.ID == sessionID {
		return fmt.Errorf("cannot delete the active session")
	}

	return s.sessionStore.Delete(sessionID)
}

// RenameSession renames a session.
func (s *SessionService) RenameSession(sessionID, title string) error {
	sess, err := s.sessionStore.Get(sessionID)
	if err != nil {
		return err
	}
	sess.Title = title
	if err := s.sessionStore.Update(sess); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeSession != nil && s.activeSession.ID == sessionID {
		s.activeSession.Title = title
	}
	return nil
}

// GetActiveSession returns the currently active session.
func (s *SessionService) GetActiveSession() *model.Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeSession == nil {
		return nil
	}
	sess := *s.activeSession
	return &sess
}

// GetActiveSessionMessages returns the messages for the active session from disk.
// Used by the frontend to restore messages after session switch.
func (s *SessionService) GetActiveSessionMessages() ([]model.PersistedMessage, error) {
	s.mu.Lock()
	activeID := ""
	if s.activeSession != nil {
		activeID = s.activeSession.ID
	}
	s.mu.Unlock()

	if activeID == "" {
		return nil, nil
	}

	return s.sessionStore.LoadMessages(activeID)
}

// SaveChatDisplay saves the frontend's rich display messages (with timeline events) for the active session.
func (s *SessionService) SaveChatDisplay(data string) error {
	s.mu.Lock()
	activeID := ""
	if s.activeSession != nil {
		activeID = s.activeSession.ID
	}
	s.mu.Unlock()

	if activeID == "" {
		return nil
	}
	return s.sessionStore.SaveDisplayData(activeID, data)
}

// LoadChatDisplay loads the frontend's rich display messages for the active session.
func (s *SessionService) LoadChatDisplay() (string, error) {
	s.mu.Lock()
	activeID := ""
	if s.activeSession != nil {
		activeID = s.activeSession.ID
	}
	s.mu.Unlock()

	if activeID == "" {
		return "", nil
	}
	return s.sessionStore.LoadDisplayData(activeID)
}

// SaveCurrentSession persists the current session's conversation to disk.
func (s *SessionService) SaveCurrentSession() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveCurrentLocked()
}

// saveCurrentLocked persists the current session (caller must hold the lock).
func (s *SessionService) saveCurrentLocked() error {
	if s.activeSession == nil || s.ctxEngine == nil {
		return nil
	}

	messages := s.ctxEngine.ExportMessages()
	if err := s.sessionStore.SaveMessages(s.activeSession.ID, messages); err != nil {
		return fmt.Errorf("failed to save messages: %w", err)
	}

	// Update session metadata
	s.activeSession.MessageCount = s.ctxEngine.MessageCount()
	if err := s.sessionStore.Update(s.activeSession); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

// EnsureDefaultSession creates a default session if none exist,
// and loads the most recent session into the context engine.
func (s *SessionService) EnsureDefaultSession() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessions, err := s.sessionStore.List()
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		// Create a default session
		sess, err := s.sessionStore.Create("Default Session")
		if err != nil {
			return fmt.Errorf("failed to create default session: %w", err)
		}
		s.activeSession = sess
		return nil
	}

	// Load the most recent session (list is sorted by UpdatedAt desc)
	mostRecent := sessions[0]
	s.activeSession = &mostRecent

	// Restore messages
	messages, err := s.sessionStore.LoadMessages(mostRecent.ID)
	if err != nil {
		return nil // non-fatal, just start with empty history
	}
	if s.ctxEngine != nil && messages != nil {
		s.ctxEngine.ImportMessages(messages)
	}

	return nil
}

// EnrichedSession extends Session with live container info for the frontend.
type EnrichedSession struct {
	model.Session
	ContainerStatus string `json:"containerStatus"`
	ContainerName   string `json:"containerName"`
	ContainerSSH    string `json:"containerSSH"`
}

// ListSessionsEnriched returns all sessions with their container info inlined.
func (s *SessionService) ListSessionsEnriched() ([]EnrichedSession, error) {
	sessions, err := s.sessionStore.List()
	if err != nil {
		return nil, err
	}

	result := make([]EnrichedSession, 0, len(sessions))
	for _, sess := range sessions {
		es := EnrichedSession{Session: sess}
		if sess.ContainerID != "" && s.containerStore != nil {
			container, cerr := s.containerStore.Get(sess.ContainerID)
			if cerr == nil && container != nil {
				es.ContainerStatus = string(container.Status)
				es.ContainerName = container.Name
				if container.SSHHost != "" {
					es.ContainerSSH = fmt.Sprintf("%s:%d", container.SSHHost, container.SSHPort)
				}
			}
		}
		result = append(result, es)
	}

	return result, nil
}
