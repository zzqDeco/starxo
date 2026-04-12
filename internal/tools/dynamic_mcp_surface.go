package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type DeferredMCPProvider interface {
	DeferredMCPState(ctx context.Context) (DeferredMCPState, error)
	LookupCatalogEntry(name string) (CatalogEntry, bool)
}

func NewDynamicMCPSurfaceMiddleware(provider DeferredMCPProvider) adk.ChatModelAgentMiddleware {
	return &dynamicMCPSurfaceMiddleware{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
		provider:                     provider,
	}
}

type dynamicMCPSurfaceMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
	provider DeferredMCPProvider
}

func (m *dynamicMCPSurfaceMiddleware) WrapModel(ctx context.Context, cm model.BaseChatModel, mc *adk.ModelContext) (model.BaseChatModel, error) {
	state, err := m.provider.DeferredMCPState(ctx)
	if err != nil {
		return nil, err
	}
	return &dynamicMCPModelWrapper{
		base:            cm,
		allTools:        mc.Tools,
		state:           state,
		catalogProvider: m.provider,
	}, nil
}

func (m *dynamicMCPSurfaceMiddleware) WrapInvokableToolCall(ctx context.Context, endpoint adk.InvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.InvokableToolCallEndpoint, error) {
	return func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
		if err := m.ensureToolCallable(ctx, tCtx.Name); err != nil {
			return "", err
		}
		return endpoint(ctx, argumentsInJSON, opts...)
	}, nil
}

func (m *dynamicMCPSurfaceMiddleware) WrapStreamableToolCall(ctx context.Context, endpoint adk.StreamableToolCallEndpoint, tCtx *adk.ToolContext) (adk.StreamableToolCallEndpoint, error) {
	return func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (*schema.StreamReader[string], error) {
		if err := m.ensureToolCallable(ctx, tCtx.Name); err != nil {
			return nil, err
		}
		return endpoint(ctx, argumentsInJSON, opts...)
	}, nil
}

func (m *dynamicMCPSurfaceMiddleware) ensureToolCallable(ctx context.Context, toolName string) error {
	state, err := m.provider.DeferredMCPState(ctx)
	if err != nil {
		return err
	}
	if toolName == "tool_search" {
		if toolSearchVisible(state) {
			return nil
		}
		return fmt.Errorf("tool_search is unavailable because no deferred MCP tools are currently searchable")
	}
	entry, ok := m.provider.LookupCatalogEntry(toolName)
	if !ok {
		return nil
	}
	if state.IsCurrentlyLoaded(entry.CanonicalName) {
		return nil
	}
	if state.IsCurrentlySearchable(entry.CanonicalName) {
		return fmt.Errorf("tool %s is not currently loaded; use tool_search first", entry.CanonicalName)
	}
	if decision, ok := state.SearchDecisions[entry.CanonicalName]; ok && decision.Reason != "" {
		return fmt.Errorf("tool %s is unavailable in the current mode or runtime: %s", entry.CanonicalName, decision.Reason)
	}
	return fmt.Errorf("tool %s is unavailable in the current mode or runtime", entry.CanonicalName)
}

type dynamicMCPModelWrapper struct {
	base            model.BaseChatModel
	allTools        []*schema.ToolInfo
	state           DeferredMCPState
	catalogProvider DeferredMCPProvider
}

func (w *dynamicMCPModelWrapper) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	msgs := injectAnnouncement(input, buildDeferredAnnouncement(w.state))
	tools := filterVisibleToolInfos(w.allTools, w.state, w.catalogProvider)
	return w.base.Generate(ctx, msgs, append(opts, model.WithTools(tools))...)
}

func (w *dynamicMCPModelWrapper) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	msgs := injectAnnouncement(input, buildDeferredAnnouncement(w.state))
	tools := filterVisibleToolInfos(w.allTools, w.state, w.catalogProvider)
	return w.base.Stream(ctx, msgs, append(opts, model.WithTools(tools))...)
}

func toolSearchVisible(state DeferredMCPState) bool {
	return len(state.SearchablePoolForMode) > 0 || len(state.PendingMCPServers) > 0
}

func buildDeferredAnnouncement(state DeferredMCPState) *schema.Message {
	if len(state.SearchablePoolForMode) == 0 {
		return nil
	}

	var b strings.Builder
	b.WriteString("<available-deferred-mcp-tools>\n")
	for _, entry := range state.SearchablePoolForMode {
		b.WriteString(entry.CanonicalName)
		b.WriteString("\n")
	}
	b.WriteString("</available-deferred-mcp-tools>")
	return schema.UserMessage(b.String())
}

func injectAnnouncement(input []*schema.Message, msg *schema.Message) []*schema.Message {
	if msg == nil {
		return input
	}
	if len(input) == 0 {
		return []*schema.Message{msg}
	}

	out := make([]*schema.Message, 0, len(input)+1)
	if input[0].Role == schema.System {
		out = append(out, input[0], msg)
		out = append(out, input[1:]...)
		return out
	}
	out = append(out, msg)
	out = append(out, input...)
	return out
}

func filterVisibleToolInfos(all []*schema.ToolInfo, state DeferredMCPState, provider DeferredMCPProvider) []*schema.ToolInfo {
	loaded := make(map[string]struct{}, len(state.CurrentLoadedTools))
	for _, entry := range state.CurrentLoadedTools {
		loaded[entry.CanonicalName] = struct{}{}
	}

	visible := make([]*schema.ToolInfo, 0, len(all))
	for _, info := range all {
		if info == nil {
			continue
		}
		if info.Name == "tool_search" {
			if toolSearchVisible(state) {
				visible = append(visible, info)
			}
			continue
		}
		if entry, ok := provider.LookupCatalogEntry(info.Name); ok {
			if _, exists := loaded[entry.CanonicalName]; exists {
				visible = append(visible, info)
			}
			continue
		}
		visible = append(visible, info)
	}
	return visible
}
