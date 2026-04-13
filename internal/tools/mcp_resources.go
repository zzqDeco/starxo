package tools

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultMCPResourceLimit = 50
	maxMCPResourceTextBytes = 32 * 1024
	maxMCPResourceBlobBytes = 32 * 1024
	ListMCPResourcesName    = "list_mcp_resources"
	ListMCPTemplatesName    = "list_mcp_resource_templates"
	ReadMCPResourceName     = "read_mcp_resource"
)

type MCPHandleSource interface {
	MCPHandleSnapshot() []*MCPServerHandle
}

type ListMCPResourcesInput struct {
	Server string `json:"server,omitempty"`
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type MCPResourceSummary struct {
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	URI         string `json:"uri"`
	MIMEType    string `json:"mimeType,omitempty"`
	Description string `json:"description,omitempty"`
	Size        int64  `json:"size,omitempty"`
}

type ListMCPResourcesOutput struct {
	Server     string               `json:"server"`
	Resources  []MCPResourceSummary `json:"resources"`
	NextCursor string               `json:"next_cursor,omitempty"`
}

type ListMCPResourceTemplatesOutput struct {
	Server     string                       `json:"server"`
	Templates  []MCPResourceTemplateSummary `json:"templates"`
	NextCursor string                       `json:"next_cursor,omitempty"`
}

type MCPResourceTemplateSummary struct {
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	URITemplate string `json:"uri_template"`
	MIMEType    string `json:"mimeType,omitempty"`
	Description string `json:"description,omitempty"`
}

type ReadMCPResourceInput struct {
	Server            string `json:"server,omitempty"`
	URI               string `json:"uri"`
	IncludeBlobBase64 bool   `json:"include_blob_base64,omitempty"`
}

type MCPReadResourceContent struct {
	URI           string `json:"uri"`
	MIMEType      string `json:"mimeType,omitempty"`
	Text          string `json:"text,omitempty"`
	BlobBase64    string `json:"blobBase64,omitempty"`
	IsBinary      bool   `json:"isBinary"`
	OriginalBytes int    `json:"originalBytes"`
	ReturnedBytes int    `json:"returnedBytes"`
	Truncated     bool   `json:"truncated"`
}

type ReadMCPResourceOutput struct {
	Server   string                   `json:"server"`
	URI      string                   `json:"uri"`
	Contents []MCPReadResourceContent `json:"contents"`
}

func NewListMCPResourcesTool(source MCPHandleSource) (tool.BaseTool, error) {
	return toolutils.InferTool(ListMCPResourcesName,
		"List cached MCP resources for a server. Returns resource names and URIs only.",
		func(ctx context.Context, input ListMCPResourcesInput) (ListMCPResourcesOutput, error) {
			handle, err := selectMCPHandle(source.MCPHandleSnapshot(), input.Server)
			if err != nil {
				return ListMCPResourcesOutput{}, err
			}
			if !handle.SupportsResources() {
				return ListMCPResourcesOutput{}, unsupportedResourceCapabilityError(source.MCPHandleSnapshot(), "resources")
			}
			start, limit := normalizeCursorAndLimit(input.Cursor, input.Limit, len(handle.Resources))
			items := handle.Resources[start:limit]
			out := ListMCPResourcesOutput{
				Server:    handle.Name,
				Resources: make([]MCPResourceSummary, 0, len(items)),
			}
			if limit < len(handle.Resources) {
				out.NextCursor = fmt.Sprintf("%d", limit)
			}
			for _, resource := range items {
				out.Resources = append(out.Resources, MCPResourceSummary{
					Name:        resource.Name,
					Title:       resource.Title,
					URI:         resource.URI,
					MIMEType:    resource.MIMEType,
					Description: resource.Description,
					Size:        resource.Size,
				})
			}
			return out, nil
		})
}

func NewListMCPResourceTemplatesTool(source MCPHandleSource) (tool.BaseTool, error) {
	return toolutils.InferTool(ListMCPTemplatesName,
		"List cached MCP resource templates for a server.",
		func(ctx context.Context, input ListMCPResourcesInput) (ListMCPResourceTemplatesOutput, error) {
			handle, err := selectMCPHandle(source.MCPHandleSnapshot(), input.Server)
			if err != nil {
				return ListMCPResourceTemplatesOutput{}, err
			}
			if !handle.SupportsResources() {
				return ListMCPResourceTemplatesOutput{}, unsupportedResourceCapabilityError(source.MCPHandleSnapshot(), "resource templates")
			}
			start, limit := normalizeCursorAndLimit(input.Cursor, input.Limit, len(handle.ResourceTemplates))
			items := handle.ResourceTemplates[start:limit]
			out := ListMCPResourceTemplatesOutput{
				Server:    handle.Name,
				Templates: make([]MCPResourceTemplateSummary, 0, len(items)),
			}
			if limit < len(handle.ResourceTemplates) {
				out.NextCursor = fmt.Sprintf("%d", limit)
			}
			for _, tpl := range items {
				out.Templates = append(out.Templates, MCPResourceTemplateSummary{
					Name:        tpl.Name,
					Title:       tpl.Title,
					URITemplate: tpl.URITemplate,
					MIMEType:    tpl.MIMEType,
					Description: tpl.Description,
				})
			}
			return out, nil
		})
}

func NewReadMCPResourceTool(source MCPHandleSource) (tool.BaseTool, error) {
	return toolutils.InferTool(ReadMCPResourceName,
		"Read an MCP resource by URI. Text is truncated to a safe limit; binary content is omitted unless include_blob_base64=true.",
		func(ctx context.Context, input ReadMCPResourceInput) (ReadMCPResourceOutput, error) {
			handle, err := selectMCPHandle(source.MCPHandleSnapshot(), input.Server)
			if err != nil {
				return ReadMCPResourceOutput{}, err
			}
			if !handle.SupportsResources() {
				return ReadMCPResourceOutput{}, unsupportedResourceCapabilityError(source.MCPHandleSnapshot(), "resources")
			}
			res, err := handle.Session.ReadResource(ctx, &mcp.ReadResourceParams{URI: input.URI})
			if err != nil {
				return ReadMCPResourceOutput{}, err
			}
			out := ReadMCPResourceOutput{
				Server:   handle.Name,
				URI:      input.URI,
				Contents: make([]MCPReadResourceContent, 0, len(res.Contents)),
			}
			for _, content := range res.Contents {
				item := MCPReadResourceContent{
					URI:      content.URI,
					MIMEType: content.MIMEType,
				}
				if len(content.Blob) > 0 {
					item.IsBinary = true
					item.OriginalBytes = len(content.Blob)
					if input.IncludeBlobBase64 {
						blob := content.Blob
						if len(blob) > maxMCPResourceBlobBytes {
							blob = blob[:maxMCPResourceBlobBytes]
							item.Truncated = true
						}
						item.ReturnedBytes = len(blob)
						item.BlobBase64 = base64.StdEncoding.EncodeToString(blob)
					}
				} else {
					item.OriginalBytes = len(content.Text)
					text := content.Text
					if len(text) > maxMCPResourceTextBytes {
						text = text[:maxMCPResourceTextBytes]
						item.Truncated = true
					}
					item.Text = text
					item.ReturnedBytes = len(text)
				}
				out.Contents = append(out.Contents, item)
			}
			return out, nil
		})
}

func NewMCPResourceCatalogEntries(source MCPHandleSource) ([]CatalogEntry, error) {
	listTool, err := NewListMCPResourcesTool(source)
	if err != nil {
		return nil, err
	}
	templateTool, err := NewListMCPResourceTemplatesTool(source)
	if err != nil {
		return nil, err
	}
	readTool, err := NewReadMCPResourceTool(source)
	if err != nil {
		return nil, err
	}

	return []CatalogEntry{
		{
			CanonicalName:   ListMCPResourcesName,
			Source:          ToolSourceMCP,
			Kind:            ToolKindResourceList,
			Title:           "List MCP Resources",
			Description:     "List available MCP resources on a connected server.",
			ToolClass:       ToolClassMCPResource,
			DeferReason:     "mcp_default",
			ShouldDefer:     true,
			IsMcp:           true,
			IsResourceTool:  true,
			ReadOnlyHint:    true,
			ReadOnlyTrusted: true,
			PermissionSpec: PermissionSpec{
				AllowSearch:  true,
				AllowExecute: true,
			},
			Tool: listTool,
		},
		{
			CanonicalName:   ListMCPTemplatesName,
			Source:          ToolSourceMCP,
			Kind:            ToolKindResourceTemplate,
			Title:           "List MCP Resource Templates",
			Description:     "List available MCP resource templates on a connected server.",
			ToolClass:       ToolClassMCPResource,
			DeferReason:     "mcp_default",
			ShouldDefer:     true,
			IsMcp:           true,
			IsResourceTool:  true,
			ReadOnlyHint:    true,
			ReadOnlyTrusted: true,
			PermissionSpec: PermissionSpec{
				AllowSearch:  true,
				AllowExecute: true,
			},
			Tool: templateTool,
		},
		{
			CanonicalName:   ReadMCPResourceName,
			Source:          ToolSourceMCP,
			Kind:            ToolKindResourceRead,
			Title:           "Read MCP Resource",
			Description:     "Read an MCP resource by URI from a connected server.",
			ToolClass:       ToolClassMCPResource,
			DeferReason:     "mcp_default",
			ShouldDefer:     true,
			IsMcp:           true,
			IsResourceTool:  true,
			ReadOnlyHint:    true,
			ReadOnlyTrusted: true,
			PermissionSpec: PermissionSpec{
				AllowSearch:  true,
				AllowExecute: true,
			},
			Tool: readTool,
		},
	}, nil
}

func selectMCPHandle(handles []*MCPServerHandle, requested string) (*MCPServerHandle, error) {
	if requested != "" {
		for _, handle := range handles {
			if handle != nil && handle.Name == requested {
				return handle, nil
			}
		}
		return nil, fmt.Errorf("mcp server %q not found", requested)
	}

	connected := make([]*MCPServerHandle, 0, len(handles))
	for _, handle := range handles {
		if handle != nil && handle.State == MCPServerStateConnected {
			connected = append(connected, handle)
		}
	}
	if len(connected) == 1 {
		return connected[0], nil
	}
	if len(connected) == 0 {
		return nil, fmt.Errorf("no connected mcp server available")
	}
	return nil, fmt.Errorf("multiple mcp servers are connected; server is required")
}

func unsupportedResourceCapabilityError(handles []*MCPServerHandle, capability string) error {
	servers := make([]string, 0, len(handles))
	for _, handle := range handles {
		if handle != nil && handle.SupportsResources() {
			servers = append(servers, handle.Name)
		}
	}
	if len(servers) == 0 {
		return fmt.Errorf("no connected mcp server supports %s", capability)
	}
	return fmt.Errorf("%s unsupported for this server; supported servers: %v", capability, servers)
}

func normalizeCursorAndLimit(cursor string, limit, total int) (int, int) {
	start := 0
	if cursor != "" {
		fmt.Sscanf(cursor, "%d", &start)
	}
	if start < 0 {
		start = 0
	}
	if start > total {
		start = total
	}
	if limit <= 0 || limit > defaultMCPResourceLimit {
		limit = defaultMCPResourceLimit
	}
	end := start + limit
	if end > total {
		end = total
	}
	return start, end
}
