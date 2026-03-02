package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionZeroValue(t *testing.T) {
	var s Session
	assert.Empty(t, s.ID)
	assert.Empty(t, s.Title)
	assert.Empty(t, s.Containers)
	assert.Empty(t, s.ActiveContainerID)
	assert.Empty(t, s.WorkspacePath)
	assert.Zero(t, s.CreatedAt)
	assert.Zero(t, s.UpdatedAt)
	assert.Zero(t, s.MessageCount)
}

func TestSessionJSONRoundTrip(t *testing.T) {
	original := Session{
		ID:                "sess-abc123",
		Title:             "Test Session",
		Containers:        []string{"ctr-xyz", "ctr-abc"},
		ActiveContainerID: "ctr-xyz",
		WorkspacePath:     "/home/user/project",
		CreatedAt:         1700000000,
		UpdatedAt:         1700003600,
		MessageCount:      42,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var restored Session
	require.NoError(t, json.Unmarshal(data, &restored))
	assert.Equal(t, original, restored)
}

func TestSessionJSONOmitsEmptyWorkspace(t *testing.T) {
	s := Session{ID: "s1", Title: "No workspace"}
	data, err := json.Marshal(s)
	require.NoError(t, err)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.NotContains(t, raw, "workspacePath")
}

func TestSessionJSONIncludesWorkspaceWhenSet(t *testing.T) {
	s := Session{ID: "s1", WorkspacePath: "/ws"}
	data, err := json.Marshal(s)
	require.NoError(t, err)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Contains(t, raw, "workspacePath")
}
