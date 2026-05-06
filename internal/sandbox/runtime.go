package sandbox

import (
	"context"
	"encoding/base64"
	"fmt"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"starxo/internal/config"
)

const (
	RuntimeAuto     = "auto"
	RuntimeBwrap    = "bwrap"
	RuntimeSeatbelt = "seatbelt"
	RuntimeDocker   = "docker"
)

const (
	DiagnosticPass    = "pass"
	DiagnosticWarn    = "warn"
	DiagnosticFail    = "fail"
	DiagnosticInfo    = "info"
	DiagnosticSkipped = "skipped"
)

type RuntimeCheckResult struct {
	Runtime         string   `json:"runtime"`
	OS              string   `json:"os"`
	Available       bool     `json:"available"`
	Installable     bool     `json:"installable"`
	Missing         []string `json:"missing"`
	Message         string   `json:"message"`
	InstallCommand  string   `json:"installCommand,omitempty"`
	WorkspaceRoot   string   `json:"workspaceRoot,omitempty"`
	CommandTimeout  int      `json:"commandTimeoutSec"`
	MemoryLimitMB   int64    `json:"memoryLimitMB"`
	NetworkEnabled  bool     `json:"networkEnabled"`
	PythonBootstrap bool     `json:"pythonBootstrap"`
}

type RuntimeInstallResult struct {
	Runtime   string `json:"runtime"`
	OS        string `json:"os"`
	Installed bool   `json:"installed"`
	Stdout    string `json:"stdout,omitempty"`
	Stderr    string `json:"stderr,omitempty"`
	Message   string `json:"message"`
}

type SandboxDiagnosticsResult struct {
	Runtime           string                   `json:"runtime"`
	OS                string                   `json:"os"`
	Available         bool                     `json:"available"`
	Summary           string                   `json:"summary"`
	Checks            []SandboxDiagnosticCheck `json:"checks"`
	Fixes             []SandboxFixSuggestion   `json:"fixes"`
	WorkspaceRoot     string                   `json:"workspaceRoot,omitempty"`
	CommandTimeoutSec int                      `json:"commandTimeoutSec"`
	MemoryLimitMB     int64                    `json:"memoryLimitMB"`
	NetworkEnabled    bool                     `json:"networkEnabled"`
}

type SandboxDiagnosticCheck struct {
	ID      string   `json:"id"`
	Label   string   `json:"label"`
	Status  string   `json:"status"`
	Message string   `json:"message"`
	Details string   `json:"details,omitempty"`
	Command string   `json:"command,omitempty"`
	Output  string   `json:"output,omitempty"`
	FixIDs  []string `json:"fixIDs,omitempty"`
}

type SandboxFixSuggestion struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Risk         string   `json:"risk"`
	Platform     string   `json:"platform,omitempty"`
	Commands     []string `json:"commands,omitempty"`
	CopyOnly     bool     `json:"copyOnly"`
	AutoRunnable bool     `json:"autoRunnable"`
}

type RuntimeCleanupResult struct {
	TmpPath        string `json:"tmpPath"`
	RemovedEntries int    `json:"removedEntries"`
	ReclaimedBytes int64  `json:"reclaimedBytes"`
}

type SandboxInstance struct {
	ID            string
	Name          string
	Runtime       string
	WorkspacePath string
	RootPath      string
	TmpPath       string
	VenvPath      string
	ProfilePath   string
}

type remoteCommandRunner interface {
	RunCommand(ctx context.Context, cmd string) (stdout, stderr string, exitCode int, err error)
}

type RemoteRuntimeManager struct {
	ssh      remoteCommandRunner
	cfg      config.SandboxConfig
	kind     string
	osName   string
	instance *SandboxInstance
	mu       sync.Mutex
}

func NewRemoteRuntimeManager(ssh *SSHClient, cfg config.SandboxConfig) *RemoteRuntimeManager {
	appCfg := &config.AppConfig{Sandbox: cfg}
	config.NormalizeAppConfig(appCfg)
	return &RemoteRuntimeManager{ssh: ssh, cfg: appCfg.Sandbox}
}

func (m *RemoteRuntimeManager) Detect(ctx context.Context) (RuntimeCheckResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.ensureKindLocked(ctx); err != nil {
		return RuntimeCheckResult{}, err
	}

	root, _ := m.resolveRootLocked(ctx)
	result := RuntimeCheckResult{
		Runtime:         m.kind,
		OS:              m.osName,
		WorkspaceRoot:   root,
		CommandTimeout:  m.cfg.CommandTimeoutSec,
		MemoryLimitMB:   m.cfg.MemoryLimitMB,
		NetworkEnabled:  m.cfg.Network,
		PythonBootstrap: m.cfg.BootstrapPython,
	}

	switch m.kind {
	case RuntimeBwrap:
		result.Installable = true
		result.InstallCommand = linuxInstallCommand()
		m.detectCommand(ctx, &result, "bwrap")
		m.detectCommand(ctx, &result, "python3")
		if len(result.Missing) == 0 {
			_, stderr, exitCode, err := m.ssh.RunCommand(ctx, "bwrap --die-with-parent --ro-bind / / --tmpfs /tmp --proc /proc --dev /dev sh -lc 'true'")
			if err != nil || exitCode != 0 {
				result.Missing = append(result.Missing, "user namespace support")
				if stderr != "" {
					result.Message = strings.TrimSpace(stderr)
				}
			}
		}
	case RuntimeSeatbelt:
		result.Installable = false
		m.detectCommand(ctx, &result, "sandbox-exec")
		m.detectCommand(ctx, &result, "python3")
		if len(result.Missing) > 0 {
			result.Message = "macOS sandbox-exec is required for Seatbelt runtime and cannot be installed by Starxo"
		}
	default:
		result.Missing = append(result.Missing, "supported runtime")
	}

	result.Available = len(result.Missing) == 0
	if result.Available && result.Message == "" {
		result.Message = "sandbox runtime is available"
	}
	return result, nil
}

