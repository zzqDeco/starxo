package service

import (
	"strings"
	"testing"

	"starxo/internal/tools"
)

func TestNormalizeRuntimeAgentInput(t *testing.T) {
	input, err := normalizeRuntimeAgentInput(runtimeAgentInput{
		Prompt:       "  do work  ",
		SubagentType: "",
		Isolation:    "",
	})
	if err != nil {
		t.Fatalf("normalize runtime agent input: %v", err)
	}
	if input.Prompt != "do work" || input.SubagentType != "general" || input.Isolation != "none" {
		t.Fatalf("unexpected normalized input: %#v", input)
	}
	if _, err := normalizeRuntimeAgentInput(runtimeAgentInput{Prompt: "x", SubagentType: "writer"}); err == nil {
		t.Fatalf("expected invalid subagent_type to fail")
	}
	if _, err := normalizeRuntimeAgentInput(runtimeAgentInput{Prompt: "x", Isolation: "container"}); err == nil {
		t.Fatalf("expected invalid isolation to fail")
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
	instruction := runtimeSubagentInstruction("code_writer", "/workspace/.starxo/worktrees/agent-1", "worktree")
	if !strings.Contains(instruction, "code_writer") ||
		!strings.Contains(instruction, "/workspace/.starxo/worktrees/agent-1") ||
		!strings.Contains(instruction, "worktree") ||
		!strings.Contains(instruction, "parent session workspace") {
		t.Fatalf("expected instruction to describe isolated workspace, got %q", instruction)
	}
}
