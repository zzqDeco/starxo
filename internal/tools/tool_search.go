package tools

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"

	"starxo/internal/model"
)

const (
	defaultToolSearchLimit = 12
	maxToolSearchLimit     = 20

	ToolSearchUnavailableNoDeferredMessage = "tool_search is unavailable because no deferred tools are currently searchable"
)

type ToolSearchInput struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

type ToolSearchOutput struct {
	Matches           []string `json:"matches"`
	Query             string   `json:"query"`
	PendingMCPServers []string `json:"pending_mcp_servers,omitempty"`
}

type ToolSearchState struct {
	SearchablePool   []CatalogEntry
	CurrentLoaded    []CatalogEntry
	PendingMCPServer []string
}

type ToolSearchProvider interface {
	ToolSearchState(ctx context.Context) (ToolSearchState, error)
	AddDiscoveredTools(ctx context.Context, records []model.DiscoveredToolRecord) error
}

func NewToolSearchTool(provider ToolSearchProvider) (tool.InvokableTool, error) {
	return toolutils.InferTool("tool_search",
		"Search deferred tools by canonical name, alias, or keywords. Supports select:<tool>, select:A,B,C, exact name matching, and +required terms.",
		func(ctx context.Context, input ToolSearchInput) (ToolSearchOutput, error) {
			state, err := provider.ToolSearchState(ctx)
			if err != nil {
				return ToolSearchOutput{}, err
			}
			output, records := ExecuteToolSearch(input, state, time.Now())
			if len(records) > 0 {
				if err := provider.AddDiscoveredTools(ctx, records); err != nil {
					return ToolSearchOutput{}, err
				}
			}
			return output, nil
		})
}

func ExecuteToolSearch(input ToolSearchInput, state ToolSearchState, now time.Time) (ToolSearchOutput, []model.DiscoveredToolRecord) {
	limit := input.Limit
	if limit <= 0 {
		limit = defaultToolSearchLimit
	}
	if limit > maxToolSearchLimit {
		limit = maxToolSearchLimit
	}

	query := strings.TrimSpace(input.Query)
	if query == "" {
		return ToolSearchOutput{Query: input.Query}, nil
	}

	if strings.HasPrefix(strings.ToLower(query), "select:") {
		return executeSelectSearch(strings.TrimSpace(query[len("select:"):]), input.Query, limit, state, now)
	}

	if output, records, ok := executeExactNameSearch(query, input.Query, state, now); ok {
		return output, records
	}

	return executeKeywordSearch(query, input.Query, limit, state, now)
}

func executeSelectSearch(query, rawQuery string, limit int, state ToolSearchState, now time.Time) (ToolSearchOutput, []model.DiscoveredToolRecord) {
	parts := strings.Split(query, ",")
	matches := make([]string, 0, len(parts))
	records := make([]model.DiscoveredToolRecord, 0, len(parts))
	seen := make(map[string]struct{})

	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}

		if entry, ok := exactMatchCatalogEntry(state.CurrentLoaded, name); ok {
			if _, exists := seen[entry.CanonicalName]; !exists {
				matches = append(matches, entry.CanonicalName)
				seen[entry.CanonicalName] = struct{}{}
			}
			continue
		}

		entry, ok := exactMatchCatalogEntry(state.SearchablePool, name)
		if !ok {
			continue
		}
		if _, exists := seen[entry.CanonicalName]; exists {
			continue
		}
		matches = append(matches, entry.CanonicalName)
		seen[entry.CanonicalName] = struct{}{}
		if entry.ShouldDefer && !entry.AlwaysLoad {
			records = append(records, model.DiscoveredToolRecord{
				CanonicalName: entry.CanonicalName,
				Server:        entry.Server,
				Kind:          entry.Kind,
				DiscoveredAt:  now.UnixMilli(),
			})
		}
		if len(matches) >= limit {
			break
		}
	}

	output := ToolSearchOutput{
		Matches: matches,
		Query:   rawQuery,
	}
	if len(matches) == 0 && len(state.PendingMCPServer) > 0 {
		output.PendingMCPServers = cloneStrings(state.PendingMCPServer)
	}
	return output, records
}

func executeExactNameSearch(query, rawQuery string, state ToolSearchState, now time.Time) (ToolSearchOutput, []model.DiscoveredToolRecord, bool) {
	if entry, ok := exactMatchCatalogEntry(state.CurrentLoaded, query); ok {
		return ToolSearchOutput{
			Matches: []string{entry.CanonicalName},
			Query:   rawQuery,
		}, nil, true
	}

	entry, ok := exactMatchCatalogEntry(state.SearchablePool, query)
	if !ok {
		return ToolSearchOutput{}, nil, false
	}

	output := ToolSearchOutput{
		Matches: []string{entry.CanonicalName},
		Query:   rawQuery,
	}
	if entry.AlwaysLoad || !entry.ShouldDefer {
		return output, nil, true
	}
	return output, []model.DiscoveredToolRecord{{
		CanonicalName: entry.CanonicalName,
		Server:        entry.Server,
		Kind:          entry.Kind,
		DiscoveredAt:  now.UnixMilli(),
	}}, true
}

