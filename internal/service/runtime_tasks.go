package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"starxo/internal/config"
	"starxo/internal/model"
	"starxo/internal/tools"
)

const (
	runtimeTaskStatusRunning   = "running"
	runtimeTaskStatusCompleted = "completed"
	runtimeTaskStatusFailed    = "failed"
	runtimeTaskStatusKilled    = "killed"
	defaultTaskOutputLimit     = 64 * 1024
)

type runtimeTask struct {
	snapshot tools.RuntimeTaskSnapshot
	cancel   context.CancelFunc
}

type runtimeTaskManager struct {
	mu        sync.RWMutex
	tasks     map[string]*runtimeTask
	taskItems map[string]tools.RuntimeTaskItem
	now       func() time.Time
	emit      func(event string, data any)
}

func newRuntimeTaskManager(now func() time.Time, emit func(event string, data any)) *runtimeTaskManager {
	if now == nil {
		now = time.Now
	}
	return &runtimeTaskManager{
		tasks:     make(map[string]*runtimeTask),
		taskItems: make(map[string]tools.RuntimeTaskItem),
		now:       now,
		emit:      emit,
	}
}

func (m *runtimeTaskManager) StartShellTask(ctx context.Context, sessionID, command, description string, runner tools.RuntimeTaskRunner) (tools.RuntimeTaskRef, error) {
	if runner == nil {
		return tools.RuntimeTaskRef{}, fmt.Errorf("runtime task runner is nil")
	}
	if strings.TrimSpace(sessionID) == "" {
		sessionID = "global"
	}
	taskID := fmt.Sprintf("task-%d", m.now().UnixNano())
	outputPath, err := runtimeTaskOutputPath(sessionID, taskID)
	if err != nil {
		return tools.RuntimeTaskRef{}, err
	}
	taskCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	snapshot := tools.RuntimeTaskSnapshot{
		ID:          taskID,
		SessionID:   sessionID,
		Type:        "bash",
		Status:      runtimeTaskStatusRunning,
		Description: description,
		Command:     command,
		OutputPath:  outputPath,
		StartedAt:   m.now().UnixMilli(),
	}
	m.mu.Lock()
	m.tasks[taskID] = &runtimeTask{snapshot: snapshot, cancel: cancel}
	m.mu.Unlock()
	m.emitEvent("runtime:task_started", snapshot)

	go func() {
		out, runErr := runner(taskCtx)
		content := formatBashTaskOutput(out, runErr)
		_ = os.MkdirAll(filepath.Dir(outputPath), 0755)
		_ = os.WriteFile(outputPath, []byte(content), 0644)

		m.mu.Lock()
		task := m.tasks[taskID]
		if task != nil {
			finishedAt := m.now().UnixMilli()
			task.snapshot.FinishedAt = finishedAt
			task.snapshot.DurationMs = maxRuntimeTaskDuration(0, finishedAt-task.snapshot.StartedAt)
			task.snapshot.OutputSize = int64(len(content))
			task.snapshot.ExitCode = out.ExitCode
			if task.snapshot.Status == runtimeTaskStatusKilled {
				// Keep explicit stop status even if the process reports a later error.
			} else if runErr != nil {
				task.snapshot.Status = runtimeTaskStatusFailed
				task.snapshot.Error = runErr.Error()
			} else if out.Interrupted {
				task.snapshot.Status = runtimeTaskStatusKilled
			} else if out.ExitCode != 0 {
				task.snapshot.Status = runtimeTaskStatusFailed
			} else {
				task.snapshot.Status = runtimeTaskStatusCompleted
			}
			task.cancel = nil
			snapshot = task.snapshot
		}
		m.mu.Unlock()
		m.emitEvent("runtime:task_completed", snapshot)
	}()

	return tools.RuntimeTaskRef{
		TaskID:     taskID,
		Status:     runtimeTaskStatusRunning,
		OutputPath: outputPath,
	}, nil
}

