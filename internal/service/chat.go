package service

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/adk"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"starxo/internal/agent"
	"starxo/internal/config"
	agentctx "starxo/internal/context"
	"starxo/internal/llm"
	"starxo/internal/logger"
	"starxo/internal/model"
	"starxo/internal/sandbox"
	checkpoint "starxo/internal/store"
	"starxo/internal/tools"
)

// Context key for propagating session identity through agent execution.
type contextKey string

const sessionIDCtxKey contextKey = "sessionID"

func contextWithSessionID(ctx context.Context, sessionID string) context.Context {
	ctx = context.WithValue(ctx, sessionIDCtxKey, sessionID)
	// Also store a plain-string key so lower-level internal packages can read
	// session scope without importing service package types (avoids import cycles).
	return context.WithValue(ctx, "sessionID", sessionID)
}

// SessionIDFromContext extracts the session ID from a context.
func SessionIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(sessionIDCtxKey).(string); ok {
		return v
	}
	return ""
}

// Default context engine parameters.
const (
	defaultSystemPrompt = "You are an intelligent coding agent that helps users write, debug, and execute code in a sandboxed environment. You have access to tools for file operations, shell commands, and code execution. Always explain your approach before taking action."
	defaultMaxTokens    = 8000
)

// PendingInterrupt holds the state needed to resume after an interrupt.
type PendingInterrupt struct {
	CheckpointID string
	InterruptID  string
	Info         any
}

// SessionRun holds per-session agent execution state.
// Each session gets its own context engine, timeline, and run lifecycle.
type SessionRun struct {
	sessionID string
	ctxEngine *agentctx.Engine
	timeline  *agentctx.TimelineCollector

	// Run lifecycle
	running          bool
	cancelFn         context.CancelFunc
	runDone          chan struct{}
	pendingInterrupt *PendingInterrupt
	streamingState   *model.StreamingState
	mode             string // "default" or "plan"
	currentAgent     string
}

// ChatService manages chat interactions between the frontend and the AI agent.
type ChatService struct {
	ctx context.Context

	// Shared across all sessions (concurrent-safe, stateless)
	deepAgent       adk.Agent
	defaultRunner   *adk.Runner
	planRunner      *adk.Runner
	mcpHandles      []*tools.MCPServerHandle
	retiredMCPs     []*tools.MCPServerHandle
	sandbox         *sandbox.SandboxManager
	store           *config.Store
	checkpointStore compose.CheckPointStore

	// Per-session execution state
	sessions        map[string]*SessionRun
	activeSessionID string

	// Service deps
	sessionService *SessionService
	onAgentDone    func(sessionID string)

	mu sync.Mutex
}

// NewChatService creates a new ChatService.
func NewChatService(store *config.Store) *ChatService {
	return &ChatService{
		store:           store,
		checkpointStore: checkpoint.NewInMemoryStore(),
		sessions:        make(map[string]*SessionRun),
	}
}

// SetContext stores the Wails application context. Called from app.go startup.
func (s *ChatService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// SetDependencies injects the sandbox manager.
// The ctxEngine parameter is accepted for backward compatibility but ignored;
// per-session context engines are managed inside SessionRun.
func (s *ChatService) SetDependencies(sbx *sandbox.SandboxManager, _ *agentctx.Engine) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sandbox = sbx
}

// UpdateSandbox updates the sandbox manager reference.
func (s *ChatService) UpdateSandbox(sbx *sandbox.SandboxManager) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sandbox = sbx
	s.invalidateRunners()
}

// InvalidateRunner forces runners to be rebuilt on the next message.
func (s *ChatService) InvalidateRunner() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.invalidateRunners()
}

func (s *ChatService) invalidateRunners() {
	s.retireMCPHandlesLocked(s.mcpHandles)
	s.mcpHandles = nil
	s.deepAgent = nil
	s.defaultRunner = nil
	s.planRunner = nil
}

// SetOnAgentDone registers a callback that fires after the agent finishes processing.
// The callback receives the sessionID of the completed run.
func (s *ChatService) SetOnAgentDone(fn func(sessionID string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onAgentDone = fn
}

// SetSessionService injects the session service.
func (s *ChatService) SetSessionService(ss *SessionService) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionService = ss
}

// ---------------------------------------------------------------------------
// Per-session state management
// ---------------------------------------------------------------------------

// getOrCreateRun returns the SessionRun for the given session.
// Creates a new one if it doesn't exist. Caller must hold s.mu.
func (s *ChatService) getOrCreateRun(sessionID string) *SessionRun {
	if run, ok := s.sessions[sessionID]; ok {
		return run
	}
	run := &SessionRun{
		sessionID: sessionID,
		ctxEngine: agentctx.NewEngine(defaultSystemPrompt, defaultMaxTokens),
		timeline:  agentctx.NewTimelineCollector(),
		mode:      "default",
	}
	s.sessions[sessionID] = run
	return run
}

func (s *ChatService) anySessionRunningLocked() bool {
	for _, run := range s.sessions {
		if run.running {
			return true
		}
	}
	return false
}

func (s *ChatService) retireMCPHandlesLocked(handles []*tools.MCPServerHandle) {
	if len(handles) == 0 {
		return
	}
	if s.anySessionRunningLocked() {
		s.retiredMCPs = append(s.retiredMCPs, handles...)
		return
	}
	s.closeMCPHandlesLocked(handles)
}

func (s *ChatService) closeMCPHandlesLocked(handles []*tools.MCPServerHandle) {
	for _, handle := range handles {
		if handle == nil {
			continue
		}
		if err := handle.Close(); err != nil {
			logger.Warn("[CHAT] Failed to close MCP handle", "server", handle.Name, "error", err)
		}
	}
}