func (m *RemoteRuntimeManager) Install(ctx context.Context) (RuntimeInstallResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.ensureKindLocked(ctx); err != nil {
		return RuntimeInstallResult{}, err
	}
	result := RuntimeInstallResult{Runtime: m.kind, OS: m.osName}
	if m.kind != RuntimeBwrap {
		result.Message = "automatic runtime installation is only supported for Linux bubblewrap hosts"
		return result, nil
	}

	stdout, stderr, exitCode, err := m.ssh.RunCommand(ctx, linuxInstallCommand())
	result.Stdout = stdout
	result.Stderr = stderr
	if err != nil {
		result.Message = err.Error()
		return result, err
	}
	if exitCode != 0 {
		result.Message = fmt.Sprintf("runtime installation failed with exit code %d", exitCode)
		return result, fmt.Errorf("%s: %s", result.Message, stderr)
	}
	result.Installed = true
	result.Message = "bubblewrap runtime installed"
	return result, nil
}

func (m *RemoteRuntimeManager) Diagnose(ctx context.Context) (SandboxDiagnosticsResult, error) {
	m.mu.Lock()
	if err := m.ensureKindLocked(ctx); err != nil {
		m.mu.Unlock()
		return SandboxDiagnosticsResult{}, err
	}
	root, err := m.resolveRootLocked(ctx)
	if err != nil {
		m.mu.Unlock()
		return SandboxDiagnosticsResult{}, err
	}
	kind := m.kind
	osName := m.osName
	cfg := m.cfg
	m.mu.Unlock()

	builder := newDiagnosticBuilder(kind, osName, root, cfg)
	switch kind {
	case RuntimeBwrap:
		m.diagnoseBwrap(ctx, builder)
	case RuntimeSeatbelt:
		m.diagnoseSeatbelt(ctx, builder)
	default:
		builder.addCheck(SandboxDiagnosticCheck{
			ID:      "runtime.supported",
			Label:   "Supported runtime",
			Status:  DiagnosticFail,
			Message: fmt.Sprintf("unsupported sandbox runtime %q", kind),
		})
	}
	return builder.finish(), nil
}

func (m *RemoteRuntimeManager) diagnoseBwrap(ctx context.Context, b *diagnosticBuilder) {
	b.addCheck(SandboxDiagnosticCheck{
		ID:      "runtime.network",
		Label:   "Network policy",
		Status:  DiagnosticInfo,
		Message: networkPolicyMessage(b.result.NetworkEnabled),
	})

	bwrapOK := m.checkCommand(ctx, b, "runtime.bwrap", "bubblewrap binary", "bwrap", "install-linux-runtime")
	pythonOK := m.checkCommand(ctx, b, "runtime.python", "Python binary", "python3", "install-linux-runtime")

	if bwrapOK {
		m.runDiagnostic(ctx, b, "runtime.bwrap.version", "bubblewrap version", "bwrap --version", DiagnosticPass, "bubblewrap version detected", nil)
	} else {
		b.addSkipped("runtime.bwrap.version", "bubblewrap version", "install bubblewrap before checking its version", []string{"install-linux-runtime"})
	}

	if pythonOK {
		m.runDiagnostic(ctx, b, "runtime.python.version", "Python version", "python3 --version", DiagnosticPass, "Python version detected", nil)
		m.runDiagnostic(ctx, b, "runtime.python.venv", "Python venv smoke", "tmp=${TMPDIR:-/tmp}/starxo-venv-check-$$; rm -rf \"$tmp\"; python3 -m venv \"$tmp/.venv\"; rc=$?; rm -rf \"$tmp\"; exit $rc", DiagnosticPass, "python3 can create virtual environments", []string{"install-linux-runtime"})
	} else {
		b.addSkipped("runtime.python.version", "Python version", "install python3 before checking its version", []string{"install-linux-runtime"})
		b.addSkipped("runtime.python.venv", "Python venv smoke", "install python3 before checking venv support", []string{"install-linux-runtime"})
	}

	m.diagnoseLinuxUserNamespaces(ctx, b)

	if bwrapOK {
		smokeCmd := bwrapDiagnosticSmokeCommand(b.result.NetworkEnabled)
		stdout, stderr, exitCode, err := m.ssh.RunCommand(ctx, smokeCmd)
		check := SandboxDiagnosticCheck{
			ID:      "runtime.bwrap.smoke",
			Label:   "bubblewrap smoke",
			Command: smokeCmd,
			Output:  diagnosticOutput(stdout, stderr),
		}
		if err == nil && exitCode == 0 {
			check.Status = DiagnosticPass
			check.Message = "bubblewrap can start an isolated process"
		} else {
			check.Status = DiagnosticFail
			check.Message = "bubblewrap failed to start an isolated process"
			check.Details = commandFailureDetails(exitCode, err)
			check.FixIDs = bwrapSmokeFixIDs(stdout, stderr)
		}
		b.addCheck(check)
	} else {
		b.addSkipped("runtime.bwrap.smoke", "bubblewrap smoke", "install bubblewrap before running the smoke test", []string{"install-linux-runtime"})
	}
}