func (m *runtimeTaskManager) StartAgentTask(ctx context.Context, sessionID, description string, runner func(ctx context.Context) (string, error)) (tools.RuntimeTaskRef, error) {
	if runner == nil {
		return tools.RuntimeTaskRef{}, fmt.Errorf("runtime agent task runner is nil")
	}
	if strings.TrimSpace(sessionID) == "" {
		sessionID = "global"
	}
	taskID := fmt.Sprintf("task-%d", m.now().UnixNano())
	outputPath, err := runtimeTaskOutputPath(sessionID, taskID)
	if err != nil {
		return tools.RuntimeTaskRef{}, err
	}
	taskCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	snapshot := tools.RuntimeTaskSnapshot{
		ID:          taskID,
		SessionID:   sessionID,
		Type:        "agent",
		Status:      runtimeTaskStatusRunning,
		Description: description,
		Command:     "Agent",
		OutputPath:  outputPath,
		StartedAt:   m.now().UnixMilli(),
	}
	m.mu.Lock()
	m.tasks[taskID] = &runtimeTask{snapshot: snapshot, cancel: cancel}
	m.mu.Unlock()
	m.emitEvent("runtime:task_started", snapshot)

	go func() {
		content, runErr := runner(taskCtx)
		if runErr != nil {
			content = strings.TrimRight(content, "\n") + "\nerror: " + runErr.Error() + "\n"
		}
		_ = os.MkdirAll(filepath.Dir(outputPath), 0755)
		_ = os.WriteFile(outputPath, []byte(content), 0644)

		m.mu.Lock()
		task := m.tasks[taskID]
		if task != nil {
			finishedAt := m.now().UnixMilli()
			task.snapshot.FinishedAt = finishedAt
			task.snapshot.DurationMs = maxRuntimeTaskDuration(0, finishedAt-task.snapshot.StartedAt)
			task.snapshot.OutputSize = int64(len(content))
			if task.snapshot.Status == runtimeTaskStatusKilled {
				// Keep explicit stop status.
			} else if runErr != nil {
				task.snapshot.Status = runtimeTaskStatusFailed
				task.snapshot.Error = runErr.Error()
				task.snapshot.ExitCode = 1
			} else {
				task.snapshot.Status = runtimeTaskStatusCompleted
			}
			task.cancel = nil
			snapshot = task.snapshot
		}
		m.mu.Unlock()
		m.emitEvent("runtime:task_completed", snapshot)
	}()

	return tools.RuntimeTaskRef{
		TaskID:     taskID,
		Status:     runtimeTaskStatusRunning,
		OutputPath: outputPath,
	}, nil
}

func (m *runtimeTaskManager) CreateTaskItem(ctx context.Context, sessionID string, input tools.TaskCreateInput) (tools.RuntimeTaskItem, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return tools.RuntimeTaskItem{}, fmt.Errorf("title is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		sessionID = "global"
	}
	status, err := normalizeRuntimeTaskItemStatus(input.Status, runtimeTaskItemStatusTodo)
	if err != nil {
		return tools.RuntimeTaskItem{}, err
	}
	now := m.now().UnixMilli()
	item := tools.RuntimeTaskItem{
		ID:          fmt.Sprintf("taskitem-%d", m.now().UnixNano()),
		SessionID:   sessionID,
		Title:       title,
		Description: strings.TrimSpace(input.Description),
		Status:      status,
		Owner:       strings.TrimSpace(input.Owner),
		Priority:    strings.TrimSpace(input.Priority),
		DependsOn:   compactRuntimeTaskDependsOn(input.DependsOn),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if runtimeTaskItemStatusClosed(status) {
		item.CompletedAt = now
	}
	m.mu.Lock()
	m.taskItems[item.ID] = item
	m.mu.Unlock()
	m.emitEvent("runtime:task_graph_changed", map[string]any{
		"action":    "created",
		"sessionId": sessionID,
		"task":      item,
	})
	return item, nil
}

func (m *runtimeTaskManager) GetTaskItem(ctx context.Context, sessionID, taskID string) (tools.RuntimeTaskItem, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return tools.RuntimeTaskItem{}, fmt.Errorf("task_id is required")
	}
	m.mu.RLock()
	item, ok := m.taskItems[taskID]
	m.mu.RUnlock()
	if !ok || (sessionID != "" && item.SessionID != sessionID) {
		return tools.RuntimeTaskItem{}, fmt.Errorf("task item %s not found", taskID)
	}
	return cloneRuntimeTaskItem(item), nil
}

