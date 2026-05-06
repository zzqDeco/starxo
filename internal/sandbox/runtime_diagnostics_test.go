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

type fakeCommandResponse struct {
	match    string
	stdout   string
	stderr   string
	exitCode int
	err      error
}

type fakeCommandRunner struct {
	responses []fakeCommandResponse
}

func (f *fakeCommandRunner) RunCommand(ctx context.Context, cmd string) (string, string, int, error) {
	for _, response := range f.responses {
		if cmd == response.match || strings.HasPrefix(cmd, response.match) {
			return response.stdout, response.stderr, response.exitCode, response.err
		}
	}
	return "", "", 127, fmt.Errorf("unexpected command: %s", cmd)
}

func TestDiagnoseBwrapReportsAppArmorUsernsFailure(t *testing.T) {
	runner := &fakeCommandRunner{responses: []fakeCommandResponse{
		{match: "uname -s", stdout: "Linux\n"},
		{match: "command -v 'bwrap'", stdout: "/usr/bin/bwrap\n"},
		{match: "command -v 'python3'", stdout: "/usr/bin/python3\n"},
		{match: "bwrap --version", stdout: "bubblewrap 0.9.0\n"},
		{match: "python3 --version", stdout: "Python 3.12.3\n"},
		{match: "tmp=${TMPDIR:-/tmp}/starxo-venv-check-", exitCode: 0},
		{match: "if [ -r /proc/sys/kernel/unprivileged_userns_clone ]", stdout: "1\n"},
		{match: "if command -v sysctl >/dev/null 2>&1", stdout: "1\n"},
		{match: "bwrap --die-with-parent --ro-bind / / --tmpfs /tmp --proc /proc --dev /dev sh -lc 'true'", stderr: "setting up uid map: Permission denied\n", exitCode: 1},
	}}
	manager := &RemoteRuntimeManager{
		ssh: runner,
		cfg: config.SandboxConfig{Runtime: RuntimeAuto, RootDir: "/tmp/starxo", Network: true, CommandTimeoutSec: 120, MemoryLimitMB: 2048},
	}

	result, err := manager.Diagnose(context.Background())
	require.NoError(t, err)

	assert.Equal(t, RuntimeBwrap, result.Runtime)
	assert.Equal(t, "Linux", result.OS)
	assert.False(t, result.Available)
	assertCheckStatus(t, result.Checks, "runtime.apparmor.userns", DiagnosticFail)
	assertFixIDs(t, result.Fixes, "relax-apparmor-userns", "enable-userns")
}

func TestDiagnoseSeatbeltMissingSandboxExec(t *testing.T) {
	runner := &fakeCommandRunner{responses: []fakeCommandResponse{
		{match: "uname -s", stdout: "Darwin\n"},
		{match: "command -v 'sandbox-exec'", exitCode: 1},
		{match: "command -v 'python3'", stdout: "/usr/bin/python3\n"},
		{match: "python3 --version", stdout: "Python 3.12.3\n"},
	}}
	manager := &RemoteRuntimeManager{
		ssh: runner,
		cfg: config.SandboxConfig{Runtime: RuntimeAuto, RootDir: "/tmp/starxo", Network: true},
	}

	result, err := manager.Diagnose(context.Background())
	require.NoError(t, err)

	assert.Equal(t, RuntimeSeatbelt, result.Runtime)
	assert.False(t, result.Available)
	assertCheckStatus(t, result.Checks, "runtime.seatbelt", DiagnosticFail)
	assertFixIDs(t, result.Fixes, "seatbelt-unavailable")
}

func TestDiagnoseBwrapAvailableWhenSmokePasses(t *testing.T) {
	runner := &fakeCommandRunner{responses: []fakeCommandResponse{
		{match: "uname -s", stdout: "Linux\n"},
		{match: "command -v 'bwrap'", stdout: "/usr/bin/bwrap\n"},
		{match: "command -v 'python3'", stdout: "/usr/bin/python3\n"},
		{match: "bwrap --version", stdout: "bubblewrap 0.9.0\n"},
		{match: "python3 --version", stdout: "Python 3.12.3\n"},
		{match: "tmp=${TMPDIR:-/tmp}/starxo-venv-check-", exitCode: 0},
		{match: "if [ -r /proc/sys/kernel/unprivileged_userns_clone ]", stdout: "1\n"},
		{match: "if command -v sysctl >/dev/null 2>&1", stdout: "0\n"},
		{match: "bwrap --die-with-parent --unshare-net --ro-bind / / --tmpfs /tmp --proc /proc --dev /dev sh -lc 'true'", exitCode: 0},
	}}
	manager := &RemoteRuntimeManager{
		ssh: runner,
		cfg: config.SandboxConfig{Runtime: RuntimeBwrap, RootDir: "/tmp/starxo", Network: false},
	}

	result, err := manager.Diagnose(context.Background())
	require.NoError(t, err)

	assert.True(t, result.Available)
	assert.True(t, strings.Contains(result.Summary, "passed"))
	assertCheckStatus(t, result.Checks, "runtime.bwrap.smoke", DiagnosticPass)
	assert.Empty(t, result.Fixes)
}

func TestCleanupTmpRefusesUnsafePath(t *testing.T) {
	manager := &RemoteRuntimeManager{
		ssh: &fakeCommandRunner{},
		instance: &SandboxInstance{
			ID:       "sbx-test",
			RootPath: "/tmp/starxo/sbx-test",
			TmpPath:  "/",
		},
	}

	_, err := manager.CleanupTmp(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsafe sandbox tmp path")
}

func TestCleanupTmpParsesResult(t *testing.T) {
	runner := &fakeCommandRunner{responses: []fakeCommandResponse{
		{match: "if [ ! -d '/tmp/starxo/sbx-test/tmp' ]", stdout: "2 4096\n"},
	}}
	manager := &RemoteRuntimeManager{
		ssh: runner,
		instance: &SandboxInstance{
			ID:       "sbx-test",
			RootPath: "/tmp/starxo/sbx-test",
			TmpPath:  "/tmp/starxo/sbx-test/tmp",
		},
	}

	result, err := manager.CleanupTmp(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "/tmp/starxo/sbx-test/tmp", result.TmpPath)
	assert.Equal(t, 2, result.RemovedEntries)
	assert.Equal(t, int64(4096), result.ReclaimedBytes)
}

func assertCheckStatus(t *testing.T, checks []SandboxDiagnosticCheck, id, status string) {
	t.Helper()
	for _, check := range checks {
		if check.ID == id {
			assert.Equal(t, status, check.Status)
			return
		}
	}
	t.Fatalf("check %s not found", id)
}

func assertFixIDs(t *testing.T, fixes []SandboxFixSuggestion, ids ...string) {
	t.Helper()
	got := make(map[string]bool, len(fixes))
	for _, fix := range fixes {
		got[fix.ID] = true
	}
	for _, id := range ids {
		assert.True(t, got[id], "fix %s not found in %#v", id, fixes)
	}
}