func (m *RemoteRuntimeManager) diagnoseSeatbelt(ctx context.Context, b *diagnosticBuilder) {
	b.addCheck(SandboxDiagnosticCheck{
		ID:      "runtime.network",
		Label:   "Network policy",
		Status:  DiagnosticInfo,
		Message: networkPolicyMessage(b.result.NetworkEnabled),
	})

	seatbeltOK := m.checkCommand(ctx, b, "runtime.seatbelt", "Seatbelt sandbox-exec", "sandbox-exec", "seatbelt-unavailable")
	pythonOK := m.checkCommand(ctx, b, "runtime.python", "Python binary", "python3", "seatbelt-python")
	if pythonOK {
		m.runDiagnostic(ctx, b, "runtime.python.version", "Python version", "python3 --version", DiagnosticPass, "Python version detected", []string{"seatbelt-python"})
	} else {
		b.addSkipped("runtime.python.version", "Python version", "install python3 before checking its version", []string{"seatbelt-python"})
	}
	if seatbeltOK {
		m.runDiagnostic(ctx, b, "runtime.seatbelt.smoke", "Seatbelt smoke", "sandbox-exec -p '(version 1) (allow default)' true", DiagnosticPass, "sandbox-exec can evaluate a minimal profile", []string{"seatbelt-unavailable"})
	} else {
		b.addSkipped("runtime.seatbelt.smoke", "Seatbelt smoke", "sandbox-exec is unavailable on this macOS host", []string{"seatbelt-unavailable"})
	}
}

func (m *RemoteRuntimeManager) diagnoseLinuxUserNamespaces(ctx context.Context, b *diagnosticBuilder) {
	stdout, stderr, exitCode, err := m.ssh.RunCommand(ctx, "if [ -r /proc/sys/kernel/unprivileged_userns_clone ]; then cat /proc/sys/kernel/unprivileged_userns_clone; else echo unavailable; fi")
	value := strings.TrimSpace(stdout)
	check := SandboxDiagnosticCheck{
		ID:      "runtime.userns",
		Label:   "Unprivileged user namespaces",
		Command: "cat /proc/sys/kernel/unprivileged_userns_clone",
		Output:  diagnosticOutput(stdout, stderr),
	}
	switch {
	case err != nil || exitCode != 0:
		check.Status = DiagnosticWarn
		check.Message = "could not read user namespace sysctl"
		check.Details = commandFailureDetails(exitCode, err)
	case value == "0":
		check.Status = DiagnosticFail
		check.Message = "unprivileged user namespaces are disabled"
		check.FixIDs = []string{"enable-userns"}
	case value == "1":
		check.Status = DiagnosticPass
		check.Message = "unprivileged user namespaces are enabled"
	default:
		check.Status = DiagnosticInfo
		check.Message = "user namespace sysctl is not exposed on this host"
	}
	b.addCheck(check)

	stdout, stderr, exitCode, err = m.ssh.RunCommand(ctx, "if command -v sysctl >/dev/null 2>&1; then sysctl -n kernel.apparmor_restrict_unprivileged_userns 2>/dev/null || echo unavailable; else echo unavailable; fi")
	value = strings.TrimSpace(stdout)
	check = SandboxDiagnosticCheck{
		ID:      "runtime.apparmor.userns",
		Label:   "AppArmor userns restriction",
		Command: "sysctl -n kernel.apparmor_restrict_unprivileged_userns",
		Output:  diagnosticOutput(stdout, stderr),
	}
	switch {
	case err != nil || exitCode != 0:
		check.Status = DiagnosticWarn
		check.Message = "could not read AppArmor user namespace restriction"
		check.Details = commandFailureDetails(exitCode, err)
	case value == "1":
		check.Status = DiagnosticFail
		check.Message = "AppArmor is blocking unprivileged user namespaces for bubblewrap"
		check.FixIDs = []string{"relax-apparmor-userns"}
	case value == "0":
		check.Status = DiagnosticPass
		check.Message = "AppArmor user namespace restriction is disabled"
	default:
		check.Status = DiagnosticInfo
		check.Message = "AppArmor user namespace restriction is not exposed on this host"
	}
	b.addCheck(check)
}

func (m *RemoteRuntimeManager) checkCommand(ctx context.Context, b *diagnosticBuilder, id, label, commandName, fixID string) bool {
	cmd := fmt.Sprintf("command -v %s", shellQuote(commandName))
	stdout, stderr, exitCode, err := m.ssh.RunCommand(ctx, cmd)
	check := SandboxDiagnosticCheck{
		ID:      id,
		Label:   label,
		Command: cmd,
		Output:  diagnosticOutput(stdout, stderr),
	}
	if err == nil && exitCode == 0 {
		check.Status = DiagnosticPass
		check.Message = fmt.Sprintf("%s is available", commandName)
		b.addCheck(check)
		return true
	}
	check.Status = DiagnosticFail
	check.Message = fmt.Sprintf("%s is missing", commandName)
	check.Details = commandFailureDetails(exitCode, err)
	check.FixIDs = []string{fixID}
	b.addCheck(check)
	return false
}

