package sandbox

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/sftp"
)

// FileTransfer provides file transfer capabilities using SFTP over SSH.
type FileTransfer struct {
	ssh *SSHClient
}

// NewFileTransfer creates a new FileTransfer backed by the given SSH client.
func NewFileTransfer(ssh *SSHClient) *FileTransfer {
	return &FileTransfer{ssh: ssh}
}

// UploadFile uploads a local file to the remote host via SFTP.
func (t *FileTransfer) UploadFile(ctx context.Context, localPath, remotePath string) error {
	sftpClient, err := t.newSFTP()
	if err != nil {
		return err
	}
	defer sftpClient.Close()

	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file %s: %w", localPath, err)
	}
	defer localFile.Close()

	// Ensure remote directory exists
	remoteDir := filepath.Dir(remotePath)
	if remoteDir != "" && remoteDir != "." && remoteDir != "/" {
		_ = sftpClient.MkdirAll(remoteDir)
	}

	remoteFile, err := sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("failed to create remote file %s: %w", remotePath, err)
	}
	defer remoteFile.Close()

	if _, err := io.Copy(remoteFile, localFile); err != nil {
		return fmt.Errorf("failed to upload file to %s: %w", remotePath, err)
	}

	return nil
}

// DownloadFile downloads a file from the remote host to a local path via SFTP.
func (t *FileTransfer) DownloadFile(ctx context.Context, remotePath, localPath string) error {
	sftpClient, err := t.newSFTP()
	if err != nil {
		return err
	}
	defer sftpClient.Close()

	remoteFile, err := sftpClient.Open(remotePath)
	if err != nil {
		return fmt.Errorf("failed to open remote file %s: %w", remotePath, err)
	}
	defer remoteFile.Close()

	// Ensure local directory exists
	localDir := filepath.Dir(localPath)
	if localDir != "" && localDir != "." {
		if err := os.MkdirAll(localDir, 0755); err != nil {
			return fmt.Errorf("failed to create local directory %s: %w", localDir, err)
		}
	}

	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file %s: %w", localPath, err)
	}
	defer localFile.Close()

	if _, err := io.Copy(localFile, remoteFile); err != nil {
		return fmt.Errorf("failed to download file from %s: %w", remotePath, err)
	}

	return nil
}

// UploadToContainer uploads a local file into the active sandbox workspace.
// The name is kept for Wails/service compatibility with the previous Docker implementation.
func (t *FileTransfer) UploadToContainer(ctx context.Context, localPath, containerPath string, runtime *RemoteRuntimeManager) error {
	remotePath, err := workspaceTransferPath(containerPath, runtime)
	if err != nil {
		return err
	}
	return t.UploadFile(ctx, localPath, remotePath)
}

// DownloadFromContainer downloads a file from the active sandbox workspace.
func (t *FileTransfer) DownloadFromContainer(ctx context.Context, containerPath, localPath string, runtime *RemoteRuntimeManager) error {
	remotePath, err := workspaceTransferPath(containerPath, runtime)
	if err != nil {
		return err
	}
	return t.DownloadFile(ctx, remotePath, localPath)
}

// newSFTP creates a new SFTP client from the SSH connection.
func (t *FileTransfer) newSFTP() (*sftp.Client, error) {
	sshClient := t.ssh.GetClient()
	if sshClient == nil {
		return nil, fmt.Errorf("SSH client is not connected")
	}
	client, err := sftp.NewClient(sshClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create SFTP client: %w", err)
	}
	return client, nil
}

// sanitizeFileName removes path separators and other unsafe characters from a file name.
func sanitizeFileName(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, " ", "_")
	return name
}

func workspaceTransferPath(filePath string, runtime *RemoteRuntimeManager) (string, error) {
	if runtime == nil || !runtime.IsActive() {
		return "", fmt.Errorf("no sandbox is active")
	}
	workspace := cleanRemotePath(runtime.WorkspacePath())
	if workspace == "" {
		return "", fmt.Errorf("sandbox workspace is not available")
	}
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
