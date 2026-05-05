package sandbox

import (
	"context"
	"encoding/base64"
	"fmt"
	"path"
	"strings"

	"github.com/cloudwego/eino-ext/components/tool/commandline"
)

// RemoteOperator implements commandline.Operator over SSH plus a lightweight
// bwrap/seatbelt runtime. File writes are restricted to the active workspace.
type RemoteOperator struct {
	runtime  *RemoteRuntimeManager
	onOutput func(stdout, stderr string, exitCode int)
}

var _ commandline.Operator = (*RemoteOperator)(nil)

func NewRemoteOperator(runtime *RemoteRuntimeManager) *RemoteOperator {
	return &RemoteOperator{runtime: runtime}
}

func (o *RemoteOperator) SetOnOutput(fn func(stdout, stderr string, exitCode int)) {
	o.onOutput = fn
}

func (o *RemoteOperator) ReadFile(ctx context.Context, filePath string) (string, error) {
	target, err := o.workspacePath(filePath)
	if err != nil {
		return "", err
	}
	stdout, stderr, exitCode, err := o.runtime.ExecInSandbox(ctx, []string{"cat", target})
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	if exitCode != 0 {
		return "", fmt.Errorf("failed to read file %s (exit %d): %s", filePath, exitCode, stderr)
	}
	return stdout, nil
}

func (o *RemoteOperator) WriteFile(ctx context.Context, filePath string, content string) error {
	target, err := o.workspacePath(filePath)
	if err != nil {
		return err
	}
	dir := parentDir(target)
	if dir != "" && dir != "." && dir != "/" {
		if _, stderr, exitCode, err := o.runtime.ExecInSandbox(ctx, []string{"mkdir", "-p", dir}); err != nil {
			return fmt.Errorf("failed to create parent directory %s: %w", dir, err)
		} else if exitCode != 0 {
			return fmt.Errorf("failed to create parent directory %s (exit %d): %s", dir, exitCode, stderr)
		}
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	cmd := fmt.Sprintf("printf %%s %s | base64 -d > %s", shellQuote(encoded), shellQuote(target))
	_, stderr, exitCode, err := o.runtime.ExecInSandbox(ctx, []string{"sh", "-lc", cmd})
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}
	if exitCode != 0 {
		return fmt.Errorf("failed to write file %s (exit %d): %s", filePath, exitCode, stderr)
	}
	return nil
}

func (o *RemoteOperator) IsDirectory(ctx context.Context, filePath string) (bool, error) {
	target, err := o.workspacePath(filePath)
	if err != nil {
		return false, err
	}
	_, _, exitCode, err := o.runtime.ExecInSandbox(ctx, []string{"test", "-d", target})
	if err != nil {
		return false, fmt.Errorf("failed to check directory %s: %w", filePath, err)
	}
	return exitCode == 0, nil
}

func (o *RemoteOperator) Exists(ctx context.Context, filePath string) (bool, error) {
	target, err := o.workspacePath(filePath)
	if err != nil {
		return false, err
	}
	_, _, exitCode, err := o.runtime.ExecInSandbox(ctx, []string{"test", "-e", target})
	if err != nil {
		return false, fmt.Errorf("failed to check existence of %s: %w", filePath, err)
	}
	return exitCode == 0, nil
}

func (o *RemoteOperator) RunCommand(ctx context.Context, command []string) (*commandline.CommandOutput, error) {
	stdout, stderr, exitCode, err := o.runtime.ExecInSandbox(ctx, command)
	if err != nil {
		return nil, fmt.Errorf("failed to execute command: %w", err)
	}
	if o.onOutput != nil {
		o.onOutput(stdout, stderr, exitCode)
	}
	return &commandline.CommandOutput{Stdout: stdout, Stderr: stderr, ExitCode: exitCode}, nil
}

func (o *RemoteOperator) workspacePath(filePath string) (string, error) {
	workspace := o.runtime.WorkspacePath()
	if workspace == "" {
		return "", fmt.Errorf("sandbox workspace is not active")
	}
	workspace = cleanRemotePath(workspace)

	p := strings.TrimSpace(filePath)
	if p == "" {
		return "", fmt.Errorf("path is empty")
	}
	if p == "/workspace" {
		p = workspace
	} else if strings.HasPrefix(p, "/workspace/") {
		p = path.Join(workspace, strings.TrimPrefix(p, "/workspace/"))
	} else if !strings.HasPrefix(p, "/") {
		p = path.Join(workspace, p)
	}
	p = cleanRemotePath(p)
	if p != workspace && !strings.HasPrefix(p, workspace+"/") {
		return "", fmt.Errorf("path %s is outside sandbox workspace %s", filePath, workspace)
	}
	return p, nil
}

func parentDir(filePath string) string {
	idx := strings.LastIndex(filePath, "/")
	if idx <= 0 {
		return "/"
	}
	return filePath[:idx]
}