func (m *runtimeTaskManager) UpdateTaskItem(ctx context.Context, sessionID string, input tools.TaskUpdateInput) (tools.RuntimeTaskItem, error) {
	taskID := strings.TrimSpace(input.TaskID)
	if taskID == "" {
		return tools.RuntimeTaskItem{}, fmt.Errorf("task_id is required")
	}
	m.mu.Lock()
	item, ok := m.taskItems[taskID]
	if !ok || (sessionID != "" && item.SessionID != sessionID) {
		m.mu.Unlock()
		return tools.RuntimeTaskItem{}, fmt.Errorf("task item %s not found", taskID)
	}
	if title := strings.TrimSpace(input.Title); title != "" {
		item.Title = title
	}
	if strings.TrimSpace(input.Description) != "" {
		item.Description = strings.TrimSpace(input.Description)
	}
	if status := strings.TrimSpace(input.Status); status != "" {
		nextStatus, err := normalizeRuntimeTaskItemStatus(status, item.Status)
		if err != nil {
			m.mu.Unlock()
			return tools.RuntimeTaskItem{}, err
		}
		item.Status = nextStatus
		if runtimeTaskItemStatusClosed(nextStatus) {
			item.CompletedAt = m.now().UnixMilli()
		} else {
			item.CompletedAt = 0
		}
	}
	if input.ClearOwner {
		item.Owner = ""
	} else if owner := strings.TrimSpace(input.Owner); owner != "" {
		item.Owner = owner
	}
	if input.ClearPriority {
		item.Priority = ""
	} else if priority := strings.TrimSpace(input.Priority); priority != "" {
		item.Priority = priority
	}
	if input.DependsOn != nil {
		item.DependsOn = compactRuntimeTaskDependsOn(input.DependsOn)
	}
	item.UpdatedAt = m.now().UnixMilli()
	m.taskItems[taskID] = item
	m.mu.Unlock()
	m.emitEvent("runtime:task_graph_changed", map[string]any{
		"action":    "updated",
		"sessionId": item.SessionID,
		"task":      item,
	})
	return cloneRuntimeTaskItem(item), nil
}

func (m *runtimeTaskManager) ListTaskItems(ctx context.Context, sessionID string, input tools.TaskListInput) ([]tools.RuntimeTaskItem, error) {
	status := strings.TrimSpace(input.Status)
	if status != "" {
		normalized, err := normalizeRuntimeTaskItemStatus(status, status)
		if err != nil {
			return nil, err
		}
		status = normalized
	}
	owner := strings.TrimSpace(input.Owner)
	m.mu.RLock()
	out := make([]tools.RuntimeTaskItem, 0, len(m.taskItems))
	for _, item := range m.taskItems {
		if sessionID != "" && item.SessionID != sessionID {
			continue
		}
		if status != "" && item.Status != status {
			continue
		}
		if status == "" && !input.IncludeClosed && runtimeTaskItemStatusClosed(item.Status) {
			continue
		}
		if owner != "" && item.Owner != owner {
			continue
		}
		out = append(out, cloneRuntimeTaskItem(item))
	}
	m.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt == out[j].CreatedAt {
			return out[i].ID < out[j].ID
		}
		return out[i].CreatedAt < out[j].CreatedAt
	})
	return out, nil
}

func (m *runtimeTaskManager) ReadTaskOutput(ctx context.Context, taskID string, offset, limit int) (tools.RuntimeTaskOutput, error) {
	if strings.TrimSpace(taskID) == "" {
		return tools.RuntimeTaskOutput{}, fmt.Errorf("task_id is required")
	}
	snapshot, ok := m.getTaskSnapshot(taskID)
	if !ok {
		return tools.RuntimeTaskOutput{}, fmt.Errorf("task %s not found", taskID)
	}
	if limit <= 0 {
		limit = defaultTaskOutputLimit
	}
	if offset < 0 {
		offset = 0
	}
	data, err := os.ReadFile(snapshot.OutputPath)
	if err != nil {
		if snapshot.Status == runtimeTaskStatusRunning && os.IsNotExist(err) {
			return tools.RuntimeTaskOutput{
				TaskID:     taskID,
				Status:     snapshot.Status,
				OutputPath: snapshot.OutputPath,
				Offset:     offset,
				NextOffset: offset,
			}, nil
		}
		return tools.RuntimeTaskOutput{}, err
	}
	size := len(data)
	if offset > size {
		offset = size
	}
	end := offset + limit
	if end > size {
		end = size
	}
	return tools.RuntimeTaskOutput{
		TaskID:     taskID,
		Status:     snapshot.Status,
		OutputPath: snapshot.OutputPath,
		Content:    string(data[offset:end]),
		Offset:     offset,
		NextOffset: end,
		Size:       int64(size),
		Truncated:  end < size,
	}, nil
}

