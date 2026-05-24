package service

import (
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"

	"starxo/internal/model"
)

func objectiveScope(userMessage string) string {
	msg := strings.ToLower(strings.TrimSpace(userMessage))
	if msg == "" {
		return "standalone"
	}
	continuationSignals := []string{
		"continue", "next step", "what next", "do the above", "above", "previous", "that task", "same task",
		"继续", "下一步", "下步", "上面", "上述", "刚才", "之前", "接着", "完成上面", "做完上面",
	}
	for _, signal := range continuationSignals {
		if strings.Contains(msg, signal) {
			return "continuation"
		}
	}
	return "standalone"
}

func cloneRunObjective(in *model.RunObjective) *model.RunObjective {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func runtimeObjectivePinnedMessages(objective *model.RunObjective) []*schema.Message {
	if objective == nil {
		return nil
	}
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

Only this current objective is active. Earlier messages are historical context unless scope is continuation. Do not resume older debugging, release, review, or test work unless this objective explicitly asks for it.`,
		objective.ID,
		objective.RunID,
		scope,
		objective.UserMessageID,
		oneLineObjective(objective.Objective),
		oneLineObjective(objective.Acceptance),
	)
	return []*schema.Message{schema.UserMessage(content)}
}

func scopeCompactForObjective(compact *model.RuntimeContextCompact, objective *model.RunObjective) *model.RuntimeContextCompact {
	if compact == nil {
		if objective == nil {
			return nil
		}
		return &model.RuntimeContextCompact{
			Version:         model.RuntimeContextCompactVersion,
			ActiveObjective: cloneRunObjective(objective),
		}
	}
	out := model.CloneRuntimeContextCompact(compact)
	out.ActiveObjective = cloneRunObjective(objective)
	if objective != nil && objective.Scope == "standalone" {
		out.Summary = ""
		out.OmittedMessageCount = 0
		out.OriginalMessageCount = 0
		out.TokenEstimate = 0
		out.Tasks = nil
		out.FileReadState = nil
		out.DiffSummaries = nil
		out.Todos = nil
		out.PlanDocument = nil
	}
	return out
}

func oneLineObjective(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > 500 {
		return text[:497] + "..."
	}
	return text
}
