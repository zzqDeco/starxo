package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"starxo/internal/logger"
	"starxo/internal/model"
	"starxo/internal/tools"
)

const (
	runtimeTurnKindUser         = "user"
	runtimeTurnKindResumeAnswer = "resume_answer"
	runtimeTurnKindResumeChoice = "resume_choice"

	runtimeTurnPreemptTimeout        = 15 * time.Second
	runtimeTurnStartupPreemptTimeout = 5 * time.Second
)

var errRuntimeResumeNeedsNormalTurn = errors.New("resume checkpoint superseded by user turn")

type runtimeTurnItem struct {
	Kind             string
	SessionID        string
	RunID            string
	UserTurnID       string
	UserMessage      string
	CreatedAt        int64
	Mode             string
	RunnerKind       RunnerKind
	BundleGeneration uint64
	Objective        *model.RunObjective
	InterruptID      string
	Answer           string
	SelectedIndex    int
	Resumed          bool
}

type runtimeCheckpointDeleter interface {
	Delete(context.Context, string) error
}

func runtimeTurnCheckpointID(sessionID string) string {
	return "runtime-turn:" + sessionID
}

func runtimeTurnLoopStopOptions(cause string) []adk.StopOption {
	return []adk.StopOption{
		adk.WithImmediate(),
		adk.WithSkipCheckpoint(),
		adk.WithStopCause(cause),
	}
}

func explicitPlanModeRequested(userMessage string) bool {
	msg := strings.ToLower(strings.TrimSpace(userMessage))
	if msg == "" {
		return false
	}
	for _, signal := range []string{
		"plan mode", "planning mode", "plan-mode",
		"计划模式", "规划模式", "进入计划", "进入规划",
	} {
		if strings.Contains(msg, signal) {
			return true
		}
	}
	return false
}

func (s *ChatService) newRuntimeUserTurnItem(sessionID, userMessage string) runtimeTurnItem {
	now := s.now()
	return runtimeTurnItem{
		Kind:        runtimeTurnKindUser,
		SessionID:   sessionID,
		RunID:       fmt.Sprintf("run-%d", now.UnixNano()),
		UserTurnID:  fmt.Sprintf("usr-%d", now.UnixNano()),
		UserMessage: userMessage,
		CreatedAt:   now.UnixMilli(),
	}
}

func (s *ChatService) newRuntimeResumeAnswerItem(sessionID string, pending *PendingInterrupt, answer string) runtimeTurnItem {
	return runtimeTurnItem{
		Kind:             runtimeTurnKindResumeAnswer,
		SessionID:        sessionID,
		RunID:            pending.CheckpointID,
		Mode:             "",
		RunnerKind:       pending.RunnerKind,
		BundleGeneration: pending.BundleGeneration,
		Objective:        cloneRunObjective(pending.Objective),
		InterruptID:      pending.InterruptID,
		Answer:           answer,
	}
}

func (s *ChatService) newRuntimeResumeChoiceItem(sessionID string, pending *PendingInterrupt, selectedIndex int) runtimeTurnItem {
	return runtimeTurnItem{
		Kind:             runtimeTurnKindResumeChoice,
		SessionID:        sessionID,
		RunID:            pending.CheckpointID,
		Mode:             "",
		RunnerKind:       pending.RunnerKind,
		BundleGeneration: pending.BundleGeneration,
		Objective:        cloneRunObjective(pending.Objective),
		InterruptID:      pending.InterruptID,
		SelectedIndex:    selectedIndex,
	}
}

func (s *ChatService) ensureRuntimeTurnLoopLocked(sessionID string, run *SessionRun) *adk.TurnLoop[runtimeTurnItem, *schema.Message] {
	if run.turnLoop != nil {
		return run.turnLoop
	}
	loop := adk.NewTurnLoop[runtimeTurnItem, *schema.Message](adk.TurnLoopConfig[runtimeTurnItem, *schema.Message]{
		Store:        s.checkpointStore,
		CheckpointID: runtimeTurnCheckpointID(sessionID),
		GenInput:     s.runtimeTurnLoopGenInput(sessionID),
		GenResume:    s.runtimeTurnLoopGenResume(sessionID),
		PrepareAgent: s.runtimeTurnLoopPrepareAgent(sessionID),
		OnAgentEvents: func(ctx context.Context, tc *adk.TurnContext[runtimeTurnItem, *schema.Message], events *adk.AsyncIterator[*adk.AgentEvent]) error {
			return s.runtimeTurnLoopOnAgentEvents(ctx, tc, events)
		},
	})
	run.turnLoop = loop
	run.turnLoopCancel = nil
	run.turnLoopDone = make(chan struct{})
	run.turnLoopStarted = false
	run.cancelFn = func() {
		loop.Stop(runtimeTurnLoopStopOptions("user_stop")...)
	}
	return loop
}

