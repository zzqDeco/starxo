package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"starxo/internal/sandbox"
)

func TestUpdateSandboxNilCancelsRunningAgentRuns(t *testing.T) {
	chat := NewChatService(nil)
	cancelled := false

	chat.mu.Lock()
	run := chat.getOrCreateRun("sess-running")
	run.running = true
	run.currentAgent = "coding_agent"
	run.cancelFn = func() { cancelled = true }
	run.activeBundleGeneration = 7
	run.activeRunnerKind = RunnerKindDefault
	chat.sandbox = &sandbox.SandboxManager{}
	chat.mu.Unlock()

	chat.UpdateSandbox(nil)

	chat.mu.Lock()
	defer chat.mu.Unlock()
	require.True(t, cancelled)
	assert.Nil(t, chat.sandbox)
	assert.True(t, run.running)
	assert.False(t, run.starting)
	assert.Equal(t, "coding_agent", run.currentAgent)
	assert.Nil(t, run.cancelFn)
	assert.Equal(t, uint64(7), run.activeBundleGeneration)
	assert.Equal(t, RunnerKindDefault, run.activeRunnerKind)
}

func TestUpdateSandboxNilCancelsStartingAgentRuns(t *testing.T) {
	chat := NewChatService(nil)
	cancelled := false
	startDone := make(chan struct{})

	chat.mu.Lock()
	run := chat.getOrCreateRun("sess-starting")
	run.starting = true
	run.startDone = startDone
	run.cancelFn = func() { cancelled = true }
	run.pendingStartBundleGeneration = 11
	chat.sandbox = &sandbox.SandboxManager{}
	chat.mu.Unlock()

	chat.UpdateSandbox(nil)

	select {
	case <-startDone:
		t.Fatal("startup wait channel should stay open until startup path unwinds")
	default:
	}
	chat.mu.Lock()
	defer chat.mu.Unlock()
	require.True(t, cancelled)
	assert.False(t, run.running)
	assert.True(t, run.starting)
	assert.Equal(t, startDone, run.startDone)
	assert.Nil(t, run.cancelFn)
	assert.Equal(t, uint64(11), run.pendingStartBundleGeneration)
}

func TestSandboxLossReportsIdlePendingInterruptClearedWithoutRepair(t *testing.T) {
	chat := NewChatService(nil)

	chat.mu.Lock()
	run := chat.getOrCreateRun("sess-idle-interrupt")
	run.pendingInterrupt = &PendingInterrupt{
		CheckpointID: runtimeTurnCheckpointID("sess-idle-interrupt"),
		InterruptID:  "interrupt-idle",
		RunnerKind:   RunnerKindDefault,
		ToolCallIDs:  []string{"call-missing-from-history"},
	}
	stopped, cleared, repaired := chat.cancelRunsForSandboxLossLocked()
	chat.mu.Unlock()

	assert.Empty(t, stopped)
	assert.Equal(t, []string{"sess-idle-interrupt"}, cleared)
	assert.Empty(t, repaired)
	assert.Nil(t, run.pendingInterrupt)
}

func TestUpdateSandboxNonNilKeepsRunningAgentRuns(t *testing.T) {
	chat := NewChatService(nil)
	cancelled := false
	mgr := &sandbox.SandboxManager{}

	chat.mu.Lock()
	run := chat.getOrCreateRun("sess-running")
	run.running = true
	run.currentAgent = "coding_agent"
	run.cancelFn = func() { cancelled = true }
	chat.mu.Unlock()

	chat.UpdateSandbox(mgr)

	chat.mu.Lock()
	defer chat.mu.Unlock()
	assert.Same(t, mgr, chat.sandbox)
	assert.True(t, run.running)
	assert.Equal(t, "coding_agent", run.currentAgent)
	assert.NotNil(t, run.cancelFn)
	assert.False(t, cancelled)
}
