package service

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/adk"
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

// PendingInterrupt holds the state needed to resume after an interrupt.
type PendingInterrupt struct {
	CheckpointID string
	InterruptID  string
	Info         any
}

// ChatService manages chat interactions between the frontend and the AI agent.
type ChatService struct {
	ctx            context.Context
	deepAgent      adk.Agent
	defaultRunner  *adk.Runner
	planRunner     *adk.Runner
	ctxEngine      *agentctx.Engine
	timeline       *agentctx.TimelineCollector
	sandbox        *sandbox.SandboxManager
	store          *config.Store
	sessionService *SessionService
	cancelFn       context.CancelFunc
	onAgentDone    func()

	mode             string // "default" or "plan"
	checkpointStore  compose.CheckPointStore
	pendingInterrupt *PendingInterrupt
	streamingState   *model.StreamingState // non-nil during active streaming

	mu sync.Mutex
}

// NewChatService creates a new ChatService.
func NewChatService(store *config.Store) *ChatService {
	return &ChatService{
		store:           store,
		mode:            "default",
		checkpointStore: checkpoint.NewInMemoryStore(),
		timeline:        agentctx.NewTimelineCollector(),
	}
}

// SetContext stores the Wails application context. Called from app.go startup.
func (s *ChatService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// SetDependencies injects the sandbox manager and context engine.
func (s *ChatService) SetDependencies(sbx *sandbox.SandboxManager, ctxEngine *agentctx.Engine) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sandbox = sbx
	s.ctxEngine = ctxEngine
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
	s.deepAgent = nil
	s.defaultRunner = nil
	s.planRunner = nil
}

// SetOnAgentDone registers a callback that fires after the agent finishes processing.
func (s *ChatService) SetOnAgentDone(fn func()) {
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

// SetMode switches between "default" and "plan" mode.
func (s *ChatService) SetMode(mode string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if mode != "default" && mode != "plan" {
		return fmt.Errorf("invalid mode: %s (must be 'default' or 'plan')", mode)
	}

	s.mode = mode
	logger.Info("[CHAT] Mode changed", "mode", mode)
	wailsruntime.EventsEmit(s.ctx, "agent:mode_changed", ModeChangedEvent{Mode: mode})
	return nil
}

// GetMode returns the current agent mode.
func (s *ChatService) GetMode() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mode
}

// SendMessage processes a user message through the agent and streams results to the frontend.
func (s *ChatService) SendMessage(userMessage string) error {
	s.mu.Lock()
	if s.ctxEngine == nil {
		s.mu.Unlock()
		return fmt.Errorf("chat service not initialized: context engine is nil")
	}

	logger.Info("[CHAT] User message received",
		"length", len(userMessage),
		"preview", truncateResult(userMessage, 100),
	)

	// Add user message to context engine
	s.ctxEngine.AddUserMessage(userMessage)

	// Record user turn in timeline collector
	s.timeline.AddUserTurn(
		fmt.Sprintf("usr-%d", time.Now().UnixNano()),
		userMessage,
		time.Now().UnixMilli(),
	)

	// Build runners if not yet built
	if s.deepAgent == nil {
		s.mu.Unlock()
		if err := s.BuildRunners(); err != nil {
			logger.Error("[CHAT] Failed to build runners", err)
			wailsruntime.EventsEmit(s.ctx, "agent:error", fmt.Sprintf("Failed to build runner: %v", err))
			return fmt.Errorf("failed to build runners: %w", err)
		}
		s.mu.Lock()
	}

	// Select runner based on mode
	runner := s.defaultRunner
	if s.mode == "plan" {
		runner = s.planRunner
	}

	ctxEngine := s.ctxEngine
	mode := s.mode

	// Create a cancellable context
	runCtx, cancel := context.WithCancel(s.ctx)
	s.cancelFn = cancel
	s.mu.Unlock()

	// Prepare messages
	messages := ctxEngine.PrepareMessages()
	checkpointID := fmt.Sprintf("run-%d", time.Now().UnixNano())

	// Launch the agent run in a goroutine
	go func() {
		defer cancel()
		defer func() {
			if r := recover(); r != nil {
				wailsruntime.EventsEmit(s.ctx, "agent:error", fmt.Sprintf("Agent panic: %v", r))
				wailsruntime.EventsEmit(s.ctx, "agent:done", nil)
			}
		}()

		logger.Info("[CHAT] Agent run started", "message_count", len(messages), "mode", mode)
		startTime := time.Now()

		events := runner.Run(runCtx, messages, adk.WithCheckPointID(checkpointID))

		lastContent, transferCount, interrupted := s.processEvents(events, checkpointID)

		if interrupted {
			return // Don't emit done — waiting for user response
		}

		// Add final assistant response to context engine
		if lastContent != "" {
			s.mu.Lock()
			if s.ctxEngine != nil {
				s.ctxEngine.AddAssistantMessage(lastContent)
			}
			s.mu.Unlock()
		}

		wailsruntime.EventsEmit(s.ctx, "agent:done", nil)

		logger.Info("[CHAT] Agent run completed",
			"duration_ms", time.Since(startTime).Milliseconds(),
			"transfer_count", transferCount,
			"has_response", lastContent != "",
		)

		// Notify listeners (e.g. session auto-save)
		s.mu.Lock()
		doneFn := s.onAgentDone
		s.mu.Unlock()
		if doneFn != nil {
			doneFn()
		}
	}()

	return nil
}

