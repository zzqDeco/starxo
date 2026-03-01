package tools

import (
	"fmt"
	"sync"

	"github.com/cloudwego/eino/components/tool"
)

// ToolRegistry is a central, thread-safe registry for managing tools from
// multiple sources: built-in tools, MCP server tools, and custom tools.
type ToolRegistry struct {
	mu       sync.RWMutex
	builtins map[string]tool.BaseTool
	mcpTools map[string][]tool.BaseTool
	custom   map[string]tool.BaseTool
}

// NewToolRegistry creates a new empty ToolRegistry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		builtins: make(map[string]tool.BaseTool),
		mcpTools: make(map[string][]tool.BaseTool),
		custom:   make(map[string]tool.BaseTool),
	}
}

// RegisterBuiltin registers a built-in tool. The tool name is extracted from
// the tool's Info method. Returns an error if the tool info cannot be retrieved.
func (r *ToolRegistry) RegisterBuiltin(t tool.BaseTool) error {
	info, err := t.Info(nil)
	if err != nil {
		return fmt.Errorf("failed to get tool info: %w", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.builtins[info.Name] = t
	return nil
}

// RegisterMCPTools registers a set of tools from an MCP server under the given
// server name. Calling this again with the same name replaces the previous set.
func (r *ToolRegistry) RegisterMCPTools(name string, tools []tool.BaseTool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mcpTools[name] = tools
}

// RegisterCustom registers a custom tool under the given name.
func (r *ToolRegistry) RegisterCustom(name string, t tool.BaseTool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.custom[name] = t
}

// GetAll returns all registered tools from all sources (builtins, MCP, custom).
func (r *ToolRegistry) GetAll() []tool.BaseTool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var all []tool.BaseTool
	for _, t := range r.builtins {
		all = append(all, t)
	}
	for _, tools := range r.mcpTools {
		all = append(all, tools...)
	}
	for _, t := range r.custom {
		all = append(all, t)
	}
	return all
}

// GetByNames returns the subset of registered tools whose names match the
// provided list. Tools are looked up across all sources. Unmatched names are
// silently ignored.
func (r *ToolRegistry) GetByNames(names ...string) []tool.BaseTool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	nameSet := make(map[string]struct{}, len(names))
	for _, n := range names {
		nameSet[n] = struct{}{}
	}

	var result []tool.BaseTool

	for name, t := range r.builtins {
		if _, ok := nameSet[name]; ok {
			result = append(result, t)
		}
	}
	for _, tools := range r.mcpTools {
		for _, t := range tools {
			info, err := t.Info(nil)
			if err != nil {
				continue
			}
			if _, ok := nameSet[info.Name]; ok {
				result = append(result, t)
			}
		}
	}
	for name, t := range r.custom {
		if _, ok := nameSet[name]; ok {
			result = append(result, t)
		}
	}
	return result
}

// Remove removes a tool by name from all sources.
func (r *ToolRegistry) Remove(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.builtins, name)
	delete(r.custom, name)

	for serverName, tools := range r.mcpTools {
		filtered := tools[:0]
		for _, t := range tools {
			info, err := t.Info(nil)
			if err != nil {
				filtered = append(filtered, t)
				continue
			}
			if info.Name != name {
				filtered = append(filtered, t)
			}
		}
		if len(filtered) == 0 {
			delete(r.mcpTools, serverName)
		} else {
			r.mcpTools[serverName] = filtered
		}
	}
}