func (m *RemoteRuntimeManager) runDiagnostic(ctx context.Context, b *diagnosticBuilder, id, label, cmd, passStatus, passMessage string, fixIDs []string) {
	stdout, stderr, exitCode, err := m.ssh.RunCommand(ctx, cmd)
	check := SandboxDiagnosticCheck{
		ID:      id,
		Label:   label,
		Command: cmd,
		Output:  diagnosticOutput(stdout, stderr),
	}
	if err == nil && exitCode == 0 {
		check.Status = passStatus
		check.Message = passMessage
	} else {
		check.Status = DiagnosticFail
		check.Message = fmt.Sprintf("%s failed", label)
		check.Details = commandFailureDetails(exitCode, err)
		check.FixIDs = fixIDs
	}
	b.addCheck(check)
}

func (m *RemoteRuntimeManager) CreateSandbox(ctx context.Context, excludeIDs []string) (*SandboxInstance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.ensureKindLocked(ctx); err != nil {
		return nil, err
	}
	check, err := m.detectLocked(ctx)
	if err != nil {
		return nil, err
	}
	if !check.Available {
		return nil, fmt.Errorf("sandbox runtime unavailable: %s", strings.Join(check.Missing, ", "))
	}

	id := fmt.Sprintf("sbx-%s", uuid.New().String()[:8])
	root, err := m.resolveRootLocked(ctx)
	if err != nil {
		return nil, err
	}
	inst := m.instanceFor(root, id, fmt.Sprintf("starxo-sandbox-%s", id))
	if err := m.createDirsLocked(ctx, inst); err != nil {
		return nil, err
	}
	if err := m.bootstrapLocked(ctx, inst); err != nil {
		return nil, err
	}
	if m.kind == RuntimeSeatbelt {
		if err := m.writeSeatbeltProfileLocked(ctx, inst); err != nil {
			return nil, err
		}
	}
	m.instance = inst
	return inst, nil
}

func (m *RemoteRuntimeManager) AttachSandbox(ctx context.Context, id, name, workspacePath string) (*SandboxInstance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.ensureKindLocked(ctx); err != nil {
		return nil, err
	}
	root, err := m.resolveRootLocked(ctx)
	if err != nil {
		return nil, err
	}
	if id == "" {
		return nil, fmt.Errorf("sandbox id is empty")
	}
	if name == "" {
		name = fmt.Sprintf("starxo-sandbox-%s", id)
	}
	inst := m.instanceFor(root, id, name)
	if workspacePath != "" {
		inst.WorkspacePath = cleanRemotePath(workspacePath)
		inst.RootPath = path.Dir(inst.WorkspacePath)
		inst.TmpPath = path.Join(inst.RootPath, "tmp")
		inst.VenvPath = path.Join(inst.RootPath, ".venv")
		inst.ProfilePath = path.Join(inst.RootPath, "seatbelt.sb")
	}

	exists, _, err := m.inspectLocked(ctx, inst)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("sandbox workspace %s no longer exists", inst.WorkspacePath)
	}
	if m.kind == RuntimeSeatbelt {
		if err := m.writeSeatbeltProfileLocked(ctx, inst); err != nil {
			return nil, err
		}
	}
	m.instance = inst
	return inst, nil
}

func (m *RemoteRuntimeManager) Deactivate() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.instance = nil
}

func (m *RemoteRuntimeManager) Destroy(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.instance == nil {
		return nil
	}
	root := m.instance.RootPath
	if root == "" || root == "/" || root == "." {
		return fmt.Errorf("refusing to remove unsafe sandbox path %q", root)
	}
	_, stderr, exitCode, err := m.ssh.RunCommand(ctx, fmt.Sprintf("rm -rf -- %s", shellQuote(root)))
	if err != nil {
		return err
	}
	if exitCode != 0 {
		return fmt.Errorf("failed to destroy sandbox %s: %s", m.instance.ID, stderr)
	}
	m.instance = nil
	return nil
}

func (m *RemoteRuntimeManager) DestroySandbox(ctx context.Context, id, workspacePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id == "" {
		return fmt.Errorf("sandbox id is empty")
	}
	root, err := m.resolveRootLocked(ctx)
	if err != nil {
		return err
	}
	targetRoot := m.instanceFor(root, id, "").RootPath
	if workspacePath != "" {
		targetRoot = path.Dir(cleanRemotePath(workspacePath))
	}
	if targetRoot == "" || targetRoot == "/" || targetRoot == "." {
		return fmt.Errorf("refusing to remove unsafe sandbox path %q", targetRoot)
	}
	_, stderr, exitCode, err := m.ssh.RunCommand(ctx, fmt.Sprintf("rm -rf -- %s", shellQuote(targetRoot)))
	if err != nil {
		return err
	}
	if exitCode != 0 {
		return fmt.Errorf("failed to destroy sandbox %s: %s", id, stderr)
	}
	if m.instance != nil && m.instance.ID == id {
		m.instance = nil
	}
	return nil
}

func (m *RemoteRuntimeManager) InspectSandbox(ctx context.Context, id, workspacePath string) (exists bool, active bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	root, err := m.resolveRootLocked(ctx)
	if err != nil {
		return false, false, err
	}
	inst := m.instanceFor(root, id, "")
	if workspacePath != "" {
		inst.WorkspacePath = cleanRemotePath(workspacePath)
	}
	exists, _, err = m.inspectLocked(ctx, inst)
	if err != nil {
		return false, false, err
	}
	active = m.instance != nil && m.instance.ID == id
	return exists, active, nil
}

