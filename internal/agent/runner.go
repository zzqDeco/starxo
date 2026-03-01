package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
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
		EnableStreaming:  true,
		CheckPointStore: checkpointStore,
	})
}

// BuildPlanRunner creates a runner for the plan mode.
// It wraps the deep agent as the executor inside a planexecute pattern,
// with a planner and replanner that also persist the plan to Markdown.
func BuildPlanRunner(ctx context.Context, mdl model.ToolCallingChatModel,
	deepAgent adk.Agent, ac AgentContext,
	checkpointStore compose.CheckPointStore) (*adk.Runner, error) {

	if checkpointStore == nil {
		checkpointStore = store.NewInMemoryStore()
	}

	// Create the planner agent
	planner, err := planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
		ToolCallingChatModel: mdl,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create planner: %w", err)
	}

	// Create the replanner agent
	replanner, err := planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
		ChatModel: mdl,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create replanner: %w", err)
	}

	// Build the plan-execute agent with deep agent as executor
	planAgent, err := planexecute.New(ctx, &planexecute.Config{
		Planner:       planner,
		Executor:      deepAgent,
		Replanner:     replanner,
		MaxIterations: 20,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create plan-execute agent: %w", err)
	}

	return adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           planAgent,
		EnableStreaming:  true,
		CheckPointStore: checkpointStore,
	}), nil
}
