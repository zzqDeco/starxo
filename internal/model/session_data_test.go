package model

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestSessionData_BackwardCompatibleWithoutDiscoveredTools(t *testing.T) {
	raw := []byte(`{"version":2,"messages":[],"display":[]}`)

	var data SessionData
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("unmarshal session data: %v", err)
	}
	if len(data.DiscoveredTools) != 0 {
		t.Fatalf("expected empty discovered tools for legacy payload, got %#v", data.DiscoveredTools)
	}
}

func TestNormalizeSessionDataAppliesV4Defaults(t *testing.T) {
	raw := []byte(`{"version":3,"messages":[],"display":[]}`)

	var data SessionData
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("unmarshal session data: %v", err)
	}

	normalized, warnings := NormalizeSessionData(&data)
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings for missing v4 fields, got %#v", warnings)
	}
	if normalized == nil {
		t.Fatal("expected normalized session data")
	}
	if normalized.Version != SessionDataVersion {
		t.Fatalf("expected version %d, got %d", SessionDataVersion, normalized.Version)
	}
	if normalized.Mode != ModeDefault {
		t.Fatalf("expected default mode, got %q", normalized.Mode)
	}
	if normalized.PlanDocument != nil || normalized.PendingPlanApproval != nil || normalized.PendingPlanAttachment != nil {
		t.Fatalf("expected nil plan state defaults, got %#v", normalized)
	}
}

func TestNormalizeSessionDataDowngradesInvalidValues(t *testing.T) {
	data := &SessionData{
		Version: 2,
		Mode:    "weird",
		PendingPlanAttachment: &PendingPlanAttachment{
			Kind:     "oops",
			Markdown: "plan",
		},
	}

	normalized, warnings := NormalizeSessionData(data)
	if normalized.Mode != ModeDefault {
		t.Fatalf("expected invalid mode to downgrade to %q, got %q", ModeDefault, normalized.Mode)
	}
	if normalized.PendingPlanAttachment != nil {
		t.Fatalf("expected invalid attachment kind to be dropped, got %#v", normalized.PendingPlanAttachment)
	}
	wantWarnings := []string{
		SessionDataWarningInvalidMode,
		SessionDataWarningInvalidPendingAttachmentKind,
	}
	if !reflect.DeepEqual(warnings, wantWarnings) {
		t.Fatalf("unexpected warnings: got %#v want %#v", warnings, wantWarnings)
	}
}

func TestNormalizeSessionDataReturnsCopy(t *testing.T) {
	data := &SessionData{
		Version: SessionDataVersion,
		Mode:    ModePlan,
		PermissionGrants: []RuntimePermissionGrant{{
			ToolName: "Bash",
			Decision: "allow_session",
		}},
		PermissionAudit: []RuntimePermissionAudit{{
			ToolName: "Bash",
			Decision: "allow_once",
		}},
		PlanDocument: &PlanDocument{
			Markdown:  "draft",
			UpdatedAt: 10,
		},
		PendingPlanApproval: &PendingPlanApproval{RequestedAt: 20},
		PendingPlanAttachment: &PendingPlanAttachment{
			Kind:      PendingPlanAttachmentKindApproved,
			Markdown:  "approved plan",
			Feedback:  "ok",
			CreatedAt: 30,
		},
	}

	normalized, warnings := NormalizeSessionData(data)
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	if normalized.PlanDocument == data.PlanDocument {
		t.Fatal("expected plan document clone")
	}
	if normalized.PendingPlanApproval == data.PendingPlanApproval {
		t.Fatal("expected pending approval clone")
	}
	if normalized.PendingPlanAttachment == data.PendingPlanAttachment {
		t.Fatal("expected pending attachment clone")
	}
	if len(normalized.PermissionGrants) != 1 || normalized.PermissionGrants[0].ToolName != "Bash" {
		t.Fatalf("expected permission grant clone, got %#v", normalized.PermissionGrants)
	}
	if len(normalized.PermissionAudit) != 1 || normalized.PermissionAudit[0].ToolName != "Bash" {
		t.Fatalf("expected permission audit clone, got %#v", normalized.PermissionAudit)
	}

	normalized.PlanDocument.Markdown = "changed"
	normalized.PendingPlanApproval.RequestedAt = 99
	normalized.PendingPlanAttachment.Markdown = "changed too"
	normalized.PermissionGrants[0].ToolName = "Write"
	normalized.PermissionAudit[0].ToolName = "Write"

	if data.PlanDocument.Markdown != "draft" {
		t.Fatalf("expected original plan document to stay unchanged, got %#v", data.PlanDocument)
	}
	if data.PendingPlanApproval.RequestedAt != 20 {
		t.Fatalf("expected original pending approval to stay unchanged, got %#v", data.PendingPlanApproval)
	}
	if data.PendingPlanAttachment.Markdown != "approved plan" {
		t.Fatalf("expected original pending attachment to stay unchanged, got %#v", data.PendingPlanAttachment)
	}
	if data.PermissionGrants[0].ToolName != "Bash" {
		t.Fatalf("expected original permission grant to stay unchanged, got %#v", data.PermissionGrants)
	}
	if data.PermissionAudit[0].ToolName != "Bash" {
		t.Fatalf("expected original permission audit to stay unchanged, got %#v", data.PermissionAudit)
	}
}

