package sandbox

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"starxo/internal/config"
)

type createSandboxCommandRunner struct {
	commands []string
	pipExit  int
}

func (r *createSandboxCommandRunner) RunCommand(ctx context.Context, cmd string) (string, string, int, error) {
	r.commands = append(r.commands, cmd)
	switch {
	case strings.Contains(cmd, "uname -s"):
		return "Linux\n", "", 0, nil
	case strings.Contains(cmd, "printf %s \"$HOME\""):
		return "/home/starxo", "", 0, nil
	case strings.Contains(cmd, "command -v 'bwrap'"):
		return "/usr/bin/bwrap\n", "", 0, nil
	case strings.Contains(cmd, "command -v 'python3'"):
		return "/usr/bin/python3\n", "", 0, nil
	case strings.Contains(cmd, "bwrap --die-with-parent"):
		return "", "", 0, nil
	case strings.Contains(cmd, "pip install --no-cache-dir"):
		if r.pipExit != 0 {
			return "", "pip stalled", r.pipExit, nil
		}
		return "installed", "", 0, nil
	case strings.Contains(cmd, "python3 -m venv"),
		strings.Contains(cmd, "pip install --upgrade pip"),
		strings.Contains(cmd, "mkdir -p --"),
		strings.Contains(cmd, "rm -rf --"):
		return "", "", 0, nil
	default:
		return "", "", 127, fmt.Errorf("unexpected command: %s", cmd)
	}
}

func TestCreateSandboxBootstrapsInObservableStepsWithTimeoutWrapper(t *testing.T) {
	runner := &createSandboxCommandRunner{}
	manager := &RemoteRuntimeManager{
		ssh: runner,
		cfg: config.SandboxConfig{
			Runtime:           RuntimeBwrap,
			RootDir:           "~/.starxo/sandboxes",
			WorkDirName:       "workspace",
			CommandTimeoutSec: 3,
			BootstrapPython:   true,
			PythonPackages:    []string{"pandas", "numpy"},
			Network:           true,
		},
	}
	var progress []string

	inst, err := manager.CreateSandbox(context.Background(), nil, func(step string, percent int) {
		progress = append(progress, fmt.Sprintf("%d:%s", percent, step))
	})
	require.NoError(t, err)

	assert.NotEmpty(t, inst.ID)
	assert.Contains(t, progress, "20:Creating sandbox directories")
	assert.Contains(t, progress, "40:Creating Python virtual environment")
	assert.Contains(t, progress, "60:Upgrading sandbox pip")
	assert.Contains(t, progress, "75:Installing sandbox Python packages")

	assertCommandContains(t, runner.commands, "timeout --kill-after=5s 3s")
	assertCommandContains(t, runner.commands, "python3 -m venv")
	assertCommandContains(t, runner.commands, "pip install --upgrade pip")
	assertCommandContains(t, runner.commands, "pip install --no-cache-dir --timeout 30 --retries 2")
}

func TestCreateSandboxCleansIncompleteRootWhenPipInstallTimesOut(t *testing.T) {
	runner := &createSandboxCommandRunner{pipExit: 124}
	manager := &RemoteRuntimeManager{
		ssh: runner,
		cfg: config.SandboxConfig{
			Runtime:           RuntimeBwrap,
			RootDir:           "~/.starxo/sandboxes",
			WorkDirName:       "workspace",
			CommandTimeoutSec: 5,
			BootstrapPython:   true,
			PythonPackages:    []string{"pandas"},
			Network:           true,
		},
	}

	inst, err := manager.CreateSandbox(context.Background(), nil, nil)
	require.Error(t, err)
	assert.Nil(t, inst)
	assert.Contains(t, err.Error(), "install sandbox Python packages timed out after 5 seconds")
	assert.Contains(t, err.Error(), "pip index")
	assert.False(t, manager.IsActive())
	assertCommandContains(t, runner.commands, "rm -rf -- '/home/starxo/.starxo/sandboxes/")
}

func assertCommandContains(t *testing.T, commands []string, needle string) {
	t.Helper()
	for _, cmd := range commands {
		if strings.Contains(cmd, needle) {
			return
		}
	}
	t.Fatalf("expected one command to contain %q, got %#v", needle, commands)
}
