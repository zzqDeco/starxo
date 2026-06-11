package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"starxo/internal/config"
	"starxo/internal/model"
	"starxo/internal/tools"
)

type runtimeTurnTestAgent struct{}

func (runtimeTurnTestAgent) Name(context.Context) string        { return "runtime_turn_test" }
func (runtimeTurnTestAgent) Description(context.Context) string { return "runtime turn test agent" }
func (runtimeTurnTestAgent) Run(context.Context, *adk.AgentInput, ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	go gen.Close()
	return iter
}

type runtimeTurnRecordingAgent struct {
	calls chan []*schema.Message
}

func (a *runtimeTurnRecordingAgent) Name(context.Context) string { return "runtime_turn_recording" }
func (a *runtimeTurnRecordingAgent) Description(context.Context) string {
	return "runtime turn recording agent"
}
func (a *runtimeTurnRecordingAgent) Run(_ context.Context, input *adk.AgentInput, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	if a.calls != nil {
		messages := append([]*schema.Message(nil), input.Messages...)
		a.calls <- messages
	}
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	go gen.Close()
	return iter
}

func TestRuntimeTurnLoopGenResumeUsesInterruptedObjective(t *testing.T) {
	chat := NewChatService(nil)
	sessionID := "sess-turn-resume"
	chat.mu.Lock()
	run := chat.getOrCreateRun(sessionID)
	chat.activeSessionID = sessionID
	run.stateMu.Lock()
	run.activeObjective = &model.RunObjective{ID: "obj-current", Objective: "new unrelated task"}
	run.stateMu.Unlock()
	chat.installedBundle = &RunnerBundle{
		Generation:   7,
		ConfigDigest: "digest",
		DefaultAgent: runtimeTurnTestAgent{},
		PlanAgent:    runtimeTurnTestAgent{},
	}
	chat.mu.Unlock()

	interruptedObjective := &model.RunObjective{ID: "obj-interrupted", Objective: "original interrupted task"}
	result, err := chat.runtimeTurnLoopGenResume(sessionID)(
		context.Background(),
		nil,
		[]runtimeTurnItem{{
			Kind:             runtimeTurnKindUser,
			SessionID:        sessionID,
			BundleGeneration: 7,
			RunnerKind:       RunnerKindDefault,
			Objective:        interruptedObjective,
			InterruptID:      "interrupt-1",
		}},
		nil,
		[]runtimeTurnItem{{
			Kind:        runtimeTurnKindResumeAnswer,
			SessionID:   sessionID,
			InterruptID: "interrupt-1",
			Answer:      "continue",
			Objective:   interruptedObjective,
		}},
	)
	if err != nil {
		t.Fatalf("gen resume: %v", err)
	}
	objective, ok := tools.RuntimeObjectiveFromContext(result.RunCtx)
	if !ok {
		t.Fatalf("expected runtime objective in resume context")
	}
	if objective.ID != interruptedObjective.ID {
		t.Fatalf("expected interrupted objective %q, got %q", interruptedObjective.ID, objective.ID)
	}
	if result.ResumeParams == nil || result.ResumeParams.Targets["interrupt-1"] == nil {
		t.Fatalf("expected resume params for original interrupt, got %#v", result.ResumeParams)
	}
	if len(result.Consumed) != 1 || !result.Consumed[0].Resumed {
		t.Fatalf("expected consumed turn to be marked resumed, got %#v", result.Consumed)
	}
}

func TestRuntimeTurnStopCauseSuppressesAdministrativeErrors(t *testing.T) {
	if !runtimeTurnStopCauseSuppressesError("user_stop") {
		t.Fatalf("expected user_stop to suppress loop exit errors")
	}
	if !runtimeTurnStopCauseSuppressesError("sandbox_lost") {
		t.Fatalf("expected sandbox_lost to suppress loop exit errors")
	}
	if runtimeTurnStopCauseSuppressesError("model_error") {
		t.Fatalf("expected non-administrative stop cause to report loop exit errors")
	}
}