func (m *RemoteRuntimeManager) ExecInSandbox(ctx context.Context, command []string) (stdout, stderr string, exitCode int, err error) {
	m.mu.Lock()
	inst := m.instance
	kind := m.kind
	cfg := m.cfg
	m.mu.Unlock()

	if inst == nil {
		return "", "", -1, fmt.Errorf("no sandbox is active")
	}
	if len(command) == 0 {
		return "", "", -1, fmt.Errorf("command is empty")
	}

	runCtx := ctx
	cancel := func() {}
	if cfg.CommandTimeoutSec > 0 {
		if _, ok := ctx.Deadline(); !ok {
			runCtx, cancel = context.WithTimeout(ctx, time.Duration(cfg.CommandTimeoutSec)*time.Second)
		}
	}
	defer cancel()

	inner := m.innerShell(command, inst, cfg)
	var remoteCmd string
	switch kind {
	case RuntimeBwrap:
		remoteCmd = m.bwrapCommand(inst, cfg, inner)
	case RuntimeSeatbelt:
		remoteCmd = m.seatbeltCommand(inst, cfg, inner)
	default:
		return "", "", -1, fmt.Errorf("unsupported sandbox runtime %q", kind)
	}
	return m.ssh.RunCommand(runCtx, remoteCmd)
}

func (m *RemoteRuntimeManager) RuntimeID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.instance == nil {
		return ""
	}
	return m.instance.ID
}

func (m *RemoteRuntimeManager) RuntimeName() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.instance == nil {
		return ""
	}
	return m.instance.Name
}

func (m *RemoteRuntimeManager) RuntimeKind() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.kind
}

func (m *RemoteRuntimeManager) WorkspacePath() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.instance == nil {
		return ""
	}
	return m.instance.WorkspacePath
}

func (m *RemoteRuntimeManager) TmpPath() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.instance == nil {
		return ""
	}
	return m.instance.TmpPath
}

func (m *RemoteRuntimeManager) SandboxRootPath() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.instance == nil {
		return ""
	}
	return m.instance.RootPath
}

func (m *RemoteRuntimeManager) IsActive() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.instance != nil
}

func (m *RemoteRuntimeManager) CleanupTmp(ctx context.Context) (RuntimeCleanupResult, error) {
	m.mu.Lock()
	inst := m.instance
	m.mu.Unlock()
	if inst == nil {
		return RuntimeCleanupResult{}, fmt.Errorf("no sandbox is active")
	}
	root := cleanRemotePath(inst.RootPath)
	tmp := cleanRemotePath(inst.TmpPath)
	if root == "" || root == "/" || root == "." {
		return RuntimeCleanupResult{}, fmt.Errorf("refusing to clean unsafe sandbox root %q", root)
	}
	if tmp == "" || tmp == "/" || tmp == "." || tmp == root || !strings.HasPrefix(tmp, root+"/") {
		return RuntimeCleanupResult{}, fmt.Errorf("refusing to clean unsafe sandbox tmp path %q", tmp)
	}

	cmd := fmt.Sprintf("if [ ! -d %s ]; then printf '0 0\\n'; exit 0; fi; "+
		"count=$(find %s -mindepth 1 -maxdepth 1 2>/dev/null | wc -l | tr -d ' '); "+
		"kb=$(du -sk %s 2>/dev/null | awk '{print $1}'); "+
		"find %s -mindepth 1 -maxdepth 1 -exec rm -rf -- {} +; "+
		"printf '%%s %%s\\n' \"${count:-0}\" \"$(( ${kb:-0} * 1024 ))\"",
		shellQuote(tmp), shellQuote(tmp), shellQuote(tmp), shellQuote(tmp))
	stdout, stderr, exitCode, err := m.ssh.RunCommand(ctx, cmd)
	if err != nil {
		return RuntimeCleanupResult{}, err
	}
	if exitCode != 0 {
		return RuntimeCleanupResult{}, fmt.Errorf("failed to clean sandbox tmp directory: %s", stderr)
	}
	fields := strings.Fields(strings.TrimSpace(stdout))
	result := RuntimeCleanupResult{TmpPath: tmp}
	if len(fields) >= 1 {
		result.RemovedEntries, _ = strconv.Atoi(fields[0])
	}
	if len(fields) >= 2 {
		result.ReclaimedBytes, _ = strconv.ParseInt(fields[1], 10, 64)
	}
	return result, nil
}

// Legacy method names kept for callers that are migrated separately.
func (m *RemoteRuntimeManager) ContainerID() string   { return m.RuntimeID() }
func (m *RemoteRuntimeManager) ContainerName() string { return m.RuntimeName() }
func (m *RemoteRuntimeManager) IsRunning() bool       { return m.IsActive() }
func (m *RemoteRuntimeManager) ClearContainer()       { m.Deactivate() }
func (m *RemoteRuntimeManager) StopContainer(ctx context.Context) error {
	m.Deactivate()
	return nil
}
func (m *RemoteRuntimeManager) StopAndRemove(ctx context.Context) error { return m.Destroy(ctx) }
func (m *RemoteRuntimeManager) ExecInContainer(ctx context.Context, cmd []string) (string, string, int, error) {
	return m.ExecInSandbox(ctx, cmd)
}
func (m *RemoteRuntimeManager) InspectContainer(ctx context.Context, id string) (bool, bool, error) {
	return m.InspectSandbox(ctx, id, "")
}

