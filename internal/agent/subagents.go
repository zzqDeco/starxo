package agent

import (
	"fmt"
	"sort"
	"strings"
)

type SubagentDefinition struct {
	Name              string
	Description       string
	Instruction       string
	AllowedTools      []string
	DefaultIsolation  string
	BackgroundAllowed bool
}

type SubagentRegistry struct {
	defs        map[string]SubagentDefinition
	defaultName string
}

func DefaultSubagentRegistry() *SubagentRegistry {
	defs := []SubagentDefinition{
		defaultGeneralSubagentDefinition(),
		{
			Name:              "code_writer",
			Description:       "Writes, edits, and refactors code in the sandbox workspace.",
			AllowedTools:      []string{"Read", "Write", "Edit", "Glob", "Grep", "Bash", "TaskOutput"},
			DefaultIsolation:  "none",
			BackgroundAllowed: true,
		},
		{
			Name:              "code_executor",
			Description:       "Runs commands and scripts, inspects output, and reports failures.",
			AllowedTools:      []string{"Read", "Glob", "Grep", "Bash", "TaskOutput", "TaskStop"},
			DefaultIsolation:  "none",
			BackgroundAllowed: true,
		},
		{
			Name:              "file_manager",
			Description:       "Handles bulk file exploration and non-code file operations.",
			AllowedTools:      []string{"Read", "Write", "Edit", "Glob", "Grep"},
			DefaultIsolation:  "none",
			BackgroundAllowed: true,
		},
		{
			Name:              "reviewer",
			Description:       "Reviews code and diffs without applying edits by default.",
			AllowedTools:      []string{"Read", "Glob", "Grep", "Bash", "TaskOutput"},
			DefaultIsolation:  "none",
			BackgroundAllowed: true,
		},
	}
	return NewSubagentRegistry(defs)
}

func resolveSubagentRegistry(registries ...*SubagentRegistry) *SubagentRegistry {
	for _, registry := range registries {
		if registry != nil {
			return registry
		}
	}
	return DefaultSubagentRegistry()
}

func NewSubagentRegistry(defs []SubagentDefinition) *SubagentRegistry {
	r := &SubagentRegistry{defs: make(map[string]SubagentDefinition, len(defs))}
	for _, def := range defs {
		name := normalizeSubagentName(def.Name)
		if name == "" {
			continue
		}
		def.Name = name
		if def.Description == "" {
			def.Description = name
		}
		if def.DefaultIsolation == "" {
			def.DefaultIsolation = "none"
		}
		if len(def.AllowedTools) > 0 {
			def.AllowedTools = normalizeToolNames(def.AllowedTools)
		}
		r.defs[name] = def
		if r.defaultName == "" || name == "general" {
			r.defaultName = name
		}
	}
	if len(r.defs) == 0 {
		def := defaultGeneralSubagentDefinition()
		r.defs[def.Name] = def
		r.defaultName = def.Name
	}
	if r.defaultName == "" {
		r.defaultName = "general"
	}
	return r
}

func defaultGeneralSubagentDefinition() SubagentDefinition {
	return SubagentDefinition{
		Name:              "general",
		Description:       "General-purpose focused coding subagent for bounded tasks.",
		AllowedTools:      []string{"Read", "Write", "Edit", "Glob", "Grep", "Bash", "TaskOutput", "TaskStop"},
		DefaultIsolation:  "none",
		BackgroundAllowed: true,
	}
}

func (r *SubagentRegistry) Get(name string) (SubagentDefinition, bool) {
	if r == nil {
		r = DefaultSubagentRegistry()
	}
	def, ok := r.defs[normalizeSubagentName(name)]
	return def, ok
}

func (r *SubagentRegistry) MustGet(name string) SubagentDefinition {
	if r == nil {
		r = DefaultSubagentRegistry()
	}
	def, ok := r.Get(name)
	if ok {
		return def
	}
	def, _ = r.Get(r.defaultName)
	if def.Name == "" {
		def = defaultGeneralSubagentDefinition()
	}
	return def
}

func (r *SubagentRegistry) Names() []string {
	if r == nil {
		r = DefaultSubagentRegistry()
	}
	names := make([]string, 0, len(r.defs))
	for name := range r.defs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (r *SubagentRegistry) DefaultName() string {
	if r == nil {
		r = DefaultSubagentRegistry()
	}
	if r.defaultName != "" {
		return r.defaultName
	}
	return "general"
}

func (r *SubagentRegistry) NamesCSV() string {
	return strings.Join(r.Names(), ", ")
}

func (r *SubagentRegistry) PromptList() string {
	if r == nil {
		r = DefaultSubagentRegistry()
	}
	defaultName := r.DefaultName()
	lines := make([]string, 0, len(r.defs))
	for _, name := range r.Names() {
		def, _ := r.Get(name)
		label := name
		if name == defaultName {
			label += " (default when omitted)"
		}
		description := strings.TrimSpace(def.Description)
		if description == "" {
			description = name
		}
		background := "background allowed"
		if !def.BackgroundAllowed {
			background = "background disabled"
		}
		allowedTools := "all permitted runtime tools"
		if len(def.AllowedTools) > 0 {
			allowedTools = strings.Join(def.AllowedTools, ", ")
		}
		lines = append(lines, fmt.Sprintf("     - %s: %s; default isolation=%s; %s; allowed tools=%s",
			label, description, def.DefaultIsolation, background, allowedTools))
	}
	return strings.Join(lines, "\n")
}

func (r *SubagentRegistry) Normalize(name string) (string, error) {
	name = normalizeSubagentName(name)
	if name == "" {
		if r == nil {
			r = DefaultSubagentRegistry()
		}
		return r.DefaultName(), nil
	}
	if _, ok := r.Get(name); ok {
		return name, nil
	}
	return "", fmt.Errorf("unsupported subagent_type %q; use one of: %s", name, strings.Join(r.Names(), ", "))
}

func NormalizeSubagentType(name string) (string, error) {
	return DefaultSubagentRegistry().Normalize(name)
}

func RuntimeSubagentPrompt(def SubagentDefinition, workspacePath, isolation string) string {
	isolationNote := "none"
	if isolation == "worktree" {
		isolationNote = "worktree (changes are isolated from the parent session workspace unless explicitly merged later)"
	}
	instruction := strings.TrimSpace(def.Instruction)
	if instruction == "" {
		instruction = defaultRuntimeSubagentInstruction(def.Name)
	}
	return fmt.Sprintf(`You are a focused Starxo runtime subagent.

Type: %s
Workspace: %s
Isolation: %s

%s

Use only the delegated scope. Prefer Read/Grep/Glob for inspection, Edit/Write for file changes, and Bash only when command execution is necessary. Keep output concise and include changed files, verification performed, and blockers.`, def.Name, workspacePath, isolationNote, instruction)
}

func defaultRuntimeSubagentInstruction(name string) string {
	switch normalizeSubagentName(name) {
	case "code_writer":
		return "Write, edit, and refactor code. Inspect current files before editing and report the concrete files changed."
	case "code_executor":
		return "Run commands and scripts, inspect outputs, explain failures, and avoid unrelated file edits."
	case "file_manager":
		return "Explore and manage files, especially non-code content and bulk file operations."
	case "reviewer":
		return "Review code, diffs, and risks. Do not edit files unless the parent explicitly requests it."
	default:
		return "Complete the delegated task independently, using the workspace tools available to you."
	}
}

func normalizeSubagentName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func normalizeToolNames(names []string) []string {
	out := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}