func TestRuntimeTurnCheckpointIDIsSessionScoped(t *testing.T) {
	if got := runtimeTurnCheckpointID("sess-1"); got != "runtime-turn:sess-1" {
		t.Fatalf("unexpected checkpoint id %q", got)
	}
}

func TestRuntimeTurnLoopGenInputPersistsUserTurnBeforeBundleFailure(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	store, err := config.NewStore()
	if err != nil {
		t.Fatalf("new config store: %v", err)
	}
	chat := NewChatService(store)
	sessionID := "sess-bundle-fail"
	chat.mu.Lock()
	run := chat.getOrCreateRun(sessionID)
	chat.activeSessionID = sessionID
	chat.mu.Unlock()
	bundleErr := errors.New("bundle setup failed")
	chat.prepareRunnerBundleFn = func(context.Context, *config.AppConfig, string, map[string]cachedMCPServerSurface) (*RunnerBundle, error) {
		return nil, bundleErr
	}

	item := chat.newRuntimeUserTurnItem(sessionID, "write a short note")
	_, err = chat.runtimeTurnLoopGenInput(sessionID)(context.Background(), nil, []runtimeTurnItem{item})
	if !errors.Is(err, bundleErr) {
		t.Fatalf("expected bundle error, got %v", err)
	}
	messages := run.ctxEngine.ExportMessages()
	if len(messages) != 1 || messages[0].Role != string(schema.User) || messages[0].Content != item.UserMessage {
		t.Fatalf("expected user message to persist before bundle failure, got %#v", messages)
	}
	objective := run.currentObjective()
	if objective == nil || objective.UserMessageID != item.UserTurnID || objective.Objective != item.UserMessage {
		t.Fatalf("expected objective to persist before bundle failure, got %#v", objective)
	}
}

func TestRuntimeTurnLoopGenInputKeepsComplexDefaultTurnInDefaultMode(t *testing.T) {
	store := newTestConfigStore(t)
	chat := NewChatService(store)
	sessionID := "sess-default-complex"
	chat.SetActiveSessionID(sessionID)
	targetDigest := mustConfigDigest(t, chat)
	chat.prepareRunnerBundleFn = func(context.Context, *config.AppConfig, string, map[string]cachedMCPServerSurface) (*RunnerBundle, error) {
		return &RunnerBundle{
			ConfigDigest: targetDigest,
			DefaultAgent: runtimeTurnTestAgent{},
			PlanAgent:    runtimeTurnTestAgent{},
		}, nil
	}
	chat.mu.Lock()
	run := chat.getOrCreateRun(sessionID)
	run.mode = model.ModeDefault
	chat.mu.Unlock()

	item := chat.newRuntimeUserTurnItem(sessionID, "write a file, then run tests, then verify the result")
	result, err := chat.runtimeTurnLoopGenInput(sessionID)(context.Background(), nil, []runtimeTurnItem{item})
	if err != nil {
		t.Fatalf("gen input: %v", err)
	}
	if len(result.Consumed) != 1 {
		t.Fatalf("expected one consumed item, got %#v", result.Consumed)
	}
	if result.Consumed[0].RunnerKind != RunnerKindDefault {
		t.Fatalf("expected default runner for complex default turn, got %s", result.Consumed[0].RunnerKind)
	}
	chat.mu.Lock()
	mode := run.mode
	chat.mu.Unlock()
	if mode != model.ModeDefault {
		t.Fatalf("expected run mode to remain default, got %s", mode)
	}
}

