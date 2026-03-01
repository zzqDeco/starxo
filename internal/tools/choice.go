package tools

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

// Option represents a single choice option presented to the user.
type Option struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// ChoiceInfo is the information presented to the user during an interrupt.
type ChoiceInfo struct {
	Question string   `json:"question"`
	Options  []Option `json:"options"`
	Selected int      `json:"selected"` // populated by the user (0-indexed)
}

func (ci *ChoiceInfo) String() string {
	s := fmt.Sprintf("Please choose: %s\n", ci.Question)
	for i, opt := range ci.Options {
		s += fmt.Sprintf("  %d. %s — %s\n", i+1, opt.Label, opt.Description)
	}
	return s
}

// ChoiceState is the state saved during the interrupt for resumption.
type ChoiceState struct {
	Question string   `json:"question"`
	Options  []Option `json:"options"`
}

// ChoiceToolInput defines the input schema for the Choice tool.
type ChoiceToolInput struct {
	Question string   `json:"question" jsonschema:"description=the question to present to the user"`
	Options  []Option `json:"options" jsonschema:"description=the list of options for the user to choose from"`
}

func init() {
	schema.Register[*ChoiceInfo]()
	schema.Register[*ChoiceState]()
}

// choice is the core function that implements the interrupt/resume pattern for choices.
func choice(ctx context.Context, input *ChoiceToolInput) (string, error) {
	wasInterrupted, _, storedState := tool.GetInterruptState[*ChoiceState](ctx)

	if !wasInterrupted {
		// First call: trigger interrupt with options
		info := &ChoiceInfo{
			Question: input.Question,
			Options:  input.Options,
		}
		state := &ChoiceState{
			Question: input.Question,
			Options:  input.Options,
		}
		return "", tool.StatefulInterrupt(ctx, info, state)
	}

	// Resumed after interrupt
	isResumeTarget, hasData, resumeData := tool.GetResumeContext[*ChoiceInfo](ctx)

	if !isResumeTarget {
		// Not the target of this resume; re-interrupt
		info := &ChoiceInfo{
			Question: storedState.Question,
			Options:  storedState.Options,
		}
		return "", tool.StatefulInterrupt(ctx, info, storedState)
	}

	if !hasData {
		return "", fmt.Errorf("tool resumed without user selection")
	}

	selected := resumeData.Selected
	if selected < 0 || selected >= len(storedState.Options) {
		return "", fmt.Errorf("invalid selection index: %d", selected)
	}

	return fmt.Sprintf("User selected: %s — %s",
		storedState.Options[selected].Label,
		storedState.Options[selected].Description,
	), nil
}

// NewChoiceTool creates the Choice tool that interrupts execution
// to present structured options for the user to select from.
func NewChoiceTool() tool.BaseTool {
	t, err := utils.InferTool(
		"ask_choice",
		"Presents a structured set of options for the user to choose from. Use this when you need the user to make a decision between specific alternatives.",
		choice,
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create ask_choice tool: %v", err))
	}
	return t
}
