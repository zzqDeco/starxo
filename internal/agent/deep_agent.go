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

// BuildDeepAgent creates the core deep agent that handles all coding tasks.
// It has direct tools (FollowUp, Choice, MCP tools) and sub-agents (code_writer,
// code_executor, file_manager) that it can delegate to via transfer.
//
// This agent is reused in both default mode (as the runner's agent directly)
// and plan mode (as the executor inside planexecute.New()).
func BuildDeepAgent(ctx context.Context, mdl model.ToolCallingChatModel,
	op commandline.Operator, extraTools []tool.BaseTool, ac AgentContext) (adk.Agent, error) {

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

	// Direct tools: interrupt tools + todo tracking + status + extra tools (MCP etc.)
	directTools := []tool.BaseTool{
		agenttools.NewFollowUpTool(),
		agenttools.NewChoiceTool(),
		agenttools.NewWriteTodosTool(),
		agenttools.NewUpdateTodoTool(),
		agenttools.NewNotifyUserTool(),
	}
	directTools = append(directTools, extraTools...)

	return deep.New(ctx, &deep.Config{
		Name:        "coding_agent",
		Description: "Autonomous coding agent with specialized sub-agents for code writing, execution, and file management.",
		Instruction: DeepAgentPrompt(ac),
		ChatModel:   mdl,
		SubAgents:   []adk.Agent{codeWriter, codeExecutor, fileManager},
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: directTools,
			},
		},
		MaxIteration: 50,
	})
}
