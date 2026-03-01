package agent

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/cloudwego/eino-ext/components/tool/commandline"
)

// Step represents a single step in the plan.
type Step struct {
	Index int    `json:"index"`
	Desc  string `json:"desc"`
}

// Plan represents a structured execution plan with ordered steps.
type Plan struct {
	Steps []Step `json:"steps"`
}

// PlanStatus represents the execution status of a plan step.
type PlanStatus string

const (
	PlanStatusTodo    PlanStatus = "todo"
	PlanStatusDoing   PlanStatus = "doing"
	PlanStatusDone    PlanStatus = "done"
	PlanStatusFailed  PlanStatus = "failed"
	PlanStatusSkipped PlanStatus = "skipped"
)

// FullPlan combines a plan step with its execution status and result.
type FullPlan struct {
	TaskID     int        `json:"task_id,omitempty"`
	Status     PlanStatus `json:"status,omitempty"`
	AgentName  string     `json:"agent_name,omitempty"`
	Desc       string     `json:"desc,omitempty"`
	ExecResult string     `json:"exec_result,omitempty"`
}

// PlanString formats a single plan step as a Markdown checkbox line.
func (p *FullPlan) PlanString(n int) string {
	switch p.Status {
	case PlanStatusDone:
		return fmt.Sprintf("- [x] %d. %s", n, p.Desc)
	case PlanStatusDoing:
		return fmt.Sprintf("- [~] %d. %s *(executing...)*", n, p.Desc)
	case PlanStatusFailed:
		return fmt.Sprintf("- [!] %d. %s *(failed)*", n, p.Desc)
	case PlanStatusSkipped:
		return fmt.Sprintf("- [-] %d. %s *(skipped)*", n, p.Desc)
	default:
		return fmt.Sprintf("- [ ] %d. %s", n, p.Desc)
	}
}

// FormatPlanMD formats a slice of FullPlan into a Markdown document.
func FormatPlanMD(plans []*FullPlan) string {
	md := "# Execution Plan\n\n"
	for i, p := range plans {
		md += p.PlanString(i+1) + "\n"
	}
	return md
}

// WritePlanMD writes the plan as a Markdown file in the given workspace directory.
func WritePlanMD(ctx context.Context, op commandline.Operator, workspacePath string, plans []*FullPlan) error {
	content := FormatPlanMD(plans)
	filePath := filepath.Join(workspacePath, "plan.md")
	return op.WriteFile(ctx, filePath, content)
}
