package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/adk"
	einomodel "github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"starxo/internal/agent"
	"starxo/internal/config"
	"starxo/internal/llm"
	"starxo/internal/logger"
	"starxo/internal/tools"
)

type runtimeBundleBuilder struct {
	chat    *ChatService
	cfg     *config.AppConfig
	digest  string
	surface *runnerBundleSurface
}

func newRuntimeBundleBuilder(chat *ChatService, cfg *config.AppConfig, digest string, surface *runnerBundleSurface) *runtimeBundleBuilder {
	return &runtimeBundleBuilder{
		chat:    chat,
		cfg:     cfg,
		digest:  digest,
		surface: surface,
	}
}

func (b *runtimeBundleBuilder) Build(ctx context.Context) (*RunnerBundle, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if b.cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	if b.surface == nil {
		return nil, fmt.Errorf("runner bundle surface is nil")
	}
	if b.chat.sandbox == nil || !b.chat.sandbox.IsConnected() {
		return b.closeWithError(fmt.Errorf("sandbox is not connected"))
	}

	op := b.chat.sandbox.Operator()
	if op == nil {
		return b.closeWithError(fmt.Errorf("sandbox operator is not available"))
	}

	mdl, err := llm.NewChatModel(ctx, b.cfg.LLM)
	if err != nil {
		logger.Error("[RUNNER] Failed to create chat model", err)
		return b.closeWithError(fmt.Errorf("failed to create chat model: %w", err))
	}
	logger.RunnerEvent("chat_model_created", "type", b.cfg.LLM.Type, "model", b.cfg.LLM.Model)
	effectiveAgenticProtocol := b.effectiveAgenticProtocol(ctx)

	bundle := &RunnerBundle{
		ConfigDigest:                  b.digest,
		MCPHandles:                    b.surface.Handles,
		SurfaceRelevantFingerprint:    b.surface.SurfaceRelevantFingerprint,
		CachedSurfaceMetadataByServer: cloneSurfaceCache(b.surface.CachedSurfaceMetadataByServer),
	}
	provider := &deferredMCPProvider{chat: b.chat, bundle: bundle}
	ac := b.chat.buildAgentContext()
	if b.chat.runtimeLSP != nil {
		b.chat.runtimeLSP.SetConfig(b.cfg.Agent.LSP)
	}
	subagentRegistry := newRuntimeSubagentRegistry(b.cfg.Agent.Runtime.Subagents)

	topLevelCatalog, err := b.buildRuntimeCatalog(ctx, provider, mdl, op, ac, subagentRegistry)
	if err != nil {
		return b.closeWithError(err)
	}
	bundle.MCPCatalog = topLevelCatalog

	extraTools := runtimeAlwaysLoadedTools(topLevelCatalog)
	defaultHandlers, planHandlers, err := b.buildRuntimeHandlers(ctx, provider, effectiveAgenticProtocol)
	if err != nil {
		return b.closeWithError(err)
	}

	unknownToolsHandler := newDeferredUnknownToolHandler(provider)
	defaultAgent, planAgent, err := buildTopLevelRuntimeAgents(ctx, b.cfg, mdl, op, extraTools, ac, defaultHandlers, planHandlers, unknownToolsHandler, subagentRegistry)
	if err != nil {
		return b.closeWithError(err)
	}

	bundle.DefaultRunner = agent.BuildDefaultRunner(ctx, defaultAgent, b.chat.checkpointStore)
	bundle.PlanRunner, err = agent.BuildPlanRunner(ctx, mdl, planAgent, ac, b.chat.checkpointStore)
	if err != nil {
		logger.Error("[RUNNER] Failed to build runtime plan runner", err)
		return b.closeWithError(fmt.Errorf("failed to build runtime plan runner: %w", err))
	}
	bundle.LastFreshnessCheckAt = b.chat.now()

	b.emitMCPHandleErrors(bundle)
	logger.RunnerEvent("runners_built", "extra_tools", len(extraTools))
	return bundle, nil
}

func (b *runtimeBundleBuilder) buildRuntimeCatalog(ctx context.Context, provider *deferredMCPProvider, mdl einomodel.ToolCallingChatModel, op commandline.Operator, ac agent.AgentContext, registry *agent.SubagentRegistry) (*tools.ToolCatalog, error) {
	topLevelCatalog := tools.NewToolCatalog()
	runtimeEntries, err := tools.NewRuntimeCoreCatalogEntries(op, ac.WorkspacePath, b.chat.runtimeTasks, b.chat.runtimeWorkspaces)
	if err != nil {
		return nil, fmt.Errorf("failed to build runtime core tools: %w", err)
	}
	if err := registerWrappedCatalogEntries(topLevelCatalog, provider, runtimeEntries, "runtime tool"); err != nil {
		return nil, err
	}

	agentEntry, err := newRuntimeSubagentRunner(b.chat).NewCatalogEntry(ctx, mdl, op, provider, ac, registry)
	if err != nil {
		return nil, fmt.Errorf("failed to build runtime Agent tool: %w", err)
	}
	agentEntry.Tool = tools.WrapMCPToolWithPermissionCheck(agentEntry, provider)
	if err := topLevelCatalog.Register(agentEntry); err != nil {
		return nil, fmt.Errorf("failed to register runtime Agent tool: %w", err)
	}

	runtimeDeferredEntries, err := tools.NewRuntimeDeferredCatalogEntries(op, ac.WorkspacePath, b.chat.runtimeWorkspaces, b.chat.runtimeLSP)
	if err != nil {
		return nil, fmt.Errorf("failed to build deferred runtime tools: %w", err)
	}
	runtimeWebEntries, err := newRuntimeWebCatalogEntries(b.cfg.Agent.WebSearch, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to build deferred web tools: %w", err)
	}
	if err := registerWrappedCatalogEntries(topLevelCatalog, provider, append(runtimeDeferredEntries, runtimeWebEntries...), "deferred runtime tool"); err != nil {
		return nil, err
	}
	if err := registerWrappedCatalogEntries(topLevelCatalog, provider, b.surface.ActionCatalog.Entries(), "MCP action tool"); err != nil {
		return nil, err
	}

	resourceEntries, err := tools.NewMCPResourceCatalogEntries(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to build MCP resource tools: %w", err)
	}
	if err := registerWrappedCatalogEntries(topLevelCatalog, provider, resourceEntries, "MCP resource tool"); err != nil {
		return nil, err
	}
	if b.chat.runtimeOptionsSnapshot().DevDeferredBuiltinSampleEnabled {
		sampleEntry, err := tools.NewDevDeferredBuiltinSampleEntry()
		if err != nil {
			return nil, fmt.Errorf("failed to build dev deferred builtin sample: %w", err)
		}
		if err := registerWrappedCatalogEntries(topLevelCatalog, provider, []tools.CatalogEntry{sampleEntry}, "dev deferred builtin sample"); err != nil {
			return nil, err
		}
	}
	return topLevelCatalog, nil
}

