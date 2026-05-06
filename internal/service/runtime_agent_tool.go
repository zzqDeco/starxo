package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/adk"
	einomodel "github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"

	"starxo/internal/agent"
	"starxo/internal/tools"
)

type runtimeAgentInput struct {
	Description  string `json:"description,omitempty" jsonschema:"description=short description of the delegated task"`
	Prompt       string `json:"prompt" jsonschema:"description=full task prompt for the subagent"`
	SubagentType string `json:"subagent_type,omitempty" jsonschema:"description=general, code_writer, code_executor, or file_manager"`
	Model        string `json:"model,omitempty" jsonschema:"description=reserved for future model override"`
	Mode         string `json:"mode,omitempty" jsonschema:"description=reserved for future mode override"`
	Background   bool   `json:"background,omitempty" jsonschema:"description=run in background and return a task id"`
	Isolation    string `json:"isolation,omitempty" jsonschema:"description=none or worktree"`
}

type runtimeAgentOutput struct {
	AgentID          string `json:"agentId"`
	Status           string `json:"status"`
	Result           string `json:"result,omitempty"`
	BackgroundTaskID string `json:"backgroundTaskId,omitempty"`
	OutputPath       string `json:"outputPath,omitempty"`
	WorktreePath     string `json:"worktreePath,omitempty"`
	WorktreeBranch   string `json:"worktreeBranch,omitempty"`
}

func (s *ChatService) newRuntimeAgentCatalogEntry(ctx context.Context, mdl einomodel.ToolCallingChatModel, op commandline.Operator, provider *deferredMCPProvider, ac agent.AgentContext) (tools.CatalogEntry, error) {
	t, err := toolutils.InferTool(tools.RuntimeToolAgent,
		"Spawn a focused runtime subagent for a well-scoped task. Supports synchronous or background execution and optional worktree isolation.",
		func(ctx context.Context, input runtimeAgentInput) (runtimeAgentOutput, error) {
			if strings.TrimSpace(input.Prompt) == "" {
				return runtimeAgentOutput{}, fmt.Errorf("prompt is required")
			}
			agentID := fmt.Sprintf("agent-%d", s.now().UnixNano())
			description := runtimeFirstNonEmpty(input.Description, input.SubagentType, "Runtime subagent")
			run := func(runCtx context.Context) (string, error) {
				return s.runRuntimeSubagent(runCtx, mdl, op, provider, ac, agentID, input)
			}
			if input.Background {
				if s.runtimeTasks == nil {
					return runtimeAgentOutput{}, fmt.Errorf("runtime task manager is not available")
				}
				ref, err := s.runtimeTasks.StartAgentTask(ctx, SessionIDFromContext(ctx), description, run)
				if err != nil {
					return runtimeAgentOutput{}, err
				}
				return runtimeAgentOutput{
					AgentID:          agentID,
					Status:           ref.Status,
					BackgroundTaskID: ref.TaskID,
					OutputPath:       ref.OutputPath,
				}, nil
			}
			result, err := run(ctx)
			if err != nil {
				return runtimeAgentOutput{}, err
			}
			return runtimeAgentOutput{
				AgentID: agentID,
				Status:  runtimeTaskStatusCompleted,
				Result:  result,
			}, nil
		})
	if err != nil {
		return tools.CatalogEntry{}, err
	}
	return tools.CatalogEntry{
		CanonicalName: tools.RuntimeToolAgent,
		Source:        tools.ToolSourceRuntime,
		Kind:          tools.ToolKindAction,
		Title:         "Agent",
		Description:   "Spawn a focused runtime subagent.",
		SearchHint:    "delegate spawn subagent background worktree isolated task",
		ToolClass:     tools.ToolClassRuntimeTask,
		AlwaysLoad:    true,
		ShouldDefer:   false,
		PermissionSpec: tools.PermissionSpec{
			AllowSearch:  true,
			AllowExecute: true,
		},
		Tool: t,
	}, nil
}

func (s *ChatService) runRuntimeSubagent(ctx context.Context, mdl einomodel.ToolCallingChatModel, op commandline.Operator, provider *deferredMCPProvider, ac agent.AgentContext, agentID string, input runtimeAgentInput) (string, error) {
	worktreeResult := tools.WorktreeOutput{}
	if input.Isolation == "worktree" {
		if s.runtimeWorkspaces == nil {
			return "", fmt.Errorf("runtime worktree manager is not available")
		}
		var err error
		worktreeResult, err = s.runtimeWorkspaces.EnterWorktree(ctx, op, ac.WorkspacePath, agentID)
		if err != nil {
			return "", err
		}
		defer func() {
			_, _ = s.runtimeWorkspaces.ExitWorktree(ctx, op, ac.WorkspacePath, "keep", false)
		}()
	}

	entries, err := tools.NewRuntimeCoreCatalogEntries(op, ac.WorkspacePath, s.runtimeTasks, s.runtimeWorkspaces)
	if err != nil {
		return "", err
	}
	subTools := make([]einotool.BaseTool, 0, len(entries))
	for _, entry := range entries {
		if entry.CanonicalName == tools.RuntimeToolAgent {
			continue
		}
		wrapped := entry
		wrapped.Tool = tools.WrapMCPToolWithPermissionCheck(wrapped, provider)
		subTools = append(subTools, wrapped.Tool)
	}
	subTools = agent.WrapToolsWithEvents(agentID, subTools, ac)

	subagentType := runtimeFirstNonEmpty(input.SubagentType, "general")
	sub, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        agentID,
		Description: "Runtime subagent for delegated coding tasks.",
		Instruction: runtimeSubagentInstruction(ac, subagentType),
		Model:       mdl,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: subTools,
			},
		},
		MaxIterations: 30,
	})
	if err != nil {
		return "", err
	}
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: sub, EnableStreaming: true})
	iter := runner.Query(ctx, input.Prompt)
	result, err := collectRuntimeAgentResult(iter)
	if err != nil {
		return result, err
	}
	if worktreeResult.WorktreePath != "" {
		result = strings.TrimRight(result, "\n") + fmt.Sprintf("\n\nworktreePath: %s\nworktreeBranch: %s", worktreeResult.WorktreePath, worktreeResult.WorktreeBranch)
	}
	return result, nil
}

func collectRuntimeAgentResult(iter *adk.AsyncIterator[*adk.AgentEvent]) (string, error) {
	var parts []string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return strings.Join(parts, "\n\n"), event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}
		msg, err := event.Output.MessageOutput.GetMessage()
		if err != nil {
			return strings.Join(parts, "\n\n"), err
		}
		if strings.TrimSpace(msg.Content) != "" {
			parts = append(parts, msg.Content)
		}
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("runtime agent returned no output")
	}
	return strings.Join(parts, "\n\n"), nil
}

func runtimeSubagentInstruction(ac agent.AgentContext, subagentType string) string {
	return fmt.Sprintf(`You are a focused Starxo runtime subagent.

Type: %s
Workspace: %s

Complete only the delegated task. Use Read/Grep/Glob for inspection, Edit/Write for file changes, and Bash for commands. Keep output concise and include changed files, verification performed, and blockers.`, subagentType, ac.WorkspacePath)
}
