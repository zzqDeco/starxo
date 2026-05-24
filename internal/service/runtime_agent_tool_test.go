package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"starxo/internal/agent"
	"starxo/internal/config"
	"starxo/internal/model"
	"starxo/internal/tools"
)

func TestNormalizeRuntimeAgentInput(t *testing.T) {
	registry := agent.DefaultSubagentRegistry()
	input, err := normalizeRuntimeAgentInput(runtimeAgentInput{
		Prompt:       "  do work  ",
		SubagentType: "",
		Isolation:    "",
	}, registry)
	if err != nil {
		t.Fatalf("normalize runtime agent input: %v", err)
	}
	if input.Prompt != "do work" || input.SubagentType != "general" || input.Isolation != "none" || !input.Fork {
		t.Fatalf("unexpected normalized input: %#v", input)
	}
	if _, err := normalizeRuntimeAgentInput(runtimeAgentInput{Prompt: "x", SubagentType: "writer"}, registry); err == nil {
		t.Fatalf("expected invalid subagent_type to fail")
	}
	if _, err := normalizeRuntimeAgentInput(runtimeAgentInput{Prompt: "x", Isolation: "container"}, registry); err == nil {
		t.Fatalf("expected invalid isolation to fail")
	}
}

func TestNormalizeRuntimeAgentInputUsesConfiguredRegistry(t *testing.T) {
	registry := agent.NewSubagentRegistry([]agent.SubagentDefinition{{
		Name:             "triage",
		Description:      "Triage work",
		DefaultIsolation: "worktree",
	}})
	input, err := normalizeRuntimeAgentInput(runtimeAgentInput{
		Prompt:       "inspect",
		SubagentType: "triage",
	}, registry)
	if err != nil {
		t.Fatalf("normalize configured subagent: %v", err)
	}
	if input.SubagentType != "triage" || input.Isolation != "worktree" || input.Fork {
		t.Fatalf("unexpected normalized configured subagent: %#v", input)
	}

	defaulted, err := normalizeRuntimeAgentInput(runtimeAgentInput{
		Prompt: "inspect",
	}, registry)
	if err != nil {
		t.Fatalf("normalize default configured subagent: %v", err)
	}
	if defaulted.SubagentType != "triage" || defaulted.Isolation != "worktree" || !defaulted.Fork {
		t.Fatalf("expected omitted subagent_type to fork with registry default policy, got %#v", defaulted)
	}
}

func TestRuntimeSubagentRegistryFromConfigPreservesPolicy(t *testing.T) {
	allowBackground := false
	registry := newRuntimeSubagentRegistry([]config.SubagentDefinitionConfig{{
		Name:              "review_only",
		Description:       "Review only",
		DefaultIsolation:  "worktree",
		AllowedTools:      []string{"Read", "Grep"},
		BackgroundAllowed: &allowBackground,
	}})
	def, ok := registry.Get("review_only")
	if !ok {
		t.Fatalf("expected configured subagent")
	}
	if def.BackgroundAllowed {
		t.Fatalf("expected background execution to be disabled")
	}
	if def.DefaultIsolation != "worktree" {
		t.Fatalf("expected configured default isolation, got %q", def.DefaultIsolation)
	}
	if !runtimeSubagentAllowsTool(def, "Read") || runtimeSubagentAllowsTool(def, "Write") {
		t.Fatalf("expected configured allowed tools to be enforced: %#v", def.AllowedTools)
	}

	normalized, err := registry.Normalize("")
	if err != nil {
		t.Fatalf("normalize empty configured subagent: %v", err)
	}
	if normalized != "review_only" {
		t.Fatalf("expected empty subagent_type to use configured default, got %q", normalized)
	}
	defaultDef := registry.MustGet("")
	if defaultDef.Name != "review_only" || runtimeSubagentAllowsTool(defaultDef, "Write") {
		t.Fatalf("expected default configured subagent to preserve policy, got %#v", defaultDef)
	}
	if _, ok := registry.Get("general"); ok {
		t.Fatalf("custom registry without general must not inject an unrestricted general")
	}
}

