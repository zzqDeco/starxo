package sandbox

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pkg/sftp"
)

// FileTransfer provides file transfer capabilities using SFTP over SSH.
type FileTransfer struct {
	ssh *SSHClient
}

type RemoteFileInfo struct {
	Name     string
	Path     string
	Size     int64
	Modified string
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

func (t *FileTransfer) ListFiles(ctx context.Context, root string, maxDepth int) ([]RemoteFileInfo, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if maxDepth <= 0 {
		maxDepth = 3
	}
	root = cleanRemotePath(root)
	if root == "" || root == "/" || root == "." {
		return nil, fmt.Errorf("refusing to list unsafe remote path %q", root)
	}

	sftpClient, err := t.newSFTP()
	if err != nil {
		return nil, err
	}
	defer sftpClient.Close()

	var files []RemoteFileInfo
	var walk func(dir, rel string, depth int) error
	walk = func(dir, rel string, depth int) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if depth >= maxDepth {
			return nil
		}
		entries, err := sftpClient.ReadDir(dir)
		if err != nil {
			if rel == "" {
				return fmt.Errorf("failed to list remote directory %s: %w", dir, err)
			}
			return nil
		}
		sort.SliceStable(entries, func(i, j int) bool {
			if entries[i].IsDir() != entries[j].IsDir() {
				return entries[i].IsDir()
			}
			return entries[i].Name() < entries[j].Name()
		})
		for _, entry := range entries {
			name := entry.Name()
			if name == "." || name == ".." {
				continue
			}
			childRel := name
			if rel != "" {
				childRel = path.Join(rel, name)
			}
			childPath := path.Join(root, childRel)
			if entry.IsDir() {
				if depth+1 < maxDepth {
					if err := walk(childPath, childRel, depth+1); err != nil {
						return err
					}
				}
				continue
			}
			files = append(files, RemoteFileInfo{
				Name:     name,
				Path:     childPath,
				Size:     entry.Size(),
				Modified: entry.ModTime().Format(time.RFC3339),
			})
		}
		return nil
	}
	if err := walk(root, "", 0); err != nil {
		return nil, err
	}
	sort.SliceStable(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
	return files, nil
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
