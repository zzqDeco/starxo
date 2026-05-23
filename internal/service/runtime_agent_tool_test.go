package service

import (
	"encoding/json"
	"strings"
	"testing"

	"starxo/internal/agent"
	"starxo/internal/config"
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
	if input.Prompt != "do work" || input.SubagentType != "general" || input.Isolation != "none" {
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
	if input.SubagentType != "triage" || input.Isolation != "worktree" {
		t.Fatalf("unexpected normalized configured subagent: %#v", input)
	}

	defaulted, err := normalizeRuntimeAgentInput(runtimeAgentInput{
		Prompt: "inspect",
	}, registry)
	if err != nil {
		t.Fatalf("normalize default configured subagent: %v", err)
	}
	if defaulted.SubagentType != "triage" || defaulted.Isolation != "worktree" {
		t.Fatalf("expected omitted subagent_type to use registry default, got %#v", defaulted)
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
	if !strings.Contains(text, "triage") || !strings.Contains(text, "registry default (triage)") {
		t.Fatalf("expected configured subagent metadata in schema, got %s", text)
	}
	if strings.Contains(text, "code_writer") || strings.Contains(text, "file_manager") {
		t.Fatalf("agent tool schema leaked builtin subagent names for custom registry: %s", text)
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
