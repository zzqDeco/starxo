package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

// FollowUpInfo is the information presented to the user during an interrupt.
// It is passed as the interrupt info and used to carry the user's answer back.
type FollowUpInfo struct {
	Questions  []string `json:"questions"`
	UserAnswer string   `json:"userAnswer"`
}

func (fi *FollowUpInfo) String() string {
	var sb strings.Builder
	sb.WriteString("We need more information. Please answer the following questions:\n")
	for i, q := range fi.Questions {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, q))
	}
	return sb.String()
}

// FollowUpState is the state saved during the interrupt for resumption.
type FollowUpState struct {
	Questions []string `json:"questions"`
}

// FollowUpToolInput defines the input schema for the FollowUp tool.
type FollowUpToolInput struct {
	Questions []string `json:"questions" jsonschema:"description=a list of questions to ask the user for clarification"`
}

func init() {
	schema.Register[*FollowUpInfo]()
	schema.Register[*FollowUpState]()
}

// followUp is the core function that implements the interrupt/resume pattern.
func followUp(ctx context.Context, input *FollowUpToolInput) (string, error) {
	wasInterrupted, _, storedState := tool.GetInterruptState[*FollowUpState](ctx)

	if !wasInterrupted {
		// First call: trigger interrupt with questions
		info := &FollowUpInfo{Questions: input.Questions}
		state := &FollowUpState{Questions: input.Questions}
		return "", tool.StatefulInterrupt(ctx, info, state)
	}

	// Resumed after interrupt
	isResumeTarget, hasData, resumeData := tool.GetResumeContext[*FollowUpInfo](ctx)

	if !isResumeTarget {
		// Not the target of this resume; re-interrupt
		info := &FollowUpInfo{Questions: storedState.Questions}
		return "", tool.StatefulInterrupt(ctx, info, storedState)
	}

	if !hasData || resumeData.UserAnswer == "" {
		return "", fmt.Errorf("tool resumed without a user answer")
	}

	return resumeData.UserAnswer, nil
}

// NewFollowUpTool creates the FollowUp tool that interrupts execution
// to ask the user clarifying questions.
func NewFollowUpTool() tool.BaseTool {
	t, err := utils.InferTool(
		"ask_user",
		"Asks the user for more information by providing a list of questions. Use this when you need clarification before proceeding.",
		followUp,
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create ask_user tool: %v", err))
	}
	return t
}
