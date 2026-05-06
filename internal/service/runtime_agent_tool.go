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

type runtimeAgentRunResult struct {
	text     string
	worktree tools.WorktreeOutput
}

func (s *ChatService) newRuntimeAgentCatalogEntry(ctx context.Context, mdl einomodel.ToolCallingChatModel, op commandline.Operator, provider *deferredMCPProvider, ac agent.AgentContext) (tools.CatalogEntry, error) {
	t, err := toolutils.InferTool(tools.RuntimeToolAgent,
		"Spawn a focused runtime subagent for a well-scoped task. Supports synchronous or background execution and optional worktree isolation.",
		func(ctx context.Context, input runtimeAgentInput) (runtimeAgentOutput, error) {
			if strings.TrimSpace(input.Prompt) == "" {
				return runtimeAgentOutput{}, fmt.Errorf("prompt is required")
			}
			normalized, err := normalizeRuntimeAgentInput(input)
			if err != nil {
				return runtimeAgentOutput{}, err
			}
			agentID := fmt.Sprintf("agent-%d", s.now().UnixNano())
			description := runtimeFirstNonEmpty(normalized.Description, normalized.SubagentType, "Runtime subagent")
			run := func(runCtx context.Context) (runtimeAgentRunResult, error) {
				return s.runRuntimeSubagent(runCtx, mdl, op, provider, ac, agentID, normalized)
			}
			if normalized.Background {
				if s.runtimeTasks == nil {
					return runtimeAgentOutput{}, fmt.Errorf("runtime task manager is not available")
				}
				ref, err := s.runtimeTasks.StartAgentTask(ctx, SessionIDFromContext(ctx), description, func(taskCtx context.Context) (string, error) {
					result, runErr := run(taskCtx)
					return formatRuntimeAgentRunResult(result), runErr
				})
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
				AgentID:        agentID,
				Status:         runtimeTaskStatusCompleted,
				Result:         formatRuntimeAgentRunResult(result),
				WorktreePath:   result.worktree.WorktreePath,
				WorktreeBranch: result.worktree.WorktreeBranch,
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

func (s *ChatService) runRuntimeSubagent(ctx context.Context, mdl einomodel.ToolCallingChatModel, op commandline.Operator, provider *deferredMCPProvider, ac agent.AgentContext, agentID string, input runtimeAgentInput) (runtimeAgentRunResult, error) {
	worktreeResult := tools.WorktreeOutput{}
	if input.Isolation == "worktree" {
		if s.runtimeWorkspaces == nil {
			return runtimeAgentRunResult{}, fmt.Errorf("runtime worktree manager is not available")
		}
		var err error
		worktreeResult, err = s.runtimeWorkspaces.CreateIsolatedWorktree(ctx, op, ac.WorkspacePath, agentID)
		if err != nil {
			return runtimeAgentRunResult{}, err
		}
		ctx = contextWithRuntimeWorkspaceOverride(ctx, worktreeResult.WorktreePath)
	}

	entries, err := tools.NewRuntimeCoreCatalogEntries(op, ac.WorkspacePath, s.runtimeTasks, s.runtimeWorkspaces)
	if err != nil {
		return runtimeAgentRunResult{}, err
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
		Instruction: runtimeSubagentInstruction(subagentType, currentRuntimeAgentWorkspace(ac.WorkspacePath, worktreeResult), input.Isolation),
		Model:       mdl,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: subTools,
			},
		},
		MaxIterations: 30,
	})
	if err != nil {
		return runtimeAgentRunResult{}, err
	}
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: sub, EnableStreaming: true})
	iter := runner.Query(ctx, input.Prompt)
	result, err := collectRuntimeAgentResult(iter)
	if err != nil {
		return runtimeAgentRunResult{text: result, worktree: worktreeResult}, err
	}
	return runtimeAgentRunResult{text: result, worktree: worktreeResult}, nil
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

func normalizeRuntimeAgentInput(input runtimeAgentInput) (runtimeAgentInput, error) {
	input.Description = strings.TrimSpace(input.Description)
	input.Prompt = strings.TrimSpace(input.Prompt)
	input.SubagentType = strings.TrimSpace(input.SubagentType)
	if input.SubagentType == "" {
		input.SubagentType = "general"
	}
	switch input.SubagentType {
	case "general", "code_writer", "code_executor", "file_manager":
	default:
		return runtimeAgentInput{}, fmt.Errorf("unsupported subagent_type %q; use general, code_writer, code_executor, or file_manager", input.SubagentType)
	}
	input.Isolation = strings.TrimSpace(input.Isolation)
	if input.Isolation == "" {
		input.Isolation = "none"
	}
	switch input.Isolation {
	case "none", "worktree":
	default:
		return runtimeAgentInput{}, fmt.Errorf("unsupported isolation %q; use none or worktree", input.Isolation)
	}
	return input, nil
}

func formatRuntimeAgentRunResult(result runtimeAgentRunResult) string {
	text := strings.TrimRight(result.text, "\n")
	if result.worktree.WorktreePath == "" {
		return text
	}
	if text != "" {
		text += "\n\n"
	}
	text += fmt.Sprintf("worktreePath: %s\nworktreeBranch: %s", result.worktree.WorktreePath, result.worktree.WorktreeBranch)
	return text
}

func currentRuntimeAgentWorkspace(defaultWorkspace string, worktree tools.WorktreeOutput) string {
	if strings.TrimSpace(worktree.WorktreePath) != "" {
		return worktree.WorktreePath
	}
	return defaultWorkspace
}

func runtimeSubagentInstruction(subagentType, workspacePath, isolation string) string {
	isolationNote := "none"
	if isolation == "worktree" {
		isolationNote = "worktree (changes are isolated from the parent session workspace unless explicitly merged later)"
	}
	return fmt.Sprintf(`You are a focused Starxo runtime subagent.

Type: %s
Workspace: %s
Isolation: %s

Complete only the delegated task. Use Read/Grep/Glob for inspection, Edit/Write for file changes, and Bash for commands. Keep output concise and include changed files, verification performed, and blockers.`, subagentType, workspacePath, isolationNote)
}
