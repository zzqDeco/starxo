package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunTerminalCommandRequiresNonEmptyCommand(t *testing.T) {
	svc := NewSandboxService(nil, nil)

	_, err := svc.RunTerminalCommand("  ")

	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "empty")
}

func TestRunTerminalCommandRequiresSSHConnection(t *testing.T) {
	svc := NewSandboxService(nil, nil)

	_, err := svc.RunTerminalCommand("pwd")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "SSH not connected")
}