func (s *ChatService) startRuntimeTurnLoopLocked(sessionID string, run *SessionRun, loop *adk.TurnLoop[runtimeTurnItem, *schema.Message]) {
	if run == nil || loop == nil || run.turnLoopStarted {
		return
	}
	run.turnLoopStarted = true
	loopCtx := s.ctx
	if loopCtx == nil {
		loopCtx = context.Background()
	}
	var cancel context.CancelFunc
	loopCtx, cancel = context.WithCancel(loopCtx)
	run.turnLoopCancel = cancel
	done := run.turnLoopDone
	loop.Run(loopCtx)
	go func() {
		exit := loop.Wait()
		s.finishRuntimeTurnLoop(sessionID, loop, exit)
		if done != nil {
			close(done)
		}
	}()
}

func (s *ChatService) resetRuntimeTurnLoopLocked(run *SessionRun) {
	if run == nil {
		return
	}
	if run.turnLoopCancel != nil {
		run.turnLoopCancel()
	}
	run.turnLoop = nil
	run.turnLoopCancel = nil
	run.turnLoopDone = nil
	run.turnLoopStarted = false
	run.cancelFn = nil
}

func (s *ChatService) runtimeTurnLoopGenInput(sessionID string) func(context.Context, *adk.TurnLoop[runtimeTurnItem, *schema.Message], []runtimeTurnItem) (*adk.GenInputResult[runtimeTurnItem, *schema.Message], error) {
	return func(ctx context.Context, loop *adk.TurnLoop[runtimeTurnItem, *schema.Message], items []runtimeTurnItem) (*adk.GenInputResult[runtimeTurnItem, *schema.Message], error) {
		if len(items) == 0 {
			return nil, fmt.Errorf("runtime turn input is empty")
		}
		item := items[0]
		if item.Kind != runtimeTurnKindUser {
			return nil, fmt.Errorf("runtime turn %s requires checkpoint resume", item.Kind)
		}
		remaining := append([]runtimeTurnItem(nil), items[1:]...)
		startCtx, startCancel := context.WithCancel(ctx)
		defer startCancel()

		s.mu.Lock()
		run := s.sessions[sessionID]
		if run == nil {
			s.mu.Unlock()
			return nil, fmt.Errorf("session %s not found", sessionID)
		}
		run.starting = true
		run.startDone = make(chan struct{})
		run.cancelFn = func() {
			startCancel()
			loop.Stop(runtimeTurnLoopStopOptions("user_stop")...)
		}
		mode := run.mode
		changedMode := false
		if mode == model.ModeDefault && explicitPlanModeRequested(item.UserMessage) {
			mode = model.ModePlan
			run.mode = mode
			changedMode = true
		}
		sessionSvc := s.sessionService
		appCtx := s.ctx
		s.mu.Unlock()
		if changedMode {
			logger.Info("[CHAT] Mode changed from explicit user request", "mode", mode, "session", sessionID)
			if appCtx != nil {
				wailsEmit(appCtx, "agent:mode_changed", ModeChangedEvent{
					Mode:      mode,
					SessionID: sessionID,
				})
			}
			if sessionSvc != nil {
				if err := sessionSvc.SaveSessionByID(sessionID); err != nil {
					logger.Warn("[CHAT] Failed to schedule explicit mode save", "session", sessionID, "error", err)
				}
			}
		}
		s.emitRunState(sessionID)

		run.addUserMessage(item.UserMessage)
		objective := run.beginObjective(item.UserTurnID, item.RunID, item.UserMessage, item.CreatedAt)
		run.addUserTurn(item.UserTurnID, item.UserMessage, item.CreatedAt)

		bundle, err := s.ensureBundleReadyForNewRun(startCtx, sessionID)
		if err != nil {
			startCancel()
			s.mu.Lock()
			s.finalizeStartupLocked(sessionID)
			s.mu.Unlock()
			s.emitRunState(sessionID)
			return nil, err
		}
		if err := startCtx.Err(); err != nil {
			s.mu.Lock()
			s.finalizeStartupLocked(sessionID)
			s.mu.Unlock()
			s.emitRunState(sessionID)
			return nil, err
		}

		runnerKind := runnerKindForMode(mode)
		if agentForKind(bundle, runnerKind) == nil {
			s.mu.Lock()
			s.finalizeStartupLocked(sessionID)
			s.mu.Unlock()
			s.emitRunState(sessionID)
			return nil, fmt.Errorf("agent %s unavailable for bundle generation %d", runnerKind, bundle.Generation)
		}

		done := make(chan struct{})
		s.mu.Lock()
		run, err = s.publishStartupLocked(sessionID, bundle, runnerKind, func() {
			loop.Stop(runtimeTurnLoopStopOptions("user_stop")...)
		}, done)
		s.mu.Unlock()
		if err != nil {
			s.emitRunState(sessionID)
			return nil, err
		}
		s.emitRunState(sessionID)

		item.Mode = mode
		item.RunnerKind = runnerKind
		item.BundleGeneration = bundle.Generation
		item.Objective = objective

		runCtx := contextWithSessionID(ctx, sessionID)
		runCtx = tools.ContextWithRuntimeObjective(runCtx, objective)
		messages := s.prepareMessagesForRun(sessionID, run)
		return &adk.GenInputResult[runtimeTurnItem, *schema.Message]{
			RunCtx: runCtx,
			Input: &adk.AgentInput{
				Messages:        messages,
				EnableStreaming: true,
			},
			Consumed:  []runtimeTurnItem{item},
			Remaining: remaining,
		}, nil
	}
}

