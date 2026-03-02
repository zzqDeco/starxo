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

// todoStore is the in-memory store for the current todo list.
// It is used by both write_todos and update_todo tools.
var todoStore struct {
	mu    sync.Mutex
	todos []TodoItem
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

			// Store todos
			todoStore.mu.Lock()
			todoStore.todos = make([]TodoItem, len(input.Todos))
			copy(todoStore.todos, input.Todos)
			todoStore.mu.Unlock()

			// Diagnostic log: verify stored IDs
			ids := make([]string, len(input.Todos))
			for i, t := range input.Todos {
				ids[i] = t.ID
			}
			logger.Info("[TODOS] Store updated", "count", len(input.Todos), "ids", strings.Join(ids, ","))

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

			// Debug log: record store state at lookup time
			storedIDs := make([]string, len(todoStore.todos))
			for i, t := range todoStore.todos {
				storedIDs[i] = t.ID
			}
			logger.Debug("[TODOS] update_todo lookup", "target", input.ID, "store_count", len(todoStore.todos), "stored_ids", strings.Join(storedIDs, ","))

			found := false
			for i := range todoStore.todos {
				if todoStore.todos[i].ID == input.ID {
					todoStore.todos[i].Status = input.Status
					if input.Title != "" {
						todoStore.todos[i].Title = input.Title
					}
					found = true
					break
				}
			}

			if !found {
				return fmt.Sprintf("Warning: todo with ID %q not found in current store (store has %d items). The todo list may need to be re-declared with write_todos.", input.ID, len(todoStore.todos)), nil
			}

			// Build summary
			counts := map[string]int{}
			for _, t := range todoStore.todos {
				counts[t.Status]++
			}
			parts := []string{}
			for _, s := range []string{"pending", "in_progress", "done", "failed", "blocked"} {
				if c, ok := counts[s]; ok && c > 0 {
					parts = append(parts, fmt.Sprintf("%d %s", c, s))
				}
			}

			data, _ := json.Marshal(todoStore.todos)
			return fmt.Sprintf("Updated todo %q to %s (%s)\n---\n%s", input.ID, input.Status, strings.Join(parts, ", "), string(data)), nil
		},
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create update_todo tool: %v", err))
	}
	return t
}