func (s *ChatService) cleanupRetiredMCPHandlesLocked() {
	if s.anySessionRunningLocked() || len(s.retiredMCPs) == 0 {
		return
	}
	s.closeMCPHandlesLocked(s.retiredMCPs)
	s.retiredMCPs = nil
}

// activeRun returns the SessionRun for the currently active session.
// Returns nil if no active session is set. Caller must hold s.mu.
func (s *ChatService) activeRun() *SessionRun {
	if s.activeSessionID == "" {
		return nil
	}
	return s.getOrCreateRun(s.activeSessionID)
}

// SetActiveSessionID sets the currently active session.
func (s *ChatService) SetActiveSessionID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeSessionID = id
}

// GetActiveSessionID returns the currently active session ID.
func (s *ChatService) GetActiveSessionID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activeSessionID
}

// GetOrCreateRun returns the SessionRun for the given session (for SessionService).
func (s *ChatService) GetOrCreateRun(sessionID string) *SessionRun {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getOrCreateRun(sessionID)
}

// RemoveSession removes a session's run state from memory.
func (s *ChatService) RemoveSession(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
}

// ---------------------------------------------------------------------------
// Mode management
// ---------------------------------------------------------------------------

// SetMode switches the active session between "default" and "plan" mode.
func (s *ChatService) SetMode(mode string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if mode != "default" && mode != "plan" {
		return fmt.Errorf("invalid mode: %s (must be 'default' or 'plan')", mode)
	}

	run := s.activeRun()
	if run == nil {
		return fmt.Errorf("no active session")
	}

	run.mode = mode
	logger.Info("[CHAT] Mode changed", "mode", mode, "session", s.activeSessionID)
	wailsruntime.EventsEmit(s.ctx, "agent:mode_changed", ModeChangedEvent{
		Mode:      mode,
		SessionID: s.activeSessionID,
	})
	return nil
}

// GetMode returns the current agent mode for the active session.
func (s *ChatService) GetMode() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	run := s.activeRun()
	if run == nil {
		return "default"
	}
	return run.mode
}

// ---------------------------------------------------------------------------
// Run status
// ---------------------------------------------------------------------------

// IsRunning returns whether the active session has a running agent.
func (s *ChatService) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	run := s.activeRun()
	return run != nil && run.running
}

// IsSessionRunning returns whether a specific session has a running agent.
func (s *ChatService) IsSessionRunning(sessionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.sessions[sessionID]
	return ok && run.running
}

// WaitForSessionDone waits for a specific session's agent run to complete.
func (s *ChatService) WaitForSessionDone(sessionID string, timeout time.Duration) error {
	s.mu.Lock()
	run, ok := s.sessions[sessionID]
	if !ok || !run.running {
		s.mu.Unlock()
		return nil
	}
	done := run.runDone
	s.mu.Unlock()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timed out waiting for session %s", sessionID)
	}
}

// ---------------------------------------------------------------------------
// SendMessage
// ---------------------------------------------------------------------------

// SendMessage processes a user message through the agent and streams results to the frontend.
func (s *ChatService) SendMessage(userMessage string) error {
	s.mu.Lock()

	if s.activeSessionID == "" {
		s.mu.Unlock()
		return fmt.Errorf("no active session")
	}

	run := s.activeRun()

	// Per-session concurrent run guard
	if run.running {
		s.mu.Unlock()
		return fmt.Errorf("agent is already running in this session")
	}

	logger.Info("[CHAT] User message received",
		"length", len(userMessage),
		"preview", truncateResult(userMessage, 100),
		"session", s.activeSessionID,
	)

	// Add user message to session's context engine
	run.ctxEngine.AddUserMessage(userMessage)

	// Record user turn in session's timeline collector
	run.timeline.AddUserTurn(
		fmt.Sprintf("usr-%d", time.Now().UnixNano()),
		userMessage,
		time.Now().UnixMilli(),
	)

	// Build runners if not yet built (under lock — no gap)
	if s.deepAgent == nil {
		if err := s.buildRunnersLocked(); err != nil {
			s.mu.Unlock()
			logger.Error("[CHAT] Failed to build runners", err)
			wailsruntime.EventsEmit(s.ctx, "agent:error", map[string]interface{}{
				"sessionId": s.activeSessionID,
				"error":     fmt.Sprintf("Failed to build runner: %v", err),
			})
			return fmt.Errorf("failed to build runners: %w", err)
		}
	}

	// Auto-escalate to plan mode for complex tasks when currently in default mode.
	// This keeps default mode flexible while enforcing strict orchestration once
	// plan mode is entered.
	if run.mode == "default" && shouldAutoPlanMode(userMessage) {
		run.mode = "plan"
		logger.Info("[CHAT] Auto-switched to plan mode",
			"session", s.activeSessionID,
			"reason", "complexity_trigger",
		)
		wailsruntime.EventsEmit(s.ctx, "agent:mode_changed", ModeChangedEvent{
			Mode:      "plan",
			SessionID: s.activeSessionID,
		})
	}

	// Select runner based on session's mode
	runner := s.defaultRunner
	if run.mode == "plan" {
		runner = s.planRunner
	}

	sessionID := run.sessionID

	// Create a cancellable context with session identity
	runCtx, cancel := context.WithCancel(s.ctx)
	runCtx = contextWithSessionID(runCtx, sessionID)
	run.cancelFn = cancel
	run.running = true
	done := make(chan struct{})
	run.runDone = done
	s.mu.Unlock()

	// Prepare messages
	messages := run.ctxEngine.PrepareMessages()
	checkpointID := fmt.Sprintf("run-%d", time.Now().UnixNano())

	// Launch the agent run in a goroutine
	go func() {
		defer close(done)
		defer func() {
			s.mu.Lock()
			run.running = false
			run.cancelFn = nil
			s.cleanupRetiredMCPHandlesLocked()
			s.mu.Unlock()
		}()
		defer cancel()
		defer func() {
			if r := recover(); r != nil {
				wailsruntime.EventsEmit(s.ctx, "agent:error", map[string]interface{}{
					"sessionId": sessionID,
					"error":     fmt.Sprintf("Agent panic: %v", r),
				})
				wailsruntime.EventsEmit(s.ctx, "agent:done", map[string]string{
					"sessionId": sessionID,
				})
			}
		}()

		logger.Info("[CHAT] Agent run started",
			"message_count", len(messages),
			"mode", run.mode,
			"session", sessionID,
		)
		startTime := time.Now()

		events := runner.Run(runCtx, messages, adk.WithCheckPointID(checkpointID))

		lastContent, transferCount, interrupted := s.processEventsForRun(events, checkpointID, run)

		if interrupted {
			return // Don't emit done — waiting for user response
		}

		// Add final assistant response to session's context engine
		if lastContent != "" {
			run.ctxEngine.AddAssistantMessage(lastContent)
		}

		wailsruntime.EventsEmit(s.ctx, "agent:done", map[string]string{
			"sessionId": sessionID,
		})

		logger.Info("[CHAT] Agent run completed",
			"duration_ms", time.Since(startTime).Milliseconds(),
			"transfer_count", transferCount,
			"has_response", lastContent != "",
			"session", sessionID,
		)

		// Notify listeners (e.g. session auto-save) with sessionID
		s.mu.Lock()
		doneFn := s.onAgentDone
		s.mu.Unlock()
		if doneFn != nil {
			doneFn(sessionID)
		}
	}()

	return nil
}

