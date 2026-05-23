package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"

	"starxo/internal/logger"
	"starxo/internal/model"
)

// TodoItem represents a single task in the DAG.
type TodoItem struct {
	ID        string   `json:"id" jsonschema:"description=unique identifier for this todo"`
	Title     string   `json:"title" jsonschema:"description=short description of the task"`
	Status    string   `json:"status" jsonschema:"description=current status: pending | in_progress | done | failed | blocked"`
	DependsOn []string `json:"depends_on,omitempty" jsonschema:"description=IDs of prerequisite tasks that must complete before this one"`
}

// WriteTodosInput is the input for the write_todos tool.
type WriteTodosInput struct {
	Todos []TodoItem `json:"todos" jsonschema:"description=the complete list of todos with their current statuses and dependencies"`
}

// todoStore is the in-memory store for todo lists.
// It is used by both write_todos and update_todo tools. Session-scoped agent
// runs use bySession; the package-level wrappers keep the original global
// bucket for compatibility with older callers/tests that do not pass sessionID.
var todoStore struct {
	mu        sync.Mutex
	todos     []TodoItem
	bySession map[string][]TodoItem
}

func todoSessionID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if sessionID, ok := ctx.Value("sessionID").(string); ok {
		return sessionID
	}
	return ""
}

func cloneTodoItems(items []TodoItem) []TodoItem {
	if len(items) == 0 {
		return nil
	}
	out := make([]TodoItem, len(items))
	for i, item := range items {
		out[i] = TodoItem{
			ID:        item.ID,
			Title:     item.Title,
			Status:    item.Status,
			DependsOn: append([]string(nil), item.DependsOn...),
		}
	}
	return out
}

func todoItemsFromRuntime(items []model.RuntimeTodoItem) []TodoItem {
	if len(items) == 0 {
		return nil
	}
	out := make([]TodoItem, len(items))
	for i, item := range items {
		out[i] = TodoItem{
			ID:        item.ID,
			Title:     item.Title,
			Status:    item.Status,
			DependsOn: append([]string(nil), item.DependsOn...),
		}
	}
	return out
}

func runtimeTodosFromTodoItems(items []TodoItem) []model.RuntimeTodoItem {
	if len(items) == 0 {
		return nil
	}
	out := make([]model.RuntimeTodoItem, len(items))
	for i, todo := range items {
		out[i] = model.RuntimeTodoItem{
			ID:        todo.ID,
			Title:     todo.Title,
			Status:    todo.Status,
			DependsOn: append([]string(nil), todo.DependsOn...),
		}
	}
	return out
}

func todosForSessionLocked(sessionID string) []TodoItem {
	if sessionID == "" {
		return todoStore.todos
	}
	return todoStore.bySession[sessionID]
}

func setTodosForSessionLocked(sessionID string, items []TodoItem) {
	if sessionID == "" {
		todoStore.todos = cloneTodoItems(items)
		return
	}
	if len(items) == 0 {
		delete(todoStore.bySession, sessionID)
		return
	}
	if todoStore.bySession == nil {
		todoStore.bySession = make(map[string][]TodoItem)
	}
	todoStore.bySession[sessionID] = cloneTodoItems(items)
}

// ClearTodosForSession resets the in-memory todo store for a session.
func ClearTodosForSession(sessionID string) {
	todoStore.mu.Lock()
	if sessionID == "" {
		todoStore.todos = nil
	} else {
		delete(todoStore.bySession, sessionID)
	}
	todoStore.mu.Unlock()
}

// ClearTodos resets the global compatibility todo store.
func ClearTodos() {
	ClearTodosForSession("")
}

// SnapshotTodosForSession returns a copy of a session's in-memory todo list
// using the model package shape so it can be persisted with session compact state.
func SnapshotTodosForSession(sessionID string) []model.RuntimeTodoItem {
	todoStore.mu.Lock()
	defer todoStore.mu.Unlock()
	return runtimeTodosFromTodoItems(todosForSessionLocked(sessionID))
}

// SnapshotTodos returns a copy of the global compatibility todo list.
func SnapshotTodos() []model.RuntimeTodoItem {
	return SnapshotTodosForSession("")
}

// RestoreTodosForSession replaces a session's in-memory todo list from
// persisted compact state.
func RestoreTodosForSession(sessionID string, items []model.RuntimeTodoItem) {
	todoStore.mu.Lock()
	defer todoStore.mu.Unlock()
	if len(items) == 0 {
		setTodosForSessionLocked(sessionID, nil)
		return
	}
	setTodosForSessionLocked(sessionID, todoItemsFromRuntime(items))
}

// RestoreTodos replaces the global compatibility todo list from persisted compact state.
func RestoreTodos(items []model.RuntimeTodoItem) {
	RestoreTodosForSession("", items)
}

