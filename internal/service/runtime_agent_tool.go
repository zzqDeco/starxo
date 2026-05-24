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
	"github.com/cloudwego/eino/schema"

	"starxo/internal/agent"
	starmodel "starxo/internal/model"
	"starxo/internal/tools"
)

type runtimeAgentInput struct {
	Description  string `json:"description,omitempty" jsonschema:"description=short description of the delegated task"`
	Prompt       string `json:"prompt" jsonschema:"description=full task prompt for the subagent"`
	SubagentType string `json:"subagent_type,omitempty" jsonschema:"description=configured subagent type; omit to use the registry default"`
	Model        string `json:"model,omitempty" jsonschema:"description=reserved for future model override"`
	Mode         string `json:"mode,omitempty" jsonschema:"description=reserved for future mode override"`
	Background   bool   `json:"background,omitempty" jsonschema:"description=run in background and return a task id"`
	Isolation    string `json:"isolation,omitempty" jsonschema:"description=none or worktree"`
	Fork         bool   `json:"-"`
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

func (s *ChatService) newRuntimeAgentCatalogEntry(ctx context.Context, mdl einomodel.ToolCallingChatModel, op commandline.Operator, provider *deferredMCPProvider, ac agent.AgentContext, registry *agent.SubagentRegistry) (tools.CatalogEntry, error) {
	if registry == nil {
		registry = agent.DefaultSubagentRegistry()
	}
	t := toolutils.NewTool(runtimeAgentToolInfo(registry),
		func(ctx context.Context, input runtimeAgentInput) (runtimeAgentOutput, error) {
			if strings.TrimSpace(input.Prompt) == "" {
				return runtimeAgentOutput{}, fmt.Errorf("prompt is required")
			}
			normalized, err := normalizeRuntimeAgentInput(input, registry)
			if err != nil {
				return runtimeAgentOutput{}, err
			}
			def := registry.MustGet(normalized.SubagentType)
			if normalized.Background && !def.BackgroundAllowed {
				return runtimeAgentOutput{}, fmt.Errorf("subagent_type %s does not allow background execution", normalized.SubagentType)
			}
			agentID := fmt.Sprintf("agent-%d", s.now().UnixNano())
			description := runtimeFirstNonEmpty(normalized.Description, normalized.SubagentType, "Runtime subagent")
			run := func(runCtx context.Context) (runtimeAgentRunResult, error) {
				return s.runRuntimeSubagent(runCtx, mdl, op, provider, ac, agentID, normalized, registry)
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

func runtimeAgentToolInfo(registry *agent.SubagentRegistry) *schema.ToolInfo {
	if registry == nil {
		registry = agent.DefaultSubagentRegistry()
	}
	subagentNames := registry.Names()
	subagentDesc := fmt.Sprintf("Configured subagent type. Omit to fork the current agent context. Supported fresh worker values: %s.",
		registry.NamesCSV())
	return &schema.ToolInfo{
		Name: tools.RuntimeToolAgent,
		Desc: "Spawn a focused runtime subagent. Omit subagent_type to fork current context; set subagent_type for a fresh worker. Supports background execution and optional worktree isolation.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"description": {
				Type: schema.String,
				Desc: "Short description of the delegated task.",
			},
			"prompt": {
				Type:     schema.String,
				Desc:     "Full task prompt for the subagent.",
				Required: true,
			},
			"subagent_type": {
				Type: schema.String,
				Desc: subagentDesc,
				Enum: subagentNames,
			},
			"model": {
				Type: schema.String,
				Desc: "Reserved for future model override.",
			},
			"mode": {
				Type: schema.String,
				Desc: "Reserved for future mode override.",
			},
			"background": {
				Type: schema.Boolean,
				Desc: "Run in background and return a task id.",
			},
			"isolation": {
				Type: schema.String,
				Desc: "Execution isolation. Omit to use the selected subagent's default isolation.",
				Enum: []string{"none", "worktree"},
			},
		}),
	}
}

func (s *ChatService) runRuntimeSubagent(ctx context.Context, mdl einomodel.ToolCallingChatModel, op commandline.Operator, provider *deferredMCPProvider, ac agent.AgentContext, agentID string, input runtimeAgentInput, registry *agent.SubagentRegistry) (runtimeAgentRunResult, error) {
	worktreeResult := tools.WorktreeOutput{}
	subAC := ac
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
		subAC = runtimeSubagentAgentContext(ac, worktreeResult)
	}

	def := registry.MustGet(input.SubagentType)
	subTools := s.runtimeSubagentTools(provider, def, input.Fork)
	if len(subTools) == 0 {
		entries, err := tools.NewRuntimeCoreCatalogEntries(op, subAC.WorkspacePath, s.runtimeTasks, s.runtimeWorkspaces)
		if err != nil {
			return runtimeAgentRunResult{}, err
		}
		for _, entry := range entries {
			if entry.CanonicalName == tools.RuntimeToolAgent {
				continue
			}
			if !input.Fork && !runtimeSubagentAllowsTool(def, entry.CanonicalName) {
				continue
			}
			wrapped := entry
			wrapped.Tool = tools.WrapMCPToolWithPermissionCheck(wrapped, provider)
			subTools = append(subTools, wrapped.Tool)
		}
	}
	subTools = agent.WrapToolsWithEvents(agentID, subTools, subAC)

	mode := s.runtimeSubagentMode(ctx, provider, input.Mode)
	handlers := []adk.ChatModelAgentMiddleware{tools.NewDynamicMCPSurfaceMiddleware(provider)}
	toolSearchHandler, err := newEinoV09ToolSearchHandlerForCatalog(
		ctx,
		runtimeSubagentCatalog(provider),
		mode,
		"client",
		"",
		runtimeSubagentCatalogEntryAllowed(def, input.Fork),
	)
	if err != nil {
		return runtimeAgentRunResult{}, err
	}
	if toolSearchHandler != nil {
		handlers = append([]adk.ChatModelAgentMiddleware{toolSearchHandler}, handlers...)
	}
	contextHandlers, err := agent.NewEinoV09ContextMiddlewares(ctx, mdl, op, subAC)
	if err != nil {
		return runtimeAgentRunResult{}, err
	}
	handlers = append([]adk.ChatModelAgentMiddleware{agent.NewRuntimeBehaviorMiddleware()}, append(contextHandlers, handlers...)...)

	instruction := agent.RuntimeSubagentPrompt(def, currentRuntimeAgentWorkspace(ac.WorkspacePath, worktreeResult), input.Isolation)
	prompt := input.Prompt
	if input.Fork {
		instruction = agent.RuntimeAgentPrompt(subAC, runtimeSubagentDeepAgentMode(mode), registry)
		if objective, ok := tools.RuntimeObjectiveFromContext(ctx); ok {
			prompt = fmt.Sprintf("Forked task for the current objective:\n\nCurrent objective: %s\n\nDelegated task: %s", objective.Objective, input.Prompt)
		}
	}

	sub, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        agentID,
		Description: def.Description,
		Instruction: instruction,
		Model:       mdl,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools:               subTools,
				UnknownToolsHandler: newDeferredUnknownToolHandler(provider),
			},
		},
		MaxIterations: 30,
		Handlers:      handlers,
	})
	if err != nil {
		return runtimeAgentRunResult{}, err
	}
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: sub, EnableStreaming: true})
	iter := runner.Query(ctx, prompt)
	result, err := collectRuntimeAgentResult(iter)
	if err != nil {
		return runtimeAgentRunResult{text: result, worktree: worktreeResult}, err
	}
	return runtimeAgentRunResult{text: result, worktree: worktreeResult}, nil
}

