package service

import (
	"fmt"
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

func TestBeforeSandboxActivationHookReceivesTargetAndCanBlock(t *testing.T) {
	svc := NewSandboxService(nil, nil)
	var gotTarget string
	svc.SetBeforeSandboxActivation(func(containerRegID string) error {
		gotTarget = containerRegID
		return fmt.Errorf("blocked")
	})

	err := svc.runBeforeSandboxActivation("ctr-target")

	require.Error(t, err)
	assert.Equal(t, "ctr-target", gotTarget)
	assert.Contains(t, err.Error(), "blocked")
}
