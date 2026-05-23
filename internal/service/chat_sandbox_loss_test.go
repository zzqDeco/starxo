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
	assert.False(t, run.running)
	assert.False(t, run.starting)
	assert.Empty(t, run.currentAgent)
	assert.Nil(t, run.cancelFn)
	assert.Zero(t, run.activeBundleGeneration)
	assert.Empty(t, run.activeRunnerKind)
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
	default:
		t.Fatal("expected starting run to be released")
	}
	chat.mu.Lock()
	defer chat.mu.Unlock()
	require.True(t, cancelled)
	assert.False(t, run.running)
	assert.False(t, run.starting)
	assert.Nil(t, run.startDone)
	assert.Nil(t, run.cancelFn)
	assert.Zero(t, run.pendingStartBundleGeneration)
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