func (m *runtimeTaskManager) StopTask(ctx context.Context, taskID string) (tools.RuntimeTaskSnapshot, error) {
	if strings.TrimSpace(taskID) == "" {
		return tools.RuntimeTaskSnapshot{}, fmt.Errorf("task_id is required")
	}
	m.mu.Lock()
	task, ok := m.tasks[taskID]
	if !ok {
		m.mu.Unlock()
		return tools.RuntimeTaskSnapshot{}, fmt.Errorf("task %s not found", taskID)
	}
	if task.snapshot.Status == runtimeTaskStatusRunning {
		task.snapshot.Status = runtimeTaskStatusKilled
		task.snapshot.FinishedAt = m.now().UnixMilli()
		task.snapshot.DurationMs = maxRuntimeTaskDuration(0, task.snapshot.FinishedAt-task.snapshot.StartedAt)
		if task.cancel != nil {
			task.cancel()
		}
	}
	snapshot := task.snapshot
	m.mu.Unlock()
	m.emitEvent("runtime:task_stopped", snapshot)
	return snapshot, nil
}

func (m *runtimeTaskManager) PersistToolResult(ctx context.Context, sessionID, prefix, content string) (string, int64, error) {
	if strings.TrimSpace(sessionID) == "" {
		sessionID = "global"
	}
	if strings.TrimSpace(prefix) == "" {
		prefix = "tool"
	}
	id := fmt.Sprintf("%s-%d", sanitizeFileComponent(prefix), m.now().UnixNano())
	outputPath, err := runtimeToolResultPath(sessionID, id)
	if err != nil {
		return "", 0, err
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return "", 0, err
	}
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return "", 0, err
	}
	return outputPath, int64(len(content)), nil
}

func (m *runtimeTaskManager) List(sessionID string) []tools.RuntimeTaskSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]tools.RuntimeTaskSnapshot, 0, len(m.tasks))
	for _, task := range m.tasks {
		if sessionID != "" && task.snapshot.SessionID != sessionID {
			continue
		}
		out = append(out, task.snapshot)
	}
	return out
}

func (m *runtimeTaskManager) CompactSnapshots(sessionID string) []model.RuntimeTaskCompact {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]model.RuntimeTaskCompact, 0, len(m.tasks))
	for _, task := range m.tasks {
		snapshot := task.snapshot
		if sessionID != "" && snapshot.SessionID != sessionID {
			continue
		}
		out = append(out, model.RuntimeTaskCompact{
			ID:          snapshot.ID,
			SessionID:   snapshot.SessionID,
			Type:        snapshot.Type,
			Status:      snapshot.Status,
			Description: snapshot.Description,
			Command:     snapshot.Command,
			OutputPath:  snapshot.OutputPath,
			OutputSize:  snapshot.OutputSize,
			StartedAt:   snapshot.StartedAt,
			FinishedAt:  snapshot.FinishedAt,
			DurationMs:  snapshot.DurationMs,
			ExitCode:    snapshot.ExitCode,
			Error:       snapshot.Error,
		})
	}
	return out
}

func (m *runtimeTaskManager) CompactTaskItems(sessionID string) []model.RuntimeTaskItemCompact {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]model.RuntimeTaskItemCompact, 0, len(m.taskItems))
	for _, item := range m.taskItems {
		if sessionID != "" && item.SessionID != sessionID {
			continue
		}
		out = append(out, model.RuntimeTaskItemCompact{
			ID:          item.ID,
			SessionID:   item.SessionID,
			Title:       item.Title,
			Description: item.Description,
			Status:      item.Status,
			Owner:       item.Owner,
			Priority:    item.Priority,
			DependsOn:   append([]string(nil), item.DependsOn...),
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
			CompletedAt: item.CompletedAt,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt == out[j].CreatedAt {
			return out[i].ID < out[j].ID
		}
		return out[i].CreatedAt < out[j].CreatedAt
	})
	return out
}