// ---------------------------------------------------------------------------
// Timeline emission
// ---------------------------------------------------------------------------

// emitTimelineForRun emits a timeline event to the frontend AND records it in the
// session's timeline collector for backend persistence.
func (s *ChatService) emitTimelineForRun(evt TimelineEvent, run *SessionRun) {
	evt.SessionID = run.sessionID
	wailsruntime.EventsEmit(s.ctx, "agent:timeline", evt)
	run.timeline.AddEvent(model.DisplayEvent{
		ID:        evt.ID,
		Type:      evt.Type,
		Agent:     evt.Agent,
		Content:   evt.Content,
		ToolName:  evt.ToolName,
		ToolArgs:  evt.ToolArgs,
		ToolID:    evt.ToolID,
		Timestamp: evt.Timestamp,
	}, evt.Agent)
}

// emitTimelineForSession emits a timeline event using a session ID lookup
// (for OnToolEvent callbacks where we only have context, not a run reference).
func (s *ChatService) emitTimelineForSession(evt TimelineEvent, sessionID string) {
	evt.SessionID = sessionID
	wailsruntime.EventsEmit(s.ctx, "agent:timeline", evt)
	s.mu.Lock()
	run, ok := s.sessions[sessionID]
	s.mu.Unlock()
	if ok {
		run.timeline.AddEvent(model.DisplayEvent{
			ID:        evt.ID,
			Type:      evt.Type,
			Agent:     evt.Agent,
			Content:   evt.Content,
			ToolName:  evt.ToolName,
			ToolArgs:  evt.ToolArgs,
			ToolID:    evt.ToolID,
			Timestamp: evt.Timestamp,
		}, evt.Agent)
	}
}

// ---------------------------------------------------------------------------
// Event processing
// ---------------------------------------------------------------------------

