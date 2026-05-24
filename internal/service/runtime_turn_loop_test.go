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