func (s *ChatService) runtimeTurnLoopGenResume(sessionID string) func(context.Context, *adk.TurnLoop[runtimeTurnItem, *schema.Message], []runtimeTurnItem, []runtimeTurnItem, []runtimeTurnItem) (*adk.GenResumeResult[runtimeTurnItem, *schema.Message], error) {
	return func(ctx context.Context, loop *adk.TurnLoop[runtimeTurnItem, *schema.Message], interruptedItems, unhandledItems, newItems []runtimeTurnItem) (*adk.GenResumeResult[runtimeTurnItem, *schema.Message], error) {
		var resume runtimeTurnItem
		found := false
		remaining := make([]runtimeTurnItem, 0, len(unhandledItems)+len(newItems))
		for _, item := range append(append([]runtimeTurnItem{}, unhandledItems...), newItems...) {
			if !found && (item.Kind == runtimeTurnKindResumeAnswer || item.Kind == runtimeTurnKindResumeChoice) {
				resume = item
				found = true
				continue
			}
			remaining = append(remaining, item)
		}
		if !found {
			if runtimeTurnItemsContainUser(remaining) {
				return nil, errRuntimeResumeNeedsNormalTurn
			}
			return nil, fmt.Errorf("resume requested without resume payload")
		}
		if len(interruptedItems) == 0 {
			return nil, fmt.Errorf("resume requested without interrupted turn")
		}
		turn := interruptedItems[0]
		if turn.Objective == nil {
			turn.Objective = cloneRunObjective(resume.Objective)
		}
		if turn.InterruptID == "" {
			turn.InterruptID = resume.InterruptID
		}
		turn.Resumed = true

		s.mu.Lock()
		run := s.sessions[sessionID]
		if run == nil {
			s.mu.Unlock()
			return nil, fmt.Errorf("session %s not found", sessionID)
		}
		run.starting = true
		run.startDone = make(chan struct{})
		run.pendingStartBundleGeneration = turn.BundleGeneration
		run.stateMu.Lock()
		run.activeObjective = cloneRunObjective(turn.Objective)
		run.stateMu.Unlock()
		run.cancelFn = func() {
			loop.Stop(runtimeTurnLoopStopOptions("user_stop")...)
		}
		s.mu.Unlock()
		s.emitRunState(sessionID)

		bundle := (*RunnerBundle)(nil)
		s.mu.Lock()
		bundle = s.findBundleByGenerationLocked(turn.BundleGeneration)
		s.mu.Unlock()
		if bundle == nil {
			s.mu.Lock()
			s.finalizeStartupLocked(sessionID)
			s.mu.Unlock()
			s.emitRunState(sessionID)
			return nil, fmt.Errorf("runner bundle generation %d is no longer available for resume", turn.BundleGeneration)
		}
		if agentForKind(bundle, turn.RunnerKind) == nil {
			s.mu.Lock()
			s.finalizeStartupLocked(sessionID)
			s.mu.Unlock()
			s.emitRunState(sessionID)
			return nil, fmt.Errorf("agent %s is unavailable for bundle generation %d", turn.RunnerKind, turn.BundleGeneration)
		}

		done := make(chan struct{})
		s.mu.Lock()
		run, err := s.publishStartupLocked(sessionID, bundle, turn.RunnerKind, func() {
			loop.Stop(runtimeTurnLoopStopOptions("user_stop")...)
		}, done)
		s.mu.Unlock()
		if err != nil {
			s.emitRunState(sessionID)
			return nil, err
		}
		s.emitRunState(sessionID)

		resumeData, err := runtimeTurnResumeData(resume)
		if err != nil {
			s.finishRuntimeTurn(sessionID, run)
			return nil, err
		}
		runCtx := contextWithSessionID(ctx, sessionID)
		runCtx = tools.ContextWithRuntimeObjective(runCtx, turn.Objective)
		interruptID := resume.InterruptID
		if interruptID == "" {
			interruptID = turn.InterruptID
		}
		if interruptID == "" {
			s.finishRuntimeTurn(sessionID, run)
			return nil, fmt.Errorf("resume requested without interrupt id")
		}
		return &adk.GenResumeResult[runtimeTurnItem, *schema.Message]{
			RunCtx: runCtx,
			ResumeParams: &adk.ResumeParams{
				Targets: map[string]any{interruptID: resumeData},
			},
			Consumed:  []runtimeTurnItem{turn},
			Remaining: remaining,
		}, nil
	}
}

