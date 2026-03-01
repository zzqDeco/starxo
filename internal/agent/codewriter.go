package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"

	agenttools "starxo/internal/tools"
)

// NewCodeWriterAgent creates a sub-agent specialized in writing and editing code files.
// It uses the str_replace_editor tool from the commandline package to create and modify files.
// It also has read_file and list_files tools so it can inspect code without delegating.
func NewCodeWriterAgent(ctx context.Context, mdl model.ToolCallingChatModel,
	op commandline.Operator, ac AgentContext) (adk.Agent, error) {

	editor, err := commandline.NewStrReplaceEditor(ctx, &commandline.EditorConfig{
		Operator: op,
	})
	if err != nil {
		return nil, err
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

	listFilesTool, err := toolutils.InferTool("list_files",
		"List files in the specified directory up to 3 levels deep.",
		func(ctx context.Context, input ListFilesInput) (ListFilesOutput, error) {
			path := input.Path
			if path == "" {
				path = ac.WorkspacePath
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
		return nil, fmt.Errorf("failed to create list_files tool: %w", err)
	}

	// Assemble tools and wrap with event emission for frontend visibility
	allTools := []tool.BaseTool{editor, readFileTool, listFilesTool, agenttools.NewFollowUpTool(), agenttools.NewChoiceTool(), agenttools.NewNotifyUserTool()}
	allTools = WrapToolsWithEvents("code_writer", allTools, ac)

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "code_writer",
		Description: "Primary agent for all code tasks: reading files, listing directories, creating new files, editing existing code, and refactoring.",
		Instruction: CodeWriterPrompt(ac),
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
