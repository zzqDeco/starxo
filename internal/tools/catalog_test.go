package tools

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/require"
)

type testTool struct {
	name string
}

func (t testTool) Info(context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: t.name}, nil
}

func (t testTool) InvokableRun(context.Context, string, ...tool.Option) (string, error) {
	return "{}", nil
}

func TestNormalizeMCPNamePart(t *testing.T) {
	t.Parallel()

	name, err := NormalizeMCPNamePart(" GitHub / Search  API ")
	require.NoError(t, err)
	require.Equal(t, "GitHub_Search_API", name)

	name, err = NormalizeMCPNamePart("a***b")
	require.NoError(t, err)
	require.Equal(t, "a_b", name)

	_, err = NormalizeMCPNamePart("   ")
	require.Error(t, err)

	_, err = NormalizeMCPNamePart("!!!")
	require.Error(t, err)
}

func TestCanonicalMCPToolName(t *testing.T) {
	t.Parallel()

	name, err := CanonicalMCPToolName("GitHub Server", "search/repo")
	require.NoError(t, err)
	require.Equal(t, "mcp__GitHub_Server__search_repo", name)
}

func TestToolCatalogRegisterAndLookup(t *testing.T) {
	t.Parallel()

	catalog := NewToolCatalog()
	entry := CatalogEntry{
		CanonicalName: "mcp__GitHub__search_repo",
		Aliases:       []string{"search_repo", "searchRepo"},
		Tool:          testTool{name: "mcp__GitHub__search_repo"},
	}

	require.NoError(t, catalog.Register(entry))

	got, ok := catalog.Get("mcp__GitHub__search_repo")
	require.True(t, ok)
	require.Equal(t, entry.CanonicalName, got.CanonicalName)

	got, ok = catalog.LookupExact("MCP__github__SEARCH_REPO")
	require.True(t, ok)
	require.Equal(t, entry.CanonicalName, got.CanonicalName)

	got, ok = catalog.LookupExact("searchrepo")
	require.True(t, ok)
	require.Equal(t, entry.CanonicalName, got.CanonicalName)
}

func TestToolCatalogCanonicalConflict(t *testing.T) {
	t.Parallel()

	catalog := NewToolCatalog()
	entry := CatalogEntry{
		CanonicalName: "mcp__GitHub__search_repo",
		Tool:          testTool{name: "mcp__GitHub__search_repo"},
	}

	require.NoError(t, catalog.Register(entry))
	require.Error(t, catalog.Register(entry))
}

func TestToolCatalogAliasConflict(t *testing.T) {
	t.Parallel()

	catalog := NewToolCatalog()
	require.NoError(t, catalog.Register(CatalogEntry{
		CanonicalName: "mcp__GitHub__search_repo",
		Aliases:       []string{"search"},
		Tool:          testTool{name: "mcp__GitHub__search_repo"},
	}))

	err := catalog.Register(CatalogEntry{
		CanonicalName: "mcp__GitHub__search_issue",
		Aliases:       []string{"SEARCH"},
		Tool:          testTool{name: "mcp__GitHub__search_issue"},
	})
	require.Error(t, err)
}
