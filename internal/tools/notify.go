package tools

import (
	"context"
	"fmt"

	toolutils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/components/tool"
)

// NotifyInput is the input for the notify_user tool.
type NotifyInput struct {
	Message string `json:"message" jsonschema:"description=a brief status update message to show the user (keep it short and informative)"`
}

// NewNotifyUserTool creates a tool that lets the agent send brief status updates
// to the user without stopping execution. The message appears as an inline
// info banner in the chat timeline.
func NewNotifyUserTool() tool.BaseTool {
	t, err := toolutils.InferTool(
		"notify_user",
		"Send a brief status update to the user without stopping your work. Use this to keep the user informed about what you are currently doing or what progress you have made. Unlike ask_user, this does NOT pause execution — you can continue working immediately.",
		func(ctx context.Context, input *NotifyInput) (string, error) {
			if input.Message == "" {
				return "No message provided.", nil
			}
			return fmt.Sprintf("[Status] %s", input.Message), nil
		},
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create notify_user tool: %v", err))
	}
	return t
}
