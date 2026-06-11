package service

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"starxo/internal/model"
	"starxo/internal/storage"
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

func TestActivateContainerRejectsSandboxOwnedByAnotherSession(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	sessionStore, err := storage.NewSessionStore()
	require.NoError(t, err)
	containerStore, err := storage.NewContainerStore()
	require.NoError(t, err)
	sessionSvc := NewSessionService(sessionStore, containerStore)
	_, err = sessionSvc.CreateSession("Active session")
	require.NoError(t, err)
	require.NoError(t, containerStore.Add(&model.Container{
		ID:        "ctr-other",
		RuntimeID: "runtime-other",
		SessionID: "different-session",
	}))

	svc := NewSandboxService(nil, containerStore)
	svc.SetSessionService(sessionSvc)

	err = svc.ActivateContainer("ctr-other")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "belongs to another session")
}
