package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"starxo/internal/model"
	"starxo/internal/tools"
)

func TestRuntimeTaskManagerStartReadAndComplete(t *testing.T) {
	now := time.UnixMilli(100)
	manager := newRuntimeTaskManager(func() time.Time {
		now = now.Add(time.Millisecond)
		return now
	}, nil)

	ref, err := manager.StartShellTask(context.Background(), "sess-runtime", "echo ok", "Echo ok", func(ctx context.Context) (tools.BashOutput, error) {
		return tools.BashOutput{Stdout: "ok\n", ExitCode: 0}, nil
	})
	if err != nil {
		t.Fatalf("start task: %v", err)
	}

	var out tools.RuntimeTaskOutput
	var tasks []tools.RuntimeTaskSnapshot
	for i := 0; i < 20; i++ {
		out, err = manager.ReadTaskOutput(context.Background(), ref.TaskID, 0, 1024)
		tasks = manager.List("sess-runtime")
		if err == nil && strings.Contains(out.Content, "ok") && len(tasks) == 1 && tasks[0].Status == runtimeTaskStatusCompleted {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("read task output: %v", err)
	}
	if !strings.Contains(out.Content, "ok") {
		t.Fatalf("expected task output, got %#v", out)
	}

	if len(tasks) != 1 || tasks[0].Status != runtimeTaskStatusCompleted {
		t.Fatalf("expected completed task, got %#v", tasks)
	}
}

func TestRuntimeTaskManagerStopTask(t *testing.T) {
	manager := newRuntimeTaskManager(time.Now, nil)
	started := make(chan struct{})
	release := make(chan struct{})

	ref, err := manager.StartShellTask(context.Background(), "sess-runtime", "sleep", "Sleep", func(ctx context.Context) (tools.BashOutput, error) {
		close(started)
		select {
		case <-ctx.Done():
			return tools.BashOutput{Interrupted: true, ExitCode: 143}, ctx.Err()
		case <-release:
			return tools.BashOutput{ExitCode: 0}, nil
		}
	})
	if err != nil {
		t.Fatalf("start task: %v", err)
	}
	<-started

	snapshot, err := manager.StopTask(context.Background(), ref.TaskID)
	if err != nil {
		t.Fatalf("stop task: %v", err)
	}
	if snapshot.Status != runtimeTaskStatusKilled {
		t.Fatalf("expected killed status, got %#v", snapshot)
	}
	close(release)
}

func TestRuntimeTaskManagerTaskGraphCreateUpdateListAndRestore(t *testing.T) {
	now := time.UnixMilli(1000)
	manager := newRuntimeTaskManager(func() time.Time {
		now = now.Add(time.Millisecond)
		return now
	}, nil)

	created, err := manager.CreateTaskItem(context.Background(), "sess-runtime", tools.TaskCreateInput{
		Title:     "Align CC task graph",
		Status:    "pending",
		Owner:     "agent",
		Priority:  "high",
		DependsOn: []string{"root", "root", " "},
	})
	if err != nil {
		t.Fatalf("create task item: %v", err)
	}
	if created.Status != runtimeTaskItemStatusTodo || len(created.DependsOn) != 1 || created.DependsOn[0] != "root" {
		t.Fatalf("unexpected created task item: %#v", created)
	}

	updated, err := manager.UpdateTaskItem(context.Background(), "sess-runtime", tools.TaskUpdateInput{
		TaskID:    created.ID,
		Status:    "done",
		DependsOn: []string{},
	})
	if err != nil {
		t.Fatalf("update task item: %v", err)
	}
	if updated.Status != runtimeTaskItemStatusCompleted || updated.CompletedAt == 0 || len(updated.DependsOn) != 0 {
		t.Fatalf("unexpected updated task item: %#v", updated)
	}

	open, err := manager.ListTaskItems(context.Background(), "sess-runtime", tools.TaskListInput{})
	if err != nil {
		t.Fatalf("list open task items: %v", err)
	}
	if len(open) != 0 {
		t.Fatalf("expected closed task hidden by default, got %#v", open)
	}
	all, err := manager.ListTaskItems(context.Background(), "sess-runtime", tools.TaskListInput{IncludeClosed: true})
	if err != nil {
		t.Fatalf("list all task items: %v", err)
	}
	if len(all) != 1 || all[0].ID != created.ID {
		t.Fatalf("expected completed task item, got %#v", all)
	}

	compact := manager.CompactTaskItems("sess-runtime")
	restored := newRuntimeTaskManager(func() time.Time { return time.UnixMilli(2000) }, nil)
	restored.RestoreCompactTaskItems("sess-runtime", compact)
	got, err := restored.GetTaskItem(context.Background(), "sess-runtime", created.ID)
	if err != nil {
		t.Fatalf("get restored task item: %v", err)
	}
	if got.Status != runtimeTaskItemStatusCompleted || got.Title != "Align CC task graph" {
		t.Fatalf("unexpected restored task item: %#v", got)
	}
}

func TestRuntimeTaskManagerTaskGraphIDsResistClockCollisions(t *testing.T) {
	fixed := time.Unix(0, 1234)
	manager := newRuntimeTaskManager(func() time.Time { return fixed }, nil)

	first, err := manager.CreateTaskItem(context.Background(), "sess-runtime", tools.TaskCreateInput{Title: "first"})
	if err != nil {
		t.Fatalf("create first task item: %v", err)
	}
	second, err := manager.CreateTaskItem(context.Background(), "sess-runtime", tools.TaskCreateInput{Title: "second"})
	if err != nil {
		t.Fatalf("create second task item: %v", err)
	}
	if first.ID == second.ID {
		t.Fatalf("expected collision-resistant ids, got %q", first.ID)
	}
	items, err := manager.ListTaskItems(context.Background(), "sess-runtime", tools.TaskListInput{})
	if err != nil {
		t.Fatalf("list task items: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected both task items to survive same-tick creates, got %#v", items)
	}
}

func TestRuntimeTaskManagerClearTaskItemsEmitsAfterUnlock(t *testing.T) {
	manager := newRuntimeTaskManager(func() time.Time { return time.UnixMilli(1000) }, nil)
	if _, err := manager.CreateTaskItem(context.Background(), "sess-runtime", tools.TaskCreateInput{Title: "clear me"}); err != nil {
		t.Fatalf("create task item: %v", err)
	}
	emitted := make(chan struct{})
	manager.onTaskGraphChanged = func(sessionID string) {
		items := manager.CompactTaskItems(sessionID)
		if len(items) != 0 {
			t.Errorf("expected callback to observe cleared task graph, got %#v", items)
		}
		close(emitted)
	}

	if removed := manager.ClearTaskItemsForSession("sess-runtime"); removed != 1 {
		t.Fatalf("expected one removed task item, got %d", removed)
	}
	select {
	case <-emitted:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for task graph clear callback")
	}
}

func TestRuntimeTaskManagerRestoreTaskItemsReplacesSessionState(t *testing.T) {
	manager := newRuntimeTaskManager(func() time.Time { return time.UnixMilli(1000) }, nil)
	stale, err := manager.CreateTaskItem(context.Background(), "sess-runtime", tools.TaskCreateInput{Title: "stale item"})
	if err != nil {
		t.Fatalf("create stale task item: %v", err)
	}
	if _, err := manager.CreateTaskItem(context.Background(), "sess-other", tools.TaskCreateInput{Title: "other item"}); err != nil {
		t.Fatalf("create other task item: %v", err)
	}

	manager.RestoreCompactTaskItems("sess-runtime", []model.RuntimeTaskItemCompact{{
		ID:        "taskitem-restored",
		SessionID: "sess-runtime",
		Title:     "restored item",
		Status:    runtimeTaskItemStatusInProgress,
		CreatedAt: 100,
		UpdatedAt: 200,
	}})

	items, err := manager.ListTaskItems(context.Background(), "sess-runtime", tools.TaskListInput{IncludeClosed: true})
	if err != nil {
		t.Fatalf("list restored task items: %v", err)
	}
	if len(items) != 1 || items[0].ID != "taskitem-restored" || items[0].Status != runtimeTaskItemStatusInProgress {
		t.Fatalf("expected restored task item to replace stale state, got %#v", items)
	}
	if _, err := manager.GetTaskItem(context.Background(), "sess-runtime", stale.ID); err == nil {
		t.Fatalf("expected stale task item %q to be removed", stale.ID)
	}

	manager.RestoreCompactTaskItems("sess-runtime", nil)
	items, err = manager.ListTaskItems(context.Background(), "sess-runtime", tools.TaskListInput{IncludeClosed: true})
	if err != nil {
		t.Fatalf("list cleared task items: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty restore to clear session task items, got %#v", items)
	}
	otherItems, err := manager.ListTaskItems(context.Background(), "sess-other", tools.TaskListInput{IncludeClosed: true})
	if err != nil {
		t.Fatalf("list other task items: %v", err)
	}
	if len(otherItems) != 1 || otherItems[0].SessionID != "sess-other" {
		t.Fatalf("expected restore to preserve other session task items, got %#v", otherItems)
	}
}
