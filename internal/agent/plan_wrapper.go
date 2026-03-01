package agent

import (
	"context"
	"encoding/json"
	"log"
	"runtime/debug"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"

	"starxo/internal/logger"
)

// PlanEventCallback is called when the plan state changes, allowing the caller
// to emit events (e.g. to the frontend via Wails).
type PlanEventCallback func(plans []*FullPlan)

// NewPlanMDWrapper wraps a planner or replanner agent to persist the plan
// state to a Markdown file and emit plan events after each run.
func NewPlanMDWrapper(a adk.Agent, writeFunc func(plans []*FullPlan), onPlan PlanEventCallback) adk.Agent {
	return &planMDWrapper{
		a:         a,
		writeFunc: writeFunc,
		onPlan:    onPlan,
	}
}

type planMDWrapper struct {
	a         adk.Agent
	writeFunc func(plans []*FullPlan)
	onPlan    PlanEventCallback
}

func (w *planMDWrapper) Name(ctx context.Context) string {
	return w.a.Name(ctx)
}

func (w *planMDWrapper) Description(ctx context.Context) string {
	return w.a.Description(ctx)
}

func (w *planMDWrapper) Run(ctx context.Context, input *adk.AgentInput, options ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	iter := w.a.Run(ctx, input, options...)
	nIter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()

	go func() {
		defer func() {
			if e := recover(); e != nil {
				log.Printf("[planMDWrapper] panic recovered: %+v, stack: %s", e, string(debug.Stack()))
			}
			gen.Close()
		}()

		for {
			e, ok := iter.Next()
			if !ok {
				break
			}
			if e.Action != nil && e.Action.Exit {
				w.persistPlan(ctx)
				gen.Send(e)
				return
			}
			gen.Send(e)
		}

		// Also persist on normal completion
		w.persistPlan(ctx)
	}()

	return nIter
}

// persistPlan reads the plan and executed steps from the session context,
// builds a FullPlan slice, and writes it to Markdown + emits events.
func (w *planMDWrapper) persistPlan(ctx context.Context) {
	plans := w.buildPlans(ctx)
	if len(plans) == 0 {
		return
	}

	// Write to MD via the provided function
	if w.writeFunc != nil {
		w.writeFunc(plans)
	}

	// Emit plan event
	if w.onPlan != nil {
		w.onPlan(plans)
	}
}

// buildPlans extracts plan + executed steps from the ADK session and builds FullPlan slice.
func (w *planMDWrapper) buildPlans(ctx context.Context) []*FullPlan {
	var plans []*FullPlan

	// Get executed steps
	executedStepsRaw, hasExecuted := adk.GetSessionValue(ctx, planexecute.ExecutedStepsSessionKey)
	if hasExecuted {
		if executedSteps, ok := executedStepsRaw.([]planexecute.ExecutedStep); ok {
			for i, step := range executedSteps {
				desc := step.Step
				// Try to parse as JSON step for desc field
				var s Step
				if err := json.Unmarshal([]byte(step.Step), &s); err == nil && s.Desc != "" {
					desc = s.Desc
				}
				plans = append(plans, &FullPlan{
					TaskID:     i + 1,
					Status:     PlanStatusDone,
					Desc:       desc,
					ExecResult: step.Result,
				})
			}
		}
	}

	// Get remaining plan steps
	planRaw, hasPlan := adk.GetSessionValue(ctx, planexecute.PlanSessionKey)
	if hasPlan {
		// The plan value is a json.RawMessage containing the plan structure
		if planBytes, ok := planRaw.(json.RawMessage); ok {
			var plan Plan
			if err := json.Unmarshal(planBytes, &plan); err == nil {
				for i, step := range plan.Steps {
					plans = append(plans, &FullPlan{
						TaskID: i + len(plans) + 1,
						Status: PlanStatusTodo,
						Desc:   step.Desc,
					})
				}
			}
		}
	}

	if len(plans) > 0 {
		logger.Info("[PLAN] Plan state updated", "total_steps", len(plans))
	}

	return plans
}
