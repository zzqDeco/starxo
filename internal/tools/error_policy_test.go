package tools

import (
	"errors"
	"strings"
	"testing"
)

func TestClassifyToolError_RecoverableViewRange(t *testing.T) {
	err := errors.New("failed to stream tool call str_replace_editor: invalid `view_range`: [900 950]. Its second element `950` should be less than the number of lines in the file: `940`")
	d := ClassifyToolError("str_replace_editor", `{"path":"/workspace/a.go","view_range":[900,950]}`, err)
	if !d.Recoverable {
		t.Fatalf("expected recoverable=true, got false")
	}
	if !strings.Contains(d.NormalizedMsg, "hint: adjust view_range") {
		t.Fatalf("expected view_range hint in normalized msg, got: %s", d.NormalizedMsg)
	}
	if !strings.Contains(d.Signature, "view_range_oob") {
		t.Fatalf("expected view_range signature, got: %s", d.Signature)
	}
}

func TestClassifyToolError_RecoverableOldStr(t *testing.T) {
	err := errors.New("str_replace_editor failed: old_str not found in file")
	d := ClassifyToolError("str_replace_editor", `{"path":"/workspace/a.go","old_str":"x"}`, err)
	if !d.Recoverable {
		t.Fatalf("expected recoverable=true, got false")
	}
	if !strings.Contains(d.Signature, "old_str_not_found") {
		t.Fatalf("expected old_str signature, got: %s", d.Signature)
	}
}

func TestClassifyToolError_FatalByDefault(t *testing.T) {
	err := errors.New("permission denied")
	d := ClassifyToolError("shell_execute", `{"command":"rm -rf /"}`, err)
	if d.Recoverable {
		t.Fatalf("expected recoverable=false, got true")
	}
	if !strings.Contains(d.Signature, "fatal") {
		t.Fatalf("expected fatal signature, got: %s", d.Signature)
	}
}

func TestClassifyToolError_RecoverableReadFileMissingPath(t *testing.T) {
	err := errors.New("failed to read file /workspace/package.json: failed to read file /workspace/package.json (exit 1): cat: /workspace/package.json: No such file or directory")
	d := ClassifyToolError("read_file", `{"path":"/workspace/package.json"}`, err)
	if !d.Recoverable {
		t.Fatalf("expected read_file missing path to be recoverable")
	}
	if !strings.Contains(d.Signature, "path_not_found") {
		t.Fatalf("expected path_not_found signature, got: %s", d.Signature)
	}
	if !strings.Contains(d.NormalizedMsg, "hint: file may not exist") {
		t.Fatalf("expected missing-path hint in message, got: %s", d.NormalizedMsg)
	}
}