func TestRuntimeTurnLoopGenInputAllowsExplicitPlanModeRequest(t *testing.T) {
	store := newTestConfigStore(t)
	chat := NewChatService(store)
	sessionID := "sess-explicit-plan"
	chat.SetActiveSessionID(sessionID)
	targetDigest := mustConfigDigest(t, chat)
	chat.prepareRunnerBundleFn = func(context.Context, *config.AppConfig, string, map[string]cachedMCPServerSurface) (*RunnerBundle, error) {
		return &RunnerBundle{
			ConfigDigest: targetDigest,
			DefaultAgent: runtimeTurnTestAgent{},
			PlanAgent:    runtimeTurnTestAgent{},
		}, nil
	}
	chat.mu.Lock()
	run := chat.getOrCreateRun(sessionID)
	run.mode = model.ModeDefault
	chat.mu.Unlock()

	item := chat.newRuntimeUserTurnItem(sessionID, "进入计划模式，先给我方案")
	result, err := chat.runtimeTurnLoopGenInput(sessionID)(context.Background(), nil, []runtimeTurnItem{item})
	if err != nil {
		t.Fatalf("gen input: %v", err)
	}
	if len(result.Consumed) != 1 {
		t.Fatalf("expected one consumed item, got %#v", result.Consumed)
	}
	if result.Consumed[0].RunnerKind != RunnerKindPlan {
		t.Fatalf("expected plan runner for explicit plan request, got %s", result.Consumed[0].RunnerKind)
	}
	chat.mu.Lock()
	mode := run.mode
	chat.mu.Unlock()
	if mode != model.ModePlan {
		t.Fatalf("expected run mode to switch to plan, got %s", mode)
	}
}

func TestSendMessageCancelsStartupBeforeReplacementTurn(t *testing.T) {
	store := newTestConfigStore(t)
	chat := NewChatService(store)
	chat.SetContext(context.Background())
	sessionID := "sess-startup-replace"
	chat.SetActiveSessionID(sessionID)
	targetDigest := mustConfigDigest(t, chat)
	enteredBuild := make(chan struct{}, 1)
	releaseBuild := make(chan struct{})
	recorder := &runtimeTurnRecordingAgent{calls: make(chan []*schema.Message, 2)}
	chat.prepareRunnerBundleFn = func(context.Context, *config.AppConfig, string, map[string]cachedMCPServerSurface) (*RunnerBundle, error) {
		select {
		case enteredBuild <- struct{}{}:
		default:
		}
		<-releaseBuild
		return &RunnerBundle{
			ConfigDigest: targetDigest,
			DefaultAgent: recorder,
			PlanAgent:    recorder,
		}, nil
	}

	if err := chat.SendMessage("first stale startup objective"); err != nil {
		t.Fatalf("send first message: %v", err)
	}
	select {
	case <-enteredBuild:
	case <-time.After(time.Second):
		t.Fatalf("expected first startup to begin bundle preparation")
	}
	if err := chat.SendMessage("second replacement objective"); err != nil {
		t.Fatalf("send replacement message: %v", err)
	}
	close(releaseBuild)

	var messages []*schema.Message
	select {
	case messages = <-recorder.calls:
	case <-time.After(time.Second):
		t.Fatalf("expected replacement turn to run after startup cancellation")
	}
	if !runtimeTurnMessagesContain(messages, "second replacement objective") {
		t.Fatalf("expected replacement objective in agent input, got %#v", messages)
	}
	select {
	case extra := <-recorder.calls:
		t.Fatalf("expected stale startup turn not to run, got extra call %#v", extra)
	case <-time.After(100 * time.Millisecond):
	}
	chat.mu.Lock()
	run := chat.sessions[sessionID]
	chat.mu.Unlock()
	objective := run.currentObjective()
	if objective == nil || objective.Objective != "second replacement objective" {
		t.Fatalf("expected active objective to be replacement, got %#v", objective)
	}
}

func runtimeTurnMessagesContain(messages []*schema.Message, content string) bool {
	for _, msg := range messages {
		if msg != nil && msg.Content == content {
			return true
		}
	}
	return false
}

type runtimeTurnTrackingCheckpointStore struct {
	mu      sync.Mutex
	deleted []string
	values  map[string][]byte
}

func newRuntimeTurnTrackingCheckpointStore() *runtimeTurnTrackingCheckpointStore {
	return &runtimeTurnTrackingCheckpointStore{values: make(map[string][]byte)}
}