func (s *ChatService) runtimeSubagentMode(ctx context.Context, provider *deferredMCPProvider, requested string) string {
	switch strings.TrimSpace(requested) {
	case starmodel.ModePlan:
		return starmodel.ModePlan
	case starmodel.ModeDefault:
		return starmodel.ModeDefault
	}
	if provider != nil {
		if _, mode, _, err := provider.sessionState(ctx); err == nil && strings.TrimSpace(mode) != "" {
			return mode
		}
	}
	return starmodel.ModeDefault
}

func runtimeSubagentDeepAgentMode(mode string) agent.DeepAgentMode {
	if mode == starmodel.ModePlan {
		return agent.DeepAgentModePlan
	}
	return agent.DeepAgentModeDefault
}

func runtimeSubagentCatalog(provider *deferredMCPProvider) *tools.ToolCatalog {
	if provider == nil || provider.bundle == nil {
		return nil
	}
	return provider.bundle.MCPCatalog
}

func runtimeSubagentCatalogEntryAllowed(def agent.SubagentDefinition, fork bool) func(tools.CatalogEntry) bool {
	if fork {
		return nil
	}
	return func(entry tools.CatalogEntry) bool {
		if entry.CanonicalName == tools.RuntimeToolAgent {
			return false
		}
		return runtimeSubagentAllowsTool(def, entry.CanonicalName)
	}
}

func (s *ChatService) runtimeSubagentTools(provider *deferredMCPProvider, def agent.SubagentDefinition, fork bool) []einotool.BaseTool {
	if provider == nil || provider.bundle == nil || provider.bundle.MCPCatalog == nil {
		return nil
	}
	entries := provider.bundle.MCPCatalog.Entries()
	subTools := make([]einotool.BaseTool, 0, len(entries))
	for _, entry := range entries {
		if entry.CanonicalName == tools.RuntimeToolAgent {
			continue
		}
		if entry.ShouldDefer && !entry.AlwaysLoad {
			continue
		}
		if !fork && !runtimeSubagentAllowsTool(def, entry.CanonicalName) {
			continue
		}
		if entry.Tool != nil {
			subTools = append(subTools, entry.Tool)
		}
	}
	return subTools
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

func normalizeRuntimeAgentInput(input runtimeAgentInput, registry *agent.SubagentRegistry) (runtimeAgentInput, error) {
	if registry == nil {
		registry = agent.DefaultSubagentRegistry()
	}
	input.Description = strings.TrimSpace(input.Description)
	input.Prompt = strings.TrimSpace(input.Prompt)
	rawSubagentType := strings.TrimSpace(input.SubagentType)
	input.SubagentType = rawSubagentType
	if rawSubagentType == "" {
		input.Fork = true
		input.SubagentType = registry.DefaultName()
	} else {
		normalizedType, err := registry.Normalize(input.SubagentType)
		if err != nil {
			return runtimeAgentInput{}, err
		}
		input.SubagentType = normalizedType
	}
	input.Isolation = strings.TrimSpace(input.Isolation)
	if input.Isolation == "" {
		def := registry.MustGet(input.SubagentType)
		input.Isolation = def.DefaultIsolation
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

func runtimeSubagentAgentContext(ac agent.AgentContext, worktree tools.WorktreeOutput) agent.AgentContext {
	if strings.TrimSpace(worktree.WorktreePath) != "" {
		ac.WorkspacePath = worktree.WorktreePath
	}
	return ac
}

func runtimeSubagentAllowsTool(def agent.SubagentDefinition, toolName string) bool {
	if len(def.AllowedTools) == 0 {
		return true
	}
	for _, allowed := range def.AllowedTools {
		if allowed == toolName {
			return true
		}
	}
	return false
}