func (m *runtimeTaskManager) RestoreCompactTasks(sessionID string, tasks []model.RuntimeTaskCompact) {
	if len(tasks) == 0 {
		return
	}
	now := m.now().UnixMilli()
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, task := range tasks {
		if strings.TrimSpace(task.ID) == "" {
			continue
		}
		if _, exists := m.tasks[task.ID]; exists {
			continue
		}
		taskSessionID := task.SessionID
		if taskSessionID == "" {
			taskSessionID = sessionID
		}
		if sessionID != "" && taskSessionID != sessionID {
			continue
		}
		status := task.Status
		finishedAt := task.FinishedAt
		errText := task.Error
		if status == runtimeTaskStatusRunning {
			status = runtimeTaskStatusFailed
			finishedAt = now
			if errText == "" {
				errText = "runtime task was active before reload and is no longer attached"
			}
		}
		m.tasks[task.ID] = &runtimeTask{snapshot: tools.RuntimeTaskSnapshot{
			ID:          task.ID,
			SessionID:   taskSessionID,
			Type:        task.Type,
			Status:      status,
			Description: task.Description,
			Command:     task.Command,
			OutputPath:  task.OutputPath,
			OutputSize:  task.OutputSize,
			StartedAt:   task.StartedAt,
			FinishedAt:  finishedAt,
			DurationMs:  task.DurationMs,
			ExitCode:    task.ExitCode,
			Error:       errText,
		}}
	}
}

func (m *runtimeTaskManager) RestoreCompactTaskItems(sessionID string, items []model.RuntimeTaskItemCompact) {
	if len(items) == 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, item := range items {
		if strings.TrimSpace(item.ID) == "" || strings.TrimSpace(item.Title) == "" {
			continue
		}
		itemSessionID := item.SessionID
		if itemSessionID == "" {
			itemSessionID = sessionID
		}
		if sessionID != "" && itemSessionID != sessionID {
			continue
		}
		status, err := normalizeRuntimeTaskItemStatus(item.Status, runtimeTaskItemStatusTodo)
		if err != nil {
			status = runtimeTaskItemStatusTodo
		}
		m.taskItems[item.ID] = tools.RuntimeTaskItem{
			ID:          item.ID,
			SessionID:   itemSessionID,
			Title:       item.Title,
			Description: item.Description,
			Status:      status,
			Owner:       item.Owner,
			Priority:    item.Priority,
			DependsOn:   compactRuntimeTaskDependsOn(item.DependsOn),
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
			CompletedAt: item.CompletedAt,
		}
	}
}

const (
	runtimeTaskItemStatusTodo       = "todo"
	runtimeTaskItemStatusInProgress = "in_progress"
	runtimeTaskItemStatusBlocked    = "blocked"
	runtimeTaskItemStatusCompleted  = "completed"
	runtimeTaskItemStatusCanceled   = "canceled"
)

func normalizeRuntimeTaskItemStatus(status, fallback string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "":
		if strings.TrimSpace(fallback) != "" {
			return fallback, nil
		}
		return runtimeTaskItemStatusTodo, nil
	case "todo", "pending", "open":
		return runtimeTaskItemStatusTodo, nil
	case "in_progress", "in-progress", "running", "active":
		return runtimeTaskItemStatusInProgress, nil
	case "blocked":
		return runtimeTaskItemStatusBlocked, nil
	case "completed", "complete", "done", "closed":
		return runtimeTaskItemStatusCompleted, nil
	case "canceled", "cancelled":
		return runtimeTaskItemStatusCanceled, nil
	default:
		return "", fmt.Errorf("unsupported task status %q", status)
	}
}

func runtimeTaskItemStatusClosed(status string) bool {
	return status == runtimeTaskItemStatusCompleted || status == runtimeTaskItemStatusCanceled
}

