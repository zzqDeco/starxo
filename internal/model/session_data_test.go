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

	normalized.PlanDocument.Markdown = "changed"
	normalized.PendingPlanApproval.RequestedAt = 99
	normalized.PendingPlanAttachment.Markdown = "changed too"

	if data.PlanDocument.Markdown != "draft" {
		t.Fatalf("expected original plan document to stay unchanged, got %#v", data.PlanDocument)
	}
	if data.PendingPlanApproval.RequestedAt != 20 {
		t.Fatalf("expected original pending approval to stay unchanged, got %#v", data.PendingPlanApproval)
	}
	if data.PendingPlanAttachment.Markdown != "approved plan" {
		t.Fatalf("expected original pending attachment to stay unchanged, got %#v", data.PendingPlanAttachment)
	}
}
