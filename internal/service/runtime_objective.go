package service

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/cloudwego/eino/schema"

	"starxo/internal/model"
)

func objectiveScope(userMessage string) string {
	msg := normalizeObjectiveScopeMessage(userMessage)
	if msg == "" {
		return "standalone"
	}
	exactContinuationSignals := map[string]bool{
		"continue": true, "continue please": true, "go on": true, "proceed": true, "resume": true,
		"next step": true, "what next": true, "do it": true, "do that": true,
		"do the above": true, "same task": true, "that task": true, "previous task": true,
		"继续": true, "下一步": true, "下步": true, "接着": true, "继续做": true, "继续执行": true,
		"完成上面": true, "做完上面": true,
	}
	if exactContinuationSignals[msg] {
		return "continuation"
	}
	if strings.HasPrefix(msg, "continue ") {
		return "continuation"
	}
	if strings.HasPrefix(msg, "resume ") && resumeContinuationIntent(msg) {
		return "continuation"
	}
	if strings.Contains(msg, "the above") && containsAny(msg, "do", "implement", "finish", "complete", "fix", "address", "continue") {
		return "continuation"
	}
	if strings.Contains(msg, "previous") && containsAny(msg, "continue", "resume", "task") {
		return "continuation"
	}
	if strings.HasPrefix(msg, "继续") || strings.HasPrefix(msg, "接着") || strings.HasPrefix(msg, "下一步") {
		return "continuation"
	}
	if containsAny(msg, "上面", "上述", "刚才", "之前") && containsAny(msg, "完成", "做", "修复", "继续", "接着", "按照") {
		return "continuation"
	}
	return "standalone"
}

func normalizeObjectiveScopeMessage(userMessage string) string {
	msg := strings.ToLower(strings.TrimSpace(userMessage))
	msg = strings.Trim(msg, " \t\r\n.!?。！？,，;；:：")
	return strings.Join(strings.Fields(msg), " ")
}

func resumeContinuationIntent(msg string) bool {
	return containsAny(msg,
		"above", "interrupted",
		"previous task", "previous work", "prior task", "prior work", "last task", "last work", "earlier task", "earlier work",
		"current task", "current work", "this task", "this work", "that task", "that work", "the task",
		"where we left", "where you left",
	)
}

func containsAny(text string, terms ...string) bool {
	for _, term := range terms {
		if containsContinuationTerm(text, term) {
			return true
		}
	}
	return false
}

func containsContinuationTerm(text, term string) bool {
	term = strings.TrimSpace(term)
	if term == "" {
		return false
	}
	for _, r := range term {
		if r > 127 {
			return strings.Contains(text, term)
		}
	}
	pattern := `(?:^|[^a-z0-9_])` + regexp.QuoteMeta(term) + `(?:$|[^a-z0-9_])`
	return regexp.MustCompile(pattern).FindStringIndex(text) != nil
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
	msg := schema.UserMessage(content)
	msg.Extra = map[string]any{"starxo_current_objective": true}
	return []*schema.Message{msg}
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