func (b *runtimeBundleBuilder) buildRuntimeHandlers(ctx context.Context, provider *deferredMCPProvider, effectiveAgenticProtocol string) ([]adk.ChatModelAgentMiddleware, []adk.ChatModelAgentMiddleware, error) {
	deferredHandler := tools.NewDynamicMCPSurfaceMiddleware(provider)
	defaultToolSearchHandler, err := newEinoV09ToolSearchHandler(ctx, provider, "default", b.cfg.Agent.Runtime.ToolSearchMode, effectiveAgenticProtocol)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build Eino v0.9 default tool_search bridge: %w", err)
	}
	planToolSearchHandler, err := newEinoV09ToolSearchHandler(ctx, provider, "plan", b.cfg.Agent.Runtime.ToolSearchMode, effectiveAgenticProtocol)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build Eino v0.9 plan tool_search bridge: %w", err)
	}
	defaultHandlers := []adk.ChatModelAgentMiddleware{deferredHandler}
	if defaultToolSearchHandler != nil {
		defaultHandlers = append([]adk.ChatModelAgentMiddleware{defaultToolSearchHandler}, defaultHandlers...)
	}
	planHandlers := []adk.ChatModelAgentMiddleware{deferredHandler}
	if planToolSearchHandler != nil {
		planHandlers = append([]adk.ChatModelAgentMiddleware{planToolSearchHandler}, planHandlers...)
	}
	return defaultHandlers, planHandlers, nil
}

func (b *runtimeBundleBuilder) effectiveAgenticProtocol(ctx context.Context) string {
	effectiveAgenticProtocol := strings.TrimSpace(b.cfg.Agent.Runtime.AgenticProtocol)
	if effectiveAgenticProtocol == "" {
		effectiveAgenticProtocol = llm.AgenticProtocolOff
	}
	if protocol := effectiveAgenticProtocol; protocol != "" && protocol != llm.AgenticProtocolOff {
		if _, agenticErr := llm.NewAgenticModel(ctx, b.cfg.LLM, protocol); agenticErr != nil {
			effectiveAgenticProtocol = llm.AgenticProtocolOff
			logger.Warn("[RUNNER] Agentic beta model disabled; falling back to Message runtime",
				"protocol", protocol, "error", agenticErr)
		} else {
			logger.RunnerEvent("agentic_beta_model_available", "protocol", protocol)
		}
	}
	return effectiveAgenticProtocol
}

func (b *runtimeBundleBuilder) emitMCPHandleErrors(bundle *RunnerBundle) {
	if bundle == nil {
		return
	}
	for _, handle := range bundle.MCPHandles {
		if handle == nil || handle.LastError == nil {
			continue
		}
		wailsruntime.EventsEmit(b.chat.ctx, "agent:error",
			fmt.Sprintf("MCP server %s unavailable (%s): %v", handle.Name, handle.State, handle.LastError))
	}
}

func (b *runtimeBundleBuilder) closeWithError(err error) (*RunnerBundle, error) {
	if b.surface != nil {
		b.chat.closeMCPHandlesLocked(b.surface.Handles)
	}
	return nil, err
}

func registerWrappedCatalogEntries(catalog *tools.ToolCatalog, provider *deferredMCPProvider, entries []tools.CatalogEntry, label string) error {
	for _, entry := range entries {
		wrapped := entry
		wrapped.Tool = tools.WrapMCPToolWithPermissionCheck(wrapped, provider)
		if err := catalog.Register(wrapped); err != nil {
			return fmt.Errorf("failed to register %s %s: %w", label, wrapped.CanonicalName, err)
		}
	}
	return nil
}

func runtimeAlwaysLoadedTools(catalog *tools.ToolCatalog) []einotool.BaseTool {
	if catalog == nil {
		return nil
	}
	extraTools := make([]einotool.BaseTool, 0, len(catalog.CanonicalNames()))
	for _, entry := range catalog.Entries() {
		if entry.ShouldDefer && !entry.AlwaysLoad {
			continue
		}
		extraTools = append(extraTools, entry.Tool)
	}
	return extraTools
}
