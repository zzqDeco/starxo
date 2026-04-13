package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/eino-contrib/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"starxo/internal/config"
)

type MCPServerState string

const (
	MCPServerStateDisabled  MCPServerState = "disabled"
	MCPServerStatePending   MCPServerState = "pending"
	MCPServerStateConnected MCPServerState = "connected"
	MCPServerStateFailed    MCPServerState = "failed"
	MCPServerStateNeedsAuth MCPServerState = "needs_auth"
)

type MCPServerHandle struct {
	Name         string
	Config       config.MCPServerConfig
	State        MCPServerState
	Session      *mcp.ClientSession
	Capabilities *mcp.ServerCapabilities

	Tools             []*mcp.Tool
	Resources         []*mcp.Resource
	ResourceTemplates []*mcp.ResourceTemplate

	LastError         error
	LastRefreshedAt   time.Time
	ToolMetadataReady bool
}

func ConnectMCPServerHandle(ctx context.Context, cfg config.MCPServerConfig) (*MCPServerHandle, error) {
	handle := &MCPServerHandle{
		Name:   cfg.Name,
		Config: cfg,
		State:  MCPServerStatePending,
	}
	if !cfg.Enabled {
		handle.State = MCPServerStateDisabled
		return handle, nil
	}

	session, err := ConnectMCPServer(ctx, cfg)
	if err != nil {
		handle.LastError = err
		handle.State = classifyMCPError(err)
		return handle, nil
	}

	handle.Session = session
	if init := session.InitializeResult(); init != nil {
		handle.Capabilities = init.Capabilities
	}
	if err := handle.RefreshMetadata(ctx); err != nil {
		handle.LastError = err
		return handle, nil
	}

	handle.State = MCPServerStateConnected
	return handle, nil
}

func (h *MCPServerHandle) Close() error {
	if h == nil || h.Session == nil {
		return nil
	}
	return h.Session.Close()
}

func (h *MCPServerHandle) SupportsResources() bool {
	return h != nil && h.Capabilities != nil && h.Capabilities.Resources != nil
}

func (h *MCPServerHandle) SupportsTools() bool {
	return h != nil && h.Capabilities != nil && h.Capabilities.Tools != nil
}

func (h *MCPServerHandle) RefreshMetadata(ctx context.Context) error {
	if h == nil || h.Session == nil {
		return fmt.Errorf("mcp session is not connected")
	}

	tools, err := listAllTools(ctx, h.Session)
	if err != nil {
		h.State = classifyMCPError(err)
		return fmt.Errorf("list tools for %s: %w", h.Name, err)
	}
	h.Tools = tools
	h.ToolMetadataReady = true

	if h.SupportsResources() {
		if resources, err := listAllResources(ctx, h.Session); err == nil {
			h.Resources = resources
		}
		if templates, err := listAllResourceTemplates(ctx, h.Session); err == nil {
			h.ResourceTemplates = templates
		}
	}

	h.LastRefreshedAt = time.Now()
	return nil
}

func listAllTools(ctx context.Context, session *mcp.ClientSession) ([]*mcp.Tool, error) {
	var out []*mcp.Tool
	cursor := ""
	for {
		res, err := session.ListTools(ctx, &mcp.ListToolsParams{Cursor: cursor})
		if err != nil {
			return nil, err
		}
		out = append(out, res.Tools...)
		if res.NextCursor == "" {
			return out, nil
		}
		cursor = res.NextCursor
	}
}

func listAllResources(ctx context.Context, session *mcp.ClientSession) ([]*mcp.Resource, error) {
	var out []*mcp.Resource
	cursor := ""
	for {
		res, err := session.ListResources(ctx, &mcp.ListResourcesParams{Cursor: cursor})
		if err != nil {
			return nil, err
		}
		out = append(out, res.Resources...)
		if res.NextCursor == "" {
			return out, nil
		}
		cursor = res.NextCursor
	}
}

func listAllResourceTemplates(ctx context.Context, session *mcp.ClientSession) ([]*mcp.ResourceTemplate, error) {
	var out []*mcp.ResourceTemplate
	cursor := ""
	for {
		res, err := session.ListResourceTemplates(ctx, &mcp.ListResourceTemplatesParams{Cursor: cursor})
		if err != nil {
			return nil, err
		}
		out = append(out, res.ResourceTemplates...)
		if res.NextCursor == "" {
			return out, nil
		}
		cursor = res.NextCursor
	}
}

func classifyMCPError(err error) MCPServerState {
	if err == nil {
		return MCPServerStateConnected
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "auth"),
		strings.Contains(msg, "unauthorized"),
		strings.Contains(msg, "forbidden"),
		strings.Contains(msg, "token"),
		strings.Contains(msg, "permission denied"):
		return MCPServerStateNeedsAuth
	case strings.Contains(msg, "deadline"),
		strings.Contains(msg, "timeout"),
		strings.Contains(msg, "tempor"),
		strings.Contains(msg, "connection reset"),
		strings.Contains(msg, "connection refused"),
		strings.Contains(msg, "eof"):
		return MCPServerStatePending
	default:
		return MCPServerStateFailed
	}
}

type MCPActionAdapter struct {
	handle       *MCPServerHandle
	canonical    string
	remoteName   string
	toolInfo     *schema.ToolInfo
	displayTitle string
}