func runtimeTurnResumeData(item runtimeTurnItem) (any, error) {
	switch item.Kind {
	case runtimeTurnKindResumeAnswer:
		return &tools.FollowUpInfo{UserAnswer: item.Answer}, nil
	case runtimeTurnKindResumeChoice:
		return &tools.ChoiceInfo{Selected: item.SelectedIndex}, nil
	default:
		return nil, fmt.Errorf("unsupported resume turn kind %s", item.Kind)
	}
}

func (s *ChatService) runtimeTurnLoopPrepareAgent(sessionID string) func(context.Context, *adk.TurnLoop[runtimeTurnItem, *schema.Message], []runtimeTurnItem) (adk.Agent, error) {
	return func(_ context.Context, _ *adk.TurnLoop[runtimeTurnItem, *schema.Message], consumed []runtimeTurnItem) (adk.Agent, error) {
		if len(consumed) == 0 {
			return nil, fmt.Errorf("runtime turn consumed no items")
		}
		item := consumed[0]
		s.mu.Lock()
		bundle := s.findBundleByGenerationLocked(item.BundleGeneration)
		s.mu.Unlock()
		agent := agentForKind(bundle, item.RunnerKind)
		if agent == nil {
			s.mu.Lock()
			run := s.sessions[sessionID]
			if run != nil {
				s.finishRuntimeTurnLocked(run)
			}
			s.cleanupRetiredBundlesLocked()
			s.mu.Unlock()
			s.emitRunState(sessionID)
			return nil, fmt.Errorf("agent %s unavailable for bundle generation %d", item.RunnerKind, item.BundleGeneration)
		}
		return agent, nil
	}
}

