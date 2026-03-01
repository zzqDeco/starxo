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

// ListFilesInput is the input for the list_files tool.
type ListFilesInput struct {
	Path string `json:"path" jsonschema:"description=the directory path to list files from"`
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

// NewFileManagerAgent creates a sub-agent specialized in managing files in the workspace.
// It provides tools for listing, reading, and writing files.
func NewFileManagerAgent(ctx context.Context, mdl model.ToolCallingChatModel,
	op commandline.Operator, ac AgentContext) (adk.Agent, error) {

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
		return nil, fmt.Errorf("failed to create write_file tool: %w", err)
	}

	// Assemble tools and wrap with event emission for frontend visibility
	allTools := []tool.BaseTool{listFilesTool, readFileTool, writeFileTool, agenttools.NewFollowUpTool(), agenttools.NewChoiceTool(), agenttools.NewNotifyUserTool()}
	allTools = WrapToolsWithEvents("file_manager", allTools, ac)

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "file_manager",
		Description: "Agent for bulk file operations, workspace exploration, and writing non-code content. Use code_writer for code-related file operations.",
		Instruction: FileManagerPrompt(ac),
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
