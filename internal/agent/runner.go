package agent

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"

	"starxo/internal/store"
)

// BuildDefaultRunner creates a runner for the default (non-plan) mode.
// The deep agent handles everything autonomously.
func BuildDefaultRunner(ctx context.Context, deepAgent adk.Agent,
	checkpointStore compose.CheckPointStore) *adk.Runner {

	if checkpointStore == nil {
		checkpointStore = store.NewInMemoryStore()
	}

	return adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           deepAgent,
		EnableStreaming: true,
		CheckPointStore: checkpointStore,
	})
}

// BuildPlanRunner creates a runner for plan mode. Plan mode is now a tool
// permission surface on the same runtime loop, not a separate plan-execute graph.
func BuildPlanRunner(ctx context.Context, mdl model.ToolCallingChatModel,
	runtimeAgent adk.Agent, ac AgentContext,
	checkpointStore compose.CheckPointStore) (*adk.Runner, error) {
	_, _ = mdl, ac

	if checkpointStore == nil {
		checkpointStore = store.NewInMemoryStore()
	}

	return adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           runtimeAgent,
		EnableStreaming: true,
		CheckPointStore: checkpointStore,
	}), nil
}
