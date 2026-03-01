package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"

	agenttools "starxo/internal/tools"
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

// NewCodeExecutorAgent creates a sub-agent specialized in executing code and shell commands.
// It uses the python_execute tool and a custom shell_execute tool.
func NewCodeExecutorAgent(ctx context.Context, mdl model.ToolCallingChatModel,
	op commandline.Operator, ac AgentContext) (adk.Agent, error) {

	pyExec, err := commandline.NewPyExecutor(ctx, &commandline.PyExecutorConfig{
		Command:  "python3",
		Operator: op,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create python executor: %w", err)
	}

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
		return nil, fmt.Errorf("failed to create shell_execute tool: %w", err)
	}

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
		return nil, fmt.Errorf("failed to create read_file tool: %w", err)
	}

	// Assemble tools and wrap with event emission for frontend visibility
	allTools := []tool.BaseTool{pyExec, shellTool, readFileTool, agenttools.NewFollowUpTool(), agenttools.NewChoiceTool(), agenttools.NewNotifyUserTool()}
	allTools = WrapToolsWithEvents("code_executor", allTools, ac)

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "code_executor",
		Description: "Expert agent for executing Python code and shell commands. Can also read files to inspect scripts and output.",
		Instruction: CodeExecutorPrompt(ac),
		Model:       mdl,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: allTools,
			},
		},
		MaxIterations: 30,
	})
	if err != nil {
		return nil, err
	}

	return agent, nil
}
