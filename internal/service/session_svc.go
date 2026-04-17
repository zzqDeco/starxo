package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"starxo/internal/model"
	"starxo/internal/storage"
	"starxo/internal/tools"
)

// SessionService manages chat sessions for the frontend.
type SessionService struct {
	ctx                context.Context
	sessionStore       *storage.SessionStore
	containerStore     *storage.ContainerStore
	chatService        *ChatService
	activeSession      *model.Session
	onSessionSwitch    func(containerRegID string)
	onDestroyContainer func(containerRegID string) error
	saveStates         map[string]*sessionSaveState
	mu                 sync.Mutex
}

type sessionSaveState struct {
	inFlight bool
	pending  bool
}

// NewSessionService creates a new SessionService.
func NewSessionService(sessionStore *storage.SessionStore, containerStore *storage.ContainerStore) *SessionService {
	return &SessionService{
		sessionStore:   sessionStore,
		containerStore: containerStore,
		saveStates:     make(map[string]*sessionSaveState),
	}
}

// SetContext stores the Wails application context.
func (s *SessionService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// SetChatService sets the chat service dependency for per-session state access.
func (s *SessionService) SetChatService(cs *ChatService) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chatService = cs
}

// SetOnSessionSwitch registers a callback fired when the active session changes.
// The callback receives the target session's ActiveContainerID (may be empty).
func (s *SessionService) SetOnSessionSwitch(fn func(containerRegID string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onSessionSwitch = fn
}

// SetOnDestroyContainer registers a callback to destroy a container (stop+remove on remote).
func (s *SessionService) SetOnDestroyContainer(fn func(containerRegID string) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onDestroyContainer = fn
}

// BindContainer associates a container with the current session as its active container.
// The container must not already be owned by another session.
func (s *SessionService) BindContainer(containerRegID, workspacePath string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.activeSession == nil {
		return
	}

	// Check ownership: reject if already bound to a different session
	if s.containerStore != nil {
		container, err := s.containerStore.Get(containerRegID)
		if err == nil && container != nil && container.SessionID != "" && container.SessionID != s.activeSession.ID {
			return // owned by another session, refuse to bind
		}
		// Set ownership on the container
		if container != nil {
			container.SessionID = s.activeSession.ID
			_ = s.containerStore.Update(container)
		}
	}

	s.activeSession.AddContainer(containerRegID)
	s.activeSession.ActiveContainerID = containerRegID
	s.activeSession.WorkspacePath = workspacePath
	_ = s.sessionStore.Update(s.activeSession)
}

// GetBoundContainerID returns the active container registry ID bound to the active session.
func (s *SessionService) GetBoundContainerID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeSession == nil {
		return ""
	}
	return s.activeSession.ActiveContainerID
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
// Does NOT cancel any running agent — the old session continues running in the background.
func (s *SessionService) CreateSession(title string) (*model.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Auto-save current session
	if s.activeSession != nil {
		s.saveCurrentLocked()
	}

	if title == "" {
		title = "New Session"
	}

	sess, err := s.sessionStore.Create(title)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Tell ChatService to switch to the new session
	// (its per-session ctxEngine will be auto-created in getOrCreateRun)
	if s.chatService != nil {
		s.chatService.SetActiveSessionID(sess.ID)
	}

	// Clear todo state for the new session
	tools.ClearTodos()

	s.activeSession = sess
	return sess, nil
}

// SwitchSession saves the current session and loads the target one.
// Does NOT cancel any running agent — background sessions continue independently.
func (s *SessionService) SwitchSession(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Don't switch to the same session
	if s.activeSession != nil && s.activeSession.ID == sessionID {
		return nil
	}

	// Save current session
	if s.activeSession != nil {
		s.saveCurrentLocked()
	}

	// Load target session
	sess, err := s.sessionStore.Get(sessionID)
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	// Load session data from disk
	sessionData, err := s.sessionStore.LoadSessionData(sessionID)
	if err != nil {
		return fmt.Errorf("failed to load session data: %w", err)
	}

	// Tell ChatService to switch active session
	if s.chatService != nil {
		s.chatService.SetActiveSessionID(sessionID)

		// Only load from disk if the session is NOT currently running
		// (a running session already has up-to-date state in its SessionRun)
		if !s.chatService.IsSessionRunning(sessionID) {
			s.chatService.restoreNormalizedSessionData(sessionID, sessionData)
		}
	}

	// Clear per-session todo state
	tools.ClearTodos()

	s.activeSession = sess

	// Build enriched session switched event with live state snapshot
	switchEvt := SessionSwitchedEvent{
		Session:     *sess,
		ContainerID: sess.ActiveContainerID,
		Mode:        model.ModeDefault,
	}
	if s.chatService != nil {
		running, currentAgent, mode, interrupt := s.chatService.GetSessionRunSnapshot(sessionID)
		switchEvt.AgentRunning = running
		switchEvt.CurrentAgent = currentAgent
		switchEvt.Mode = mode
		switchEvt.HasInterrupt = interrupt != nil
		switchEvt.Interrupt = interrupt
	}

	// Emit event so frontend can update
	if s.ctx != nil {
		wailsruntime.EventsEmit(s.ctx, "session:switched", switchEvt)
	}

	// Notify listeners (e.g. sandbox auto-reconnect or disconnect)
	if s.onSessionSwitch != nil {
		containerID := sess.ActiveContainerID
		s.mu.Unlock()
		s.onSessionSwitch(containerID)
		s.mu.Lock() // Re-lock so deferred Unlock is balanced
		return nil
	}

	return nil
}

// DeleteSession deletes a session and cascade-destroys all its owned containers.
// If the session has a running agent, it is stopped first.
func (s *SessionService) DeleteSession(sessionID string) error {
	s.mu.Lock()

	if s.activeSession != nil && s.activeSession.ID == sessionID {
		s.mu.Unlock()
		return fmt.Errorf("cannot delete the active session")
	}

	// Stop any running agent in this session
	if s.chatService != nil && s.chatService.IsSessionRunning(sessionID) {
		s.mu.Unlock()
		s.chatService.StopSessionGeneration(sessionID)
		_ = s.chatService.WaitForSessionDone(sessionID, 10*time.Second)
		s.mu.Lock()
	}

	// Clean up ChatService session state
	if s.chatService != nil {
		s.chatService.RemoveSession(sessionID)
	}

	// Load session to get its container list
	sess, err := s.sessionStore.Get(sessionID)
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("failed to load session: %w", err)
	}

	destroyFn := s.onDestroyContainer
	s.mu.Unlock()

	// Cascade destroy all owned containers (best-effort)
	for _, cid := range sess.Containers {
		if destroyFn != nil {
			_ = destroyFn(cid) // best-effort: remote may be unreachable
		}
		// Remove from registry regardless
		_ = s.containerStore.Remove(cid)
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

// LoadSessionData loads the unified session data (messages + display + streaming) for the active session.
// This is the preferred method for frontend session restore.
func (s *SessionService) LoadSessionData() (*model.SessionData, error) {
	s.mu.Lock()
	activeID := ""
	if s.activeSession != nil {
		activeID = s.activeSession.ID
	}
	s.mu.Unlock()

	if activeID == "" {
		return nil, nil
	}
	return s.sessionStore.LoadSessionData(activeID)
}

// SaveCurrentSession persists the current session's conversation to disk.
func (s *SessionService) SaveCurrentSession() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveCurrentLocked()
}

// SaveSessionByID schedules a best-effort asynchronous save for the given session.
// Saves are coalesced per session so at most one save is in-flight and one trailing
// save is queued.
func (s *SessionService) SaveSessionByID(sessionID string) error {
	if sessionID == "" {
		return nil
	}

	s.mu.Lock()
	state := s.saveStates[sessionID]
	if state == nil {
		state = &sessionSaveState{}
		s.saveStates[sessionID] = state
	}
	if state.inFlight {
		state.pending = true
		s.mu.Unlock()
		return nil
	}
	state.inFlight = true
	s.mu.Unlock()

	go s.runCoalescedSave(sessionID)
	return nil
}

func (s *SessionService) runCoalescedSave(sessionID string) {
	for {
		_ = s.saveSessionByIDBlocking(sessionID)

		s.mu.Lock()
		state := s.saveStates[sessionID]
		if state == nil {
			s.mu.Unlock()
			return
		}
		if state.pending {
			state.pending = false
			s.mu.Unlock()
			continue
		}
		state.inFlight = false
		delete(s.saveStates, sessionID)
		s.mu.Unlock()
		return
	}
}

// saveCurrentLocked persists the current session (caller must hold the lock).
func (s *SessionService) saveCurrentLocked() error {
	if s.activeSession == nil || s.chatService == nil {
		return nil
	}

	sessionID := s.activeSession.ID
	return s.saveSessionByIDBlockingLocked(sessionID)
}

func (s *SessionService) saveSessionByIDBlocking(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveSessionByIDBlockingLocked(sessionID)
}

// SaveSessionByIDBlocking persists a session snapshot synchronously.
func (s *SessionService) SaveSessionByIDBlocking(sessionID string) error {
	return s.saveSessionByIDBlocking(sessionID)
}

func (s *SessionService) saveSessionByIDBlockingLocked(sessionID string) error {
	if sessionID == "" || s.chatService == nil {
		return nil
	}

	snapshot, err := s.chatService.ExportSessionSnapshot(sessionID)
	if err != nil {
		return fmt.Errorf("failed to export session snapshot: %w", err)
	}
	if snapshot == nil || snapshot.SessionData == nil {
		return nil
	}
	if s.chatService != nil {
		snapshot.SessionData.DiscoveredTools = s.chatService.PruneDiscoveredToolsForSave(sessionID, snapshot.SessionData.DiscoveredTools)
	}

	if err := s.sessionStore.SaveSessionData(sessionID, snapshot.SessionData); err != nil {
		return fmt.Errorf("failed to save session data: %w", err)
	}
	if s.chatService != nil {
		s.chatService.ReplaceDiscoveredTools(sessionID, snapshot.SessionData.DiscoveredTools)
	}

	var sess *model.Session
	if s.activeSession != nil && s.activeSession.ID == sessionID {
		copySess := *s.activeSession
		sess = &copySess
	} else {
		loaded, err := s.sessionStore.Get(sessionID)
		if err != nil {
			return fmt.Errorf("failed to load session metadata: %w", err)
		}
		sess = loaded
	}

	sess.MessageCount = snapshot.MessageCount
	if err := s.sessionStore.Update(sess); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}
	if s.activeSession != nil && s.activeSession.ID == sessionID {
		s.activeSession.MessageCount = sess.MessageCount
		s.activeSession.UpdatedAt = sess.UpdatedAt
	}

	return nil
}

// EnsureDefaultSession creates a default session if none exist,
// and loads the most recent session into the per-session context engine.
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

	// Restore messages and timeline into per-session state via ChatService
	sessionData, err := s.sessionStore.LoadSessionData(mostRecent.ID)
	if err != nil {
		return nil // non-fatal, just start with empty history
	}
	if s.chatService != nil && sessionData != nil {
		s.chatService.restoreNormalizedSessionData(mostRecent.ID, sessionData)
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

// ListSessionsEnriched returns all sessions with their active container info inlined.
func (s *SessionService) ListSessionsEnriched() ([]EnrichedSession, error) {
	sessions, err := s.sessionStore.List()
	if err != nil {
		return nil, err
	}

	result := make([]EnrichedSession, 0, len(sessions))
	for _, sess := range sessions {
		es := EnrichedSession{Session: sess}
		if sess.ActiveContainerID != "" && s.containerStore != nil {
			container, cerr := s.containerStore.Get(sess.ActiveContainerID)
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
