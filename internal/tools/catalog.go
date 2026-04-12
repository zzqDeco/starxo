package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/cloudwego/eino/components/tool"
)

const (
	ToolSourceBuiltin = "builtin"
	ToolSourceMCP     = "mcp"
	ToolSourceCustom  = "custom"

	ToolKindAction           = "action"
	ToolKindResourceList     = "resource_list"
	ToolKindResourceRead     = "resource_read"
	ToolKindResourceTemplate = "resource_template"
	ToolKindMeta             = "meta"
)

type PermissionSpec struct {
	AllowSearch  bool
	AllowExecute bool
}

type CatalogEntry struct {
	CanonicalName string
	RemoteName    string
	Aliases       []string

	Source         string
	Server         string
	Kind           string
	Title          string
	Description    string
	SearchHint     string
	AlwaysLoad     bool
	ShouldDefer    bool
	IsMcp          bool
	IsResourceTool bool

	ReadOnlyHint    bool
	ReadOnlyTrusted bool

	PermissionSpec PermissionSpec
	Tool           tool.BaseTool
}

type ToolCatalog struct {
	mu         sync.RWMutex
	ordered    []string
	entries    map[string]CatalogEntry
	exactIndex map[string]string
}

func NewToolCatalog() *ToolCatalog {
	return &ToolCatalog{
		entries:    make(map[string]CatalogEntry),
		exactIndex: make(map[string]string),
	}
}

func NormalizeMCPNamePart(name string) (string, error) {
	s := strings.TrimSpace(name)
	if s == "" {
		return "", fmt.Errorf("name is empty")
	}

	var b strings.Builder
	b.Grow(len(s))
	lastUnderscore := false
	for _, r := range s {
		allowed := (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-'
		if !allowed {
			r = '_'
		}
		if r == '_' {
			if lastUnderscore {
				continue
			}
			lastUnderscore = true
		} else {
			lastUnderscore = false
		}
		b.WriteRune(r)
	}

	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "", fmt.Errorf("normalized name is empty")
	}
	return out, nil
}

func CanonicalMCPToolName(serverName, toolName string) (string, error) {
	serverPart, err := NormalizeMCPNamePart(serverName)
	if err != nil {
		return "", fmt.Errorf("normalize server name: %w", err)
	}
	toolPart, err := NormalizeMCPNamePart(toolName)
	if err != nil {
		return "", fmt.Errorf("normalize tool name: %w", err)
	}
	return "mcp__" + serverPart + "__" + toolPart, nil
}

func exactMatchKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func (c *ToolCatalog) Register(entry CatalogEntry) error {
	if entry.CanonicalName == "" {
		return fmt.Errorf("catalog entry canonical name is required")
	}
	if entry.Tool == nil {
		return fmt.Errorf("catalog entry tool is required")
	}

	entry.Aliases = cloneStrings(entry.Aliases)

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.entries[entry.CanonicalName]; exists {
		return fmt.Errorf("catalog canonical conflict: %s", entry.CanonicalName)
	}

	if err := c.registerExactKeyLocked(entry.CanonicalName, entry.CanonicalName); err != nil {
		return err
	}
	for _, alias := range entry.Aliases {
		if strings.TrimSpace(alias) == "" {
			continue
		}
		if err := c.registerExactKeyLocked(alias, entry.CanonicalName); err != nil {
			return err
		}
	}

	c.entries[entry.CanonicalName] = entry
	c.ordered = append(c.ordered, entry.CanonicalName)
	return nil
}

func (c *ToolCatalog) registerExactKeyLocked(name, canonical string) error {
	key := exactMatchKey(name)
	if key == "" {
		return fmt.Errorf("exact match key is empty")
	}
	if existing, ok := c.exactIndex[key]; ok && existing != canonical {
		return fmt.Errorf("catalog alias conflict: %s", name)
	}
	c.exactIndex[key] = canonical
	return nil
}

func (c *ToolCatalog) Get(canonical string) (CatalogEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[canonical]
	return entry, ok
}

func (c *ToolCatalog) LookupExact(name string) (CatalogEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	canonical, ok := c.exactIndex[exactMatchKey(name)]
	if !ok {
		return CatalogEntry{}, false
	}
	entry, ok := c.entries[canonical]
	return entry, ok
}

func (c *ToolCatalog) Entries() []CatalogEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]CatalogEntry, 0, len(c.ordered))
	for _, canonical := range c.ordered {
		out = append(out, c.entries[canonical])
	}
	return out
}

func (c *ToolCatalog) Tools() []tool.BaseTool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]tool.BaseTool, 0, len(c.ordered))
	for _, canonical := range c.ordered {
		out = append(out, c.entries[canonical].Tool)
	}
	return out
}

func (c *ToolCatalog) Filter(fn func(CatalogEntry) bool) []CatalogEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]CatalogEntry, 0, len(c.ordered))
	for _, canonical := range c.ordered {
		entry := c.entries[canonical]
		if fn == nil || fn(entry) {
			out = append(out, entry)
		}
	}
	return out
}

func (c *ToolCatalog) CanonicalNames() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := append([]string(nil), c.ordered...)
	return out
}

func (c *ToolCatalog) SortedEntries() []CatalogEntry {
	entries := c.Entries()
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CanonicalName < entries[j].CanonicalName
	})
	return entries
}

func InfoName(ctx context.Context, t tool.BaseTool) (string, error) {
	info, err := t.Info(ctx)
	if err != nil {
		return "", err
	}
	if info == nil || info.Name == "" {
		return "", fmt.Errorf("tool info missing name")
	}
	return info.Name, nil
}

func cloneStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}