func (m *RemoteRuntimeManager) detectLocked(ctx context.Context) (RuntimeCheckResult, error) {
	m.mu.Unlock()
	result, err := m.Detect(ctx)
	m.mu.Lock()
	return result, err
}

func (m *RemoteRuntimeManager) ensureKindLocked(ctx context.Context) error {
	if m.kind != "" {
		return nil
	}
	stdout, stderr, exitCode, err := m.ssh.RunCommand(ctx, "uname -s")
	if err != nil {
		return fmt.Errorf("failed to detect remote OS: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("failed to detect remote OS: %s", stderr)
	}
	m.osName = strings.TrimSpace(stdout)
	requested := strings.ToLower(strings.TrimSpace(m.cfg.Runtime))
	if requested == "" {
		requested = RuntimeAuto
	}
	switch requested {
	case RuntimeAuto:
		switch strings.ToLower(m.osName) {
		case "linux":
			m.kind = RuntimeBwrap
		case "darwin":
			m.kind = RuntimeSeatbelt
		default:
			return fmt.Errorf("unsupported remote OS %q for auto sandbox runtime", m.osName)
		}
	case RuntimeBwrap, RuntimeSeatbelt:
		m.kind = requested
	default:
		return fmt.Errorf("unsupported sandbox runtime %q", requested)
	}
	return nil
}

func (m *RemoteRuntimeManager) detectCommand(ctx context.Context, result *RuntimeCheckResult, cmd string) {
	_, _, exitCode, err := m.ssh.RunCommand(ctx, fmt.Sprintf("command -v %s >/dev/null 2>&1", shellQuote(cmd)))
	if err != nil || exitCode != 0 {
		result.Missing = append(result.Missing, cmd)
	}
}

func (m *RemoteRuntimeManager) resolveRootLocked(ctx context.Context) (string, error) {
	root := strings.TrimSpace(m.cfg.RootDir)
	if root == "" {
		root = "~/.starxo/sandboxes"
	}
	if root == "~" || strings.HasPrefix(root, "~/") {
		home, stderr, exitCode, err := m.ssh.RunCommand(ctx, "printf %s \"$HOME\"")
		if err != nil {
			return "", err
		}
		if exitCode != 0 {
			return "", fmt.Errorf("failed to resolve remote home: %s", stderr)
		}
		root = strings.TrimSpace(home) + strings.TrimPrefix(root, "~")
	}
	return cleanRemotePath(root), nil
}

func (m *RemoteRuntimeManager) instanceFor(root, id, name string) *SandboxInstance {
	workDir := strings.Trim(m.cfg.WorkDirName, "/")
	if workDir == "" {
		workDir = "workspace"
	}
	rootPath := path.Join(root, id)
	if name == "" {
		name = fmt.Sprintf("starxo-sandbox-%s", id)
	}
	return &SandboxInstance{
		ID:            id,
		Name:          name,
		Runtime:       m.kind,
		RootPath:      rootPath,
		WorkspacePath: path.Join(rootPath, workDir),
		TmpPath:       path.Join(rootPath, "tmp"),
		VenvPath:      path.Join(rootPath, ".venv"),
		ProfilePath:   path.Join(rootPath, "seatbelt.sb"),
	}
}

func (m *RemoteRuntimeManager) createDirsLocked(ctx context.Context, inst *SandboxInstance) error {
	cmd := fmt.Sprintf("mkdir -p -- %s %s", shellQuote(inst.WorkspacePath), shellQuote(inst.TmpPath))
	_, stderr, exitCode, err := m.ssh.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		return fmt.Errorf("failed to create sandbox directories: %s", stderr)
	}
	return nil
}

func (m *RemoteRuntimeManager) bootstrapLocked(ctx context.Context, inst *SandboxInstance) error {
	if !m.cfg.BootstrapPython {
		return nil
	}
	packages := strings.TrimSpace(strings.Join(shellQuoteArgs(m.cfg.PythonPackages), " "))
	cmd := fmt.Sprintf("python3 -m venv %s", shellQuote(inst.VenvPath))
	if packages != "" {
		cmd += fmt.Sprintf(" && %s/bin/python -m pip install --upgrade pip", shellQuote(inst.VenvPath))
		cmd += fmt.Sprintf(" && %s/bin/pip install --no-cache-dir %s", shellQuote(inst.VenvPath), packages)
	}
	stdout, stderr, exitCode, err := m.ssh.RunCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to bootstrap Python sandbox: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("failed to bootstrap Python sandbox (exit %d): stdout=%s stderr=%s", exitCode, stdout, stderr)
	}
	return nil
}

func (m *RemoteRuntimeManager) inspectLocked(ctx context.Context, inst *SandboxInstance) (bool, bool, error) {
	if inst == nil || inst.WorkspacePath == "" {
		return false, false, nil
	}
	_, _, exitCode, err := m.ssh.RunCommand(ctx, fmt.Sprintf("test -d %s", shellQuote(inst.WorkspacePath)))
	if err != nil {
		return false, false, err
	}
	return exitCode == 0, false, nil
}