func TestNormalizeSessionDataClonesRuntimeContextCompact(t *testing.T) {
	data := &SessionData{
		Version: SessionDataVersion,
		Mode:    ModeDefault,
		RuntimeContextCompact: &RuntimeContextCompact{
			Version:              RuntimeContextCompactVersion,
			OriginalMessageCount: 30,
			OmittedMessageCount:  18,
			ToolSearch: RuntimeToolSearchCompact{
				DiscoveredTools: []DiscoveredToolRecord{{
					CanonicalName: "WebSearch",
					Kind:          "action",
				}},
				DeferredAnnouncementState: &DeferredAnnouncementState{
					AnnouncedSearchableCanonicalNames: []string{"LSP"},
				},
				MCPInstructionsDeltaState: &MCPInstructionsDeltaState{
					LastAnnouncedSearchableServers: []string{"alpha"},
				},
			},
			PermissionGrants: []RuntimePermissionGrant{{
				ToolName: "Bash",
				Decision: "allow_session",
			}},
			PermissionAudit: []RuntimePermissionAudit{{
				ToolName: "Bash",
				Decision: "allow_once",
			}},
			Tasks: []RuntimeTaskCompact{{
				ID:     "task-1",
				Status: "running",
			}},
			TaskItems: []RuntimeTaskItemCompact{{
				ID:        "taskitem-1",
				Title:     "Ship task graph",
				Status:    "todo",
				DependsOn: []string{"taskitem-0"},
			}},
			FileReadState: []RuntimeFileReadState{{
				FilePath: "/workspace/main.go",
			}},
			DiffSummaries: []RuntimeDiffSummary{{
				FilePath: "/workspace/main.go",
				Summary:  "edited",
			}},
			Todos: []RuntimeTodoItem{{
				ID:        "todo-1",
				Title:     "ship",
				Status:    "pending",
				DependsOn: []string{"todo-0"},
			}},
			PlanDocument: &PlanDocument{Markdown: "plan"},
			Workspace:    &RuntimeWorkspaceCompact{Active: true, WorktreePath: "/workspace/.starxo/worktrees/a"},
			ActiveObjective: &RunObjective{
				ID:        "obj-1",
				RunID:     "run-1",
				Scope:     "standalone",
				Objective: "current task",
			},
		},
	}

	normalized, warnings := NormalizeSessionData(data)
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	if normalized.RuntimeContextCompact == nil {
		t.Fatal("expected runtime compact state")
	}

	normalized.RuntimeContextCompact.ToolSearch.DiscoveredTools[0].CanonicalName = "mutated"
	normalized.RuntimeContextCompact.ToolSearch.DeferredAnnouncementState.AnnouncedSearchableCanonicalNames[0] = "mutated"
	normalized.RuntimeContextCompact.ToolSearch.MCPInstructionsDeltaState.LastAnnouncedSearchableServers[0] = "mutated"
	normalized.RuntimeContextCompact.PermissionGrants[0].ToolName = "Write"
	normalized.RuntimeContextCompact.PermissionAudit[0].ToolName = "Write"
	normalized.RuntimeContextCompact.Tasks[0].Status = "failed"
	normalized.RuntimeContextCompact.TaskItems[0].DependsOn[0] = "changed"
	normalized.RuntimeContextCompact.FileReadState[0].FilePath = "changed"
	normalized.RuntimeContextCompact.DiffSummaries[0].Summary = "changed"
	normalized.RuntimeContextCompact.Todos[0].DependsOn[0] = "changed"
	normalized.RuntimeContextCompact.PlanDocument.Markdown = "changed"
	normalized.RuntimeContextCompact.Workspace.WorktreePath = "changed"
	normalized.RuntimeContextCompact.ActiveObjective.Objective = "changed"

	orig := data.RuntimeContextCompact
	if orig.ToolSearch.DiscoveredTools[0].CanonicalName != "WebSearch" ||
		orig.ToolSearch.DeferredAnnouncementState.AnnouncedSearchableCanonicalNames[0] != "LSP" ||
		orig.ToolSearch.MCPInstructionsDeltaState.LastAnnouncedSearchableServers[0] != "alpha" ||
		orig.PermissionGrants[0].ToolName != "Bash" ||
		orig.PermissionAudit[0].ToolName != "Bash" ||
		orig.Tasks[0].Status != "running" ||
		orig.TaskItems[0].DependsOn[0] != "taskitem-0" ||
		orig.FileReadState[0].FilePath != "/workspace/main.go" ||
		orig.DiffSummaries[0].Summary != "edited" ||
		orig.Todos[0].DependsOn[0] != "todo-0" ||
		orig.PlanDocument.Markdown != "plan" ||
		orig.Workspace.WorktreePath != "/workspace/.starxo/worktrees/a" ||
		orig.ActiveObjective.Objective != "current task" {
		t.Fatalf("expected original compact state to stay unchanged, got %#v", orig)
	}
}
