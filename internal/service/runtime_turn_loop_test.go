package service

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/adk"

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