// processEventsForRun consumes the event stream for a specific session run,
// emits frontend events, and detects interrupts.
// Returns the last message content, transfer count, and whether an interrupt occurred.
func (s *ChatService) processEventsForRun(events *adk.AsyncIterator[*adk.AgentEvent], checkpointID string, run *SessionRun) (string, int, bool) {
	var allContents []string
	var transferCount int
	lastContentByAgent := make(map[string]string) // dedup

	// Track pending tool_call_ids to detect orphans (tool calls without results)
	pendingToolCalls := make(map[string]bool)

	sessionID := run.sessionID

	// Debounced intermediate save: persist at most once per 10 seconds during agent execution
	var lastSaveTime time.Time
	maybeSave := func() {
		if time.Since(lastSaveTime) > 10*time.Second {
			s.mu.Lock()
			ss := s.sessionService
			s.mu.Unlock()
			if ss != nil {
				go func() { _ = ss.SaveCurrentSession() }()
			}
			lastSaveTime = time.Now()
		}
	}

	for {
		event, ok := events.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			logger.Error("[CHAT] Agent event error", event.Err, "agent", event.AgentName, "session", sessionID)
			// Emit error as timeline info event so the user sees it, but do NOT break
			// the event loop — subsequent events (including sub-agent work) may follow.
			s.emitTimelineForRun(TimelineEvent{
				ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
				Type:      "info",
				Agent:     event.AgentName,
				Content:   fmt.Sprintf("Error: %v", event.Err),
				Timestamp: time.Now().UnixMilli(),
			}, run)
			continue
		}

		// Handle agent actions (tool calls, transfers, interrupts)
		if event.Action != nil {
			// Interrupt detection
			if event.Action.Interrupted != nil && len(event.Action.Interrupted.InterruptContexts) > 0 {
				interruptCtx := event.Action.Interrupted.InterruptContexts[0]
				s.handleInterruptForRun(interruptCtx, checkpointID, run)
				return strings.Join(allContents, "\n\n"), transferCount, true
			}

			if event.Action.TransferToAgent != nil {
				transferCount++
				destName := event.Action.TransferToAgent.DestAgentName
				logger.Transfer(event.AgentName, destName,
					"transfer_count", transferCount,
				)

				// Agent descriptions for enriched transfer events
				agentDescs := map[string]string{
					"code_writer":   "代码读写与编辑",
					"code_executor": "命令与脚本执行",
					"file_manager":  "文件批量操作",
				}

				s.emitTimelineForRun(TimelineEvent{
					ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
					Type:      "transfer",
					Agent:     event.AgentName,
					Content:   destName,
					ToolArgs:  agentDescs[destName],
					Timestamp: time.Now().UnixMilli(),
				}, run)

				// Emit thinking indicator so the user sees the sub-agent is active
				s.emitTimelineForRun(TimelineEvent{
					ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
					Type:      "thinking",
					Agent:     destName,
					Timestamp: time.Now().UnixMilli(),
				}, run)
			}
		}

		// Handle message output
		if event.Output != nil && event.Output.MessageOutput != nil {
			mv := event.Output.MessageOutput
			var msg *schema.Message
			var err error
			var wasStreamed bool

			if mv.IsStreaming && mv.MessageStream != nil {
				msg, err = s.drainStreamForRun(mv.MessageStream, event.AgentName, run)
				wasStreamed = true
			} else {
				msg, err = mv.GetMessage()
			}

			if err != nil {
				wailsruntime.EventsEmit(s.ctx, "agent:error", map[string]interface{}{
					"sessionId": sessionID,
					"error":     fmt.Sprintf("failed to get message: %v", err),
				})
				continue
			}
			if msg == nil {
				continue
			}

			// Emit tool call timeline events
			if len(msg.ToolCalls) > 0 {
				// Surface the LLM's reasoning text before tool calls.
				if msg.Content != "" {
					logger.Info("[CHAT] Reasoning text found with tool calls",
						"agent", event.AgentName,
						"content_len", len(msg.Content),
						"preview", truncateResult(msg.Content, 100),
					)
					s.emitTimelineForRun(TimelineEvent{
						ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
						Type:      "reasoning",
						Agent:     event.AgentName,
						Content:   msg.Content,
						Timestamp: time.Now().UnixMilli(),
					}, run)
				} else {
					logger.Info("[CHAT] No reasoning text with tool calls (Content empty)",
						"agent", event.AgentName,
						"tool_count", len(msg.ToolCalls),
					)
				}

				for _, tc := range msg.ToolCalls {
					s.emitTimelineForRun(TimelineEvent{
						ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
						Type:      "tool_call",
						Agent:     event.AgentName,
						ToolName:  tc.Function.Name,
						ToolArgs:  tc.Function.Arguments,
						ToolID:    tc.ID,
						Timestamp: time.Now().UnixMilli(),
					}, run)
				}

				// Store tool call message in session's context history
				run.ctxEngine.AddMessage(&schema.Message{
					Role:      schema.Assistant,
					Content:   msg.Content,
					ToolCalls: msg.ToolCalls,
				})
				// Track pending tool call IDs
				for _, tc := range msg.ToolCalls {
					pendingToolCalls[tc.ID] = true
				}
				continue // Don't fall through to allContents — tool call content is already stored
			}

			// Emit tool result events
			if msg.Role == schema.Tool && msg.ToolCallID != "" {
				s.emitTimelineForRun(TimelineEvent{
					ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
					Type:      "tool_result",
					Agent:     event.AgentName,
					Content:   truncateResult(msg.Content, 1000),
					ToolID:    msg.ToolCallID,
					Timestamp: time.Now().UnixMilli(),
				}, run)

				// Store tool result in session's context history
				run.ctxEngine.AddToolResult(msg.ToolCallID, msg.Content)
				// Mark this tool call as resolved
				delete(pendingToolCalls, msg.ToolCallID)

				// Debounced intermediate save
				maybeSave()

				// Emit thinking indicator after sub-agent tool result
				if event.AgentName != "coding_agent" {
					s.emitTimelineForRun(TimelineEvent{
						ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
						Type:      "thinking",
						Agent:     event.AgentName,
						Timestamp: time.Now().UnixMilli(),
					}, run)
				}

				continue
			}

			// Emit message event (assistant messages only)
			if msg.Content != "" && msg.Role == schema.Assistant {
				// Dedup: skip if same agent sent identical content
				if prev, ok := lastContentByAgent[event.AgentName]; ok && prev == msg.Content {
					continue
				}
				lastContentByAgent[event.AgentName] = msg.Content
				allContents = append(allContents, msg.Content)

				// Only emit timeline event if content was NOT already streamed
				if !wasStreamed {
					s.emitTimelineForRun(TimelineEvent{
						ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
						Type:      "message",
						Agent:     event.AgentName,
						Content:   msg.Content,
						Timestamp: time.Now().UnixMilli(),
					}, run)
				}
			}
		}
	}

	// Fix orphaned tool calls: inject synthetic error responses for any tool_call_ids
	// that were stored but never received a matching tool result.
	if len(pendingToolCalls) > 0 {
		for toolCallID := range pendingToolCalls {
			logger.Warn("[CHAT] Injecting synthetic tool result for orphaned tool_call",
				"tool_call_id", toolCallID, "session", sessionID)
			run.ctxEngine.AddToolResult(toolCallID, "Error: tool execution failed or was interrupted")
		}
	}

	return strings.Join(allContents, "\n\n"), transferCount, false
}

// ---------------------------------------------------------------------------
// Interrupt handling
// ---------------------------------------------------------------------------

