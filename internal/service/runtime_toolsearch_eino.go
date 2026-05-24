package service

import (
	"context"

	"github.com/cloudwego/eino/adk"
	einotoolsearch "github.com/cloudwego/eino/adk/middlewares/dynamictool/toolsearch"
	einotool "github.com/cloudwego/eino/components/tool"

	"starxo/internal/tools"
)

func newEinoV09ToolSearchHandler(ctx context.Context, provider *deferredMCPProvider, mode, toolSearchMode, agenticProtocol string) (adk.ChatModelAgentMiddleware, error) {
	if provider == nil || provider.bundle == nil || provider.bundle.MCPCatalog == nil {
		return nil, nil
	}
	return newEinoV09ToolSearchHandlerForCatalog(ctx, provider.bundle.MCPCatalog, mode, toolSearchMode, agenticProtocol, nil)
}

func newEinoV09ToolSearchHandlerForCatalog(ctx context.Context, catalog *tools.ToolCatalog, mode, toolSearchMode, agenticProtocol string, allowEntry func(tools.CatalogEntry) bool) (adk.ChatModelAgentMiddleware, error) {
	dynamicTools := einoV09ToolSearchCandidatesFiltered(catalog, mode, allowEntry)
	if len(dynamicTools) == 0 {
		return nil, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return einotoolsearch.New(ctx, &einotoolsearch.Config{
		DynamicTools:       dynamicTools,
		UseModelToolSearch: useEinoV09ModelToolSearch(toolSearchMode, agenticProtocol),
	})
}

func einoV09ToolSearchCandidates(catalog *tools.ToolCatalog, mode string) []einotool.BaseTool {
	return einoV09ToolSearchCandidatesFiltered(catalog, mode, nil)
}

func einoV09ToolSearchCandidatesFiltered(catalog *tools.ToolCatalog, mode string, allowEntry func(tools.CatalogEntry) bool) []einotool.BaseTool {
	if catalog == nil {
		return nil
	}
	entries := catalog.Entries()
	dynamicTools := make([]einotool.BaseTool, 0, len(entries))
	for _, entry := range entries {
		if !entry.ShouldDefer || entry.AlwaysLoad || entry.Tool == nil {
			continue
		}
		if !einoV09PotentiallySearchable(entry, mode) {
			continue
		}
		if allowEntry != nil && !allowEntry(entry) {
			continue
		}
		dynamicTools = append(dynamicTools, entry.Tool)
	}
	return dynamicTools
}

func einoV09PotentiallySearchable(entry tools.CatalogEntry, mode string) bool {
	if !entry.PermissionSpec.AllowSearch {
		return false
	}
	if mode == "plan" && !entry.ReadOnlyEligible() {
		return false
	}
	return true
}

func useEinoV09ModelToolSearch(toolSearchMode, agenticProtocol string) bool {
	// Starxo still gates deferred tool execution through per-session discovery state.
	// Eino's model-native path puts dynamic tools in DeferredToolInfos, which can let
	// a model invoke them before Starxo has recorded discovery. Keep client-side
	// tool_search as the only active path until native search can pre-grant discovery.
	return false
}
