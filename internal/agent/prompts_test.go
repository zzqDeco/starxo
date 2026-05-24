package agent

import (
	"strings"
	"testing"
)

func TestDeepAgentPromptsUseConfiguredSubagentRegistry(t *testing.T) {
	registry := NewSubagentRegistry([]SubagentDefinition{{
		Name:              "triage",
		Description:       "Triage and route work",
		AllowedTools:      []string{"Read", "Grep"},
		DefaultIsolation:  "worktree",
		BackgroundAllowed: false,
	}})
	ac := DefaultAgentContext()

	for name, prompt := range map[string]string{
		"default": DeepAgentPrompt(ac, registry),
		"plan":    DeepAgentPlanPrompt(ac, registry),
	} {
		if !strings.Contains(prompt, "triage (default when omitted)") ||
			!strings.Contains(prompt, "Triage and route work") ||
			!strings.Contains(prompt, "allowed tools=Read, Grep") {
			t.Fatalf("%s prompt did not include configured subagent metadata:\n%s", name, prompt)
		}
		if strings.Contains(prompt, "code_writer, code_executor") ||
			strings.Contains(prompt, "- code_writer:") {
			t.Fatalf("%s prompt leaked hard-coded builtin subagent list:\n%s", name, prompt)
		}
	}
}

func TestRuntimeForkAgentPromptDoesNotAdvertiseAgentDelegation(t *testing.T) {
	prompt := RuntimeForkAgentPrompt(DefaultAgentContext(), DeepAgentModePlan)
	if !strings.Contains(prompt, "Agent delegation is not available inside this fork") {
		t.Fatalf("expected fork prompt to explain Agent unavailability:\n%s", prompt)
	}
	if strings.Contains(prompt, "SUBAGENTS:") || strings.Contains(prompt, "If Agent subagent_type") {
		t.Fatalf("fork prompt should not advertise recursive Agent delegation:\n%s", prompt)
	}
	if !strings.Contains(prompt, "Mode: plan") || !strings.Contains(prompt, "write_todos/update_todo") {
		t.Fatalf("fork prompt lost mode or direct tool instructions:\n%s", prompt)
	}
}
