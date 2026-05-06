package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// FileService handles file upload and download between local machine and sandbox container.
type FileService struct {
	ctx            context.Context
	sandbox        *SandboxService
	sessionService *SessionService
}

// NewFileService creates a new FileService.
func NewFileService(sandbox *SandboxService) *FileService {
	return &FileService{
		sandbox: sandbox,
	}
}

// SetContext stores the Wails application context. Called from app.go startup.
func (s *FileService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// SetSessionService injects the session service for workspace path resolution.
func (s *FileService) SetSessionService(ss *SessionService) {
	s.sessionService = ss
}

// workspacePath returns the current workspace path from the session or a default.
func (s *FileService) workspacePath() string {
	if s.sandbox != nil {
		if mgr := s.sandbox.Manager(); mgr != nil {
			if workspace := mgr.WorkspacePath(); workspace != "" {
				return workspace
			}
		}
	}
	if s.sessionService != nil {
		return s.sessionService.GetWorkspacePath()
	}
	return "/workspace"
}

// SelectAndUploadFile opens a native file dialog, then uploads the selected file
// to the sandbox container's /workspace directory.
func (s *FileService) SelectAndUploadFile() (FileInfoDTO, error) {
	localPath, err := wailsruntime.OpenFileDialog(s.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select File to Upload",
	})
	if err != nil {
		return FileInfoDTO{}, fmt.Errorf("file dialog failed: %w", err)
	}
	if localPath == "" {
		return FileInfoDTO{}, fmt.Errorf("no file selected")
	}

	return s.UploadFile(localPath)
}

// UploadFile uploads a file at the given local path to the sandbox container.
func (s *FileService) UploadFile(localPath string) (FileInfoDTO, error) {
	mgr := s.sandbox.Manager()
	if mgr == nil || !mgr.IsConnected() {
		return FileInfoDTO{}, fmt.Errorf("sandbox is not connected")
	}

	transfer := mgr.Transfer()
	if transfer == nil {
		return FileInfoDTO{}, fmt.Errorf("file transfer is not available")
	}

	runtime := mgr.Runtime()
	if runtime == nil {
		return FileInfoDTO{}, fmt.Errorf("sandbox runtime manager is not available")
	}

	// Get file info
	info, err := os.Stat(localPath)
	if err != nil {
		return FileInfoDTO{}, fmt.Errorf("failed to stat file %s: %w", localPath, err)
	}

	baseName := filepath.Base(localPath)
	containerPath := path.Join(s.workspacePath(), baseName)

	// Upload to active sandbox workspace.
	if err := transfer.UploadToContainer(s.ctx, localPath, containerPath, runtime); err != nil {
		return FileInfoDTO{}, fmt.Errorf("upload failed: %w", err)
	}

	return FileInfoDTO{
		Name:     baseName,
		Path:     containerPath,
		Size:     info.Size(),
		IsOutput: false,
	}, nil
}

// DownloadFile opens a save dialog, then downloads a file from the sandbox container
// to the selected local path.
func (s *FileService) DownloadFile(containerPath string) error {
	mgr := s.sandbox.Manager()
	if mgr == nil || !mgr.IsConnected() {
		return fmt.Errorf("sandbox is not connected")
	}

	transfer := mgr.Transfer()
	if transfer == nil {
		return fmt.Errorf("file transfer is not available")
	}

	runtime := mgr.Runtime()
	if runtime == nil {
		return fmt.Errorf("sandbox runtime manager is not available")
	}

	baseName := filepath.Base(containerPath)

	localPath, err := wailsruntime.SaveFileDialog(s.ctx, wailsruntime.SaveDialogOptions{
		Title:           "Save File",
		DefaultFilename: baseName,
	})
	if err != nil {
		return fmt.Errorf("save dialog failed: %w", err)
	}
	if localPath == "" {
		return fmt.Errorf("no save location selected")
	}

	if err := transfer.DownloadFromContainer(s.ctx, containerPath, localPath, runtime); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	return nil
}