func NewMCPActionAdapter(handle *MCPServerHandle, raw *mcp.Tool) (*MCPActionAdapter, CatalogEntry, error) {
	if handle == nil || raw == nil {
		return nil, CatalogEntry{}, fmt.Errorf("mcp handle and tool are required")
	}

	canonical, err := CanonicalMCPToolName(handle.Name, raw.Name)
	if err != nil {
		return nil, CatalogEntry{}, err
	}

	marshaledInputSchema, err := sonic.Marshal(raw.InputSchema)
	if err != nil {
		return nil, CatalogEntry{}, fmt.Errorf("marshal input schema for %s: %w", raw.Name, err)
	}
	inputSchema := &jsonschema.Schema{}
	if err := sonic.Unmarshal(marshaledInputSchema, inputSchema); err != nil {
		return nil, CatalogEntry{}, fmt.Errorf("parse input schema for %s: %w", raw.Name, err)
	}

	info := &schema.ToolInfo{
		Name:        canonical,
		Desc:        raw.Description,
		ParamsOneOf: schema.NewParamsOneOfByJSONSchema(inputSchema),
	}

	entry := CatalogEntry{
		CanonicalName: canonical,
		RemoteName:    raw.Name,
		Aliases:       []string{raw.Name},
		Source:        ToolSourceMCP,
		Server:        handle.Name,
		Kind:          ToolKindAction,
		Title:         firstNonEmpty(raw.Title, annotationTitle(raw.Annotations), raw.Name),
		Description:   raw.Description,
		ToolClass:     ToolClassMCPAction,
		DeferReason:   "mcp_default",
		ShouldDefer:   true,
		IsMcp:         true,
		ReadOnlyHint:  raw.Annotations != nil && raw.Annotations.ReadOnlyHint,
		ReadOnlyTrusted: raw.Annotations != nil &&
			raw.Annotations.ReadOnlyHint,
		PermissionSpec: PermissionSpec{
			AllowSearch:  true,
			AllowExecute: true,
		},
		Tool: &MCPActionAdapter{
			handle:       handle,
			canonical:    canonical,
			remoteName:   raw.Name,
			toolInfo:     info,
			displayTitle: firstNonEmpty(raw.Title, annotationTitle(raw.Annotations), raw.Name),
		},
	}
	return entry.Tool.(*MCPActionAdapter), entry, nil
}

func (a *MCPActionAdapter) Info(context.Context) (*schema.ToolInfo, error) {
	return a.toolInfo, nil
}

func (a *MCPActionAdapter) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	if a.handle == nil || a.handle.Session == nil {
		return "", fmt.Errorf("mcp server %s is not connected", a.remoteName)
	}

	result, err := a.handle.Session.CallTool(ctx, &mcp.CallToolParams{
		Name:      a.remoteName,
		Arguments: json.RawMessage(argumentsInJSON),
	})
	if err != nil {
		return "", fmt.Errorf("failed to call mcp tool %s: %w", a.remoteName, err)
	}

	marshaled, err := sonic.MarshalString(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal mcp tool result: %w", err)
	}
	if result.IsError {
		return "", fmt.Errorf("mcp tool %s returned error: %s", a.remoteName, marshaled)
	}
	return marshaled, nil
}

func BuildMCPActionCatalog(ctx context.Context, cfgs []config.MCPServerConfig) ([]*MCPServerHandle, *ToolCatalog, error) {
	catalog := NewToolCatalog()
	handles := make([]*MCPServerHandle, 0, len(cfgs))

	for _, cfg := range cfgs {
		if !cfg.Enabled {
			continue
		}
		handle, err := ConnectMCPServerHandle(ctx, cfg)
		if err != nil {
			return nil, nil, err
		}
		handles = append(handles, handle)

		if !handle.ToolMetadataReady {
			continue
		}
		for _, raw := range handle.Tools {
			_, entry, err := NewMCPActionAdapter(handle, raw)
			if err != nil {
				return nil, nil, err
			}
			if err := catalog.Register(entry); err != nil {
				return nil, nil, err
			}
		}
	}

	return handles, catalog, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func annotationTitle(ann *mcp.ToolAnnotations) string {
	if ann == nil {
		return ""
	}
	return ann.Title
}

// ConnectMCPServer establishes a connection to an MCP server based on the
// provided configuration. It supports two transport types:
//   - "stdio": starts a subprocess and communicates over stdin/stdout
//   - "sse": connects to an HTTP SSE endpoint
func ConnectMCPServer(ctx context.Context, cfg config.MCPServerConfig) (*mcp.ClientSession, error) {
	client := mcp.NewClient(&mcp.Implementation{
		Name:    cfg.Name,
		Version: "1.0.0",
	}, nil)

	var transport mcp.Transport

	switch cfg.Transport {
	case "stdio":
		if cfg.Command == "" {
			return nil, fmt.Errorf("MCP server %q: command is required for stdio transport", cfg.Name)
		}
		cmd := exec.CommandContext(ctx, cfg.Command, cfg.Args...)
		for k, v := range cfg.Env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
		transport = &mcp.CommandTransport{
			Command: cmd,
		}

	case "sse":
		if cfg.URL == "" {
			return nil, fmt.Errorf("MCP server %q: URL is required for SSE transport", cfg.Name)
		}
		transport = &mcp.SSEClientTransport{
			Endpoint: cfg.URL,
		}

	default:
		return nil, fmt.Errorf("MCP server %q: unsupported transport type %q (supported: stdio, sse)", cfg.Name, cfg.Transport)
	}

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MCP server %q: %w", cfg.Name, err)
	}

	return session, nil
}