func (m *RemoteRuntimeManager) innerShell(command []string, inst *SandboxInstance, cfg config.SandboxConfig) string {
	inner := strings.Join(shellQuoteArgs(command), " ")
	prefix := fmt.Sprintf("cd %s", shellQuote(inst.WorkspacePath))
	if cfg.BootstrapPython {
		prefix += fmt.Sprintf(" && if [ -d %s ]; then export PATH=%s/bin:$PATH VIRTUAL_ENV=%s; fi",
			shellQuote(inst.VenvPath), shellQuote(inst.VenvPath), shellQuote(inst.VenvPath))
	}
	prefix += fmt.Sprintf(" && export HOME=%s TMPDIR=%s", shellQuote(inst.WorkspacePath), shellQuote(inst.TmpPath))
	if cfg.MemoryLimitMB > 0 {
		prefix += fmt.Sprintf(" && ulimit -v %d 2>/dev/null || true", cfg.MemoryLimitMB*1024)
	}
	return prefix + " && exec " + inner
}

func (m *RemoteRuntimeManager) bwrapCommand(inst *SandboxInstance, cfg config.SandboxConfig, inner string) string {
	args := []string{
		"bwrap", "--die-with-parent", "--new-session",
		"--unshare-pid", "--unshare-uts", "--unshare-ipc",
	}
	if !cfg.Network {
		args = append(args, "--unshare-net")
	}
	args = append(args,
		"--ro-bind", "/", "/",
		"--bind", inst.WorkspacePath, inst.WorkspacePath,
		"--bind", inst.TmpPath, inst.TmpPath,
		"--tmpfs", "/tmp",
		"--proc", "/proc",
		"--dev", "/dev",
		"--chdir", inst.WorkspacePath,
		"sh", "-lc", inner,
	)
	return strings.Join(shellQuoteArgs(args), " ")
}

func (m *RemoteRuntimeManager) seatbeltCommand(inst *SandboxInstance, cfg config.SandboxConfig, inner string) string {
	return fmt.Sprintf("sandbox-exec -f %s sh -lc %s", shellQuote(inst.ProfilePath), shellQuote(inner))
}

func (m *RemoteRuntimeManager) writeSeatbeltProfileLocked(ctx context.Context, inst *SandboxInstance) error {
	networkRule := "(allow network*)"
	if !m.cfg.Network {
		networkRule = "(deny network*)"
	}
	profile := fmt.Sprintf(`(version 1)
(deny default)
(allow process*)
(allow sysctl-read)
(allow file-read*)
(allow file-write*
  (subpath %q)
  (subpath %q))
%s
`, inst.WorkspacePath, inst.TmpPath, networkRule)
	return m.writeRemoteFileLocked(ctx, inst.ProfilePath, profile)
}

func (m *RemoteRuntimeManager) writeRemoteFileLocked(ctx context.Context, remotePath, content string) error {
	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	cmd := fmt.Sprintf("mkdir -p -- %s && printf %%s %s | base64 -d > %s",
		shellQuote(path.Dir(remotePath)), shellQuote(encoded), shellQuote(remotePath))
	_, stderr, exitCode, err := m.ssh.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		return fmt.Errorf("failed to write remote file %s: %s", remotePath, stderr)
	}
	return nil
}

func linuxInstallCommand() string {
	return "if command -v apt-get >/dev/null 2>&1; then sudo apt-get update && sudo apt-get install -y bubblewrap python3 python3-venv python3-pip; " +
		"elif command -v dnf >/dev/null 2>&1; then sudo dnf install -y bubblewrap python3 python3-pip; " +
		"elif command -v yum >/dev/null 2>&1; then sudo yum install -y bubblewrap python3 python3-pip; " +
		"elif command -v pacman >/dev/null 2>&1; then sudo pacman -Sy --noconfirm bubblewrap python python-pip; " +
		"elif command -v zypper >/dev/null 2>&1; then sudo zypper --non-interactive install bubblewrap python3 python3-pip; " +
		"else echo 'unsupported Linux package manager; install bubblewrap and python3 manually' >&2; exit 2; fi"
}

func cleanRemotePath(p string) string {
	if p == "" {
		return ""
	}
	return path.Clean(p)
}

func shellQuoteArgs(args []string) []string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		quoted[i] = shellQuote(arg)
	}
	return quoted
}

// shellQuote wraps a string in single quotes for safe POSIX shell usage.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

type diagnosticBuilder struct {
	result   SandboxDiagnosticsResult
	fixes    map[string]SandboxFixSuggestion
	fixOrder []string
}

func newDiagnosticBuilder(kind, osName, root string, cfg config.SandboxConfig) *diagnosticBuilder {
	return &diagnosticBuilder{
		result: SandboxDiagnosticsResult{
			Runtime:           kind,
			OS:                osName,
			WorkspaceRoot:     root,
			CommandTimeoutSec: cfg.CommandTimeoutSec,
			MemoryLimitMB:     cfg.MemoryLimitMB,
			NetworkEnabled:    cfg.Network,
		},
		fixes: make(map[string]SandboxFixSuggestion),
	}
}

func (b *diagnosticBuilder) addCheck(check SandboxDiagnosticCheck) {
	for _, fixID := range check.FixIDs {
		b.addKnownFix(fixID)
	}
	b.result.Checks = append(b.result.Checks, check)
}

