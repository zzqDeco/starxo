package tools

import (
	"context"
	"fmt"
	"strings"

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
	SupportsResources     bool
}

type PermissionDecision struct {
	Allowed bool
	Reason  string
}

type ToolPermissionProvider interface {
	ToolPermissionContext(ctx context.Context) (ToolPermissionContext, error)
}

const (
	ToolPermissionDecisionAllowOnce    = "allow_once"
	ToolPermissionDecisionAllowSession = "allow_session"
	ToolPermissionDecisionDeny         = "deny"
)

type ToolPermissionRequest struct {
	RequestID   string `json:"requestId"`
	SessionID   string `json:"sessionId,omitempty"`
	ToolName    string `json:"toolName"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	ToolClass   string `json:"toolClass,omitempty"`
	Source      string `json:"source,omitempty"`
	Risk        string `json:"risk"`
	Input       string `json:"input,omitempty"`
	CreatedAt   int64  `json:"createdAt"`
}

type ToolPermissionResolution struct {
	Decision string `json:"decision"`
}

type ToolExecutionPermissionProvider interface {
	RequestToolPermission(ctx context.Context, entry CatalogEntry, argumentsInJSON string) (ToolPermissionResolution, error)
}

func (e CatalogEntry) ReadOnlyEligible() bool {
	return e.ReadOnlyTrusted && e.ReadOnlyHint
}

func CanSearchCatalogEntry(entry CatalogEntry, ctx ToolPermissionContext) PermissionDecision {
	if !entry.IsMcp {
		if !entry.PermissionSpec.AllowSearch {
			return PermissionDecision{Allowed: false, Reason: "search is disabled"}
		}
		if ctx.Mode == "plan" && !entry.ReadOnlyEligible() {
			return PermissionDecision{Allowed: false, Reason: "tool is not read-only in plan mode"}
		}
		return PermissionDecision{Allowed: true}
	}
	if !entry.PermissionSpec.AllowSearch {
		return PermissionDecision{Allowed: false, Reason: "search is disabled"}
	}
	if ctx.Mode == "plan" && !entry.ReadOnlyEligible() {
		return PermissionDecision{Allowed: false, Reason: "tool is not read-only in plan mode"}
	}

	if entry.IsResourceTool && entry.Server == "" {
		for _, server := range ctx.Servers {
			if !server.SupportsResources {
				continue
			}
			switch server.State {
			case MCPServerStateConnected:
				return PermissionDecision{Allowed: true}
			case MCPServerStatePending:
				if server.HasCachedToolMetadata {
					return PermissionDecision{Allowed: true}
				}
			}
		}
		return PermissionDecision{Allowed: false, Reason: "no searchable MCP resource server available"}
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
		if !entry.PermissionSpec.AllowExecute {
			return PermissionDecision{Allowed: false, Reason: "execution is disabled"}
		}
		if ctx.Mode == "plan" && !entry.ReadOnlyEligible() {
			return PermissionDecision{Allowed: false, Reason: "tool is not read-only in plan mode"}
		}
		return PermissionDecision{Allowed: true}
	}
	if !entry.PermissionSpec.AllowExecute {
		return PermissionDecision{Allowed: false, Reason: "execution is disabled"}
	}
	if ctx.Mode == "plan" && !entry.ReadOnlyEligible() {
		return PermissionDecision{Allowed: false, Reason: "tool is not read-only in plan mode"}
	}

	if entry.IsResourceTool && entry.Server == "" {
		for _, server := range ctx.Servers {
			if server.SupportsResources && server.State == MCPServerStateConnected {
				return PermissionDecision{Allowed: true}
			}
		}
		return PermissionDecision{Allowed: false, Reason: "no loadable MCP resource server available"}
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
	if err := p.requestExecutionPermission(ctx, argumentsInJSON); err != nil {
		return "", err
	}
	return inv.InvokableRun(ctx, argumentsInJSON, opts...)
}

func (p *permissionedTool) requestExecutionPermission(ctx context.Context, argumentsInJSON string) error {
	executionProvider, ok := p.provider.(ToolExecutionPermissionProvider)
	if !ok || p.entry.ReadOnlyEligible() {
		return nil
	}
	resolution, err := executionProvider.RequestToolPermission(ctx, p.entry, argumentsInJSON)
	if err != nil {
		return err
	}
	switch strings.TrimSpace(resolution.Decision) {
	case ToolPermissionDecisionAllowOnce, ToolPermissionDecisionAllowSession:
		return nil
	case ToolPermissionDecisionDeny:
		return fmt.Errorf("tool %s was denied by the user", p.entry.CanonicalName)
	case "":
		return fmt.Errorf("tool %s permission request returned an empty decision", p.entry.CanonicalName)
	default:
		return fmt.Errorf("tool %s permission request returned unsupported decision %q", p.entry.CanonicalName, resolution.Decision)
	}
}
