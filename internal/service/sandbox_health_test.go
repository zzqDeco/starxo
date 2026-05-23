package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"starxo/internal/sandbox"
)

func TestSandboxHealthCheckKeepsManagerAfterSingleSSHFailure(t *testing.T) {
	svc := NewSandboxService(nil, nil)
	mgr := &sandbox.SandboxManager{}
	var deactivated int
	svc.SetOnContainerDeactivated(func() { deactivated++ })
	svc.healthSSHProbe = func(context.Context, *sandbox.SandboxManager) error {
		return errors.New("transient ssh failure")
	}
	svc.healthSandboxProbe = func(context.Context, *sandbox.SandboxManager) (bool, error) {
		return true, nil
	}
	svc.mu.Lock()
	svc.manager = mgr
	svc.activeContainerRegID = "sbx-1"
	svc.healthGeneration = 1
	svc.mu.Unlock()

	svc.healthCheck(1)

	svc.mu.RLock()
	defer svc.mu.RUnlock()
	assert.Same(t, mgr, svc.manager)
	assert.Equal(t, "sbx-1", svc.activeContainerRegID)
	assert.Equal(t, 1, svc.healthSSHFailures)
	assert.Equal(t, 0, deactivated)
}

func TestSandboxHealthCheckDisconnectsAfterConsecutiveSSHFailures(t *testing.T) {
	svc := NewSandboxService(nil, nil)
	mgr := &sandbox.SandboxManager{}
	var deactivated int
	svc.SetOnContainerDeactivated(func() { deactivated++ })
	svc.healthSSHProbe = func(context.Context, *sandbox.SandboxManager) error {
		return errors.New("ssh is down")
	}
	svc.mu.Lock()
	svc.manager = mgr
	svc.activeContainerRegID = "sbx-1"
	svc.healthGeneration = 1
	svc.mu.Unlock()

	svc.healthCheck(1)
	svc.healthCheck(1)

	svc.mu.RLock()
	defer svc.mu.RUnlock()
	assert.Nil(t, svc.manager)
	assert.Empty(t, svc.activeContainerRegID)
	assert.Equal(t, 0, svc.healthSSHFailures)
	assert.Equal(t, 1, deactivated)
}

func TestSandboxHealthCheckDeactivatesMissingSandboxWithoutDroppingSSH(t *testing.T) {
	svc := NewSandboxService(nil, nil)
	mgr := &sandbox.SandboxManager{}
	var deactivated int
	svc.SetOnContainerDeactivated(func() { deactivated++ })
	svc.healthSSHProbe = func(context.Context, *sandbox.SandboxManager) error {
		return nil
	}
	svc.healthSandboxProbe = func(context.Context, *sandbox.SandboxManager) (bool, error) {
		return false, nil
	}
	svc.mu.Lock()
	svc.manager = mgr
	svc.activeContainerRegID = "sbx-1"
	svc.healthGeneration = 3
	svc.healthSSHFailures = 1
	svc.mu.Unlock()

	svc.healthCheck(3)

	svc.mu.RLock()
	defer svc.mu.RUnlock()
	assert.Same(t, mgr, svc.manager)
	assert.Empty(t, svc.activeContainerRegID)
	assert.Equal(t, 0, svc.healthSSHFailures)
	assert.Equal(t, 1, deactivated)
}

func TestSandboxHealthCheckIgnoresStaleGeneration(t *testing.T) {
	svc := NewSandboxService(nil, nil)
	current := &sandbox.SandboxManager{}
	stale := &sandbox.SandboxManager{}
	called := false
	svc.healthSSHProbe = func(context.Context, *sandbox.SandboxManager) error {
		called = true
		return errors.New("should not run")
	}
	svc.mu.Lock()
	svc.manager = current
	svc.healthGeneration = 4
	svc.mu.Unlock()

	svc.markDisconnectedIfCurrent(3, stale, "stale health check")
	svc.healthCheck(3)

	svc.mu.RLock()
	defer svc.mu.RUnlock()
	require.Same(t, current, svc.manager)
	assert.False(t, called)
}

func TestSandboxHealthCheckDoesNotDeactivateNewActiveSandboxFromStaleProbe(t *testing.T) {
	svc := NewSandboxService(nil, nil)
	mgr := &sandbox.SandboxManager{}
	var deactivated int
	svc.SetOnContainerDeactivated(func() { deactivated++ })
	svc.healthSSHProbe = func(context.Context, *sandbox.SandboxManager) error {
		return nil
	}
	svc.healthSandboxProbe = func(context.Context, *sandbox.SandboxManager) (bool, error) {
		svc.mu.Lock()
		svc.activeContainerRegID = "sbx-new"
		svc.mu.Unlock()
		return false, nil
	}
	svc.mu.Lock()
	svc.manager = mgr
	svc.activeContainerRegID = "sbx-old"
	svc.healthGeneration = 5
	svc.mu.Unlock()

	svc.healthCheck(5)

	svc.mu.RLock()
	defer svc.mu.RUnlock()
	require.Same(t, mgr, svc.manager)
	assert.Equal(t, "sbx-new", svc.activeContainerRegID)
	assert.Equal(t, 0, deactivated)
}