func TestRuntimeAgentToolSchemaUsesConfiguredRegistry(t *testing.T) {
	registry := agent.NewSubagentRegistry([]agent.SubagentDefinition{{
		Name:             "triage",
		Description:      "Triage work",
		DefaultIsolation: "worktree",
		AllowedTools:     []string{"Read", "Grep"},
	}})
	entry, err := NewChatService(nil).newRuntimeAgentCatalogEntry(nil, nil, nil, nil, agent.DefaultAgentContext(), registry)
	if err != nil {
		t.Fatalf("new runtime agent entry: %v", err)
	}
	info, err := entry.Tool.Info(nil)
	if err != nil {
		t.Fatalf("agent tool info: %v", err)
	}
	schema, err := info.ParamsOneOf.ToJSONSchema()
	if err != nil {
		t.Fatalf("agent tool schema: %v", err)
	}
	raw, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("marshal agent tool schema: %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, "triage") || !strings.Contains(text, "Omit to fork the current agent context") {
		t.Fatalf("expected configured subagent metadata in schema, got %s", text)
	}
	if strings.Contains(text, "code_writer") || strings.Contains(text, "file_manager") {
		t.Fatalf("agent tool schema leaked builtin subagent names for custom registry: %s", text)
	}
}

func TestRuntimeSubagentToolsForkIgnoresDefaultAllowlist(t *testing.T) {
	catalog := tools.NewToolCatalog()
	for _, entry := range []tools.CatalogEntry{
		{
			CanonicalName: tools.RuntimeToolRead,
			AlwaysLoad:    true,
			Tool:          &stubTool{name: tools.RuntimeToolRead},
		},
		{
			CanonicalName: tools.RuntimeToolWrite,
			AlwaysLoad:    true,
			Tool:          &stubTool{name: tools.RuntimeToolWrite},
		},
	} {
		if err := catalog.Register(entry); err != nil {
			t.Fatalf("register catalog entry: %v", err)
		}
	}
	provider := &deferredMCPProvider{bundle: &RunnerBundle{MCPCatalog: catalog}}
	def := agent.SubagentDefinition{
		Name:         "narrow",
		AllowedTools: []string{tools.RuntimeToolRead},
	}

	normalTools := NewChatService(nil).runtimeSubagentTools(provider, def, false)
	if len(normalTools) != 1 {
		t.Fatalf("expected non-fork to honor allowlist, got %d tools", len(normalTools))
	}
	forkTools := NewChatService(nil).runtimeSubagentTools(provider, def, true)
	if len(forkTools) != 2 {
		t.Fatalf("expected fork to inherit all always-loaded context tools, got %d", len(forkTools))
	}
}

func TestRuntimeSubagentToolSearchCandidatesHonorAllowlist(t *testing.T) {
	catalog := tools.NewToolCatalog()
	for _, entry := range []tools.CatalogEntry{
		{
			CanonicalName:  tools.RuntimeToolLSP,
			ShouldDefer:    true,
			PermissionSpec: tools.PermissionSpec{AllowSearch: true, AllowExecute: true},
			Tool:           &stubTool{name: tools.RuntimeToolLSP},
		},
		{
			CanonicalName:  tools.RuntimeToolWebSearch,
			ShouldDefer:    true,
			PermissionSpec: tools.PermissionSpec{AllowSearch: true, AllowExecute: true},
			Tool:           &stubTool{name: tools.RuntimeToolWebSearch},
		},
	} {
		if err := catalog.Register(entry); err != nil {
			t.Fatalf("register catalog entry: %v", err)
		}
	}
	def := agent.SubagentDefinition{Name: "reviewer", AllowedTools: []string{tools.RuntimeToolLSP}}

	restricted := einoV09ToolSearchCandidatesFiltered(catalog, "default", runtimeSubagentCatalogEntryAllowed(def, false))
	if len(restricted) != 1 {
		t.Fatalf("expected restricted ToolSearch candidates to honor allowlist, got %d", len(restricted))
	}
	info, err := restricted[0].Info(context.Background())
	if err != nil {
		t.Fatalf("tool info: %v", err)
	}
	if info.Name != tools.RuntimeToolLSP {
		t.Fatalf("expected only LSP candidate, got %q", info.Name)
	}

	forked := einoV09ToolSearchCandidatesFiltered(catalog, "default", runtimeSubagentCatalogEntryAllowed(def, true))
	if len(forked) != 2 {
		t.Fatalf("expected forked ToolSearch to inherit all candidates, got %d", len(forked))
	}
}

