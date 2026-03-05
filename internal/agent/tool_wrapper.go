package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	agenttools "starxo/internal/tools"
)

// eventEmittingTool wraps a BaseTool/InvokableTool and emits timeline events
// for tool calls and results so the frontend can display sub-agent activity.
type eventEmittingTool struct {
	inner     tool.BaseTool
	agentName string
	toolName  string
	onEvent   func(ctx context.Context, agentName, eventType, toolName, toolArgs, toolID, result string)

	mu                  sync.Mutex
	recoverableErrCount map[string]int // key: sessionScope|signature
}

const recoverableErrorEscalationThreshold = 3

func (t *eventEmittingTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return t.inner.Info(ctx)
}

func (t *eventEmittingTool) sessionScope(ctx context.Context) string {
	// Service layer injects sessionID into run context; fallback keeps behavior safe
	// even when session information is unavailable.
	if v, ok := ctx.Value("sessionID").(string); ok && v != "" {
		return v
	}
	return "global"
}

func (t *eventEmittingTool) incrementRecoverableError(sessionScope, signature string) int {
	key := sessionScope + "|" + signature
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.recoverableErrCount == nil {
		t.recoverableErrCount = make(map[string]int)
	}
	t.recoverableErrCount[key]++
	return t.recoverableErrCount[key]
}

func (t *eventEmittingTool) clearRecoverableErrorsForSession(sessionScope string) {
	prefix := sessionScope + "|"
	t.mu.Lock()
	defer t.mu.Unlock()
	for k := range t.recoverableErrCount {
		if strings.HasPrefix(k, prefix) {
			delete(t.recoverableErrCount, k)
		}
	}
}

func (t *eventEmittingTool) clearRecoverableError(sessionScope, signature string) {
	key := sessionScope + "|" + signature
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.recoverableErrCount, key)
}

func (t *eventEmittingTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	callID := fmt.Sprintf("cb-%d", time.Now().UnixNano())
	sessionScope := t.sessionScope(ctx)

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

	finalResult := result
	finalErr := err
	resultStr := result

	if err != nil {
		decision := agenttools.ClassifyToolError(t.toolName, argumentsInJSON, err)
		if decision.NormalizedMsg != "" {
			resultStr = decision.NormalizedMsg
		} else {
			resultStr = fmt.Sprintf("Error: %v", err)
		}

		if decision.Recoverable {
			count := t.incrementRecoverableError(sessionScope, decision.Signature)
			if count < recoverableErrorEscalationThreshold {
				finalResult = resultStr
				finalErr = nil
			} else {
				// Prevent unbounded loops when the same recoverable error repeats.
				resultStr = fmt.Sprintf("%s; hint: repeated %d times, escalating to node failure", resultStr, count)
				finalResult = ""
				finalErr = fmt.Errorf("%w (escalated after %d repeated recoverable failures)", err, count)
				t.clearRecoverableError(sessionScope, decision.Signature)
			}
		}
	} else {
		// Successful invocation resets recoverable-error backoff within this session.
		t.clearRecoverableErrorsForSession(sessionScope)
	}

	// Emit tool_result event
	if t.onEvent != nil {
		t.onEvent(ctx, t.agentName, "tool_result", t.toolName, "", callID, resultStr)
	}

	return finalResult, finalErr
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
			inner:               t,
			agentName:           agentName,
			toolName:            name,
			onEvent:             ac.OnToolEvent,
			recoverableErrCount: make(map[string]int),
		}
	}
	return wrapped
}
