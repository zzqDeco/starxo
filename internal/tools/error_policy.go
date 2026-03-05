package tools

import (
	"fmt"
	"hash/fnv"
	"strings"
)

// ToolErrorDecision represents how tool errors should be propagated to the
// agent runtime.
type ToolErrorDecision struct {
	Recoverable      bool
	NormalizedMsg    string
	Signature        string
	EscalationReason string
}

// ClassifyToolError decides whether a tool error can be surfaced back to the
// model as a normal tool result (recoverable) or must fail the node (fatal).
func ClassifyToolError(toolName, toolArgs string, err error) ToolErrorDecision {
	if err == nil {
		return ToolErrorDecision{}
	}

	raw := strings.TrimSpace(err.Error())
	normalized := fmt.Sprintf("Error: %s", raw)
	lower := strings.ToLower(raw)

	switch toolName {
	case "str_replace_editor":
		return classifyStrReplaceEditorError(lower, toolName, toolArgs, normalized, raw)
	case "read_file":
		if isMissingPathError(lower) {
			return ToolErrorDecision{
				Recoverable:   true,
				NormalizedMsg: normalized + "; hint: file may not exist at this path, list workspace files and retry with a valid absolute path",
				Signature:     fmt.Sprintf("%s:path_not_found:%s", toolName, shortHash(toolArgs)),
			}
		}
	case "list_files":
		if isMissingPathError(lower) {
			return ToolErrorDecision{
				Recoverable:   true,
				NormalizedMsg: normalized + "; hint: directory may not exist, verify path and retry",
				Signature:     fmt.Sprintf("%s:path_not_found:%s", toolName, shortHash(toolArgs)),
			}
		}
	}

	return ToolErrorDecision{
		Recoverable:   false,
		NormalizedMsg: normalized,
		Signature:     fmt.Sprintf("%s:fatal:%s", toolName, shortHash(raw)),
	}
}

func classifyStrReplaceEditorError(lower, toolName, toolArgs, normalized, raw string) ToolErrorDecision {
	if strings.Contains(lower, "invalid `view_range`") ||
		(strings.Contains(lower, "view_range") && strings.Contains(lower, "should be less than the number of lines")) {
		return ToolErrorDecision{
			Recoverable:   true,
			NormalizedMsg: normalized + "; hint: adjust view_range to the file's actual line count and retry",
			Signature:     fmt.Sprintf("%s:view_range_oob:%s", toolName, shortHash(toolArgs)),
		}
	}

	if strings.Contains(lower, "old_str") && strings.Contains(lower, "not found") {
		return ToolErrorDecision{
			Recoverable:   true,
			NormalizedMsg: normalized + "; hint: refresh file content and retry with an exact old_str match",
			Signature:     fmt.Sprintf("%s:old_str_not_found:%s", toolName, shortHash(toolArgs)),
		}
	}

	if strings.Contains(lower, "invalid line range") {
		return ToolErrorDecision{
			Recoverable:   true,
			NormalizedMsg: normalized + "; hint: use a valid line range within file bounds and retry",
			Signature:     fmt.Sprintf("%s:invalid_line_range:%s", toolName, shortHash(toolArgs)),
		}
	}

	return ToolErrorDecision{
		Recoverable:   false,
		NormalizedMsg: normalized,
		Signature:     fmt.Sprintf("%s:fatal:%s", toolName, shortHash(raw)),
	}
}

func isMissingPathError(lowerErr string) bool {
	return strings.Contains(lowerErr, "no such file or directory") ||
		strings.Contains(lowerErr, "cannot find the file specified") ||
		strings.Contains(lowerErr, "file does not exist")
}

func shortHash(s string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum64())
}