// ListWorkspaceFiles lists files in the container's /workspace directory up to 3 levels deep.
func (s *FileService) ListWorkspaceFiles() ([]FileInfoDTO, error) {
	mgr := s.sandbox.Manager()
	if mgr == nil || !mgr.IsConnected() {
		return nil, fmt.Errorf("sandbox is not connected")
	}

	op := mgr.Operator()
	if op == nil {
		return nil, fmt.Errorf("sandbox operator is not available")
	}

	script := `import json, os, time
root = os.getcwd()
max_depth = 3
items = []
for dirpath, dirnames, filenames in os.walk(root):
    rel_dir = os.path.relpath(dirpath, root)
    depth = 0 if rel_dir == "." else rel_dir.count(os.sep) + 1
    if depth >= max_depth:
        dirnames[:] = []
        continue
    if depth >= max_depth - 1:
        dirnames[:] = []
    for name in filenames:
        full = os.path.join(dirpath, name)
        try:
            st = os.stat(full)
        except OSError:
            continue
        rel = os.path.relpath(full, root).replace(os.sep, "/")
        items.append({
            "name": name,
            "path": os.path.join(root, rel).replace(os.sep, "/"),
            "size": st.st_size,
            "modified": time.strftime("%Y-%m-%dT%H:%M:%S%z", time.localtime(st.st_mtime)),
        })
items.sort(key=lambda item: item["path"])
print(json.dumps(items))`
	output, err := op.RunCommand(s.ctx, []string{"python3", "-c", script})
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	if output.ExitCode != 0 {
		return nil, fmt.Errorf("failed to list files (exit %d): %s", output.ExitCode, output.Stderr)
	}

	var files []FileInfoDTO
	if stdout := strings.TrimSpace(output.Stdout); stdout != "" {
		if err := json.Unmarshal([]byte(stdout), &files); err != nil {
			return nil, fmt.Errorf("failed to parse workspace file list: %w", err)
		}
	}
	sort.SliceStable(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
	return files, nil
}

func (s *FileService) GetWorkspaceInfo() (WorkspaceInfoDTO, error) {
	info := WorkspaceInfoDTO{RefreshedAt: time.Now().UnixMilli()}
	mgr := s.sandbox.Manager()
	if mgr == nil {
		return info, nil
	}

	host, port := mgr.SSHHostPort()
	info.SSHConnected = mgr.SSHConnected()
	info.SSHHost = host
	info.SSHPort = port

	runtime := mgr.Runtime()
	if runtime == nil {
		return info, nil
	}
	info.Active = runtime.IsActive()
	info.SandboxID = runtime.RuntimeID()
	info.SandboxName = runtime.RuntimeName()
	info.Runtime = runtime.RuntimeKind()
	info.WorkspacePath = runtime.WorkspacePath()
	if !info.Active {
		return info, nil
	}

	op := mgr.Operator()
	if op == nil {
		return info, nil
	}
	output, err := op.RunCommand(s.ctx, []string{"sh", "-lc", "files=$(find . -type f 2>/dev/null | wc -l | tr -d ' '); bytes=$(du -sk . 2>/dev/null | awk '{printf \"%d\", $1 * 1024}'); printf '%s %s\\n' \"${files:-0}\" \"${bytes:-0}\""})
	if err != nil {
		return info, fmt.Errorf("failed to inspect workspace: %w", err)
	}
	if output.ExitCode != 0 {
		return info, fmt.Errorf("failed to inspect workspace (exit %d): %s", output.ExitCode, output.Stderr)
	}
	fields := strings.Fields(strings.TrimSpace(output.Stdout))
	if len(fields) >= 1 {
		_, _ = fmt.Sscanf(fields[0], "%d", &info.FileCount)
	}
	if len(fields) >= 2 {
		_, _ = fmt.Sscanf(fields[1], "%d", &info.TotalSize)
	}
	return info, nil
}

func (s *FileService) CleanupSandboxTmp() (WorkspaceCleanupResultDTO, error) {
	mgr := s.sandbox.Manager()
	if mgr == nil || !mgr.SSHConnected() {
		return WorkspaceCleanupResultDTO{}, fmt.Errorf("sandbox is not connected")
	}
	runtime := mgr.Runtime()
	if runtime == nil || !runtime.IsActive() {
		return WorkspaceCleanupResultDTO{}, fmt.Errorf("no sandbox is active")
	}
	result, err := runtime.CleanupTmp(s.ctx)
	if err != nil {
		return WorkspaceCleanupResultDTO{}, err
	}
	return WorkspaceCleanupResultDTO{
		TmpPath:        result.TmpPath,
		RemovedEntries: result.RemovedEntries,
		ReclaimedBytes: result.ReclaimedBytes,
	}, nil
}

// ReadFilePreview reads the first N bytes of a file in the container for preview.
func (s *FileService) ReadFilePreview(containerPath string) (string, error) {
	mgr := s.sandbox.Manager()
	if mgr == nil || !mgr.IsConnected() {
		return "", fmt.Errorf("sandbox is not connected")
	}

	op := mgr.Operator()
	if op == nil {
		return "", fmt.Errorf("sandbox operator is not available")
	}

	content, err := op.ReadFile(s.ctx, containerPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Limit preview to 4KB
	if len(content) > 4096 {
		content = content[:4096] + "\n... (truncated)"
	}

	return content, nil
}
