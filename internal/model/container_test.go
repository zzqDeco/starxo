package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainerStatusConstants(t *testing.T) {
	assert.Equal(t, ContainerStatus("running"), ContainerRunning)
	assert.Equal(t, ContainerStatus("stopped"), ContainerStopped)
	assert.Equal(t, ContainerStatus("unknown"), ContainerUnknown)
	assert.Equal(t, ContainerStatus("destroyed"), ContainerDestroyed)
}

func TestContainerZeroValue(t *testing.T) {
	var c Container
	assert.Empty(t, c.ID)
	assert.Empty(t, c.DockerID)
	assert.Equal(t, ContainerStatus(""), c.Status)
	assert.False(t, c.SetupComplete)
	assert.Zero(t, c.SSHPort)
}

func TestContainerJSONRoundTrip(t *testing.T) {
	original := Container{
		ID:            "ctr-1",
		DockerID:      "docker-abc",
		Name:          "test-container",
		Image:         "python:3.11-slim",
		SSHHost:       "localhost",
		SSHPort:       2222,
		Status:        ContainerRunning,
		SetupComplete: true,
		CreatedAt:     1700000000,
		LastUsedAt:    1700003600,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var restored Container
	require.NoError(t, json.Unmarshal(data, &restored))
	assert.Equal(t, original, restored)
}

func TestContainerStatusTransitions(t *testing.T) {
	tests := []struct {
		name string
		from ContainerStatus
		to   ContainerStatus
	}{
		{"unknown to running", ContainerUnknown, ContainerRunning},
		{"running to stopped", ContainerRunning, ContainerStopped},
		{"stopped to running", ContainerStopped, ContainerRunning},
		{"running to destroyed", ContainerRunning, ContainerDestroyed},
		{"stopped to destroyed", ContainerStopped, ContainerDestroyed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Container{Status: tt.from}
			c.Status = tt.to
			assert.Equal(t, tt.to, c.Status)
		})
	}
}

func TestContainerStatusJSONValues(t *testing.T) {
	tests := []struct {
		status   ContainerStatus
		expected string
	}{
		{ContainerRunning, `"running"`},
		{ContainerStopped, `"stopped"`},
		{ContainerUnknown, `"unknown"`},
		{ContainerDestroyed, `"destroyed"`},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			data, err := json.Marshal(tt.status)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(data))

			var restored ContainerStatus
			require.NoError(t, json.Unmarshal(data, &restored))
			assert.Equal(t, tt.status, restored)
		})
	}
}