func (b *diagnosticBuilder) addSkipped(id, label, message string, fixIDs []string) {
	b.addCheck(SandboxDiagnosticCheck{
		ID:      id,
		Label:   label,
		Status:  DiagnosticSkipped,
		Message: message,
		FixIDs:  fixIDs,
	})
}

func (b *diagnosticBuilder) addKnownFix(id string) {
	var fix SandboxFixSuggestion
	switch id {
	case "install-linux-runtime":
		fix = SandboxFixSuggestion{
			ID:           id,
			Title:        "Install Linux sandbox dependencies",
			Description:  "Installs bubblewrap plus Python runtime packages using the detected Linux package manager.",
			Risk:         "sudo",
			Platform:     "linux",
			Commands:     []string{linuxInstallCommand()},
			CopyOnly:     false,
			AutoRunnable: true,
		}
	case "enable-userns":
		fix = SandboxFixSuggestion{
			ID:          id,
			Title:       "Enable unprivileged user namespaces",
			Description: "Allows non-root users to create user namespaces, which bubblewrap needs for isolation.",
			Risk:        "security",
			Platform:    "linux",
			Commands: []string{
				"sudo sysctl -w kernel.unprivileged_userns_clone=1",
				"printf 'kernel.unprivileged_userns_clone=1\\n' | sudo tee /etc/sysctl.d/99-starxo-sandbox.conf && sudo sysctl --system",
			},
			CopyOnly:     true,
			AutoRunnable: false,
		}
	case "relax-apparmor-userns":
		fix = SandboxFixSuggestion{
			ID:          id,
			Title:       "Relax AppArmor userns restriction",
			Description: "Ubuntu AppArmor can block unprivileged user namespaces even when bubblewrap is installed. This lowers that host-level restriction.",
			Risk:        "security",
			Platform:    "linux",
			Commands: []string{
				"sudo sysctl -w kernel.apparmor_restrict_unprivileged_userns=0",
				"printf 'kernel.apparmor_restrict_unprivileged_userns=0\\n' | sudo tee /etc/sysctl.d/99-starxo-sandbox.conf && sudo sysctl --system",
			},
			CopyOnly:     true,
			AutoRunnable: false,
		}
	case "seatbelt-unavailable":
		fix = SandboxFixSuggestion{
			ID:           id,
			Title:        "Use a macOS host with sandbox-exec",
			Description:  "Seatbelt uses Apple's sandbox-exec tool. Starxo cannot install it automatically when the host does not provide it.",
			Risk:         "safe",
			Platform:     "darwin",
			CopyOnly:     true,
			AutoRunnable: false,
		}
	case "seatbelt-python":
		fix = SandboxFixSuggestion{
			ID:           id,
			Title:        "Install Python 3 on macOS",
			Description:  "Seatbelt sandboxes still need python3 for agent tooling and workspace file inspection.",
			Risk:         "safe",
			Platform:     "darwin",
			Commands:     []string{"brew install python"},
			CopyOnly:     true,
			AutoRunnable: false,
		}
	default:
		return
	}
	if _, exists := b.fixes[id]; exists {
		return
	}
	b.fixes[id] = fix
	b.fixOrder = append(b.fixOrder, id)
}

func (b *diagnosticBuilder) finish() SandboxDiagnosticsResult {
	failures := 0
	warnings := 0
	for _, check := range b.result.Checks {
		switch check.Status {
		case DiagnosticFail:
			failures++
		case DiagnosticWarn:
			warnings++
		}
	}
	b.result.Available = failures == 0
	switch {
	case failures > 0:
		b.result.Summary = fmt.Sprintf("%d blocking sandbox runtime issue(s) found", failures)
	case warnings > 0:
		b.result.Summary = fmt.Sprintf("sandbox runtime is available with %d warning(s)", warnings)
	default:
		b.result.Summary = "sandbox runtime diagnostics passed"
	}
	for _, id := range b.fixOrder {
		b.result.Fixes = append(b.result.Fixes, b.fixes[id])
	}
	return b.result
}

func diagnosticOutput(stdout, stderr string) string {
	combined := strings.TrimSpace(strings.TrimSpace(stdout) + "\n" + strings.TrimSpace(stderr))
	if len(combined) <= 4096 {
		return combined
	}
	return combined[:4096] + "\n... (truncated)"
}

func commandFailureDetails(exitCode int, err error) string {
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("exit code %d", exitCode)
}

func bwrapSmokeFixIDs(stdout, stderr string) []string {
	output := strings.ToLower(stdout + "\n" + stderr)
	if strings.Contains(output, "setting up uid map") || strings.Contains(output, "permission denied") || strings.Contains(output, "operation not permitted") {
		return []string{"enable-userns", "relax-apparmor-userns"}
	}
	return []string{"install-linux-runtime"}
}

func bwrapDiagnosticSmokeCommand(networkEnabled bool) string {
	if networkEnabled {
		return "bwrap --die-with-parent --ro-bind / / --tmpfs /tmp --proc /proc --dev /dev sh -lc 'true'"
	}
	return "bwrap --die-with-parent --unshare-net --ro-bind / / --tmpfs /tmp --proc /proc --dev /dev sh -lc 'true'"
}

func networkPolicyMessage(enabled bool) string {
	if enabled {
		return "sandbox commands may use the remote host network"
	}
	return "sandbox commands run with network namespace isolation when the runtime supports it"
}
