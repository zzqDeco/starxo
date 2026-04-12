package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"

	agenttools "starxo/internal/tools"
)

// DeepAgentMode controls orchestration behavior and direct tool permission at
// the top-level coding_agent.
type DeepAgentMode string

const (
	DeepAgentModeDefault DeepAgentMode = "default"
	DeepAgentModePlan    DeepAgentMode = "plan"
)

// BuildDeepAgent creates the core deep agent that handles all coding tasks.
// It has direct tools (FollowUp, Choice, MCP tools) and sub-agents (code_writer,
// code_executor, file_manager) that it can delegate to via transfer.
//
// This agent is reused in both default mode (as the runner's agent directly)
// and plan mode (as the executor inside planexecute.New()).
func BuildDeepAgent(ctx context.Context, mdl model.ToolCallingChatModel,
	op commandline.Operator, extraTools []tool.BaseTool, ac AgentContext) (adk.Agent, error) {
	return BuildDeepAgentForMode(ctx, mdl, op, extraTools, ac, DeepAgentModeDefault, nil, nil)
}

// BuildDeepAgentForMode creates the core deep agent with mode-specific direct
// tool permissions and prompt constraints.
func BuildDeepAgentForMode(ctx context.Context, mdl model.ToolCallingChatModel,
	op commandline.Operator, extraTools []tool.BaseTool, ac AgentContext, mode DeepAgentMode,
	handlers []adk.ChatModelAgentMiddleware,
	unknownToolsHandler func(ctx context.Context, name, input string) (string, error),
) (adk.Agent, error) {

	// Build sub-agents (no Exit tool — deep agent manages their lifecycle)
	codeWriter, err := NewCodeWriterAgent(ctx, mdl, op, ac)
	if err != nil {
		return nil, fmt.Errorf("failed to create code_writer agent: %w", err)
	}

	codeExecutor, err := NewCodeExecutorAgent(ctx, mdl, op, ac)
	if err != nil {
		return nil, fmt.Errorf("failed to create code_executor agent: %w", err)
	}

	fileManager, err := NewFileManagerAgent(ctx, mdl, op, ac)
	if err != nil {
		return nil, fmt.Errorf("failed to create file_manager agent: %w", err)
	}

	// Direct orchestration tools always available on the top-level agent.
	directTools := []tool.BaseTool{
		agenttools.NewFollowUpTool(),
		agenttools.NewChoiceTool(),
		agenttools.NewNotifyUserTool(),
	}

	instruction := DeepAgentPrompt(ac)

	switch mode {
	case DeepAgentModePlan:
		// In plan mode, the main agent owns task-list lifecycle and acceptance.
		directTools = append(directTools,
			agenttools.NewWriteTodosTool(),
			agenttools.NewUpdateTodoTool(),
		)
		directTools = append(directTools, extraTools...)
		instruction = DeepAgentPlanPrompt(ac)
	case DeepAgentModeDefault:
		// In default mode keep existing behavior, including extra tools.
		directTools = append(directTools,
			agenttools.NewWriteTodosTool(),
			agenttools.NewUpdateTodoTool(),
		)
		directTools = append(directTools, extraTools...)
	default:
		return nil, fmt.Errorf("unsupported deep agent mode: %s", mode)
	}

	return deep.New(ctx, &deep.Config{
		Name:        "coding_agent",
		Description: "Autonomous coding agent with specialized sub-agents for code writing, execution, and file management.",
		Instruction: instruction,
		ChatModel:   mdl,
		SubAgents:   []adk.Agent{codeWriter, codeExecutor, fileManager},
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools:               directTools,
				UnknownToolsHandler: unknownToolsHandler,
			},
		},
		MaxIteration:      50,
		WithoutWriteTodos: true,
		Handlers:          handlers,
	})
}
