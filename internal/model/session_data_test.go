package model

import (
	"encoding/json"
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
