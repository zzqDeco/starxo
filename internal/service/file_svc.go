package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

	docker := mgr.Docker()
	if docker == nil {
		return FileInfoDTO{}, fmt.Errorf("docker manager is not available")
	}

	// Get file info
	info, err := os.Stat(localPath)
	if err != nil {
		return FileInfoDTO{}, fmt.Errorf("failed to stat file %s: %w", localPath, err)
	}

	baseName := filepath.Base(localPath)
	containerPath := s.workspacePath() + "/" + baseName

	// Upload to container
	if err := transfer.UploadToContainer(s.ctx, localPath, containerPath, docker); err != nil {
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

	docker := mgr.Docker()
	if docker == nil {
		return fmt.Errorf("docker manager is not available")
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

	if err := transfer.DownloadFromContainer(s.ctx, containerPath, localPath, docker); err != nil {
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

	// Use stat to get file size along with path
	output, err := op.RunCommand(s.ctx, []string{
		"find", s.workspacePath(), "-maxdepth", "3", "-type", "f",
		"-exec", "stat", "-c", "%s %n", "{}", ";",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output.Stdout), "\n")
	var files []FileInfoDTO
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Parse "size path" format from stat output
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		var size int64
		fmt.Sscanf(parts[0], "%d", &size)
		filePath := parts[1]
		files = append(files, FileInfoDTO{
			Name: filepath.Base(filePath),
			Path: filePath,
			Size: size,
		})
	}

	return files, nil
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
