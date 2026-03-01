package sandbox

import (
	"context"
	"fmt"
	"io"
	"os"
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

// UploadToContainer uploads a local file into a Docker container.
// Flow: local file -> SFTP to remote host /tmp -> docker cp into container.
func (t *FileTransfer) UploadToContainer(ctx context.Context, localPath, containerPath string, docker *RemoteDockerManager) error {
	if docker.ContainerID() == "" {
		return fmt.Errorf("no container is running")
	}

	// Generate a unique temp path on the remote host
	baseName := filepath.Base(localPath)
	tmpPath := fmt.Sprintf("/tmp/eino-upload-%s-%s", docker.ContainerID()[:8], sanitizeFileName(baseName))

	// Step 1: Upload to remote host via SFTP
	if err := t.UploadFile(ctx, localPath, tmpPath); err != nil {
		return fmt.Errorf("failed to upload to remote host: %w", err)
	}

	// Step 2: docker cp from host into container
	err := docker.CopyToContainer(ctx, tmpPath, containerPath)

	// Step 3: Clean up temp file on remote host
	cleanupCmd := fmt.Sprintf("rm -f %s", shellQuote(tmpPath))
	_, _, _, _ = t.ssh.RunCommand(ctx, cleanupCmd)

	if err != nil {
		return fmt.Errorf("failed to copy into container: %w", err)
	}

	return nil
}

// DownloadFromContainer downloads a file from a Docker container to a local path.
// Flow: docker cp from container to remote host /tmp -> SFTP download to local.
func (t *FileTransfer) DownloadFromContainer(ctx context.Context, containerPath, localPath string, docker *RemoteDockerManager) error {
	if docker.ContainerID() == "" {
		return fmt.Errorf("no container is running")
	}

	// Generate a unique temp path on the remote host
	baseName := filepath.Base(containerPath)
	tmpPath := fmt.Sprintf("/tmp/eino-download-%s-%s", docker.ContainerID()[:8], sanitizeFileName(baseName))

	// Step 1: docker cp from container to remote host
	if err := docker.CopyFromContainer(ctx, containerPath, tmpPath); err != nil {
		return fmt.Errorf("failed to copy from container: %w", err)
	}

	// Step 2: Download from remote host to local via SFTP
	err := t.DownloadFile(ctx, tmpPath, localPath)

	// Step 3: Clean up temp file on remote host
	cleanupCmd := fmt.Sprintf("rm -f %s", shellQuote(tmpPath))
	_, _, _, _ = t.ssh.RunCommand(ctx, cleanupCmd)

	if err != nil {
		return fmt.Errorf("failed to download from remote host: %w", err)
	}

	return nil
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