func (s *runtimeTurnTrackingCheckpointStore) Get(_ context.Context, key string) ([]byte, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.values[key]
	return append([]byte{}, v...), ok, nil
}

func (s *runtimeTurnTrackingCheckpointStore) Set(_ context.Context, key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = append([]byte{}, value...)
	return nil
}

func (s *runtimeTurnTrackingCheckpointStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleted = append(s.deleted, key)
	delete(s.values, key)
	return nil
}

func (s *runtimeTurnTrackingCheckpointStore) deletedKey(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, deleted := range s.deleted {
		if deleted == key {
			return true
		}
	}
	return false
}

var _ compose.CheckPointStore = (*runtimeTurnTrackingCheckpointStore)(nil)
var _ runtimeCheckpointDeleter = (*runtimeTurnTrackingCheckpointStore)(nil)

type runtimeTurnTombstoneCheckpointStore struct {
	mu     sync.Mutex
	values map[string][]byte
}

func newRuntimeTurnTombstoneCheckpointStore() *runtimeTurnTombstoneCheckpointStore {
	return &runtimeTurnTombstoneCheckpointStore{values: make(map[string][]byte)}
}

func (s *runtimeTurnTombstoneCheckpointStore) Get(_ context.Context, key string) ([]byte, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.values[key]
	return append([]byte{}, v...), ok, nil
}

func (s *runtimeTurnTombstoneCheckpointStore) Set(_ context.Context, key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = append([]byte{}, value...)
	return nil
}

var _ compose.CheckPointStore = (*runtimeTurnTombstoneCheckpointStore)(nil)

func TestSendMessageDeletesStaleRuntimeCheckpointBeforeNormalTurn(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	store, err := config.NewStore()
	if err != nil {
		t.Fatalf("new config store: %v", err)
	}
	chat := NewChatService(store)
	checkpoints := newRuntimeTurnTrackingCheckpointStore()
	chat.checkpointStore = checkpoints
	chat.SetActiveSessionID("sess-stale")

	_, digest, err := chat.currentConfigSnapshot()
	if err != nil {
		t.Fatalf("config snapshot: %v", err)
	}
	chat.mu.Lock()
	chat.installedBundle = &RunnerBundle{
		Generation:           1,
		ConfigDigest:         digest,
		DefaultAgent:         runtimeTurnTestAgent{},
		PlanAgent:            runtimeTurnTestAgent{},
		LastFreshnessCheckAt: time.Now(),
	}
	chat.nextGeneration = 1
	chat.mu.Unlock()

	if err := chat.SendMessage("start a fresh task"); err != nil {
		t.Fatalf("send message: %v", err)
	}
	if !checkpoints.deletedKey(runtimeTurnCheckpointID("sess-stale")) {
		t.Fatalf("expected normal user turn to delete stale runtime checkpoint")
	}
}

func TestDeleteRuntimeTurnCheckpointTombstonesWhenDeleteUnsupported(t *testing.T) {
	chat := NewChatService(nil)
	checkpoints := newRuntimeTurnTombstoneCheckpointStore()
	chat.checkpointStore = checkpoints
	key := runtimeTurnCheckpointID("sess-tombstone")
	if err := checkpoints.Set(context.Background(), key, []byte("stale-checkpoint")); err != nil {
		t.Fatalf("seed checkpoint: %v", err)
	}

	if err := chat.deleteRuntimeTurnCheckpoint("sess-tombstone"); err != nil {
		t.Fatalf("delete checkpoint: %v", err)
	}
	value, ok, err := checkpoints.Get(context.Background(), key)
	if err != nil {
		t.Fatalf("get checkpoint: %v", err)
	}
	if !ok {
		t.Fatalf("expected tombstone value to remain present")
	}
	if len(value) != 0 {
		t.Fatalf("expected zero-length tombstone, got %q", string(value))
	}
}

