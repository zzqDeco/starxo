package tools

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/tool"
)

type invokableTodoTool interface {
	InvokableRun(context.Context, string, ...tool.Option) (string, error)
}

func TestTodoToolsUseSessionBuckets(t *testing.T) {
	ClearTodos()
	ClearTodosForSession("session-a")
	ClearTodosForSession("session-b")
	t.Cleanup(func() {
		ClearTodos()
		ClearTodosForSession("session-a")
		ClearTodosForSession("session-b")
	})

	writeTool, ok := NewWriteTodosTool().(invokableTodoTool)
	if !ok {
		t.Fatalf("write_todos is not invokable")
	}
	updateTool, ok := NewUpdateTodoTool().(invokableTodoTool)
	if !ok {
		t.Fatalf("update_todo is not invokable")
	}

	ctxA := context.WithValue(context.Background(), "sessionID", "session-a")
	ctxB := context.WithValue(context.Background(), "sessionID", "session-b")

	if _, err := writeTool.InvokableRun(ctxA, `{"todos":[{"id":"a","title":"A","status":"pending"}]}`); err != nil {
		t.Fatalf("write session-a todos: %v", err)
	}
	if _, err := writeTool.InvokableRun(ctxB, `{"todos":[{"id":"b","title":"B","status":"pending"}]}`); err != nil {
		t.Fatalf("write session-b todos: %v", err)
	}
	if _, err := updateTool.InvokableRun(ctxA, `{"id":"a","status":"done"}`); err != nil {
		t.Fatalf("update session-a todo: %v", err)
	}

	todosA := SnapshotTodosForSession("session-a")
	if len(todosA) != 1 || todosA[0].ID != "a" || todosA[0].Status != "done" {
		t.Fatalf("unexpected session-a todos: %#v", todosA)
	}
	todosB := SnapshotTodosForSession("session-b")
	if len(todosB) != 1 || todosB[0].ID != "b" || todosB[0].Status != "pending" {
		t.Fatalf("unexpected session-b todos: %#v", todosB)
	}
	if got := SnapshotTodos(); len(got) != 0 {
		t.Fatalf("expected global compatibility store to remain empty, got %#v", got)
	}
}

func TestTodoToolsFallBackToGlobalBucketWithoutSession(t *testing.T) {
	ClearTodos()
	t.Cleanup(ClearTodos)

	writeTool, ok := NewWriteTodosTool().(invokableTodoTool)
	if !ok {
		t.Fatalf("write_todos is not invokable")
	}
	updateTool, ok := NewUpdateTodoTool().(invokableTodoTool)
	if !ok {
		t.Fatalf("update_todo is not invokable")
	}

	if _, err := writeTool.InvokableRun(context.Background(), `{"todos":[{"id":"global","title":"Global","status":"pending"}]}`); err != nil {
		t.Fatalf("write global todos: %v", err)
	}
	if _, err := updateTool.InvokableRun(context.Background(), `{"id":"global","status":"in_progress"}`); err != nil {
		t.Fatalf("update global todo: %v", err)
	}

	got := SnapshotTodos()
	if len(got) != 1 || got[0].ID != "global" || got[0].Status != "in_progress" {
		t.Fatalf("unexpected global todos: %#v", got)
	}
}
