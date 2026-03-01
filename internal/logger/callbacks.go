package logger

import (
	"context"
	"time"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	template "github.com/cloudwego/eino/utils/callbacks"
)

// contextKey is used to store timing info in context for duration measurement.
type contextKey string

const toolStartTimeKey contextKey = "tool_start_time"
const modelStartTimeKey contextKey = "model_start_time"

// RegisterGlobalCallbacks registers Eino framework-level callback handlers
// that log all model calls, tool calls, and errors. Call this once at startup.
func RegisterGlobalCallbacks() {
	handler := template.NewHandlerHelper().
		ChatModel(&template.ModelCallbackHandler{
			OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *model.CallbackInput) context.Context {
				agentName := extractAgentName(info)
				msgCount := 0
				if input != nil {
					msgCount = len(input.Messages)
				}
				ModelCall(agentName, msgCount,
					"component", info.Name,
				)
				return context.WithValue(ctx, modelStartTimeKey, time.Now())
			},
			OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *model.CallbackOutput) context.Context {
				agentName := extractAgentName(info)
				if output == nil {
					ModelResult(agentName, false, 0)
					return ctx
				}

				hasToolCalls := false
				contentLen := 0
				if output.Message != nil {
					hasToolCalls = len(output.Message.ToolCalls) > 0
					contentLen = len(output.Message.Content)

					// Log individual tool calls from model response
					if hasToolCalls {
						for _, tc := range output.Message.ToolCalls {
							L().Debug("[MODEL_TOOL_CALL]",
								"agent", agentName,
								"tool_name", tc.Function.Name,
								"tool_args", truncate(tc.Function.Arguments, 300),
								"tool_call_id", tc.ID,
							)
						}
					}
				}

				// Log token usage if available
				if output.TokenUsage != nil {
					TokenUsage(agentName,
						int64(output.TokenUsage.PromptTokens),
						int64(output.TokenUsage.CompletionTokens),
						int64(output.TokenUsage.TotalTokens),
					)
				}

				duration := time.Duration(0)
				if startTime, ok := ctx.Value(modelStartTimeKey).(time.Time); ok {
					duration = time.Since(startTime)
				}

				ModelResult(agentName, hasToolCalls, contentLen,
					"duration_ms", duration.Milliseconds(),
				)
				return ctx
			},
			OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
				agentName := extractAgentName(info)
				Error("[MODEL_ERROR]", err,
					"agent", agentName,
					"component", info.Name,
				)
				return ctx
			},
		}).
		Tool(&template.ToolCallbackHandler{
			OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *tool.CallbackInput) context.Context {
				agentName := extractAgentName(info)
				args := ""
				if input != nil {
					args = input.ArgumentsInJSON
				}
				ToolCall(agentName, info.Name, args)
				return context.WithValue(ctx, toolStartTimeKey, time.Now())
			},
			OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *tool.CallbackOutput) context.Context {
				agentName := extractAgentName(info)
				duration := time.Duration(0)
				if startTime, ok := ctx.Value(toolStartTimeKey).(time.Time); ok {
					duration = time.Since(startTime)
				}

				result := ""
				if output != nil {
					result = output.Response
				}
				ToolResult(agentName, info.Name, result, duration)
				return ctx
			},
			OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
				agentName := extractAgentName(info)
				duration := time.Duration(0)
				if startTime, ok := ctx.Value(toolStartTimeKey).(time.Time); ok {
					duration = time.Since(startTime)
				}
				ToolError(agentName, info.Name, err, duration)
				return ctx
			},
		}).
		Handler()

	callbacks.AppendGlobalHandlers(handler)
	Info("Eino global callbacks registered")
}

// extractAgentName tries to get the agent/component name from RunInfo.
func extractAgentName(info *callbacks.RunInfo) string {
	if info == nil {
		return "unknown"
	}
	if info.Name != "" {
		return info.Name
	}
	return "unknown"
}