func TestRuntimeTurnLoopRecoversUserTurnWhenResumePayloadMissing(t *testing.T) {
	store := newTestConfigStore(t)
	chat := NewChatService(store)
	chat.SetContext(context.Background())
	sessionID := "sess-resume-superseded"
	chat.SetActiveSessionID(sessionID)
	recorder := &runtimeTurnRecordingAgent{calls: make(chan []*schema.Message, 1)}
	chat.mu.Lock()
	run := chat.getOrCreateRun(sessionID)
	loop := chat.ensureRuntimeTurnLoopLocked(sessionID, run)
	chat.installedBundle = &RunnerBundle{
		Generation:           1,
		ConfigDigest:         mustConfigDigest(t, chat),
		DefaultAgent:         recorder,
		PlanAgent:            recorder,
		LastFreshnessCheckAt: time.Now(),
	}
	chat.nextGeneration = 1
	chat.mu.Unlock()

	replacement := chat.newRuntimeUserTurnItem(sessionID, "replace stale interrupt with a new objective")
	exit := &adk.TurnLoopExitState[runtimeTurnItem, *schema.Message]{
		ExitReason:     errRuntimeResumeNeedsNormalTurn,
		UnhandledItems: []runtimeTurnItem{replacement},
	}
	chat.finishRuntimeTurnLoop(sessionID, loop, exit)

	var messages []*schema.Message
	select {
	case messages = <-recorder.calls:
	case <-time.After(time.Second):
		t.Fatalf("expected superseding user turn to run after stale checkpoint cleanup")
	}
	if !runtimeTurnMessagesContain(messages, replacement.UserMessage) {
		t.Fatalf("expected replacement objective in agent input, got %#v", messages)
	}
}

func TestProcessEventsForRunPreemptDropsUnresolvedToolCallHistory(t *testing.T) {
	chat := NewChatService(nil)
	sessionID := "sess-preempt"
	chat.mu.Lock()
	run := chat.getOrCreateRun(sessionID)
	chat.mu.Unlock()

	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	go func() {
		gen.Send(&adk.AgentEvent{
			AgentName: "coding_agent",
			Output: &adk.AgentOutput{MessageOutput: &adk.MessageVariant{Message: &schema.Message{
				Role: schema.Assistant,
				ToolCalls: []schema.ToolCall{{
					ID:       "call-preempt",
					Function: schema.FunctionCall{Name: "Bash", Arguments: `{"command":"sleep 10"}`},
				}},
			}}},
		})
		gen.Close()
	}()
	preempted := make(chan struct{})
	close(preempted)

	_, _, interrupted, wasPreempted := chat.processEventsForRun(iter, runtimeTurnCheckpointID(sessionID), run, preempted)
	if interrupted {
		t.Fatalf("did not expect interrupt")
	}
	if !wasPreempted {
		t.Fatalf("expected preempted turn")
	}
	messages := run.ctxEngine.ExportMessages()
	for _, msg := range messages {
		if msg.ToolCallID == "call-preempt" || len(msg.ToolCalls) > 0 {
			t.Fatalf("expected preempted unresolved tool call history to be removed, got %#v", messages)
		}
		if msg.Content == "Error: tool execution failed or was interrupted" {
			t.Fatalf("expected no synthetic tool failure for preempted turn, got %#v", messages)
		}
	}
}

