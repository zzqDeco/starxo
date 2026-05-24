package service

import (
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"

	"starxo/internal/model"
	"starxo/internal/tools"
)

func TestStandaloneObjectiveDoesNotCarryOldTaskHistory(t *testing.T) {
	chat := NewChatService(nil)
	sessionID := "sess-objective-standalone"
	chat.mu.Lock()
	run := chat.getOrCreateRun(sessionID)
	chat.mu.Unlock()

	run.addUserMessage("debug the failing tests in package alpha")
	run.addAssistantMessage("I will inspect the failures.")
	run.addUserMessage("create hello.txt with hello and read it back")
	obj := run.beginObjective("usr-new", "run-new", "create hello.txt with hello and read it back", time.Now().UnixMilli())

	messages := chat.prepareMessagesForRun(sessionID, run)
	joined := joinMessageContent(messages)
	if obj.Scope != "standalone" {
		t.Fatalf("expected standalone objective, got %#v", obj)
	}
	if strings.Contains(joined, "debug the failing tests") || strings.Contains(joined, "inspect the failures") {
		t.Fatalf("standalone objective carried old task history:\n%s", joined)
	}
	if !strings.Contains(joined, "<current-objective>") || !strings.Contains(joined, "create hello.txt") {
		t.Fatalf("expected pinned current objective and current user message, got:\n%s", joined)
	}
}

func TestContinuationObjectiveKeepsRecentTaskHistory(t *testing.T) {
	chat := NewChatService(nil)
	sessionID := "sess-objective-continuation"
	chat.mu.Lock()
	run := chat.getOrCreateRun(sessionID)
	chat.mu.Unlock()

	run.addUserMessage("debug the failing tests in package alpha")
	run.addAssistantMessage("I found a likely failing assertion.")
	run.addUserMessage("继续")
	obj := run.beginObjective("usr-continue", "run-continue", "继续", time.Now().UnixMilli())

	messages := chat.prepareMessagesForRun(sessionID, run)
	joined := joinMessageContent(messages)
	if obj.Scope != "continuation" {
		t.Fatalf("expected continuation objective, got %#v", obj)
	}
	if !strings.Contains(joined, "debug the failing tests") || !strings.Contains(joined, "I found a likely failing assertion") {
		t.Fatalf("continuation objective lost recent task history:\n%s", joined)
	}
}

func TestObjectiveScopeAvoidsGenericAbovePreviousFalsePositives(t *testing.T) {
	if got := objectiveScope("summarize the above section in a new file"); got != "standalone" {
		t.Fatalf("expected generic above request to remain standalone, got %q", got)
	}
	if got := objectiveScope("document the above XML marker"); got != "standalone" {
		t.Fatalf("expected document above request to remain standalone, got %q", got)
	}
	if got := objectiveScope("compare this with the previous implementation"); got != "standalone" {
		t.Fatalf("expected generic previous request to remain standalone, got %q", got)
	}
	if got := objectiveScope("compare this with the previous implementation work"); got != "standalone" {
		t.Fatalf("expected generic previous work request to remain standalone, got %q", got)
	}
	if got := objectiveScope("do the above"); got != "continuation" {
		t.Fatalf("expected explicit do-the-above request to continue, got %q", got)
	}
	if got := objectiveScope("continue previous task"); got != "continuation" {
		t.Fatalf("expected explicit previous task continuation, got %q", got)
	}
	if got := objectiveScope("continue."); got != "continuation" {
		t.Fatalf("expected punctuated continue request to continue, got %q", got)
	}
	if got := objectiveScope("go on?"); got != "continuation" {
		t.Fatalf("expected punctuated go-on request to continue, got %q", got)
	}
	if got := objectiveScope("resume"); got != "continuation" {
		t.Fatalf("expected bare resume request to continue, got %q", got)
	}
	if got := objectiveScope("resume previous task"); got != "continuation" {
		t.Fatalf("expected explicit resume-previous request to continue, got %q", got)
	}
	if got := objectiveScope("resume parser design"); got != "standalone" {
		t.Fatalf("expected resume parser design to remain standalone, got %q", got)
	}
	if got := objectiveScope("resume the container service"); got != "standalone" {
		t.Fatalf("expected resume container service to remain standalone, got %q", got)
	}
	if got := objectiveScope("resume last mile implementation"); got != "standalone" {
		t.Fatalf("expected resume last-mile implementation to remain standalone, got %q", got)
	}
}

func TestScopeCompactForStandaloneObjectiveDropsOldActiveState(t *testing.T) {
	compact := &model.RuntimeContextCompact{
		Version: model.RuntimeContextCompactVersion,
		Summary: "old debug summary",
		ToolSearch: model.RuntimeToolSearchCompact{
			DiscoveredTools: []model.DiscoveredToolRecord{{CanonicalName: tools.RuntimeToolWebSearch}},
		},
		PermissionGrants: []model.RuntimePermissionGrant{{ToolName: tools.RuntimeToolBash}},
		Tasks:            []model.RuntimeTaskCompact{{ID: "task-old", Status: "running"}},
		Todos:            []model.RuntimeTodoItem{{ID: "todo-old", Title: "debug old tests", Status: "in_progress"}},
		PlanDocument:     &model.PlanDocument{Markdown: "old plan"},
	}
	objective := &model.RunObjective{
		ID:        "obj-new",
		RunID:     "run-new",
		Scope:     "standalone",
		Objective: "create hello.txt",
	}

	scoped := scopeCompactForObjective(compact, objective)
	if scoped == nil || scoped.ActiveObjective == nil || scoped.ActiveObjective.ID != "obj-new" {
		t.Fatalf("expected active objective in scoped compact, got %#v", scoped)
	}
	if scoped.Summary != "" || len(scoped.Tasks) != 0 || len(scoped.Todos) != 0 || scoped.PlanDocument != nil {
		t.Fatalf("expected old active work to be stripped, got %#v", scoped)
	}
	if len(scoped.ToolSearch.DiscoveredTools) != 1 || len(scoped.PermissionGrants) != 1 {
		t.Fatalf("expected tool discovery and permission grants to be preserved, got %#v", scoped)
	}
}

func joinMessageContent(messages []*schema.Message) string {
	parts := make([]string, 0, len(messages))
	for _, msg := range messages {
		if msg != nil {
			parts = append(parts, msg.Content)
		}
	}
	return strings.Join(parts, "\n")
}