func TestRuntimeSubagentModeInheritsParentPlanMode(t *testing.T) {
	chat := NewChatService(nil)
	sessionID := "sess-plan-subagent"
	chat.mu.Lock()
	run := chat.getOrCreateRun(sessionID)
	run.mode = model.ModePlan
	chat.mu.Unlock()

	provider := &deferredMCPProvider{chat: chat}
	ctx := contextWithSessionID(context.Background(), sessionID)
	if got := chat.runtimeSubagentMode(ctx, provider, ""); got != model.ModePlan {
		t.Fatalf("expected subagent to inherit parent plan mode, got %q", got)
	}
	if got := runtimeSubagentDeepAgentMode(model.ModePlan); got != agent.DeepAgentModePlan {
		t.Fatalf("expected plan prompt mode, got %q", got)
	}
	if got := chat.runtimeSubagentMode(ctx, provider, model.ModeDefault); got != model.ModeDefault {
		t.Fatalf("expected explicit default mode override, got %q", got)
	}
}

func TestRuntimeSubagentWorktreeUsesIsolatedAgentContext(t *testing.T) {
	ac := agent.DefaultAgentContext()
	ac.WorkspacePath = "/workspace"
	worktree := tools.WorktreeOutput{WorktreePath: "/workspace/.starxo/worktrees/agent-1"}
	subAC := runtimeSubagentAgentContext(ac, worktree)
	if subAC.WorkspacePath != worktree.WorktreePath || ac.WorkspacePath != "/workspace" {
		t.Fatalf("expected isolated subagent context without mutating parent, parent=%q sub=%q", ac.WorkspacePath, subAC.WorkspacePath)
	}
}

func TestFormatRuntimeAgentRunResultIncludesWorktreeMetadata(t *testing.T) {
	out := formatRuntimeAgentRunResult(runtimeAgentRunResult{
		text: "done\n",
		worktree: tools.WorktreeOutput{
			WorktreePath:   "/workspace/.starxo/worktrees/agent-1",
			WorktreeBranch: "starxo/agent-1",
		},
	})
	if !strings.Contains(out, "done") ||
		!strings.Contains(out, "worktreePath: /workspace/.starxo/worktrees/agent-1") ||
		!strings.Contains(out, "worktreeBranch: starxo/agent-1") {
		t.Fatalf("expected worktree metadata in result, got %q", out)
	}
}

func TestRuntimeSubagentInstructionMentionsIsolationWorkspace(t *testing.T) {
	def := agent.DefaultSubagentRegistry().MustGet("code_writer")
	instruction := agent.RuntimeSubagentPrompt(def, "/workspace/.starxo/worktrees/agent-1", "worktree")
	if !strings.Contains(instruction, "code_writer") ||
		!strings.Contains(instruction, "/workspace/.starxo/worktrees/agent-1") ||
		!strings.Contains(instruction, "worktree") ||
		!strings.Contains(instruction, "parent session workspace") {
		t.Fatalf("expected instruction to describe isolated workspace, got %q", instruction)
	}
}
