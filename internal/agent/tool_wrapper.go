package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// eventEmittingTool wraps a BaseTool/InvokableTool and emits timeline events
// for tool calls and results so the frontend can display sub-agent activity.
type eventEmittingTool struct {
	inner     tool.BaseTool
	agentName string
	toolName  string
	onEvent   func(ctx context.Context, agentName, eventType, toolName, toolArgs, toolID, result string)
}

func (t *eventEmittingTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return t.inner.Info(ctx)
}

func (t *eventEmittingTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	callID := fmt.Sprintf("cb-%d", time.Now().UnixNano())

	// Emit tool_call event
	if t.onEvent != nil {
		t.onEvent(ctx, t.agentName, "tool_call", t.toolName, argumentsInJSON, callID, "")
	}

	// Execute the actual tool
	var result string
	var err error
	if inv, ok := t.inner.(tool.InvokableTool); ok {
		result, err = inv.InvokableRun(ctx, argumentsInJSON, opts...)
	} else {
		err = fmt.Errorf("inner tool %s does not implement InvokableTool", t.toolName)
	}

	// Emit tool_result event
	if t.onEvent != nil {
		resultStr := result
		if err != nil {
			resultStr = fmt.Sprintf("Error: %v", err)
		}
		t.onEvent(ctx, t.agentName, "tool_result", t.toolName, "", callID, resultStr)
	}

	return result, err
}

// WrapToolsWithEvents wraps each tool with an event-emitting layer.
// This makes sub-agent tool calls visible to the frontend.
func WrapToolsWithEvents(agentName string, tools []tool.BaseTool, ac AgentContext) []tool.BaseTool {
	if ac.OnToolEvent == nil {
		return tools
	}

	wrapped := make([]tool.BaseTool, len(tools))
	for i, t := range tools {
		info, err := t.Info(context.Background())
		name := "unknown"
		if err == nil && info != nil {
			name = info.Name
		}
		wrapped[i] = &eventEmittingTool{
			inner:     t,
			agentName: agentName,
			toolName:  name,
			onEvent:   ac.OnToolEvent,
		}
	}
	return wrapped
}