// handleInterruptForRun processes an interrupt context for a specific session run.
func (s *ChatService) handleInterruptForRun(interruptCtx *adk.InterruptCtx, checkpointID string, run *SessionRun) {
	s.mu.Lock()
	run.pendingInterrupt = &PendingInterrupt{
		CheckpointID: checkpointID,
		InterruptID:  interruptCtx.ID,
		Info:         interruptCtx.Info,
	}
	s.mu.Unlock()

	// Determine interrupt type and emit event
	var evt InterruptEvent
	evt.InterruptID = interruptCtx.ID
	evt.CheckpointID = checkpointID
	evt.SessionID = run.sessionID

	switch info := interruptCtx.Info.(type) {
	case *tools.FollowUpInfo:
		evt.Type = "followup"
		evt.Questions = info.Questions
		logger.Info("[CHAT] Interrupt: follow-up questions", "count", len(info.Questions), "session", run.sessionID)
	case *tools.ChoiceInfo:
		evt.Type = "choice"
		evt.Question = info.Question
		for _, opt := range info.Options {
			evt.Options = append(evt.Options, InterruptOption{
				Label:       opt.Label,
				Description: opt.Description,
			})
		}
		logger.Info("[CHAT] Interrupt: choice", "question", info.Question, "options", len(info.Options), "session", run.sessionID)
	default:
		logger.Warn("[CHAT] Unknown interrupt type", "type", fmt.Sprintf("%T", interruptCtx.Info))
		evt.Type = "followup"
		evt.Questions = []string{fmt.Sprintf("%v", interruptCtx.Info)}
	}

	wailsruntime.EventsEmit(s.ctx, "agent:interrupt", evt)
	s.emitTimelineForRun(TimelineEvent{
		ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
		Type:      "interrupt",
		Agent:     "system",
		Content:   fmt.Sprintf("Waiting for user input: %s", evt.Type),
		Timestamp: time.Now().UnixMilli(),
	}, run)
}

// ---------------------------------------------------------------------------
// Resume
// ---------------------------------------------------------------------------

// ResumeWithAnswer resumes execution after the user answers follow-up questions.
func (s *ChatService) ResumeWithAnswer(answer string) error {
	s.mu.Lock()
	run := s.activeRun()
	if run == nil {
		s.mu.Unlock()
		return fmt.Errorf("no active session")
	}

	pending := run.pendingInterrupt
	if pending == nil {
		s.mu.Unlock()
		return fmt.Errorf("no pending interrupt to resume")
	}
	run.pendingInterrupt = nil

	if run.running {
		s.mu.Unlock()
		return fmt.Errorf("agent is already running in this session")
	}

	runner := s.defaultRunner
	if run.mode == "plan" {
		runner = s.planRunner
	}

	sessionID := run.sessionID
	s.mu.Unlock()

	// Build resume data with user's answer
	resumeData := &tools.FollowUpInfo{
		UserAnswer: answer,
	}
	if info, ok := pending.Info.(*tools.FollowUpInfo); ok {
		resumeData.Questions = info.Questions
	}

	runCtx, cancel := context.WithCancel(s.ctx)
	runCtx = contextWithSessionID(runCtx, sessionID)
	s.mu.Lock()
	run.cancelFn = cancel
	run.running = true
	done := make(chan struct{})
	run.runDone = done
	s.mu.Unlock()

	go func() {
		defer close(done)
		defer func() {
			s.mu.Lock()
			run.running = false
			run.cancelFn = nil
			s.cleanupRetiredMCPHandlesLocked()
			s.mu.Unlock()
		}()
		defer cancel()
		defer func() {
			if r := recover(); r != nil {
				wailsruntime.EventsEmit(s.ctx, "agent:error", map[string]interface{}{
					"sessionId": sessionID,
					"error":     fmt.Sprintf("Resume panic: %v", r),
				})
				wailsruntime.EventsEmit(s.ctx, "agent:done", map[string]string{
					"sessionId": sessionID,
				})
			}
		}()

		logger.Info("[CHAT] Resuming after follow-up", "answer_length", len(answer), "session", sessionID)
		startTime := time.Now()

		events, err := runner.ResumeWithParams(runCtx, pending.CheckpointID, &adk.ResumeParams{
			Targets: map[string]any{
				pending.InterruptID: resumeData,
			},
		})
		if err != nil {
			wailsruntime.EventsEmit(s.ctx, "agent:error", map[string]interface{}{
				"sessionId": sessionID,
				"error":     fmt.Sprintf("Resume failed: %v", err),
			})
			wailsruntime.EventsEmit(s.ctx, "agent:done", map[string]string{
				"sessionId": sessionID,
			})
			return
		}

		lastContent, transferCount, interrupted := s.processEventsForRun(events, pending.CheckpointID, run)

		if interrupted {
			return
		}

		if lastContent != "" {
			run.ctxEngine.AddAssistantMessage(lastContent)
		}

		wailsruntime.EventsEmit(s.ctx, "agent:done", map[string]string{
			"sessionId": sessionID,
		})
		logger.Info("[CHAT] Resume completed",
			"duration_ms", time.Since(startTime).Milliseconds(),
			"transfer_count", transferCount,
			"session", sessionID,
		)

		s.mu.Lock()
		doneFn := s.onAgentDone
		s.mu.Unlock()
		if doneFn != nil {
			doneFn(sessionID)
		}
	}()

	return nil
}

