package sandbox

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/cloudwego/eino-ext/components/tool/commandline"
)

// RemoteOperator implements commandline.Operator over SSH+Docker.
// It delegates all file and command operations to the remote container.
type RemoteOperator struct {
	docker   *RemoteDockerManager
	onOutput func(stdout, stderr string, exitCode int) // optional callback for terminal output
}

// Compile-time check that RemoteOperator implements commandline.Operator.
var _ commandline.Operator = (*RemoteOperator)(nil)

// NewRemoteOperator creates a new RemoteOperator backed by the given Docker manager.
func NewRemoteOperator(docker *RemoteDockerManager) *RemoteOperator {
	return &RemoteOperator{docker: docker}
}

// SetOnOutput sets a callback that is invoked whenever a command produces output.
// This is used to forward command output to the frontend terminal.
func (o *RemoteOperator) SetOnOutput(fn func(stdout, stderr string, exitCode int)) {
	o.onOutput = fn
}

// ReadFile reads the contents of a file at the given path inside the container.
func (o *RemoteOperator) ReadFile(ctx context.Context, path string) (string, error) {
	stdout, stderr, exitCode, err := o.docker.ExecInContainer(ctx, []string{"cat", path})
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", path, err)
	}
	if exitCode != 0 {
		return "", fmt.Errorf("failed to read file %s (exit %d): %s", path, exitCode, stderr)
	}
	return stdout, nil
}

// WriteFile writes content to a file at the given path inside the container.
// For small content, it uses base64 encoding piped through docker exec.
// For large content (>64KB), it uses SFTP to write a temp file on the host,
// then docker cp into the container.
func (o *RemoteOperator) WriteFile(ctx context.Context, path string, content string) error {
	const largeThreshold = 64 * 1024

	if len(content) > largeThreshold {
		return o.writeFileLarge(ctx, path, content)
	}

	return o.writeFileSmall(ctx, path, content)
}

// writeFileSmall writes small files using base64-encoded content piped through docker exec.
func (o *RemoteOperator) writeFileSmall(ctx context.Context, path string, content string) error {
	// Use base64 encoding to safely transport content with special characters
	encoded := base64.StdEncoding.EncodeToString([]byte(content))

	cid := o.docker.ContainerID()
	if cid == "" {
		return fmt.Errorf("no container is running")
	}

	// Ensure parent directory exists
	dir := parentDir(path)
	if dir != "" && dir != "." && dir != "/" {
		mkdirCmd := fmt.Sprintf("%s exec %s mkdir -p %s", o.docker.dockerCmd(), cid, shellQuote(dir))
		_, _, _, _ = o.docker.ssh.RunCommand(ctx, mkdirCmd)
	}

	// Write via base64 decode
	cmd := fmt.Sprintf("echo %s | %s exec -i %s sh -c 'base64 -d > %s'",
		shellQuote(encoded), o.docker.dockerCmd(), cid, shellQuote(path))
	_, stderr, exitCode, err := o.docker.ssh.RunCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}
	if exitCode != 0 {
		return fmt.Errorf("failed to write file %s (exit %d): %s", path, exitCode, stderr)
	}

	return nil
}

// writeFileLarge writes large files by SFTP-ing to a temp file on the host,
// then using docker cp to copy into the container.
func (o *RemoteOperator) writeFileLarge(ctx context.Context, path string, content string) error {
	cid := o.docker.ContainerID()
	if cid == "" {
		return fmt.Errorf("no container is running")
	}

	// Write content to a temp file on the remote host via base64
	tmpPath := fmt.Sprintf("/tmp/eino-upload-%s", strings.ReplaceAll(path, "/", "_"))
	encoded := base64.StdEncoding.EncodeToString([]byte(content))

	// Split large base64 into chunks and write via shell
	writeCmd := fmt.Sprintf("echo %s | base64 -d > %s", shellQuote(encoded), shellQuote(tmpPath))
	_, stderr, exitCode, err := o.docker.ssh.RunCommand(ctx, writeCmd)
	if err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("failed to write temp file (exit %d): %s", exitCode, stderr)
	}

	// Ensure parent directory exists in container
	dir := parentDir(path)
	if dir != "" && dir != "." && dir != "/" {
		mkdirCmd := fmt.Sprintf("%s exec %s mkdir -p %s", o.docker.dockerCmd(), cid, shellQuote(dir))
		_, _, _, _ = o.docker.ssh.RunCommand(ctx, mkdirCmd)
	}

	// docker cp from host to container
	err = o.docker.CopyToContainer(ctx, tmpPath, path)

	// Clean up temp file
	cleanupCmd := fmt.Sprintf("rm -f %s", shellQuote(tmpPath))
	_, _, _, _ = o.docker.ssh.RunCommand(ctx, cleanupCmd)

	if err != nil {
		return fmt.Errorf("failed to copy file into container: %w", err)
	}

	return nil
}

// IsDirectory checks whether the given path is a directory inside the container.
func (o *RemoteOperator) IsDirectory(ctx context.Context, path string) (bool, error) {
	stdout, _, exitCode, err := o.docker.ExecInContainer(ctx, []string{"test", "-d", path, "&&", "echo", "true", "||", "echo", "false"})
	if err != nil {
		return false, fmt.Errorf("failed to check directory %s: %w", path, err)
	}

	// test -d returns exit code 0 if directory, 1 if not
	// But since we chain with && and ||, parse stdout instead
	if exitCode != 0 {
		// Fallback: run a simpler check
		_, _, code, err2 := o.docker.ExecInContainer(ctx, []string{"test", "-d", path})
		if err2 != nil {
			return false, fmt.Errorf("failed to check directory %s: %w", path, err2)
		}
		return code == 0, nil
	}

	return strings.TrimSpace(stdout) == "true", nil
}

// Exists checks whether a file or directory exists at the given path inside the container.
func (o *RemoteOperator) Exists(ctx context.Context, path string) (bool, error) {
	_, _, exitCode, err := o.docker.ExecInContainer(ctx, []string{"test", "-e", path})
	if err != nil {
		return false, fmt.Errorf("failed to check existence of %s: %w", path, err)
	}
	return exitCode == 0, nil
}

// RunCommand executes a command inside the container and returns a CommandOutput.
func (o *RemoteOperator) RunCommand(ctx context.Context, command []string) (*commandline.CommandOutput, error) {
	stdout, stderr, exitCode, err := o.docker.ExecInContainer(ctx, command)
	if err != nil {
		return nil, fmt.Errorf("failed to execute command: %w", err)
	}

	// Forward output to terminal callback if set
	if o.onOutput != nil {
		o.onOutput(stdout, stderr, exitCode)
	}

	return &commandline.CommandOutput{
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: exitCode,
	}, nil
}

// parentDir returns the parent directory of the given path.
func parentDir(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx <= 0 {
		return "/"
	}
	return path[:idx]
}