func compactRuntimeTaskDependsOn(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, value := range in {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func cloneRuntimeTaskItem(in tools.RuntimeTaskItem) tools.RuntimeTaskItem {
	in.DependsOn = append([]string(nil), in.DependsOn...)
	return in
}

func maxRuntimeTaskDuration(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func (m *runtimeTaskManager) getTaskSnapshot(taskID string) (tools.RuntimeTaskSnapshot, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	task, ok := m.tasks[taskID]
	if !ok {
		return tools.RuntimeTaskSnapshot{}, false
	}
	return task.snapshot, true
}

func (m *runtimeTaskManager) emitEvent(event string, data any) {
	if m.emit != nil {
		m.emit(event, data)
	}
}

func formatBashTaskOutput(out tools.BashOutput, err error) string {
	var b strings.Builder
	if out.Stdout != "" {
		b.WriteString(out.Stdout)
		if !strings.HasSuffix(out.Stdout, "\n") {
			b.WriteString("\n")
		}
	}
	if out.Stderr != "" {
		b.WriteString(out.Stderr)
		if !strings.HasSuffix(out.Stderr, "\n") {
			b.WriteString("\n")
		}
	}
	if err != nil {
		b.WriteString("error: ")
		b.WriteString(err.Error())
		b.WriteString("\n")
	}
	if out.Interrupted {
		b.WriteString("interrupted: true\n")
	}
	b.WriteString(fmt.Sprintf("exit_code: %d\n", out.ExitCode))
	return b.String()
}

func runtimeTaskOutputPath(sessionID, taskID string) (string, error) {
	base, err := starxoHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "sessions", sanitizeFileComponent(sessionID), "runtime-tasks", sanitizeFileComponent(taskID)+".output"), nil
}

func runtimeToolResultPath(sessionID, id string) (string, error) {
	base, err := starxoHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "sessions", sanitizeFileComponent(sessionID), "tool-results", sanitizeFileComponent(id)+".txt"), nil
}

func starxoHomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".starxo"), nil
}

func sanitizeFileComponent(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "item"
	}
	return out
}

func (s *ChatService) ListRuntimeTasks(sessionID string) ([]tools.RuntimeTaskSnapshot, error) {
	s.mu.Lock()
	manager := s.runtimeTasks
	s.mu.Unlock()
	if manager == nil {
		return nil, fmt.Errorf("runtime task manager is not available")
	}
	return manager.List(sessionID), nil
}

func (s *ChatService) GetRuntimeLSPStatus(sessionID string) (RuntimeLSPStatus, error) {
	s.mu.Lock()
	manager := s.runtimeLSP
	store := s.store
	s.mu.Unlock()
	if manager == nil {
		return RuntimeLSPStatus{}, fmt.Errorf("runtime LSP manager is not available")
	}
	if store != nil {
		cfg := store.Get()
		if cfg != nil {
			config.NormalizeAppConfig(cfg)
			manager.SetConfig(cfg.Agent.LSP)
		}
	}
	return manager.Status(sessionID), nil
}

func (s *ChatService) ReadRuntimeTaskOutput(taskID string, offset int, limit int) (tools.RuntimeTaskOutput, error) {
	s.mu.Lock()
	manager := s.runtimeTasks
	s.mu.Unlock()
	if manager == nil {
		return tools.RuntimeTaskOutput{}, fmt.Errorf("runtime task manager is not available")
	}
	return manager.ReadTaskOutput(context.Background(), taskID, offset, limit)
}

func (s *ChatService) StopRuntimeTask(taskID string) (tools.RuntimeTaskSnapshot, error) {
	s.mu.Lock()
	manager := s.runtimeTasks
	s.mu.Unlock()
	if manager == nil {
		return tools.RuntimeTaskSnapshot{}, fmt.Errorf("runtime task manager is not available")
	}
	return manager.StopTask(context.Background(), taskID)
}

func (s *ChatService) ApproveToolPermission(requestID string, decision string) error {
	if strings.TrimSpace(decision) == "" {
		decision = tools.ToolPermissionDecisionAllowOnce
	}
	return s.resolvePermissionRequest(requestID, decision)
}

func (s *ChatService) DenyToolPermission(requestID string) error {
	return s.resolvePermissionRequest(requestID, tools.ToolPermissionDecisionDeny)
}

func wailsEmit(ctx context.Context, event string, data any) {
	if ctx == nil || ctx.Value("events") == nil {
		return
	}
	// Keep Wails isolated in this helper so tests can call the service methods
	// without requiring an application runtime context.
	wailsruntime.EventsEmit(ctx, event, data)
}