func TestProcessEventsForRunPreemptKeepsOlderDuplicateToolCallHistory(t *testing.T) {
	chat := NewChatService(nil)
	sessionID := "sess-preempt-duplicate"
	chat.mu.Lock()
	run := chat.getOrCreateRun(sessionID)
	chat.mu.Unlock()
	run.addMessage(&schema.Message{
		Role:    schema.Assistant,
		Content: "older completed tool call",
		ToolCalls: []schema.ToolCall{{
			ID:       "call-preempt",
			Function: schema.FunctionCall{Name: "Read", Arguments: `{"file_path":"old.txt"}`},
		}},
	})
	run.addMessage(&schema.Message{
		Role:       schema.Tool,
		ToolCallID: "call-preempt",
		Content:    "older result",
	})

	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	go func() {
		gen.Send(&adk.AgentEvent{
			AgentName: "coding_agent",
			Output: &adk.AgentOutput{MessageOutput: &adk.MessageVariant{Message: &schema.Message{
				Role: schema.Assistant,
				ToolCalls: []schema.ToolCall{{
					ID:       "call-preempt",
					Function: schema.FunctionCall{Name: "Bash", Arguments: `{"command":"sleep 10"}`},
				}},
			}}},
		})
		gen.Close()
	}()
	preempted := make(chan struct{})
	close(preempted)

	_, _, interrupted, wasPreempted := chat.processEventsForRun(iter, runtimeTurnCheckpointID(sessionID), run, preempted)
	if interrupted {
		t.Fatalf("did not expect interrupt")
	}
	if !wasPreempted {
		t.Fatalf("expected preempted turn")
	}
	messages := run.ctxEngine.ExportMessages()
	assistantToolCalls := 0
	toolResults := 0
	for _, msg := range messages {
		if msg.Content == "Error: tool execution failed or was interrupted" {
			t.Fatalf("expected no synthetic tool failure for preempted turn, got %#v", messages)
		}
		for _, tc := range msg.ToolCalls {
			if tc.ID == "call-preempt" {
				assistantToolCalls++
				if msg.Content != "older completed tool call" {
					t.Fatalf("expected only older duplicate tool call to remain, got %#v", messages)
				}
			}
		}
		if msg.ToolCallID == "call-preempt" {
			toolResults++
			if msg.Content != "older result" {
				t.Fatalf("expected only older duplicate tool result to remain, got %#v", messages)
			}
		}
	}
	if assistantToolCalls != 1 || toolResults != 1 {
		t.Fatalf("expected older duplicate call/result to remain exactly once, calls=%d results=%d messages=%#v", assistantToolCalls, toolResults, messages)
	}
}

func TestProcessEventsForRunInterruptTracksPendingToolCallIDs(t *testing.T) {
	chat := NewChatService(nil)
	sessionID := "sess-interrupt-tool-call"
	chat.mu.Lock()
	run := chat.getOrCreateRun(sessionID)
	chat.mu.Unlock()

	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	go func() {
		gen.Send(&adk.AgentEvent{
			AgentName: "coding_agent",
			Output: &adk.AgentOutput{MessageOutput: &adk.MessageVariant{Message: &schema.Message{
				Role: schema.Assistant,
				ToolCalls: []schema.ToolCall{{
					ID:       "call-ask",
					Function: schema.FunctionCall{Name: "ask_user", Arguments: `{"questions":["path?"]}`},
				}},
			}}},
		})
		gen.Send(&adk.AgentEvent{
			AgentName: "coding_agent",
			Action: &adk.AgentAction{Interrupted: &adk.InterruptInfo{InterruptContexts: []*adk.InterruptCtx{{
				ID:   "interrupt-ask",
				Info: &tools.FollowUpInfo{Questions: []string{"path?"}},
			}}}},
		})
		gen.Close()
	}()

	_, _, interrupted, _ := chat.processEventsForRun(iter, runtimeTurnCheckpointID(sessionID), run, nil)
	if !interrupted {
		t.Fatalf("expected business interrupt")
	}
	chat.mu.Lock()
	pending := run.pendingInterrupt
	chat.mu.Unlock()
	if pending == nil {
		t.Fatalf("expected pending interrupt")
	}
	if len(pending.ToolCallIDs) != 1 || pending.ToolCallIDs[0] != "call-ask" {
		t.Fatalf("expected pending interrupt to record tool call id, got %#v", pending.ToolCallIDs)
	}
}