func (s *ChatService) runtimeTurnLoopOnAgentEvents(_ context.Context, tc *adk.TurnContext[runtimeTurnItem, *schema.Message], events *adk.AsyncIterator[*adk.AgentEvent]) error {
	if len(tc.Consumed) == 0 {
		return fmt.Errorf("runtime turn consumed no items")
	}
	item := tc.Consumed[0]
	s.mu.Lock()
	run := s.sessions[item.SessionID]
	s.mu.Unlock()
	if run == nil {
		return fmt.Errorf("session %s not found", item.SessionID)
	}

	logger.Info("[CHAT] Agent turn started",
		"mode", item.Mode,
		"session", item.SessionID,
		"bundle_generation", item.BundleGeneration,
	)
	startTime := s.now()
	lastContent, transferCount, interrupted, preempted := s.processEventsForRun(events, runtimeTurnCheckpointID(item.SessionID), run, tc.Preempted)
	if interrupted {
		s.finishRuntimeTurn(item.SessionID, run)
		return nil
	}
	if preempted || runtimeTurnSignalClosed(tc.Preempted) {
		logger.Info("[CHAT] Agent turn preempted",
			"duration_ms", time.Since(startTime).Milliseconds(),
			"transfer_count", transferCount,
			"session", item.SessionID,
		)
		s.finishRuntimeTurn(item.SessionID, run)
		return nil
	}

	if lastContent != "" {
		run.addAssistantMessage(lastContent)
	}
	if item.Resumed {
		s.deleteRuntimeTurnCheckpoint(item.SessionID)
	}
	wailsEmit(s.ctx, "agent:done", map[string]string{
		"sessionId": item.SessionID,
	})
	logger.Info("[CHAT] Agent turn completed",
		"duration_ms", time.Since(startTime).Milliseconds(),
		"transfer_count", transferCount,
		"has_response", lastContent != "",
		"session", item.SessionID,
	)

	s.mu.Lock()
	doneFn := s.onAgentDone
	s.mu.Unlock()
	if doneFn != nil {
		doneFn(item.SessionID)
	}
	s.finishRuntimeTurn(item.SessionID, run)
	return nil
}

func (s *ChatService) finishRuntimeTurn(sessionID string, run *SessionRun) {
	s.mu.Lock()
	if current := s.sessions[sessionID]; current == run && current != nil {
		s.finishRuntimeTurnLocked(current)
	}
	s.cleanupRetiredBundlesLocked()
	s.mu.Unlock()
	s.emitRunState(sessionID)
}

func (s *ChatService) finishRuntimeTurnLocked(run *SessionRun) {
	run.running = false
	run.starting = false
	run.cancelFn = nil
	run.currentAgent = ""
	run.activeBundleGeneration = 0
	run.activeRunnerKind = ""
	run.pendingStartBundleGeneration = 0
	if run.startDone != nil {
		close(run.startDone)
		run.startDone = nil
	}
	if run.runDone != nil {
		close(run.runDone)
		run.runDone = nil
	}
}