// NewWriteTodosTool creates a tool that tracks task progress as a DAG.
// The agent calls this to declare/update the task list. The frontend renders it
// as a visual DAG component.
func NewWriteTodosTool() tool.BaseTool {
	t, err := toolutils.InferTool(
		"write_todos",
		"Declare or update the task list for the current request. Each todo has an id, title, status (pending/in_progress/done/failed/blocked), and optional depends_on list of prerequisite task IDs. Call this whenever you start a multi-step task to show progress, and update it as tasks complete.",
		func(ctx context.Context, input *WriteTodosInput) (string, error) {
			if len(input.Todos) == 0 {
				return "No todos provided.", nil
			}

			// Validate DAG: check for unknown dependency IDs
			idSet := make(map[string]bool)
			for _, t := range input.Todos {
				idSet[t.ID] = true
			}
			for _, t := range input.Todos {
				for _, dep := range t.DependsOn {
					if !idSet[dep] {
						return "", fmt.Errorf("todo %q depends on unknown ID %q", t.ID, dep)
					}
				}
			}

			sessionID := todoSessionID(ctx)

			// Store todos
			todoStore.mu.Lock()
			setTodosForSessionLocked(sessionID, input.Todos)
			todoStore.mu.Unlock()

			// Diagnostic log: verify stored IDs
			ids := make([]string, len(input.Todos))
			for i, t := range input.Todos {
				ids[i] = t.ID
			}
			logger.Info("[TODOS] Store updated", "session", sessionID, "count", len(input.Todos), "ids", strings.Join(ids, ","))

			// Build summary
			counts := map[string]int{}
			for _, t := range input.Todos {
				counts[t.Status]++
			}

			parts := []string{}
			for _, s := range []string{"pending", "in_progress", "done", "failed", "blocked"} {
				if c, ok := counts[s]; ok && c > 0 {
					parts = append(parts, fmt.Sprintf("%d %s", c, s))
				}
			}

			// Also return the full JSON so the frontend can render
			data, _ := json.Marshal(input.Todos)
			return fmt.Sprintf("Updated %d todos (%s)\n---\n%s", len(input.Todos), strings.Join(parts, ", "), string(data)), nil
		},
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create write_todos tool: %v", err))
	}
	return t
}

// UpdateTodoInput is the input for the update_todo tool.
type UpdateTodoInput struct {
	ID     string `json:"id" jsonschema:"description=the ID of the todo to update"`
	Status string `json:"status" jsonschema:"description=new status: pending | in_progress | done | failed | blocked"`
	Title  string `json:"title,omitempty" jsonschema:"description=optional new title for the todo"`
}

// NewUpdateTodoTool creates a tool that updates a single todo item's status.
// After updating, it returns the full updated todo list so the frontend can re-render.
func NewUpdateTodoTool() tool.BaseTool {
	t, err := toolutils.InferTool(
		"update_todo",
		"Update the status of a single todo item by its ID. Use this to mark tasks as in_progress when starting, done when complete, or failed if something went wrong. Returns the full updated todo list.",
		func(ctx context.Context, input *UpdateTodoInput) (string, error) {
			if input.ID == "" {
				return "", fmt.Errorf("todo ID is required")
			}
			validStatuses := map[string]bool{
				"pending": true, "in_progress": true, "done": true, "failed": true, "blocked": true,
			}
			if !validStatuses[input.Status] {
				return "", fmt.Errorf("invalid status %q, must be one of: pending, in_progress, done, failed, blocked", input.Status)
			}

			todoStore.mu.Lock()
			defer todoStore.mu.Unlock()
			sessionID := todoSessionID(ctx)
			todos := todosForSessionLocked(sessionID)

			// Debug log: record store state at lookup time
			storedIDs := make([]string, len(todos))
			for i, t := range todos {
				storedIDs[i] = t.ID
			}
			logger.Debug("[TODOS] update_todo lookup", "session", sessionID, "target", input.ID, "store_count", len(todos), "stored_ids", strings.Join(storedIDs, ","))

			found := false
			for i := range todos {
				if todos[i].ID == input.ID {
					todos[i].Status = input.Status
					if input.Title != "" {
						todos[i].Title = input.Title
					}
					found = true
					break
				}
			}

			if !found {
				return fmt.Sprintf("Warning: todo with ID %q not found in current store (store has %d items). The todo list may need to be re-declared with write_todos.", input.ID, len(todos)), nil
			}
			setTodosForSessionLocked(sessionID, todos)

			// Build summary
			counts := map[string]int{}
			for _, t := range todos {
				counts[t.Status]++
			}
			parts := []string{}
			for _, s := range []string{"pending", "in_progress", "done", "failed", "blocked"} {
				if c, ok := counts[s]; ok && c > 0 {
					parts = append(parts, fmt.Sprintf("%d %s", c, s))
				}
			}

			data, _ := json.Marshal(todos)
			return fmt.Sprintf("Updated todo %q to %s (%s)\n---\n%s", input.ID, input.Status, strings.Join(parts, ", "), string(data)), nil
		},
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create update_todo tool: %v", err))
	}
	return t
}