// ResumeWithChoice resumes execution after the user selects a choice.
func (s *ChatService) ResumeWithChoice(selectedIndex int) error {
	s.mu.Lock()
	run := s.activeRun()
	if run == nil {
		s.mu.Unlock()
		return fmt.Errorf("no active session")
	}

	pending := run.pendingInterrupt
	if pending == nil {
		s.mu.Unlock()
		return fmt.Errorf("no pending interrupt to resume")
	}
	run.pendingInterrupt = nil

	if run.running {
		s.mu.Unlock()
		return fmt.Errorf("agent is already running in this session")
	}

	runner := s.defaultRunner
	if run.mode == "plan" {
		runner = s.planRunner
	}

	sessionID := run.sessionID
	s.mu.Unlock()

	// Build resume data with user's selection
	resumeData := &tools.ChoiceInfo{
		Selected: selectedIndex,
	}
	if info, ok := pending.Info.(*tools.ChoiceInfo); ok {
		resumeData.Question = info.Question
		resumeData.Options = info.Options
	}

	runCtx, cancel := context.WithCancel(s.ctx)
	runCtx = contextWithSessionID(runCtx, sessionID)
	s.mu.Lock()
	run.cancelFn = cancel
	run.running = true
	done := make(chan struct{})
	run.runDone = done
	s.mu.Unlock()

	go func() {
		defer close(done)
		defer func() {
			s.mu.Lock()
			run.running = false
			run.cancelFn = nil
			s.cleanupRetiredMCPHandlesLocked()
			s.mu.Unlock()
		}()
		defer cancel()
		defer func() {
			if r := recover(); r != nil {
				wailsruntime.EventsEmit(s.ctx, "agent:error", map[string]interface{}{
					"sessionId": sessionID,
					"error":     fmt.Sprintf("Resume panic: %v", r),
				})
				wailsruntime.EventsEmit(s.ctx, "agent:done", map[string]string{
					"sessionId": sessionID,
				})
			}
		}()

		logger.Info("[CHAT] Resuming after choice", "selected", selectedIndex, "session", sessionID)
		startTime := time.Now()

		events, err := runner.ResumeWithParams(runCtx, pending.CheckpointID, &adk.ResumeParams{
			Targets: map[string]any{
				pending.InterruptID: resumeData,
			},
		})
		if err != nil {
			wailsruntime.EventsEmit(s.ctx, "agent:error", map[string]interface{}{
				"sessionId": sessionID,
				"error":     fmt.Sprintf("Resume failed: %v", err),
			})
			wailsruntime.EventsEmit(s.ctx, "agent:done", map[string]string{
				"sessionId": sessionID,
			})
			return
		}

		lastContent, transferCount, interrupted := s.processEventsForRun(events, pending.CheckpointID, run)

		if interrupted {
			return
		}

		if lastContent != "" {
			run.ctxEngine.AddAssistantMessage(lastContent)
		}

		wailsruntime.EventsEmit(s.ctx, "agent:done", map[string]string{
			"sessionId": sessionID,
		})
		logger.Info("[CHAT] Resume completed",
			"duration_ms", time.Since(startTime).Milliseconds(),
			"transfer_count", transferCount,
			"session", sessionID,
		)

		s.mu.Lock()
		doneFn := s.onAgentDone
		s.mu.Unlock()
		if doneFn != nil {
			doneFn(sessionID)
		}
	}()

	return nil
}

// ---------------------------------------------------------------------------
// Stop
// ---------------------------------------------------------------------------

// StopGeneration cancels the currently running agent in the active session.
func (s *ChatService) StopGeneration() error {
	s.mu.Lock()
	run := s.activeRun()
	if run == nil || !run.running {
		s.mu.Unlock()
		return nil
	}
	if run.cancelFn != nil {
		run.cancelFn()
	}
	run.pendingInterrupt = nil
	done := run.runDone
	s.mu.Unlock()

	if done != nil {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			logger.Warn("[CHAT] StopGeneration timed out", "session", s.activeSessionID)
		}
	}
	return nil
}

// StopSessionGeneration cancels a running agent in a specific session.
func (s *ChatService) StopSessionGeneration(sessionID string) {
	s.mu.Lock()
	run, ok := s.sessions[sessionID]
	if !ok || !run.running {
		s.mu.Unlock()
		return
	}
	if run.cancelFn != nil {
		run.cancelFn()
	}
	run.pendingInterrupt = nil
	done := run.runDone
	s.mu.Unlock()

	if done != nil {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			logger.Warn("[CHAT] StopSessionGeneration timed out", "session", sessionID)
		}
	}
}

// ---------------------------------------------------------------------------
// History & state access
// ---------------------------------------------------------------------------

// ClearHistory resets the conversation history for the active session.
func (s *ChatService) ClearHistory() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	run := s.activeRun()
	if run == nil {
		return nil
	}

	run.ctxEngine.ClearHistory()
	run.timeline.Clear()
	run.streamingState = nil
	run.pendingInterrupt = nil
	s.invalidateRunners()
	tools.ClearTodos()
	return nil
}

// Timeline returns the timeline collector for the active session.
func (s *ChatService) Timeline() *agentctx.TimelineCollector {
	s.mu.Lock()
	defer s.mu.Unlock()
	run := s.activeRun()
	if run == nil {
		return agentctx.NewTimelineCollector() // return empty if no session
	}
	return run.timeline
}

// StreamingState returns the current streaming state for the active session (nil if not streaming).
func (s *ChatService) StreamingState() *model.StreamingState {
	s.mu.Lock()
	defer s.mu.Unlock()
	run := s.activeRun()
	if run == nil {
		return nil
	}
	if run.streamingState == nil {
		return nil
	}
	ss := *run.streamingState
	return &ss
}

// CtxEngine returns the context engine for the active session.
func (s *ChatService) CtxEngine() *agentctx.Engine {
	s.mu.Lock()
	defer s.mu.Unlock()
	run := s.activeRun()
	if run == nil {
		return nil
	}
	return run.ctxEngine
}

// SessionCtxEngine returns the context engine for a specific session.
func (s *ChatService) SessionCtxEngine(sessionID string) *agentctx.Engine {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.sessions[sessionID]
	if !ok {
		return nil
	}
	return run.ctxEngine
}