func TestSendMessageSupersedesPendingInterruptRepairsOrphanToolHistory(t *testing.T) {
	store := newTestConfigStore(t)
	chat := NewChatService(store)
	chat.SetContext(context.Background())
	sessionID := "sess-supersede-interrupt"
	chat.SetActiveSessionID(sessionID)
	chat.mu.Lock()
	run := chat.getOrCreateRun(sessionID)
	run.addMessage(&schema.Message{
		Role:    schema.Assistant,
		Content: "need input",
		ToolCalls: []schema.ToolCall{{
			ID:       "call-ask",
			Function: schema.FunctionCall{Name: "ask_user", Arguments: `{"questions":["path?"]}`},
		}},
	})
	run.pendingInterrupt = &PendingInterrupt{
		CheckpointID:     runtimeTurnCheckpointID(sessionID),
		InterruptID:      "interrupt-ask",
		BundleGeneration: 1,
		RunnerKind:       RunnerKindDefault,
		Info:             &tools.FollowUpInfo{Questions: []string{"path?"}},
		ToolCallIDs:      []string{"call-ask"},
	}
	chat.installedBundle = &RunnerBundle{
		Generation:           1,
		ConfigDigest:         mustConfigDigest(t, chat),
		DefaultAgent:         runtimeTurnTestAgent{},
		PlanAgent:            runtimeTurnTestAgent{},
		LastFreshnessCheckAt: time.Now(),
	}
	chat.nextGeneration = 1
	chat.mu.Unlock()

	if err := chat.SendMessage("write a new standalone file"); err != nil {
		t.Fatalf("send superseding message: %v", err)
	}
	chat.mu.Lock()
	pending := run.pendingInterrupt
	chat.mu.Unlock()
	if pending != nil {
		t.Fatalf("expected pending interrupt to be cleared")
	}
	messages := run.ctxEngine.ExportMessages()
	assertPersistedToolPairing(t, messages)
	foundRepair := false
	for i, msg := range messages {
		if msg.Role == string(schema.Tool) && msg.ToolCallID == "call-ask" {
			foundRepair = true
			if msg.Content != orphanRepairSupersededReason {
				t.Fatalf("unexpected repair content %q", msg.Content)
			}
			if i == 0 || len(messages[i-1].ToolCalls) == 0 {
				t.Fatalf("repair result is not adjacent to assistant tool call: %#v", messages)
			}
		}
	}
	if !foundRepair {
		t.Fatalf("expected synthetic repair result for interrupted ask_user call, got %#v", messages)
	}
}

func TestStopGenerationRepairsIdlePendingInterrupt(t *testing.T) {
	chat := NewChatService(nil)
	sessionID := "sess-stop-interrupt"
	chat.SetActiveSessionID(sessionID)
	chat.mu.Lock()
	run := chat.getOrCreateRun(sessionID)
	run.addMessage(&schema.Message{
		Role: schema.Assistant,
		ToolCalls: []schema.ToolCall{{
			ID:       "call-stop",
			Function: schema.FunctionCall{Name: "ask_choice", Arguments: `{}`},
		}},
	})
	run.pendingInterrupt = &PendingInterrupt{
		CheckpointID: runtimeTurnCheckpointID(sessionID),
		InterruptID:  "interrupt-stop",
		RunnerKind:   RunnerKindDefault,
		ToolCallIDs:  []string{"call-stop"},
	}
	chat.mu.Unlock()

	if err := chat.StopGeneration(); err != nil {
		t.Fatalf("stop generation: %v", err)
	}
	messages := run.ctxEngine.ExportMessages()
	assertPersistedToolPairing(t, messages)
	found := false
	for _, msg := range messages {
		if msg.Role == string(schema.Tool) && msg.ToolCallID == "call-stop" {
			found = true
			if msg.Content != orphanRepairStoppedReason {
				t.Fatalf("unexpected stop repair content %q", msg.Content)
			}
		}
	}
	if !found {
		t.Fatalf("expected stop repair result, got %#v", messages)
	}
}

