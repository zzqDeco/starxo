package service

import (
	"context"
	"strings"
	"testing"
	"time"

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
