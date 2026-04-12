package tools

import (
	"testing"
	"time"

	"starxo/internal/model"
)

func TestExecuteToolSearch_ExactNamePrefersCurrentLoadedAndDoesNotWriteDiscovery(t *testing.T) {
	loaded := stubCatalogEntry("mcp__fs__grep")
	loaded.Aliases = []string{"grep"}

	output, records := ExecuteToolSearch(ToolSearchInput{Query: "GREP"}, ToolSearchState{
		CurrentLoaded: []CatalogEntry{loaded},
		SearchablePool: []CatalogEntry{
			loaded,
			stubCatalogEntry("mcp__fs__find"),
		},
	}, time.UnixMilli(123))

	if len(output.Matches) != 1 || output.Matches[0] != loaded.CanonicalName {
		t.Fatalf("unexpected exact match output: %#v", output)
	}
	if len(records) != 0 {
		t.Fatalf("expected no discovery records for already loaded tool, got %#v", records)
	}
}

func TestExecuteToolSearch_ReturnsCanonicalNamesForAliasMatches(t *testing.T) {
	entry := stubCatalogEntry("mcp__git__status")
	entry.Aliases = []string{"git_status", "status"}

	output, records := ExecuteToolSearch(ToolSearchInput{Query: "STATUS"}, ToolSearchState{
		SearchablePool: []CatalogEntry{entry},
	}, time.UnixMilli(456))

	if len(output.Matches) != 1 || output.Matches[0] != entry.CanonicalName {
		t.Fatalf("expected canonical name match, got %#v", output)
	}
	if len(records) != 1 || records[0].CanonicalName != entry.CanonicalName {
		t.Fatalf("expected discovery record for canonical name, got %#v", records)
	}
}

func TestExecuteToolSearch_SelectPartialMatchesDoNotFail(t *testing.T) {
	entry := stubCatalogEntry("mcp__git__status")
	entry.Aliases = []string{"status"}

	output, records := ExecuteToolSearch(ToolSearchInput{Query: "select:missing,STATUS,other"}, ToolSearchState{
		SearchablePool: []CatalogEntry{entry},
		PendingMCPServer: []string{
			"pending-server",
		},
	}, time.UnixMilli(789))

	if len(output.Matches) != 1 || output.Matches[0] != entry.CanonicalName {
		t.Fatalf("expected partial match success, got %#v", output)
	}
	if len(output.PendingMCPServers) != 0 {
		t.Fatalf("expected pending_mcp_servers to be omitted on non-empty matches, got %#v", output.PendingMCPServers)
	}
	if len(records) != 1 || records[0].CanonicalName != entry.CanonicalName {
		t.Fatalf("expected discovery record for matched deferred tool, got %#v", records)
	}
}

func TestExecuteToolSearch_ZeroMatchesIncludesPendingServers(t *testing.T) {
	output, records := ExecuteToolSearch(ToolSearchInput{Query: "select:missing"}, ToolSearchState{
		PendingMCPServer: []string{"alpha", "beta"},
	}, time.UnixMilli(111))

	if len(output.Matches) != 0 {
		t.Fatalf("expected zero matches, got %#v", output.Matches)
	}
	if len(output.PendingMCPServers) != 2 {
		t.Fatalf("expected pending servers on zero-match path, got %#v", output.PendingMCPServers)
	}
	if len(records) != 0 {
		t.Fatalf("expected no discovery records, got %#v", records)
	}
}

func TestExecuteToolSearch_AlwaysLoadedExactMatchDoesNotWriteDiscovery(t *testing.T) {
	entry := stubCatalogEntry("mcp__meta__resource_index")
	entry.AlwaysLoad = true

	output, records := ExecuteToolSearch(ToolSearchInput{Query: entry.CanonicalName}, ToolSearchState{
		CurrentLoaded: []CatalogEntry{entry},
		SearchablePool: []CatalogEntry{
			entry,
		},
	}, time.Now())

	if len(output.Matches) != 1 || output.Matches[0] != entry.CanonicalName {
		t.Fatalf("expected always-loaded canonical match, got %#v", output)
	}
	if len(records) != 0 {
		t.Fatalf("expected no discovery records for always-loaded tool, got %#v", records)
	}
}

func TestExecuteToolSearch_KeywordSearchUsesCanonicalMatches(t *testing.T) {
	entry := stubCatalogEntry("mcp__repo__open_issue")
	entry.Title = "Open Issue"
	entry.Description = "Open an issue in the repository"
	entry.SearchHint = "issue tracker"

	output, records := ExecuteToolSearch(ToolSearchInput{Query: "+issue tracker"}, ToolSearchState{
		SearchablePool: []CatalogEntry{entry},
	}, time.UnixMilli(222))

	if len(output.Matches) != 1 || output.Matches[0] != entry.CanonicalName {
		t.Fatalf("expected keyword match to return canonical name, got %#v", output)
	}
	if len(records) != 1 || records[0] != (model.DiscoveredToolRecord{
		CanonicalName: entry.CanonicalName,
		Server:        entry.Server,
		Kind:          entry.Kind,
		DiscoveredAt:  222,
	}) {
		t.Fatalf("unexpected discovery record: %#v", records)
	}
}
