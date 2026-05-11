package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino-ext/components/tool/commandline"

	"starxo/internal/tools"
)

type RuntimeWorktreeStateDTO struct {
	SessionID      string `json:"sessionId"`
	Active         bool   `json:"active"`
	WorkspacePath  string `json:"workspacePath,omitempty"`
	CurrentPath    string `json:"currentPath,omitempty"`
	WorktreePath   string `json:"worktreePath,omitempty"`
	WorktreeBranch string `json:"worktreeBranch,omitempty"`
	Message        string `json:"message"`
}

func (s *ChatService) GetRuntimeWorktreeState(sessionID string) (RuntimeWorktreeStateDTO, error) {
	sessionID = s.runtimeWorktreeSessionID(sessionID)
	defaultWorkspace := s.getWorkspacePath()
	out := RuntimeWorktreeStateDTO{
		SessionID:     sessionID,
		WorkspacePath: defaultWorkspace,
		CurrentPath:   defaultWorkspace,
		Message:       "No active runtime worktree for this session.",
	}
	if sessionID == "" {
		out.Message = "No active session."
		return out, nil
	}
	if s.runtimeWorkspaces == nil {
		return out, nil
	}
	state, ok := s.runtimeWorkspaces.activeState(contextWithSessionID(context.Background(), sessionID))
	if !ok {
		return out, nil
	}
	out.Active = true
	out.WorkspacePath = state.OriginalWorkspace
	out.CurrentPath = state.WorktreePath
	out.WorktreePath = state.WorktreePath
	out.WorktreeBranch = state.WorktreeBranch
	out.Message = fmt.Sprintf("Session is using worktree %s on branch %s.", state.WorktreePath, state.WorktreeBranch)
	return out, nil
}

func (s *ChatService) ReviewRuntimeWorktree(sessionID string, includePatch bool, maxBytes int) (tools.WorktreeDiffOutput, error) {
	ctx, sid, err := s.runtimeWorktreeContext(sessionID)
	if err != nil {
		return tools.WorktreeDiffOutput{}, err
	}
	op, err := s.runtimeWorktreeOperator()
	if err != nil {
		return tools.WorktreeDiffOutput{}, err
	}
	out, err := s.runtimeWorkspaces.DiffWorktree(ctx, op, s.getWorkspacePath(), includePatch, maxBytes)
	if err == nil {
		wailsEmit(s.ctx, "runtime:worktree_reviewed", map[string]string{"sessionId": sid})
	}
	return out, err
}

func (s *ChatService) MergeRuntimeWorktree(sessionID string, commitMessage string, removeWorktree bool) (tools.WorktreeMergeOutput, error) {
	ctx, sid, err := s.runtimeWorktreeContext(sessionID)
	if err != nil {
		return tools.WorktreeMergeOutput{}, err
	}
	op, err := s.runtimeWorktreeOperator()
	if err != nil {
		return tools.WorktreeMergeOutput{}, err
	}
	out, err := s.runtimeWorkspaces.MergeWorktree(ctx, op, s.getWorkspacePath(), commitMessage, removeWorktree)
	if err == nil {
		wailsEmit(s.ctx, "runtime:worktree_changed", map[string]string{"sessionId": sid, "action": "merge"})
	}
	return out, err
}

func (s *ChatService) ExitRuntimeWorktree(sessionID string, action string, discardChanges bool) (tools.WorktreeOutput, error) {
	ctx, sid, err := s.runtimeWorktreeContext(sessionID)
	if err != nil {
		return tools.WorktreeOutput{}, err
	}
	op, err := s.runtimeWorktreeOperator()
	if err != nil {
		return tools.WorktreeOutput{}, err
	}
	out, err := s.runtimeWorkspaces.ExitWorktree(ctx, op, s.getWorkspacePath(), action, discardChanges)
	if err == nil {
		wailsEmit(s.ctx, "runtime:worktree_changed", map[string]string{"sessionId": sid, "action": out.Action})
	}
	return out, err
}

func (s *ChatService) runtimeWorktreeSessionID(sessionID string) string {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID != "" {
		return sessionID
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activeSessionID
}

func (s *ChatService) runtimeWorktreeContext(sessionID string) (context.Context, string, error) {
	sessionID = s.runtimeWorktreeSessionID(sessionID)
	if sessionID == "" {
		return nil, "", fmt.Errorf("no active session")
	}
	if s.runtimeWorkspaces == nil {
		return nil, "", fmt.Errorf("runtime worktree manager is not available")
	}
	ctx := contextWithSessionID(context.Background(), sessionID)
	if _, ok := s.runtimeWorkspaces.activeState(ctx); !ok {
		return nil, "", fmt.Errorf("no active worktree for this session")
	}
	return ctx, sessionID, nil
}

func (s *ChatService) runtimeWorktreeOperator() (commandline.Operator, error) {
	if s.sandbox == nil || !s.sandbox.IsConnected() {
		return nil, fmt.Errorf("sandbox is not connected")
	}
	op := s.sandbox.Operator()
	if op == nil {
		return nil, fmt.Errorf("sandbox operator is not available")
	}
	return op, nil
}