func runtimeTurnSignalClosed(ch <-chan struct{}) bool {
	if ch == nil {
		return false
	}
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func runtimeTurnItemsContainUser(items []runtimeTurnItem) bool {
	for _, item := range items {
		if item.Kind == runtimeTurnKindUser {
			return true
		}
	}
	return false
}

func runtimeTurnUserItems(items []runtimeTurnItem) []runtimeTurnItem {
	out := make([]runtimeTurnItem, 0, len(items))
	for _, item := range items {
		if item.Kind == runtimeTurnKindUser {
			out = append(out, item)
		}
	}
	return out
}

func (s *ChatService) finishRuntimeTurnLoop(sessionID string, loop *adk.TurnLoop[runtimeTurnItem, *schema.Message], exit *adk.TurnLoopExitState[runtimeTurnItem, *schema.Message]) {
	var emitDone bool
	var emitErr error
	var recoverUserTurns []runtimeTurnItem
	s.mu.Lock()
	run := s.sessions[sessionID]
	currentLoop := run != nil && run.turnLoop == loop
	if currentLoop {
		run.turnLoop = nil
		if run.turnLoopCancel != nil {
			run.turnLoopCancel()
			run.turnLoopCancel = nil
		}
		run.turnLoopDone = nil
		run.turnLoopStarted = false
		if run.running || run.starting {
			s.finishRuntimeTurnLocked(run)
			emitDone = true
		}
	}
	if currentLoop && exit != nil && errors.Is(exit.ExitReason, errRuntimeResumeNeedsNormalTurn) {
		recoverUserTurns = runtimeTurnUserItems(exit.UnhandledItems)
		if len(recoverUserTurns) > 0 {
			emitDone = false
		}
	}
	if currentLoop && exit != nil && exit.ExitReason != nil {
		var interruptErr *adk.InterruptError
		supersededByUserTurn := errors.Is(exit.ExitReason, errRuntimeResumeNeedsNormalTurn) && len(recoverUserTurns) > 0
		if !errors.As(exit.ExitReason, &interruptErr) && !supersededByUserTurn && !runtimeTurnStopCauseSuppressesError(exit.StopCause) {
			emitErr = exit.ExitReason
		}
	}
	s.cleanupRetiredBundlesLocked()
	s.mu.Unlock()

	if len(recoverUserTurns) > 0 {
		if err := s.deleteRuntimeTurnCheckpoint(sessionID); err != nil {
			wailsEmit(s.ctx, "agent:error", map[string]interface{}{
				"sessionId": sessionID,
				"error":     fmt.Sprintf("failed to discard stale runtime checkpoint: %v", err),
			})
			s.emitRunState(sessionID)
			return
		}
		if err := s.enqueueRuntimeTurnItems(sessionID, recoverUserTurns); err != nil {
			wailsEmit(s.ctx, "agent:error", map[string]interface{}{
				"sessionId": sessionID,
				"error":     err.Error(),
			})
			s.emitRunState(sessionID)
		}
		return
	}

	if emitErr != nil {
		wailsEmit(s.ctx, "agent:error", map[string]interface{}{
			"sessionId": sessionID,
			"error":     emitErr.Error(),
		})
		emitDone = true
	}
	if emitDone {
		wailsEmit(s.ctx, "agent:done", map[string]string{"sessionId": sessionID})
		s.emitRunState(sessionID)
	}
}

func runtimeTurnStopCauseSuppressesError(cause string) bool {
	switch strings.TrimSpace(cause) {
	case "user_stop", "sandbox_lost", "sandbox_changed":
		return true
	default:
		return false
	}
}

func (s *ChatService) enqueueRuntimeTurnItems(sessionID string, items []runtimeTurnItem) error {
	if len(items) == 0 {
		return nil
	}
	s.mu.Lock()
	run := s.sessions[sessionID]
	if run == nil {
		s.mu.Unlock()
		return fmt.Errorf("session %s not found", sessionID)
	}
	loop := s.ensureRuntimeTurnLoopLocked(sessionID, run)
	for _, item := range items {
		if item.Kind != runtimeTurnKindUser {
			continue
		}
		if ok, _ := loop.Push(item); !ok {
			if run.turnLoop == loop {
				s.resetRuntimeTurnLoopLocked(run)
			}
			loop = s.ensureRuntimeTurnLoopLocked(sessionID, run)
			if ok, _ = loop.Push(item); !ok {
				s.mu.Unlock()
				return fmt.Errorf("runtime turn loop is stopped for session %s", sessionID)
			}
		}
	}
	if run.turnLoop == loop {
		s.startRuntimeTurnLoopLocked(sessionID, run, loop)
	}
	s.mu.Unlock()
	s.emitRunState(sessionID)
	return nil
}

func (s *ChatService) deleteRuntimeTurnCheckpoint(sessionID string) error {
	if s.checkpointStore == nil {
		return nil
	}
	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	checkpointID := runtimeTurnCheckpointID(sessionID)
	if deleter, ok := s.checkpointStore.(runtimeCheckpointDeleter); ok {
		if err := deleter.Delete(ctx, checkpointID); err == nil {
			return nil
		} else {
			logger.Warn("[CHAT] Failed to delete runtime turn checkpoint; falling back to tombstone", "session", sessionID, "error", err)
		}
	}
	if err := s.checkpointStore.Set(ctx, checkpointID, nil); err != nil {
		logger.Warn("[CHAT] Failed to tombstone runtime turn checkpoint", "session", sessionID, "error", err)
		return err
	}
	return nil
}