// emitTimeline emits a timeline event to the frontend AND records it in the
// timeline collector for backend persistence.
func (s *ChatService) emitTimeline(evt TimelineEvent) {
	wailsruntime.EventsEmit(s.ctx, "agent:timeline", evt)
	s.timeline.AddEvent(model.DisplayEvent{
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

// processEvents consumes the event stream, emits frontend events, and detects interrupts.
// Returns the last message content, transfer count, and whether an interrupt occurred.
func (s *ChatService) processEvents(events *adk.AsyncIterator[*adk.AgentEvent], checkpointID string) (string, int, bool) {
	var allContents []string
	var transferCount int
	lastContentByAgent := make(map[string]string) // dedup

	// Track pending tool_call_ids to detect orphans (tool calls without results)
	pendingToolCalls := make(map[string]bool)

	// Debounced intermediate save: persist at most once per 10 seconds during agent execution
	var lastSaveTime time.Time
	maybeSave := func() {
		if time.Since(lastSaveTime) > 10*time.Second {
			if s.sessionService != nil {
				go func() { _ = s.sessionService.SaveCurrentSession() }()
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
			logger.Error("[CHAT] Agent event error", event.Err, "agent", event.AgentName)
			// Emit error as timeline info event so the user sees it, but do NOT break
			// the event loop — subsequent events (including sub-agent work) may follow.
			s.emitTimeline(TimelineEvent{
				ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
				Type:      "info",
				Agent:     event.AgentName,
				Content:   fmt.Sprintf("Error: %v", event.Err),
				Timestamp: time.Now().UnixMilli(),
			})
			continue
		}

		// Handle agent actions (tool calls, transfers, interrupts)
		if event.Action != nil {
			// Interrupt detection
			if event.Action.Interrupted != nil && len(event.Action.Interrupted.InterruptContexts) > 0 {
				interruptCtx := event.Action.Interrupted.InterruptContexts[0]
				s.handleInterrupt(interruptCtx, checkpointID)
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

				s.emitTimeline(TimelineEvent{
					ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
					Type:      "transfer",
					Agent:     event.AgentName,
					Content:   destName,
					ToolArgs:  agentDescs[destName],
					Timestamp: time.Now().UnixMilli(),
				})

				// Emit thinking indicator so the user sees the sub-agent is active
				s.emitTimeline(TimelineEvent{
					ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
					Type:      "thinking",
					Agent:     destName,
					Timestamp: time.Now().UnixMilli(),
				})
			}
		}

		// Handle message output
		if event.Output != nil && event.Output.MessageOutput != nil {
			mv := event.Output.MessageOutput
			var msg *schema.Message
			var err error
			var wasStreamed bool

			if mv.IsStreaming && mv.MessageStream != nil {
				msg, err = s.drainStream(mv.MessageStream, event.AgentName)
				wasStreamed = true
			} else {
				msg, err = mv.GetMessage()
			}

			if err != nil {
				wailsruntime.EventsEmit(s.ctx, "agent:error",
					fmt.Sprintf("failed to get message: %v", err))
				continue
			}
			if msg == nil {
				continue
			}

			// Emit tool call timeline events
			if len(msg.ToolCalls) > 0 {
				// Surface the LLM's reasoning text before tool calls.
				// The LLM often explains its intent (e.g. "I'll read the file to understand...")
				// but this content was previously discarded by the continue below.
				if msg.Content != "" {
					logger.Info("[CHAT] Reasoning text found with tool calls",
						"agent", event.AgentName,
						"content_len", len(msg.Content),
						"preview", truncateResult(msg.Content, 100),
					)
					s.emitTimeline(TimelineEvent{
						ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
						Type:      "reasoning",
						Agent:     event.AgentName,
						Content:   msg.Content,
						Timestamp: time.Now().UnixMilli(),
					})
				} else {
					logger.Info("[CHAT] No reasoning text with tool calls (Content empty)",
						"agent", event.AgentName,
						"tool_count", len(msg.ToolCalls),
					)
				}

				for _, tc := range msg.ToolCalls {
					s.emitTimeline(TimelineEvent{
						ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
						Type:      "tool_call",
						Agent:     event.AgentName,
						ToolName:  tc.Function.Name,
						ToolArgs:  tc.Function.Arguments,
						ToolID:    tc.ID,
						Timestamp: time.Now().UnixMilli(),
					})
				}

				// Store tool call message in context history for persistence
				s.mu.Lock()
				if s.ctxEngine != nil {
					s.ctxEngine.AddMessage(&schema.Message{
						Role:      schema.Assistant,
						Content:   msg.Content,
						ToolCalls: msg.ToolCalls,
					})
				}
				s.mu.Unlock()
				// Track pending tool call IDs
				for _, tc := range msg.ToolCalls {
					pendingToolCalls[tc.ID] = true
				}
				continue // Don't fall through to allContents — tool call content is already stored
			}

			// Emit tool result events
			if msg.Role == schema.Tool && msg.ToolCallID != "" {
				s.emitTimeline(TimelineEvent{
					ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
					Type:      "tool_result",
					Agent:     event.AgentName,
					Content:   truncateResult(msg.Content, 1000),
					ToolID:    msg.ToolCallID,
					Timestamp: time.Now().UnixMilli(),
				})

				// Store tool result in context history for persistence
				s.mu.Lock()
				if s.ctxEngine != nil {
					s.ctxEngine.AddToolResult(msg.ToolCallID, msg.Content)
				}
				s.mu.Unlock()
				// Mark this tool call as resolved
				delete(pendingToolCalls, msg.ToolCallID)

				// Debounced intermediate save
				maybeSave()

				// Emit thinking indicator after sub-agent tool result
				// so the user knows the sub-agent is still working
				if event.AgentName != "coding_agent" {
					s.emitTimeline(TimelineEvent{
						ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
						Type:      "thinking",
						Agent:     event.AgentName,
						Timestamp: time.Now().UnixMilli(),
					})
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
					s.emitTimeline(TimelineEvent{
						ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
						Type:      "message",
						Agent:     event.AgentName,
						Content:   msg.Content,
						Timestamp: time.Now().UnixMilli(),
					})
				}
			}
		}
	}

	// Fix orphaned tool calls: inject synthetic error responses for any tool_call_ids
	// that were stored but never received a matching tool result. This prevents the
	// OpenAI API from rejecting the next request with "tool_calls must be followed by
	// tool messages responding to each tool_call_id".
	if len(pendingToolCalls) > 0 {
		s.mu.Lock()
		if s.ctxEngine != nil {
			for toolCallID := range pendingToolCalls {
				logger.Warn("[CHAT] Injecting synthetic tool result for orphaned tool_call",
					"tool_call_id", toolCallID)
				s.ctxEngine.AddToolResult(toolCallID, "Error: tool execution failed or was interrupted")
			}
		}
		s.mu.Unlock()
	}

	return strings.Join(allContents, "\n\n"), transferCount, false
}

// handleInterrupt processes an interrupt context and emits events to the frontend.
func (s *ChatService) handleInterrupt(interruptCtx *adk.InterruptCtx, checkpointID string) {
	s.mu.Lock()
	s.pendingInterrupt = &PendingInterrupt{
		CheckpointID: checkpointID,
		InterruptID:  interruptCtx.ID,
		Info:         interruptCtx.Info,
	}
	s.mu.Unlock()

	// Determine interrupt type and emit event
	var evt InterruptEvent
	evt.InterruptID = interruptCtx.ID
	evt.CheckpointID = checkpointID

	switch info := interruptCtx.Info.(type) {
	case *tools.FollowUpInfo:
		evt.Type = "followup"
		evt.Questions = info.Questions
		logger.Info("[CHAT] Interrupt: follow-up questions", "count", len(info.Questions))
	case *tools.ChoiceInfo:
		evt.Type = "choice"
		evt.Question = info.Question
		for _, opt := range info.Options {
			evt.Options = append(evt.Options, InterruptOption{
				Label:       opt.Label,
				Description: opt.Description,
			})
		}
		logger.Info("[CHAT] Interrupt: choice", "question", info.Question, "options", len(info.Options))
	default:
		logger.Warn("[CHAT] Unknown interrupt type", "type", fmt.Sprintf("%T", interruptCtx.Info))
		evt.Type = "followup"
		evt.Questions = []string{fmt.Sprintf("%v", interruptCtx.Info)}
	}

	wailsruntime.EventsEmit(s.ctx, "agent:interrupt", evt)
	s.emitTimeline(TimelineEvent{
		ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
		Type:      "interrupt",
		Agent:     "system",
		Content:   fmt.Sprintf("Waiting for user input: %s", evt.Type),
		Timestamp: time.Now().UnixMilli(),
	})
}

// ResumeWithAnswer resumes execution after the user answers follow-up questions.
func (s *ChatService) ResumeWithAnswer(answer string) error {
	s.mu.Lock()
	pending := s.pendingInterrupt
	if pending == nil {
		s.mu.Unlock()
		return fmt.Errorf("no pending interrupt to resume")
	}
	s.pendingInterrupt = nil

	runner := s.defaultRunner
	if s.mode == "plan" {
		runner = s.planRunner
	}
	s.mu.Unlock()

	// Build resume data with user's answer
	resumeData := &tools.FollowUpInfo{
		UserAnswer: answer,
	}
	if info, ok := pending.Info.(*tools.FollowUpInfo); ok {
		resumeData.Questions = info.Questions
	}

	runCtx, cancel := context.WithCancel(s.ctx)
	s.mu.Lock()
	s.cancelFn = cancel
	s.mu.Unlock()

	go func() {
		defer cancel()
		defer func() {
			if r := recover(); r != nil {
				wailsruntime.EventsEmit(s.ctx, "agent:error", fmt.Sprintf("Resume panic: %v", r))
				wailsruntime.EventsEmit(s.ctx, "agent:done", nil)
			}
		}()

		logger.Info("[CHAT] Resuming after follow-up", "answer_length", len(answer))
		startTime := time.Now()

		events, err := runner.ResumeWithParams(runCtx, pending.CheckpointID, &adk.ResumeParams{
			Targets: map[string]any{
				pending.InterruptID: resumeData,
			},
		})
		if err != nil {
			wailsruntime.EventsEmit(s.ctx, "agent:error", fmt.Sprintf("Resume failed: %v", err))
			wailsruntime.EventsEmit(s.ctx, "agent:done", nil)
			return
		}

		lastContent, transferCount, interrupted := s.processEvents(events, pending.CheckpointID)

		if interrupted {
			return
		}

		if lastContent != "" {
			s.mu.Lock()
			if s.ctxEngine != nil {
				s.ctxEngine.AddAssistantMessage(lastContent)
			}
			s.mu.Unlock()
		}

		wailsruntime.EventsEmit(s.ctx, "agent:done", nil)
		logger.Info("[CHAT] Resume completed",
			"duration_ms", time.Since(startTime).Milliseconds(),
			"transfer_count", transferCount,
		)

		s.mu.Lock()
		doneFn := s.onAgentDone
		s.mu.Unlock()
		if doneFn != nil {
			doneFn()
		}
	}()

	return nil
}

// ResumeWithChoice resumes execution after the user selects a choice.
func (s *ChatService) ResumeWithChoice(selectedIndex int) error {
	s.mu.Lock()
	pending := s.pendingInterrupt
	if pending == nil {
		s.mu.Unlock()
		return fmt.Errorf("no pending interrupt to resume")
	}
	s.pendingInterrupt = nil

	runner := s.defaultRunner
	if s.mode == "plan" {
		runner = s.planRunner
	}
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
	s.mu.Lock()
	s.cancelFn = cancel
	s.mu.Unlock()

	go func() {
		defer cancel()
		defer func() {
			if r := recover(); r != nil {
				wailsruntime.EventsEmit(s.ctx, "agent:error", fmt.Sprintf("Resume panic: %v", r))
				wailsruntime.EventsEmit(s.ctx, "agent:done", nil)
			}
		}()

		logger.Info("[CHAT] Resuming after choice", "selected", selectedIndex)
		startTime := time.Now()

		events, err := runner.ResumeWithParams(runCtx, pending.CheckpointID, &adk.ResumeParams{
			Targets: map[string]any{
				pending.InterruptID: resumeData,
			},
		})
		if err != nil {
			wailsruntime.EventsEmit(s.ctx, "agent:error", fmt.Sprintf("Resume failed: %v", err))
			wailsruntime.EventsEmit(s.ctx, "agent:done", nil)
			return
		}

		lastContent, transferCount, interrupted := s.processEvents(events, pending.CheckpointID)

		if interrupted {
			return
		}

		if lastContent != "" {
			s.mu.Lock()
			if s.ctxEngine != nil {
				s.ctxEngine.AddAssistantMessage(lastContent)
			}
			s.mu.Unlock()
		}

		wailsruntime.EventsEmit(s.ctx, "agent:done", nil)
		logger.Info("[CHAT] Resume completed",
			"duration_ms", time.Since(startTime).Milliseconds(),
			"transfer_count", transferCount,
		)

		s.mu.Lock()
		doneFn := s.onAgentDone
		s.mu.Unlock()
		if doneFn != nil {
			doneFn()
		}
	}()

	return nil
}

// StopGeneration cancels the currently running agent generation.
func (s *ChatService) StopGeneration() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancelFn != nil {
		s.cancelFn()
		s.cancelFn = nil
	}
	s.pendingInterrupt = nil
	return nil
}

// ClearHistory resets the conversation history.
func (s *ChatService) ClearHistory() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ctxEngine != nil {
		s.ctxEngine.ClearHistory()
	}
	s.timeline.Clear()
	s.streamingState = nil
	s.invalidateRunners()
	s.checkpointStore = checkpoint.NewInMemoryStore()
	s.pendingInterrupt = nil
	return nil
}

// Timeline returns the timeline collector for session persistence.
func (s *ChatService) Timeline() *agentctx.TimelineCollector {
	return s.timeline
}

// StreamingState returns the current streaming state (nil if not streaming).
func (s *ChatService) StreamingState() *model.StreamingState {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.streamingState == nil {
		return nil
	}
	ss := *s.streamingState
	return &ss
}

// BuildRunners builds the deep agent and both runners using the current config.
func (s *ChatService) BuildRunners() error {
	s.mu.Lock()
	defer s.mu.Unlock()

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

	// Connect MCP servers and load their tools
	for _, serverCfg := range cfg.MCP.Servers {
		if !serverCfg.Enabled {
			continue
		}
		session, err := tools.ConnectMCPServer(ctx, serverCfg)
		if err != nil {
			wailsruntime.EventsEmit(s.ctx, "agent:error",
				fmt.Sprintf("MCP server %s connection failed: %v", serverCfg.Name, err))
			continue
		}
		mcpTools, err := tools.LoadMCPTools(ctx, session, nil)
		if err != nil {
			wailsruntime.EventsEmit(s.ctx, "agent:error",
				fmt.Sprintf("MCP server %s tool loading failed: %v", serverCfg.Name, err))
			continue
		}
		registry.RegisterMCPTools(serverCfg.Name, mcpTools)
	}

	extraTools := registry.GetAll()

	// Build agent context
	ac := s.buildAgentContext()

	// Build the core deep agent
	deepAgent, err := agent.BuildDeepAgent(ctx, mdl, op, extraTools, ac)
	if err != nil {
		logger.Error("[RUNNER] Failed to build deep agent", err)
		return fmt.Errorf("failed to build deep agent: %w", err)
	}

	// Build default runner
	defaultRunner := agent.BuildDefaultRunner(ctx, deepAgent, s.checkpointStore)

	// Build plan runner
	planRunner, err := agent.BuildPlanRunner(ctx, mdl, deepAgent, ac, s.checkpointStore)
	if err != nil {
		logger.Error("[RUNNER] Failed to build plan runner", err)
		return fmt.Errorf("failed to build plan runner: %w", err)
	}

	s.deepAgent = deepAgent
	s.defaultRunner = defaultRunner
	s.planRunner = planRunner

	logger.RunnerEvent("runners_built", "extra_tools", len(extraTools))
	return nil
}

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
	// This makes sub-agent internal activity visible to the frontend.
	ac.OnToolEvent = func(agentName, eventType, toolName, toolArgs, toolID, result string) {
		s.emitTimeline(TimelineEvent{
			ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
			Type:      eventType,
			Agent:     agentName,
			ToolName:  toolName,
			ToolArgs:  toolArgs,
			ToolID:    toolID,
			Content:   result,
			Timestamp: time.Now().UnixMilli(),
		})
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

// drainStream iterates a MessageStream, batching stream_chunk timeline events
// with a 50ms window to reduce IPC frequency, and returns the final concatenated message.
func (s *ChatService) drainStream(stream adk.MessageStream, agentName string) (*schema.Message, error) {
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
		s.emitTimeline(TimelineEvent{
			ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
			Type:      "stream_chunk",
			Agent:     agentName,
			Content:   merged,
			Timestamp: time.Now().UnixMilli(),
		})
	}

	done := false
	for !done {
		select {
		case <-ticker.C:
			flushChunks()
			// Update streaming state for mid-stream saves
			s.mu.Lock()
			s.streamingState = &model.StreamingState{
				PartialContent: streamingContent.String(),
				AgentName:      agentName,
			}
			s.mu.Unlock()
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

	// Clear streaming state
	s.mu.Lock()
	s.streamingState = nil
	s.mu.Unlock()

	// Signal stream end
	s.emitTimeline(TimelineEvent{
		ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
		Type:      "stream_end",
		Agent:     agentName,
		Timestamp: time.Now().UnixMilli(),
	})

	if len(chunks) == 0 {
		return nil, nil
	}

	return schema.ConcatMessages(chunks)
}
