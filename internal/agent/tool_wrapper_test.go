package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type scriptedToolRun struct {
	result string
	err    error
}

type scriptedInvokableTool struct {
	name    string
	runs    []scriptedToolRun
	current int
}

func (s *scriptedInvokableTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: s.name}, nil
}

func (s *scriptedInvokableTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	if s.current >= len(s.runs) {
		return "", errors.New("unexpected extra call")
	}
	r := s.runs[s.current]
	s.current++
	return r.result, r.err
}

func viewRangeErr() error {
	return errors.New("invalid `view_range`: [900 950]. Its second element `950` should be less than the number of lines in the file: `940`")
}

func missingFileErr() error {
	return errors.New("failed to read file /workspace/package.json: failed to read file /workspace/package.json (exit 1): cat: /workspace/package.json: No such file or directory")
}

func TestEventEmittingTool_RecoverableErrorReturnsNilErr(t *testing.T) {
	inner := &scriptedInvokableTool{
		name: "str_replace_editor",
		runs: []scriptedToolRun{
			{err: viewRangeErr()},
		},
	}
	w := &eventEmittingTool{
		inner:               inner,
		agentName:           "code_writer",
		toolName:            "str_replace_editor",
		recoverableErrCount: make(map[string]int),
	}

	ctx := context.WithValue(context.Background(), "sessionID", "s1")
	result, err := w.InvokableRun(ctx, `{"path":"/workspace/a.go","view_range":[900,950]}`)
	if err != nil {
		t.Fatalf("expected nil err for recoverable tool error, got: %v", err)
	}
	if !strings.Contains(result, "hint: adjust view_range") {
		t.Fatalf("expected recoverable hint in result, got: %s", result)
	}
}

func TestEventEmittingTool_RecoverableEscalatesAfterThreshold(t *testing.T) {
	inner := &scriptedInvokableTool{
		name: "str_replace_editor",
		runs: []scriptedToolRun{
			{err: viewRangeErr()},
			{err: viewRangeErr()},
			{err: viewRangeErr()},
		},
	}
	w := &eventEmittingTool{
		inner:               inner,
		agentName:           "code_writer",
		toolName:            "str_replace_editor",
		recoverableErrCount: make(map[string]int),
	}
	ctx := context.WithValue(context.Background(), "sessionID", "s1")
	args := `{"path":"/workspace/a.go","view_range":[900,950]}`

	for i := 0; i < 2; i++ {
		if _, err := w.InvokableRun(ctx, args); err != nil {
			t.Fatalf("expected call %d to be recoverable, got err: %v", i+1, err)
		}
	}
	if _, err := w.InvokableRun(ctx, args); err == nil {
		t.Fatalf("expected third repeated recoverable error to escalate")
	}
}

func TestEventEmittingTool_SuccessResetsRecoverableBackoff(t *testing.T) {
	inner := &scriptedInvokableTool{
		name: "str_replace_editor",
		runs: []scriptedToolRun{
			{err: viewRangeErr()},
			{err: viewRangeErr()},
			{result: "ok", err: nil},
			{err: viewRangeErr()},
			{err: viewRangeErr()},
			{err: viewRangeErr()},
		},
	}
	w := &eventEmittingTool{
		inner:               inner,
		agentName:           "code_writer",
		toolName:            "str_replace_editor",
		recoverableErrCount: make(map[string]int),
	}
	ctx := context.WithValue(context.Background(), "sessionID", "s1")
	args := `{"path":"/workspace/a.go","view_range":[900,950]}`

	if _, err := w.InvokableRun(ctx, args); err != nil {
		t.Fatalf("first recoverable call should not fail: %v", err)
	}
	if _, err := w.InvokableRun(ctx, args); err != nil {
		t.Fatalf("second recoverable call should not fail: %v", err)
	}

	if got, err := w.InvokableRun(ctx, `{"path":"/workspace/a.go","command":"view"}`); err != nil {
		t.Fatalf("success call should not fail: %v", err)
	} else if got != "ok" {
		t.Fatalf("unexpected success result: %q", got)
	}

	if _, err := w.InvokableRun(ctx, args); err != nil {
		t.Fatalf("recoverable backoff should be reset after success, got err: %v", err)
	}
	if _, err := w.InvokableRun(ctx, args); err != nil {
		t.Fatalf("recoverable backoff should still be below threshold, got err: %v", err)
	}
	if _, err := w.InvokableRun(ctx, args); err == nil {
		t.Fatalf("expected escalation only after threshold is reached again")
	}
}

func TestEventEmittingTool_NonRecoverableStillFails(t *testing.T) {
	inner := &scriptedInvokableTool{
		name: "shell_execute",
		runs: []scriptedToolRun{
			{err: errors.New("permission denied")},
		},
	}
	w := &eventEmittingTool{
		inner:               inner,
		agentName:           "code_executor",
		toolName:            "shell_execute",
		recoverableErrCount: make(map[string]int),
	}
	ctx := context.WithValue(context.Background(), "sessionID", "s1")

	if _, err := w.InvokableRun(ctx, `{"command":"rm -rf /"}`); err == nil {
		t.Fatalf("expected non-recoverable error to fail")
	}
}

func TestEventEmittingTool_ReadFileMissingPathIsRecoverable(t *testing.T) {
	inner := &scriptedInvokableTool{
		name: "read_file",
		runs: []scriptedToolRun{
			{err: missingFileErr()},
		},
	}
	w := &eventEmittingTool{
		inner:               inner,
		agentName:           "code_writer",
		toolName:            "read_file",
		recoverableErrCount: make(map[string]int),
	}
	ctx := context.WithValue(context.Background(), "sessionID", "s1")

	result, err := w.InvokableRun(ctx, `{"path":"/workspace/package.json"}`)
	if err != nil {
		t.Fatalf("expected missing read_file path to be recoverable, got err: %v", err)
	}
	if !strings.Contains(result, "file may not exist at this path") {
		t.Fatalf("expected missing-path hint in result, got: %s", result)
	}
}
