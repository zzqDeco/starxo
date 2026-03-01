package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
)

// ShellInput is the input for the shell_execute tool.
type ShellInput struct {
	Command string `json:"command" jsonschema:"description=the shell command to execute"`
}

// ShellOutput is the output of the shell_execute tool.
type ShellOutput struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

// ListFilesInput is the input for the list_files tool.
type ListFilesInput struct {
	Path string `json:"path" jsonschema:"description=the directory path to list files from (defaults to /workspace)"`
}

// ListFilesOutput is the output of the list_files tool.
type ListFilesOutput struct {
	Files []string `json:"files"`
}

// ReadFileInput is the input for the read_file tool.
type ReadFileInput struct {
	Path string `json:"path" jsonschema:"description=the absolute path of the file to read"`
}

// ReadFileOutput is the output of the read_file tool.
type ReadFileOutput struct {
	Content string `json:"content"`
}

// WriteFileInput is the input for the write_file tool.
type WriteFileInput struct {
	Path    string `json:"path" jsonschema:"description=the absolute path of the file to write"`
	Content string `json:"content" jsonschema:"description=the content to write to the file"`
}

// WriteFileOutput is the output of the write_file tool.
type WriteFileOutput struct {
	Success bool `json:"success"`
}

// RegisterBuiltinTools creates and registers all built-in tools into the registry.
// This includes custom InferTool-based tools and Eino built-in commandline tools.
// The workspacePath parameter sets the default directory for list_files.
func RegisterBuiltinTools(registry *ToolRegistry, op commandline.Operator, workspacePath string) error {
	ctx := context.Background()

	// shell_execute tool
	shellTool, err := toolutils.InferTool("shell_execute",
		"Execute a shell command in the sandbox environment and return stdout, stderr, and exit code.",
		func(ctx context.Context, input ShellInput) (ShellOutput, error) {
			output, err := op.RunCommand(ctx, []string{"sh", "-c", input.Command})
			if err != nil {
				return ShellOutput{}, fmt.Errorf("shell execution failed: %w", err)
			}
			return ShellOutput{
				Stdout:   output.Stdout,
				Stderr:   output.Stderr,
				ExitCode: output.ExitCode,
			}, nil
		})
	if err != nil {
		return fmt.Errorf("failed to create shell_execute tool: %w", err)
	}
	if err := registry.RegisterBuiltin(shellTool); err != nil {
		return err
	}

	// list_files tool
	listFilesTool, err := toolutils.InferTool("list_files",
		"List files in the specified directory up to 3 levels deep.",
		func(ctx context.Context, input ListFilesInput) (ListFilesOutput, error) {
			path := input.Path
			if path == "" {
				path = workspacePath
			}
			output, err := op.RunCommand(ctx, []string{
				"find", path, "-maxdepth", "3", "-type", "f",
			})
			if err != nil {
				return ListFilesOutput{}, fmt.Errorf("failed to list files: %w", err)
			}
			files := strings.Split(strings.TrimSpace(output.Stdout), "\n")
			if len(files) == 1 && files[0] == "" {
				files = []string{}
			}
			return ListFilesOutput{Files: files}, nil
		})
	if err != nil {
		return fmt.Errorf("failed to create list_files tool: %w", err)
	}
	if err := registry.RegisterBuiltin(listFilesTool); err != nil {
		return err
	}

	// read_file tool
	readFileTool, err := toolutils.InferTool("read_file",
		"Read the content of a file at the specified path.",
		func(ctx context.Context, input ReadFileInput) (ReadFileOutput, error) {
			content, err := op.ReadFile(ctx, input.Path)
			if err != nil {
				return ReadFileOutput{}, fmt.Errorf("failed to read file %s: %w", input.Path, err)
			}
			return ReadFileOutput{Content: content}, nil
		})
	if err != nil {
		return fmt.Errorf("failed to create read_file tool: %w", err)
	}
	if err := registry.RegisterBuiltin(readFileTool); err != nil {
		return err
	}

	// write_file tool
	writeFileTool, err := toolutils.InferTool("write_file",
		"Write content to a file at the specified path, creating it if it does not exist.",
		func(ctx context.Context, input WriteFileInput) (WriteFileOutput, error) {
			err := op.WriteFile(ctx, input.Path, input.Content)
			if err != nil {
				return WriteFileOutput{Success: false}, fmt.Errorf("failed to write file %s: %w", input.Path, err)
			}
			return WriteFileOutput{Success: true}, nil
		})
	if err != nil {
		return fmt.Errorf("failed to create write_file tool: %w", err)
	}
	if err := registry.RegisterBuiltin(writeFileTool); err != nil {
		return err
	}

	// str_replace_editor (Eino built-in) — wrapped with argument sanitizer
	editor, err := commandline.NewStrReplaceEditor(ctx, &commandline.EditorConfig{
		Operator: op,
	})
	if err != nil {
		return fmt.Errorf("failed to create str_replace_editor: %w", err)
	}
	if err := registry.RegisterBuiltin(&sanitizedTool{inner: editor}); err != nil {
		return err
	}

	// python_execute (Eino built-in)
	pyExec, err := commandline.NewPyExecutor(ctx, &commandline.PyExecutorConfig{
		Command:  "python3",
		Operator: op,
	})
	if err != nil {
		return fmt.Errorf("failed to create python_execute tool: %w", err)
	}
	if err := registry.RegisterBuiltin(pyExec); err != nil {
		return err
	}

	return nil
}

// sanitizedTool wraps an InvokableTool and strips control characters from
// string fields in the JSON arguments before forwarding to the inner tool.
// This prevents failures caused by LLMs occasionally emitting stray control
// characters (e.g. \x06, \x01) in tool-call arguments.
type sanitizedTool struct {
	inner tool.BaseTool
}

func (s *sanitizedTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return s.inner.Info(ctx)
}

func (s *sanitizedTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	cleaned := sanitizeJSONStringValues(argumentsInJSON)
	if inv, ok := s.inner.(tool.InvokableTool); ok {
		return inv.InvokableRun(ctx, cleaned, opts...)
	}
	return "", fmt.Errorf("inner tool does not implement InvokableTool")
}

// sanitizeJSONStringValues parses the JSON, strips control characters from
// all string values, and re-serializes. Falls back to the original if parsing fails.
func sanitizeJSONStringValues(input string) string {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(input), &m); err != nil {
		return input
	}
	sanitizeMapValues(m)
	out, err := json.Marshal(m)
	if err != nil {
		return input
	}
	return string(out)
}

func sanitizeMapValues(m map[string]interface{}) {
	for k, v := range m {
		switch val := v.(type) {
		case string:
			m[k] = stripControlChars(val)
		case map[string]interface{}:
			sanitizeMapValues(val)
		case []interface{}:
			for i, item := range val {
				if s, ok := item.(string); ok {
					val[i] = stripControlChars(s)
				}
			}
		}
	}
}

// stripControlChars removes control characters (0x00-0x08, 0x0B, 0x0C, 0x0E-0x1F)
// while preserving \n (0x0A), \t (0x09), and \r (0x0D).
func stripControlChars(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\n' && r != '\t' && r != '\r' {
			return -1 // drop
		}
		if unicode.Is(unicode.Co, r) { // private use / unassigned
			return -1
		}
		return r
	}, s)
}
