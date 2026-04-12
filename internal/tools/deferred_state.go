package tools

import (
	"sort"

	"starxo/internal/model"
)

type DeferredMCPState struct {
	SearchablePoolForMode []CatalogEntry
	LoadablePoolForMode   []CatalogEntry
	EffectiveDiscovered   []CatalogEntry
	CurrentLoadedTools    []CatalogEntry
	PendingMCPServers     []string
}

func ComputeDeferredMCPState(
	catalog *ToolCatalog,
	discovered map[string]model.DiscoveredToolRecord,
	permCtx ToolPermissionContext,
) DeferredMCPState {
	if catalog == nil {
		return DeferredMCPState{}
	}

	allEntries := catalog.Entries()
	searchable := make([]CatalogEntry, 0, len(allEntries))
	loadable := make([]CatalogEntry, 0, len(allEntries))
	pendingSet := make(map[string]struct{})
	for serverName, server := range permCtx.Servers {
		if server.State == MCPServerStatePending {
			pendingSet[serverName] = struct{}{}
		}
	}

	for _, entry := range allEntries {
		searchDecision := CanSearchCatalogEntry(entry, permCtx)
		if searchDecision.Allowed {
			searchable = append(searchable, entry)
		}

		loadDecision := CanLoadCatalogEntry(entry, permCtx)
		if loadDecision.Allowed {
			loadable = append(loadable, entry)
		}
	}

	loadableByName := make(map[string]CatalogEntry, len(loadable))
	for _, entry := range loadable {
		loadableByName[entry.CanonicalName] = entry
	}

	effective := make([]CatalogEntry, 0, len(discovered))
	for canonical := range discovered {
		if entry, ok := loadableByName[canonical]; ok {
			effective = append(effective, entry)
		}
	}
	sortEntriesByCanonical(effective)

	currentLoaded := make([]CatalogEntry, 0, len(effective))
	currentLoaded = append(currentLoaded, effective...)

	seenLoaded := make(map[string]struct{}, len(currentLoaded))
	for _, entry := range currentLoaded {
		seenLoaded[entry.CanonicalName] = struct{}{}
	}
	for _, entry := range allEntries {
		if !entry.AlwaysLoad {
			continue
		}
		if _, exists := seenLoaded[entry.CanonicalName]; exists {
			continue
		}
		currentLoaded = append(currentLoaded, entry)
		seenLoaded[entry.CanonicalName] = struct{}{}
	}
	sortEntriesByCanonical(currentLoaded)

	pending := make([]string, 0, len(pendingSet))
	for server := range pendingSet {
		pending = append(pending, server)
	}
	sort.Strings(pending)

	sortEntriesByCanonical(searchable)
	sortEntriesByCanonical(loadable)

	return DeferredMCPState{
		SearchablePoolForMode: searchable,
		LoadablePoolForMode:   loadable,
		EffectiveDiscovered:   effective,
		CurrentLoadedTools:    currentLoaded,
		PendingMCPServers:     pending,
	}
}

func sortEntriesByCanonical(entries []CatalogEntry) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CanonicalName < entries[j].CanonicalName
	})
}
