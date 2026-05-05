package sandbox

import (
	"context"
	"encoding/base64"
	"fmt"
	"path"
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

type RemoteRuntimeManager struct {
	ssh      *SSHClient
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

func (m *RemoteRuntimeManager) IsActive() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.instance != nil
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