func TestRestoreSessionDataRepairsLegacyOrphanToolHistory(t *testing.T) {
	chat := NewChatService(nil)
	sessionID := "sess-legacy-orphan"
	chat.RestoreSessionData(sessionID, &model.SessionData{
		Version: model.SessionDataVersion,
		Messages: []model.PersistedMessage{
			{
				Role:    string(schema.Assistant),
				Content: "need input",
				ToolCalls: []model.PersistedToolCall{{
					ID: "call-legacy",
					Function: model.PersistedToolCallFunction{
						Name:      "ask_user",
						Arguments: `{"questions":["path?"]}`,
					},
				}},
			},
			{Role: string(schema.User), Content: "new task"},
		},
	})

	snapshot, err := chat.ExportSessionSnapshot(sessionID)
	if err != nil {
		t.Fatalf("export snapshot: %v", err)
	}
	messages := snapshot.SessionData.Messages
	assertPersistedToolPairing(t, messages)
	if len(messages) != 3 {
		t.Fatalf("expected repaired assistant/tool/user history, got %#v", messages)
	}
	if messages[1].Role != string(schema.Tool) || messages[1].ToolCallID != "call-legacy" {
		t.Fatalf("expected repaired tool result before user message, got %#v", messages)
	}
	if messages[2].Role != string(schema.User) || messages[2].Content != "new task" {
		t.Fatalf("expected user message after repaired tool group, got %#v", messages)
	}
}

func assertPersistedToolPairing(t *testing.T, messages []model.PersistedMessage) {
	t.Helper()
	for i := 0; i < len(messages); i++ {
		msg := messages[i]
		if msg.Role == string(schema.Tool) && msg.ToolCallID != "" {
			t.Fatalf("tool result %q at index %d is not attached to an assistant tool call group: %#v", msg.ToolCallID, i, messages)
		}
		if len(msg.ToolCalls) == 0 {
			continue
		}
		for _, tc := range msg.ToolCalls {
			i++
			if i >= len(messages) {
				t.Fatalf("missing tool result for %q at end of messages: %#v", tc.ID, messages)
			}
			result := messages[i]
			if result.Role != string(schema.Tool) || result.ToolCallID != tc.ID {
				t.Fatalf("expected tool result for %q at index %d, got %#v in %#v", tc.ID, i, result, messages)
			}
		}
	}
}

func TestRemoveSessionStopsIdleRuntimeTurnLoop(t *testing.T) {
	chat := NewChatService(nil)
	sessionID := "sess-remove-loop"
	chat.mu.Lock()
	run := chat.getOrCreateRun(sessionID)
	loop := chat.ensureRuntimeTurnLoopLocked(sessionID, run)
	chat.startRuntimeTurnLoopLocked(sessionID, run, loop)
	done := run.turnLoopDone
	chat.mu.Unlock()
	if done == nil {
		t.Fatalf("expected runtime turn loop done channel")
	}

	chat.RemoveSession(sessionID)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("expected idle runtime turn loop to stop when session is removed")
	}
	chat.mu.Lock()
	_, exists := chat.sessions[sessionID]
	chat.mu.Unlock()
	if exists {
		t.Fatalf("expected session to be removed")
	}
}

func TestRemoveSessionFinalizesActiveRuntimeTurnLoopRun(t *testing.T) {
	chat := NewChatService(nil)
	sessionID := "sess-remove-active-loop"
	chat.mu.Lock()
	run := chat.getOrCreateRun(sessionID)
	loop := chat.ensureRuntimeTurnLoopLocked(sessionID, run)
	chat.startRuntimeTurnLoopLocked(sessionID, run, loop)
	run.running = true
	run.runDone = make(chan struct{})
	runDone := run.runDone
	loopDone := run.turnLoopDone
	chat.mu.Unlock()
	if runDone == nil || loopDone == nil {
		t.Fatalf("expected active run and turn loop done channels")
	}

	chat.RemoveSession(sessionID)
	select {
	case <-runDone:
	case <-time.After(time.Second):
		t.Fatalf("expected active run completion channel to close when session is removed")
	}
	select {
	case <-loopDone:
	case <-time.After(time.Second):
		t.Fatalf("expected active runtime turn loop to stop when session is removed")
	}
}
