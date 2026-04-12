package tools

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type ToolPermissionContext struct {
	SessionID string
	Mode      string
	Servers   map[string]MCPServerPermissionState
}

type MCPServerPermissionState struct {
	State                 MCPServerState
	HasCachedToolMetadata bool
}

type PermissionDecision struct {
	Allowed bool
	Reason  string
}

type ToolPermissionProvider interface {
	ToolPermissionContext(ctx context.Context) (ToolPermissionContext, error)
}

func (e CatalogEntry) ReadOnlyEligible() bool {
	return e.ReadOnlyTrusted && e.ReadOnlyHint
}

func CanSearchCatalogEntry(entry CatalogEntry, ctx ToolPermissionContext) PermissionDecision {
	if !entry.IsMcp {
		return PermissionDecision{Allowed: true}
	}
	if !entry.PermissionSpec.AllowSearch {
		return PermissionDecision{Allowed: false, Reason: "search is disabled"}
	}
	if ctx.Mode == "plan" && !entry.ReadOnlyEligible() {
		return PermissionDecision{Allowed: false, Reason: "tool is not read-only in plan mode"}
	}

	server := ctx.Servers[entry.Server]
	switch server.State {
	case MCPServerStateConnected:
		return PermissionDecision{Allowed: true}
	case MCPServerStatePending:
		if server.HasCachedToolMetadata {
			return PermissionDecision{Allowed: true}
		}
		return PermissionDecision{Allowed: false, Reason: "server metadata is still pending"}
	case MCPServerStateNeedsAuth:
		return PermissionDecision{Allowed: false, Reason: "server needs authentication"}
	case MCPServerStateDisabled:
		return PermissionDecision{Allowed: false, Reason: "server is disabled"}
	default:
		return PermissionDecision{Allowed: false, Reason: "server is unavailable"}
	}
}

func CanLoadCatalogEntry(entry CatalogEntry, ctx ToolPermissionContext) PermissionDecision {
	if !entry.IsMcp {
		return PermissionDecision{Allowed: true}
	}
	if !entry.PermissionSpec.AllowExecute {
		return PermissionDecision{Allowed: false, Reason: "execution is disabled"}
	}
	if ctx.Mode == "plan" && !entry.ReadOnlyEligible() {
		return PermissionDecision{Allowed: false, Reason: "tool is not read-only in plan mode"}
	}

	server := ctx.Servers[entry.Server]
	switch server.State {
	case MCPServerStateConnected:
		return PermissionDecision{Allowed: true}
	case MCPServerStatePending:
		return PermissionDecision{Allowed: false, Reason: "server is pending"}
	case MCPServerStateNeedsAuth:
		return PermissionDecision{Allowed: false, Reason: "server needs authentication"}
	case MCPServerStateDisabled:
		return PermissionDecision{Allowed: false, Reason: "server is disabled"}
	default:
		return PermissionDecision{Allowed: false, Reason: "server is unavailable"}
	}
}

type permissionedTool struct {
	inner    tool.BaseTool
	entry    CatalogEntry
	provider ToolPermissionProvider
}

func WrapMCPToolWithPermissionCheck(entry CatalogEntry, provider ToolPermissionProvider) tool.BaseTool {
	if provider == nil {
		return entry.Tool
	}
	return &permissionedTool{
		inner:    entry.Tool,
		entry:    entry,
		provider: provider,
	}
}

func (p *permissionedTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return p.inner.Info(ctx)
}

func (p *permissionedTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	inv, ok := p.inner.(tool.InvokableTool)
	if !ok {
		return "", fmt.Errorf("inner tool does not implement InvokableTool")
	}
	permCtx, err := p.provider.ToolPermissionContext(ctx)
	if err != nil {
		return "", err
	}
	decision := CanLoadCatalogEntry(p.entry, permCtx)
	if !decision.Allowed {
		return "", fmt.Errorf("tool %s is not permitted: %s", p.entry.CanonicalName, decision.Reason)
	}
	return inv.InvokableRun(ctx, argumentsInJSON, opts...)
}