// SessionTimeline returns the timeline collector for a specific session.
func (s *ChatService) SessionTimeline(sessionID string) *agentctx.TimelineCollector {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.sessions[sessionID]
	if !ok {
		return nil
	}
	return run.timeline
}

// SessionStreamingState returns the streaming state for a specific session.
func (s *ChatService) SessionStreamingState(sessionID string) *model.StreamingState {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.sessions[sessionID]
	if !ok || run.streamingState == nil {
		return nil
	}
	ss := *run.streamingState
	return &ss
}

// GetSessionRunSnapshot returns a snapshot of a session's run state for the
// SessionSwitchedEvent. Safe to call from SessionService.
func (s *ChatService) GetSessionRunSnapshot(sessionID string) (running bool, currentAgent, mode string, interrupt *InterruptEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.sessions[sessionID]
	if !ok {
		return false, "", "default", nil
	}
	running = run.running
	currentAgent = run.currentAgent
	mode = run.mode
	if run.pendingInterrupt != nil {
		interrupt = s.buildInterruptEvent(run.pendingInterrupt, sessionID)
	}
	return
}

// buildInterruptEvent converts a PendingInterrupt into an InterruptEvent.
// Caller must hold s.mu.
func (s *ChatService) buildInterruptEvent(pi *PendingInterrupt, sessionID string) *InterruptEvent {
	evt := &InterruptEvent{
		InterruptID:  pi.InterruptID,
		CheckpointID: pi.CheckpointID,
		SessionID:    sessionID,
	}
	switch info := pi.Info.(type) {
	case *tools.FollowUpInfo:
		evt.Type = "followup"
		evt.Questions = info.Questions
	case *tools.ChoiceInfo:
		evt.Type = "choice"
		evt.Question = info.Question
		for _, opt := range info.Options {
			evt.Options = append(evt.Options, InterruptOption{
				Label:       opt.Label,
				Description: opt.Description,
			})
		}
	default:
		evt.Type = "followup"
		evt.Questions = []string{fmt.Sprintf("%v", pi.Info)}
	}
	return evt
}

// ---------------------------------------------------------------------------
// Runner building
// ---------------------------------------------------------------------------

// BuildRunners builds the deep agent and both runners using the current config.
func (s *ChatService) BuildRunners() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buildRunnersLocked()
}

// buildRunnersLocked builds the deep agent and both runners (caller must hold s.mu).
func (s *ChatService) buildRunnersLocked() error {
	if s.sandbox == nil || !s.sandbox.IsConnected() {
		return fmt.Errorf("sandbox is not connected")
	}

	op := s.sandbox.Operator()
	if op == nil {
		return fmt.Errorf("sandbox operator is not available")
	}

	cfg := s.store.Get()
	ctx := s.ctx

	// Create LLM chat model
	mdl, err := llm.NewChatModel(ctx, cfg.LLM)
	if err != nil {
		logger.Error("[RUNNER] Failed to create chat model", err)
		return fmt.Errorf("failed to create chat model: %w", err)
	}
	logger.RunnerEvent("chat_model_created", "type", cfg.LLM.Type, "model", cfg.LLM.Model)

	// Register built-in tools
	registry := tools.NewToolRegistry()
	if err := tools.RegisterBuiltinTools(registry, op, s.getWorkspacePath()); err != nil {
		return fmt.Errorf("failed to register builtin tools: %w", err)
	}

	mcpHandles, mcpCatalog, err := tools.BuildMCPActionCatalog(ctx, cfg.MCP.Servers)
	if err != nil {
		return fmt.Errorf("failed to build MCP action catalog: %w", err)
	}
	for _, handle := range mcpHandles {
		if handle == nil {
			continue
		}
		if handle.LastError != nil {
			wailsruntime.EventsEmit(s.ctx, "agent:error",
				fmt.Sprintf("MCP server %s unavailable (%s): %v", handle.Name, handle.State, handle.LastError))
		}
	}
	toolsByServer := make(map[string][]einotool.BaseTool)
	for _, entry := range mcpCatalog.Entries() {
		toolsByServer[entry.Server] = append(toolsByServer[entry.Server], entry.Tool)
	}
	for serverName, serverTools := range toolsByServer {
		registry.RegisterMCPTools(serverName, serverTools)
	}

	extraTools := registry.GetAll()

	// Build agent context
	ac := s.buildAgentContext()

	// Build the core deep agents with mode-specific permission boundaries.
	deepAgentDefault, err := agent.BuildDeepAgentForMode(ctx, mdl, op, extraTools, ac, agent.DeepAgentModeDefault)
	if err != nil {
		s.closeMCPHandlesLocked(mcpHandles)
		logger.Error("[RUNNER] Failed to build default deep agent", err)
		return fmt.Errorf("failed to build default deep agent: %w", err)
	}

	deepAgentPlan, err := agent.BuildDeepAgentForMode(ctx, mdl, op, extraTools, ac, agent.DeepAgentModePlan)
	if err != nil {
		s.closeMCPHandlesLocked(mcpHandles)
		logger.Error("[RUNNER] Failed to build plan deep agent", err)
		return fmt.Errorf("failed to build plan deep agent: %w", err)
	}

	// Build default runner
	defaultRunner := agent.BuildDefaultRunner(ctx, deepAgentDefault, s.checkpointStore)

	// Build plan runner
	planRunner, err := agent.BuildPlanRunner(ctx, mdl, deepAgentPlan, ac, s.checkpointStore)
	if err != nil {
		s.closeMCPHandlesLocked(mcpHandles)
		logger.Error("[RUNNER] Failed to build plan runner", err)
		return fmt.Errorf("failed to build plan runner: %w", err)
	}

	oldHandles := s.mcpHandles
	s.deepAgent = deepAgentDefault
	s.defaultRunner = defaultRunner
	s.planRunner = planRunner
	s.mcpHandles = mcpHandles
	s.retireMCPHandlesLocked(oldHandles)

	logger.RunnerEvent("runners_built", "extra_tools", len(extraTools))
	return nil
}

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