func executeKeywordSearch(query, rawQuery string, limit int, state ToolSearchState, now time.Time) (ToolSearchOutput, []model.DiscoveredToolRecord) {
	required, optional := parseSearchTerms(query)
	results := make([]rankedCatalogEntry, 0, len(state.SearchablePool))
	for _, entry := range state.SearchablePool {
		haystack := searchFields(entry)
		if !matchesRequiredTerms(required, haystack) {
			continue
		}
		score := rankCatalogEntry(entry, query, optional)
		if score <= 0 {
			continue
		}
		results = append(results, rankedCatalogEntry{
			entry: entry,
			score: score,
		})
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].score != results[j].score {
			return results[i].score > results[j].score
		}
		return strings.ToLower(results[i].entry.CanonicalName) < strings.ToLower(results[j].entry.CanonicalName)
	})

	matches := make([]string, 0, min(limit, len(results)))
	records := make([]model.DiscoveredToolRecord, 0, min(limit, len(results)))
	for _, result := range results {
		if len(matches) >= limit {
			break
		}
		matches = append(matches, result.entry.CanonicalName)
		if result.entry.ShouldDefer && !result.entry.AlwaysLoad {
			records = append(records, model.DiscoveredToolRecord{
				CanonicalName: result.entry.CanonicalName,
				Server:        result.entry.Server,
				Kind:          result.entry.Kind,
				DiscoveredAt:  now.UnixMilli(),
			})
		}
	}

	output := ToolSearchOutput{
		Matches: matches,
		Query:   rawQuery,
	}
	if len(matches) == 0 && len(state.PendingMCPServer) > 0 {
		output.PendingMCPServers = cloneStrings(state.PendingMCPServer)
	}
	return output, records
}

type rankedCatalogEntry struct {
	entry CatalogEntry
	score int
}

func exactMatchCatalogEntry(entries []CatalogEntry, name string) (CatalogEntry, bool) {
	query := exactMatchKey(name)
	for _, entry := range entries {
		if exactMatchKey(entry.CanonicalName) == query {
			return entry, true
		}
		for _, alias := range entry.Aliases {
			if exactMatchKey(alias) == query {
				return entry, true
			}
		}
	}
	return CatalogEntry{}, false
}

func parseSearchTerms(query string) (required []string, optional []string) {
	for _, term := range strings.Fields(strings.ToLower(query)) {
		if strings.HasPrefix(term, "+") {
			term = strings.TrimPrefix(term, "+")
			if term != "" {
				required = append(required, term)
			}
			continue
		}
		optional = append(optional, term)
	}
	return required, optional
}

func searchFields(entry CatalogEntry) []string {
	fields := []string{
		strings.ToLower(entry.CanonicalName),
		strings.ToLower(entry.Title),
		strings.ToLower(entry.Description),
		strings.ToLower(entry.SearchHint),
	}
	for _, alias := range entry.Aliases {
		fields = append(fields, strings.ToLower(alias))
	}
	return fields
}

func matchesRequiredTerms(required []string, fields []string) bool {
	for _, term := range required {
		matched := false
		for _, field := range fields {
			if strings.Contains(field, term) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func rankCatalogEntry(entry CatalogEntry, query string, optionalTerms []string) int {
	q := strings.ToLower(strings.TrimSpace(query))
	score := 0

	if q != "" {
		if strings.EqualFold(entry.CanonicalName, query) {
			score += 1000
		} else if strings.HasPrefix(strings.ToLower(entry.CanonicalName), q) {
			score += 850
		} else if strings.Contains(strings.ToLower(entry.CanonicalName), q) {
			score += 650
		}

		for _, alias := range entry.Aliases {
			aliasLower := strings.ToLower(alias)
			switch {
			case strings.EqualFold(alias, query):
				score += 980
			case strings.HasPrefix(aliasLower, q):
				score += 820
			case strings.Contains(aliasLower, q):
				score += 620
			}
		}

		switch title := strings.ToLower(entry.Title); {
		case title == q:
			score += 900
		case strings.HasPrefix(title, q):
			score += 700
		case strings.Contains(title, q):
			score += 560
		}
	}

	for _, term := range optionalTerms {
		if term == "" || strings.HasPrefix(term, "+") {
			continue
		}
		if strings.Contains(strings.ToLower(entry.CanonicalName), term) {
			score += 120
		}
		for _, alias := range entry.Aliases {
			if strings.Contains(strings.ToLower(alias), term) {
				score += 110
				break
			}
		}
		if strings.Contains(strings.ToLower(entry.Title), term) {
			score += 90
		}
		if strings.Contains(strings.ToLower(entry.SearchHint), term) {
			score += 80
		}
		if strings.Contains(strings.ToLower(entry.Description), term) {
			score += 60
		}
	}

	return score
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
