package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"starxo/internal/model"
	agenttools "starxo/internal/tools"
)

const runtimeObjectiveExtraKey = "starxo_current_objective"

type RuntimeBehaviorMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
}

func NewRuntimeBehaviorMiddleware() adk.ChatModelAgentMiddleware {
	return &RuntimeBehaviorMiddleware{BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{}}
}

func (m *RuntimeBehaviorMiddleware) BeforeModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState, mc *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
	objective, ok := agenttools.RuntimeObjectiveFromContext(ctx)
	if !ok || state == nil {
		return ctx, state, nil
	}
	state.Messages = upsertRuntimeObjectiveMessage(state.Messages, objective)
	return ctx, state, nil
}

func (m *RuntimeBehaviorMiddleware) WrapInvokableToolCall(ctx context.Context, endpoint adk.InvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.InvokableToolCallEndpoint, error) {
	toolName := ""
	if tCtx != nil {
		toolName = tCtx.Name
	}
	return func(callCtx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
		if msg, ok := agenttools.RuntimeObjectiveToolGuard(callCtx, toolName, argumentsInJSON); !ok {
			return msg, nil
		}
		return endpoint(callCtx, argumentsInJSON, opts...)
	}, nil
}

func upsertRuntimeObjectiveMessage(messages []*schema.Message, objective *model.RunObjective) []*schema.Message {
	if objective == nil {
		return messages
	}
	msg := runtimeObjectiveMessage(objective)
	if len(messages) == 0 {
		return []*schema.Message{msg}
	}
	out := make([]*schema.Message, 0, len(messages)+1)
	inserted := false
	for i, m := range messages {
		if isRuntimeObjectiveMessage(m) {
			if !inserted {
				out = append(out, msg)
				inserted = true
			}
			continue
		}
		out = append(out, m)
		if !inserted && i == 0 && m.Role == schema.System {
			out = append(out, msg)
			inserted = true
		}
	}
	if !inserted {
		out = append([]*schema.Message{msg}, out...)
	}
	return out
}

func runtimeObjectiveMessage(objective *model.RunObjective) *schema.Message {
	scope := strings.TrimSpace(objective.Scope)
	if scope == "" {
		scope = "standalone"
	}
	content := fmt.Sprintf(`<current-objective>
id: %s
run_id: %s
scope: %s
user_message_id: %s
objective: %s
acceptance: %s
</current-objective>

Only act on this current objective. Older conversation is historical context, not an active task, unless scope is continuation. If a tool call or clarification would pursue unrelated old work, stop and return to this objective.`,
		objective.ID,
		objective.RunID,
		scope,
		objective.UserMessageID,
		oneLineRuntimeObjective(objective.Objective),
		oneLineRuntimeObjective(objective.Acceptance),
	)
	msg := schema.UserMessage(content)
	msg.Extra = map[string]any{runtimeObjectiveExtraKey: true}
	return msg
}

func isRuntimeObjectiveMessage(msg *schema.Message) bool {
	if msg == nil {
		return false
	}
	if msg.Extra != nil {
		if v, ok := msg.Extra[runtimeObjectiveExtraKey].(bool); ok && v {
			return true
		}
	}
	return false
}

func oneLineRuntimeObjective(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > 500 {
		return text[:497] + "..."
	}
	return text
}