// formatToolCall formats a tool call for display purposes.
func formatToolCall(tc schema.ToolCall) string {
	detail := fmt.Sprintf("%s(%s)", tc.Function.Name, tc.Function.Arguments)
	if len(detail) > 200 {
		detail = detail[:200] + "..."
	}
	return detail
}

// truncateResult truncates a tool result to maxLen characters.
func truncateResult(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "... (truncated)"
}

// shouldAutoPlanMode applies a deterministic heuristic to decide whether a
// user request is complex enough to auto-enter plan mode.
func shouldAutoPlanMode(userMessage string) bool {
	msg := strings.ToLower(strings.TrimSpace(userMessage))
	if msg == "" {
		return false
	}

	// Explicit intent to plan.
	explicitPlanSignals := []string{
		"plan mode", "planning mode", "plan-mode",
		"计划模式", "规划模式", "进入计划", "进入规划",
	}
	for _, k := range explicitPlanSignals {
		if strings.Contains(msg, k) {
			return true
		}
	}

	stepSignals := []string{
		"and then", "then ", "after that", "step by step",
		"先", "然后", "再", "并且", "同时", "步骤",
	}
	workSignals := []string{
		"write", "edit", "refactor", "implement", "fix", "debug",
		"run", "test", "verify", "validate", "build",
		"写", "改", "重构", "实现", "修复", "调试",
		"运行", "测试", "验证", "构建", "编译",
	}

	stepCount := 0
	for _, k := range stepSignals {
		if strings.Contains(msg, k) {
			stepCount++
		}
	}

	workCount := 0
	for _, k := range workSignals {
		if strings.Contains(msg, k) {
			workCount++
		}
	}

	// Complex multi-action intent: contains sequencing and multiple work signals.
	return stepCount > 0 && workCount >= 2
}

// buildAgentContext constructs an AgentContext from the current session and sandbox state.
func (s *ChatService) buildAgentContext() agent.AgentContext {
	ac := agent.DefaultAgentContext()

	if s.sessionService != nil {
		ac.WorkspacePath = s.sessionService.GetWorkspacePath()
	}

	// Inject SSH info from config
	cfg := s.store.Get()
	if cfg != nil {
		ac.SSHHost = cfg.SSH.Host
		ac.SSHPort = cfg.SSH.Port
		ac.SSHUser = cfg.SSH.User
	}

	if s.sandbox != nil {
		docker := s.sandbox.Docker()
		if docker != nil {
			ac.ContainerName = docker.ContainerName()
			cid := docker.ContainerID()
			if len(cid) > 12 {
				cid = cid[:12]
			}
			ac.ContainerID = cid
		}
	}

	// Set up OnToolEvent to emit sub-agent tool calls as timeline events.
	// The context carries session identity for proper event routing.
	ac.OnToolEvent = func(ctx context.Context, agentName, eventType, toolName, toolArgs, toolID, result string) {
		sessionID := SessionIDFromContext(ctx)
		s.emitTimelineForSession(TimelineEvent{
			ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
			Type:      eventType,
			Agent:     agentName,
			ToolName:  toolName,
			ToolArgs:  toolArgs,
			ToolID:    toolID,
			Content:   result,
			Timestamp: time.Now().UnixMilli(),
		}, sessionID)
	}

	return ac
}

// getWorkspacePath returns the workspace path from the session service or a default.
func (s *ChatService) getWorkspacePath() string {
	if s.sessionService != nil {
		return s.sessionService.GetWorkspacePath()
	}
	return "/workspace"
}

// drainStreamForRun iterates a MessageStream for a specific session run,
// batching stream_chunk timeline events with a 50ms window to reduce IPC frequency,
// and returns the final concatenated message.
func (s *ChatService) drainStreamForRun(stream adk.MessageStream, agentName string, run *SessionRun) (*schema.Message, error) {
	defer stream.Close()

	var chunks []*schema.Message
	var streamingContent strings.Builder

	const batchInterval = 50 * time.Millisecond
	ticker := time.NewTicker(batchInterval)
	defer ticker.Stop()

	var pendingText strings.Builder

	flushChunks := func() {
		if pendingText.Len() == 0 {
			return
		}
		merged := pendingText.String()
		pendingText.Reset()
		s.emitTimelineForRun(TimelineEvent{
			ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
			Type:      "stream_chunk",
			Agent:     agentName,
			Content:   merged,
			Timestamp: time.Now().UnixMilli(),
		}, run)
	}

	done := false
	for !done {
		select {
		case <-ticker.C:
			flushChunks()
			// Update session's streaming state for mid-stream saves
			run.streamingState = &model.StreamingState{
				PartialContent: streamingContent.String(),
				AgentName:      agentName,
			}
		default:
			chunk, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					done = true
					break
				}
				flushChunks()
				return nil, err
			}
			chunks = append(chunks, chunk)

			if chunk.Content != "" {
				pendingText.WriteString(chunk.Content)
				streamingContent.WriteString(chunk.Content)
			}
		}
	}

	// Flush remaining
	flushChunks()

	// Clear session's streaming state
	run.streamingState = nil

	// Signal stream end
	s.emitTimelineForRun(TimelineEvent{
		ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
		Type:      "stream_end",
		Agent:     agentName,
		Timestamp: time.Now().UnixMilli(),
	}, run)

	if len(chunks) == 0 {
		return nil, nil
	}

	return schema.ConcatMessages(chunks)
}
