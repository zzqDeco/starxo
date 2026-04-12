package tools

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type stubInvokableTool struct {
	name string
}

func (t *stubInvokableTool) Info(context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: t.name}, nil
}

func (t *stubInvokableTool) InvokableRun(context.Context, string, ...tool.Option) (string, error) {
	return "ok", nil
}

func stubCatalogEntry(name string) CatalogEntry {
	return CatalogEntry{
		CanonicalName: name,
		Source:        ToolSourceMCP,
		Kind:          ToolKindAction,
		ShouldDefer:   true,
		IsMcp:         true,
		PermissionSpec: PermissionSpec{
			AllowSearch:  true,
			AllowExecute: true,
		},
		Tool: &stubInvokableTool{name: name},
	}
}
