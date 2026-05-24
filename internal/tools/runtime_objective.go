package tools

import (
	"context"
	"strings"

	"starxo/internal/model"
)

type runtimeObjectiveContextKey string

const runtimeObjectiveCtxKey runtimeObjectiveContextKey = "starxoRuntimeObjective"

func ContextWithRuntimeObjective(ctx context.Context, objective *model.RunObjective) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if objective == nil {
		return ctx
	}
	cp := *objective
	return context.WithValue(ctx, runtimeObjectiveCtxKey, &cp)
}

func RuntimeObjectiveFromContext(ctx context.Context) (*model.RunObjective, bool) {
	if ctx == nil {
		return nil, false
	}
	obj, ok := ctx.Value(runtimeObjectiveCtxKey).(*model.RunObjective)
	if !ok || obj == nil {
		return nil, false
	}
	cp := *obj
	return &cp, true
}

func RuntimeObjectivePromptGuard(ctx context.Context, text string) (string, bool) {
	obj, ok := RuntimeObjectiveFromContext(ctx)
	if !ok || strings.TrimSpace(obj.Objective) == "" {
		return "", true
	}
	if objectiveQuestionLooksStale(obj.Objective, text) {
		return "This clarification is unrelated to the current user objective. Stay on the current objective or provide the final answer for it.", false
	}
	return "", true
}

func RuntimeObjectiveToolGuard(ctx context.Context, toolName, arguments string) (string, bool) {
	obj, ok := RuntimeObjectiveFromContext(ctx)
	if !ok || strings.TrimSpace(obj.Objective) == "" {
		return "", true
	}
	text := toolName + "\n" + arguments
	if objectiveQuestionLooksStale(obj.Objective, text) {
		return "Tool call rejected: it appears to pursue an older unrelated task. Return to the current user objective or finish it instead.", false
	}
	return "", true
}

func objectiveQuestionLooksStale(objective, text string) bool {
	objective = strings.ToLower(objective)
	text = strings.ToLower(text)
	if strings.TrimSpace(text) == "" {
		return false
	}
	staleFamilies := [][]string{
		{"debug", "failing test", "failing tests", "test failure", "tests failing", "测试失败", "失败的测试"},
		{"release", "tag", "publish", "发布"},
		{"review", "codex review", "pr review", "审查"},
	}
	for _, family := range staleFamilies {
		textHit := false
		objectiveHit := false
		for _, term := range family {
			if strings.Contains(text, term) {
				textHit = true
			}
			if strings.Contains(objective, term) {
				objectiveHit = true
			}
		}
		if textHit && !objectiveHit {
			return true
		}
	}
	return false
}
