package tools

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
)

const (
	RuntimeToolBash          = "Bash"
	RuntimeToolAgent         = "Agent"
	RuntimeToolRead          = "Read"
	RuntimeToolWrite         = "Write"
	RuntimeToolEdit          = "Edit"
	RuntimeToolGlob          = "Glob"
	RuntimeToolGrep          = "Grep"
	RuntimeToolTaskOutput    = "TaskOutput"
	RuntimeToolTaskStop      = "TaskStop"
	RuntimeToolExitPlanMode  = "ExitPlanMode"
	RuntimeToolEnterWorktree = "EnterWorktree"
	RuntimeToolExitWorktree  = "ExitWorktree"
	RuntimeToolWorktreeDiff  = "WorktreeDiff"
	RuntimeToolWorktreeMerge = "WorktreeMerge"

	runtimeLargeOutputThreshold = 32 * 1024
	runtimeToolPatchLimit       = 20 * 1024
	runtimeWriteOldPreviewLimit = runtimeToolPatchLimit / 3
)

type runtimeFilePreviewReader interface {
	ReadFilePreview(ctx context.Context, path string, maxBytes int) (content string, bytes int64, lines int, exists bool, truncated bool, err error)
}

type RuntimeTaskRef struct {
	TaskID     string `json:"taskId"`
	Status     string `json:"status"`
	OutputPath string `json:"outputPath,omitempty"`
}

type RuntimeTaskSnapshot struct {
	ID          string `json:"id"`
	SessionID   string `json:"sessionId"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Description string `json:"description"`
	Command     string `json:"command,omitempty"`
	OutputPath  string `json:"outputPath,omitempty"`
	OutputSize  int64  `json:"outputSize,omitempty"`
	StartedAt   int64  `json:"startedAt"`
	FinishedAt  int64  `json:"finishedAt,omitempty"`
	DurationMs  int64  `json:"durationMs,omitempty"`
	ExitCode    int    `json:"exitCode,omitempty"`
	Error       string `json:"error,omitempty"`
}

type RuntimeTaskOutput struct {
	TaskID     string `json:"taskId"`
	Status     string `json:"status"`
	OutputPath string `json:"outputPath,omitempty"`
	Content    string `json:"content"`
	Offset     int    `json:"offset"`
	NextOffset int    `json:"nextOffset"`
	Size       int64  `json:"size"`
	Truncated  bool   `json:"truncated"`
}

type RuntimeTaskRunner func(ctx context.Context) (BashOutput, error)

type RuntimeTaskManager interface {
	StartShellTask(ctx context.Context, sessionID, command, description string, runner RuntimeTaskRunner) (RuntimeTaskRef, error)
	ReadTaskOutput(ctx context.Context, taskID string, offset, limit int) (RuntimeTaskOutput, error)
	StopTask(ctx context.Context, taskID string) (RuntimeTaskSnapshot, error)
	PersistToolResult(ctx context.Context, sessionID, prefix, content string) (path string, size int64, err error)
}

type RuntimeWorkspaceManager interface {
	CurrentWorkspace(ctx context.Context, defaultWorkspace string) string
	EnterWorktree(ctx context.Context, op commandline.Operator, defaultWorkspace, name string) (WorktreeOutput, error)
	ExitWorktree(ctx context.Context, op commandline.Operator, defaultWorkspace, action string, discardChanges bool) (WorktreeOutput, error)
	DiffWorktree(ctx context.Context, op commandline.Operator, defaultWorkspace string, includePatch bool, maxBytes int) (WorktreeDiffOutput, error)
	MergeWorktree(ctx context.Context, op commandline.Operator, defaultWorkspace string, commitMessage string, removeWorktree bool) (WorktreeMergeOutput, error)
}

type BashInput struct {
	Command         string `json:"command" jsonschema:"description=the shell command to execute"`
	Description     string `json:"description,omitempty" jsonschema:"description=short active-voice description of what this command does"`
	TimeoutMs       int    `json:"timeout_ms,omitempty" jsonschema:"description=optional timeout in milliseconds"`
	RunInBackground bool   `json:"run_in_background,omitempty" jsonschema:"description=set true to run this command in the background"`
}

type BashOutput struct {
	Stdout              string `json:"stdout"`
	Stderr              string `json:"stderr"`
	ExitCode            int    `json:"exit_code"`
	Interrupted         bool   `json:"interrupted"`
	BackgroundTaskID    string `json:"backgroundTaskId,omitempty"`
	OutputPath          string `json:"outputPath,omitempty"`
	PersistedOutputPath string `json:"persistedOutputPath,omitempty"`
	PersistedOutputSize int64  `json:"persistedOutputSize,omitempty"`
}

type ReadInput struct {
	FilePath string `json:"file_path" jsonschema:"description=file path inside the sandbox workspace"`
	Offset   int    `json:"offset,omitempty" jsonschema:"description=zero-based line offset"`
	Limit    int    `json:"limit,omitempty" jsonschema:"description=max number of lines to return"`
}

type ReadOutput struct {
	FilePath   string `json:"filePath"`
	Content    string `json:"content"`
	NumLines   int    `json:"numLines"`
	StartLine  int    `json:"startLine"`
	TotalLines int    `json:"totalLines"`
}

type WriteInput struct {
	FilePath string `json:"file_path" jsonschema:"description=file path inside the sandbox workspace"`
	Content  string `json:"content" jsonschema:"description=content to write"`
}

type WriteOutput struct {
	FilePath     string `json:"filePath"`
	Created      bool   `json:"created"`
	Bytes        int    `json:"bytes"`
	LinesAdded   int    `json:"linesAdded"`
	LinesRemoved int    `json:"linesRemoved,omitempty"`
	Patch        string `json:"patch,omitempty"`
	Truncated    bool   `json:"truncated,omitempty"`
}

type EditInput struct {
	FilePath   string `json:"file_path" jsonschema:"description=file path inside the sandbox workspace"`
	OldString  string `json:"old_string" jsonschema:"description=exact text to replace"`
	NewString  string `json:"new_string" jsonschema:"description=replacement text"`
	ReplaceAll bool   `json:"replace_all,omitempty" jsonschema:"description=replace every occurrence instead of exactly one"`
}

type EditOutput struct {
	FilePath     string `json:"filePath"`
	Replacements int    `json:"replacements"`
	LinesAdded   int    `json:"linesAdded"`
	LinesRemoved int    `json:"linesRemoved"`
	Patch        string `json:"patch,omitempty"`
	Truncated    bool   `json:"truncated,omitempty"`
}

type GlobInput struct {
	Pattern string `json:"pattern" jsonschema:"description=glob pattern to match files"`
	Path    string `json:"path,omitempty" jsonschema:"description=directory to search in; defaults to workspace"`
	Limit   int    `json:"limit,omitempty" jsonschema:"description=max files to return"`
	Offset  int    `json:"offset,omitempty" jsonschema:"description=files to skip before returning results"`
}

type GlobOutput struct {
	DurationMs int      `json:"durationMs"`
	NumFiles   int      `json:"numFiles"`
	Filenames  []string `json:"filenames"`
	Truncated  bool     `json:"truncated"`
}

type GrepInput struct {
	Pattern    string `json:"pattern" jsonschema:"description=regular expression pattern to search for"`
	Path       string `json:"path,omitempty" jsonschema:"description=file or directory to search in; defaults to workspace"`
	Glob       string `json:"glob,omitempty" jsonschema:"description=optional rg --glob filter"`
	OutputMode string `json:"output_mode,omitempty" jsonschema:"description=content, files_with_matches, or count"`
	Context    int    `json:"context,omitempty" jsonschema:"description=context lines around each match"`
	IgnoreCase bool   `json:"ignore_case,omitempty" jsonschema:"description=case insensitive search"`
	HeadLimit  int    `json:"head_limit,omitempty" jsonschema:"description=max output lines or entries; default 250"`
	Offset     int    `json:"offset,omitempty" jsonschema:"description=lines or entries to skip before applying head_limit"`
}

type GrepOutput struct {
	Mode          string   `json:"mode"`
	NumFiles      int      `json:"numFiles"`
	Filenames     []string `json:"filenames"`
	Content       string   `json:"content,omitempty"`
	NumLines      int      `json:"numLines,omitempty"`
	NumMatches    int      `json:"numMatches,omitempty"`
	AppliedLimit  int      `json:"appliedLimit,omitempty"`
	AppliedOffset int      `json:"appliedOffset,omitempty"`
}

type TaskOutputInput struct {
	TaskID string `json:"task_id" jsonschema:"description=background task id"`
	Offset int    `json:"offset,omitempty" jsonschema:"description=byte offset to start reading at"`
	Limit  int    `json:"limit,omitempty" jsonschema:"description=max bytes to return"`
}

type TaskStopInput struct {
	TaskID string `json:"task_id" jsonschema:"description=background task id to stop"`
}

type ExitPlanModeInput struct {
	Plan string `json:"plan,omitempty" jsonschema:"description=implementation plan approved by the user"`
}

type ExitPlanModeOutput struct {
	Plan    string `json:"plan,omitempty"`
	Message string `json:"message"`
}

type EnterWorktreeInput struct {
	Name string `json:"name,omitempty" jsonschema:"description=optional short worktree name; safe characters only"`
}

type ExitWorktreeInput struct {
	Action         string `json:"action,omitempty" jsonschema:"description=keep or remove; defaults to keep"`
	DiscardChanges bool   `json:"discard_changes,omitempty" jsonschema:"description=required to remove a dirty worktree"`
}

type WorktreeOutput struct {
	Action         string `json:"action"`
	WorkspacePath  string `json:"workspacePath"`
	WorktreePath   string `json:"worktreePath,omitempty"`
	WorktreeBranch string `json:"worktreeBranch,omitempty"`
	Message        string `json:"message"`
}

type WorktreeDiffInput struct {
	IncludePatch bool `json:"include_patch,omitempty" jsonschema:"description=include patch content in addition to status and stat"`
	MaxBytes     int  `json:"max_bytes,omitempty" jsonschema:"description=max patch bytes when include_patch is true; default 20000"`
}

type WorktreeDiffOutput struct {
	WorkspacePath      string `json:"workspacePath"`
	WorktreePath       string `json:"worktreePath"`
	WorktreeBranch     string `json:"worktreeBranch"`
	Status             string `json:"status"`
	DiffStat           string `json:"diffStat"`
	Diff               string `json:"diff,omitempty"`
	UntrackedDiff      string `json:"untrackedDiff,omitempty"`
	Truncated          bool   `json:"truncated,omitempty"`
	UntrackedTruncated bool   `json:"untrackedTruncated,omitempty"`
	Message            string `json:"message"`
}

type WorktreeMergeInput struct {
	CommitMessage  string `json:"commit_message,omitempty" jsonschema:"description=commit and merge message; defaults to Starxo runtime worktree merge"`
	RemoveWorktree bool   `json:"remove_worktree,omitempty" jsonschema:"description=remove the worktree and delete its local branch after a successful merge"`
}

type WorktreeMergeOutput struct {
	Action         string `json:"action"`
	WorkspacePath  string `json:"workspacePath"`
	WorktreePath   string `json:"worktreePath"`
	WorktreeBranch string `json:"worktreeBranch"`
	CommitMessage  string `json:"commitMessage,omitempty"`
	Removed        bool   `json:"removed"`
	Message        string `json:"message"`
}

func NewRuntimeCoreCatalogEntries(op commandline.Operator, workspacePath string, tasks RuntimeTaskManager, workspaces RuntimeWorkspaceManager) ([]CatalogEntry, error) {
	builders := []func() (CatalogEntry, error){
		func() (CatalogEntry, error) { return newBashCatalogEntry(op, workspacePath, tasks, workspaces) },
		func() (CatalogEntry, error) { return newReadCatalogEntry(op, workspacePath, workspaces) },
		func() (CatalogEntry, error) { return newWriteCatalogEntry(op, workspacePath, workspaces) },
		func() (CatalogEntry, error) { return newEditCatalogEntry(op, workspacePath, workspaces) },
		func() (CatalogEntry, error) { return newGlobCatalogEntry(op, workspacePath, workspaces) },
		func() (CatalogEntry, error) { return newGrepCatalogEntry(op, workspacePath, workspaces) },
		func() (CatalogEntry, error) { return newTaskOutputCatalogEntry(tasks) },
		func() (CatalogEntry, error) { return newTaskStopCatalogEntry(tasks) },
		newExitPlanModeCatalogEntry,
		func() (CatalogEntry, error) { return newEnterWorktreeCatalogEntry(op, workspacePath, workspaces) },
		func() (CatalogEntry, error) { return newExitWorktreeCatalogEntry(op, workspacePath, workspaces) },
		func() (CatalogEntry, error) { return newWorktreeDiffCatalogEntry(op, workspacePath, workspaces) },
		func() (CatalogEntry, error) { return newWorktreeMergeCatalogEntry(op, workspacePath, workspaces) },
	}
	entries := make([]CatalogEntry, 0, len(builders))
	for _, build := range builders {
		entry, err := build()
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func runtimeCatalogEntry(name, title, description, class string, readOnly bool, t tool.BaseTool, aliases ...string) CatalogEntry {
	return runtimeCatalogEntryWithLoading(name, title, description, class, readOnly, true, false, t, aliases...)
}

func deferredRuntimeCatalogEntry(name, title, description, class string, readOnly bool, t tool.BaseTool, aliases ...string) CatalogEntry {
	return runtimeCatalogEntryWithLoading(name, title, description, class, readOnly, false, true, t, aliases...)
}

func runtimeCatalogEntryWithLoading(name, title, description, class string, readOnly bool, alwaysLoad bool, shouldDefer bool, t tool.BaseTool, aliases ...string) CatalogEntry {
	return CatalogEntry{
		CanonicalName:   name,
		Aliases:         aliases,
		Source:          ToolSourceRuntime,
		Kind:            ToolKindAction,
		Title:           title,
		Description:     description,
		SearchHint:      description,
		ToolClass:       class,
		AlwaysLoad:      alwaysLoad,
		ShouldDefer:     shouldDefer,
		ReadOnlyHint:    readOnly,
		ReadOnlyTrusted: readOnly,
		PermissionSpec: PermissionSpec{
			AllowSearch:  true,
			AllowExecute: true,
		},
		Tool: t,
	}
}

func newBashCatalogEntry(op commandline.Operator, workspacePath string, tasks RuntimeTaskManager, workspaces RuntimeWorkspaceManager) (CatalogEntry, error) {
	t, err := toolutils.InferTool(RuntimeToolBash,
		"Execute a shell command in the sandbox workspace. Supports timeout, concise descriptions, and background execution.",
		func(ctx context.Context, input BashInput) (BashOutput, error) {
			if strings.TrimSpace(input.Command) == "" {
				return BashOutput{}, fmt.Errorf("command is required")
			}
			run := func(runCtx context.Context) (BashOutput, error) {
				return runBashCommand(runCtx, op, workspacePath, workspaces, input)
			}
			if input.RunInBackground {
				if tasks == nil {
					return BashOutput{}, fmt.Errorf("background tasks are not available")
				}
				ref, err := tasks.StartShellTask(ctx, sessionIDFromContext(ctx), input.Command, runtimeFirstNonEmpty(input.Description, input.Command), run)
				if err != nil {
					return BashOutput{}, err
				}
				return BashOutput{
					BackgroundTaskID: ref.TaskID,
					OutputPath:       ref.OutputPath,
				}, nil
			}
			out, err := run(ctx)
			if err != nil {
				return out, err
			}
			if tasks != nil {
				out = maybePersistLargeBashOutput(ctx, tasks, out)
			}
			return out, nil
		})
	if err != nil {
		return CatalogEntry{}, err
	}
	return runtimeCatalogEntry(RuntimeToolBash, "Bash", "Run shell commands in the sandbox workspace.", ToolClassRuntimeExec, false, t, "shell_execute"), nil
}

func runBashCommand(ctx context.Context, op commandline.Operator, workspacePath string, workspaces RuntimeWorkspaceManager, input BashInput) (BashOutput, error) {
	if input.TimeoutMs > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(input.TimeoutMs)*time.Millisecond)
		defer cancel()
	}
	command := input.Command
	if workspace := currentWorkspace(ctx, workspacePath, workspaces); workspace != "" {
		command = "cd " + shellQuote(workspace) + " && " + command
	}
	output, err := op.RunCommand(ctx, []string{"sh", "-lc", command})
	out := BashOutput{}
	if output != nil {
		out.Stdout = output.Stdout
		out.Stderr = output.Stderr
		out.ExitCode = output.ExitCode
	}
	if ctx.Err() == context.DeadlineExceeded || ctx.Err() == context.Canceled {
		out.Interrupted = true
	}
	if err != nil {
		return out, err
	}
	return out, nil
}

func maybePersistLargeBashOutput(ctx context.Context, tasks RuntimeTaskManager, out BashOutput) BashOutput {
	combined := strings.TrimRight(out.Stdout+"\n"+out.Stderr, "\n")
	if len(combined) <= runtimeLargeOutputThreshold {
		return out
	}
	path, size, err := tasks.PersistToolResult(ctx, sessionIDFromContext(ctx), "bash", combined)
	if err != nil {
		return out
	}
	out.PersistedOutputPath = path
	out.PersistedOutputSize = size
	out.Stdout = truncateEnd(out.Stdout, runtimeLargeOutputThreshold/2)
	out.Stderr = truncateEnd(out.Stderr, runtimeLargeOutputThreshold/2)
	return out
}

func newReadCatalogEntry(op commandline.Operator, workspacePath string, workspaces RuntimeWorkspaceManager) (CatalogEntry, error) {
	t, err := toolutils.InferTool(RuntimeToolRead,
		"Read a file from the sandbox workspace with optional line offset and limit.",
		func(ctx context.Context, input ReadInput) (ReadOutput, error) {
			if strings.TrimSpace(input.FilePath) == "" {
				return ReadOutput{}, fmt.Errorf("file_path is required")
			}
			target, err := workspaceFilePath(ctx, input.FilePath, workspacePath, workspaces)
			if err != nil {
				return ReadOutput{}, err
			}
			content, err := op.ReadFile(ctx, target)
			if err != nil {
				return ReadOutput{}, err
			}
			lines := splitLines(content)
			offset := max(0, input.Offset)
			if offset > len(lines) {
				offset = len(lines)
			}
			limit := input.Limit
			end := len(lines)
			if limit > 0 && offset+limit < end {
				end = offset + limit
			}
			return ReadOutput{
				FilePath:   target,
				Content:    strings.Join(lines[offset:end], "\n"),
				NumLines:   end - offset,
				StartLine:  offset + 1,
				TotalLines: len(lines),
			}, nil
		})
	if err != nil {
		return CatalogEntry{}, err
	}
	return runtimeCatalogEntry(RuntimeToolRead, "Read", "Read files with line paging.", ToolClassRuntimeFile, true, t, "read_file"), nil
}

func newWriteCatalogEntry(op commandline.Operator, workspacePath string, workspaces RuntimeWorkspaceManager) (CatalogEntry, error) {
	t, err := toolutils.InferTool(RuntimeToolWrite,
		"Write a file inside the sandbox workspace.",
		func(ctx context.Context, input WriteInput) (WriteOutput, error) {
			if strings.TrimSpace(input.FilePath) == "" {
				return WriteOutput{}, fmt.Errorf("file_path is required")
			}
			target, err := workspaceFilePath(ctx, input.FilePath, workspacePath, workspaces)
			if err != nil {
				return WriteOutput{}, err
			}
			previous, err := readExistingFilePreview(ctx, op, target, runtimeWriteOldPreviewLimit)
			if err != nil {
				return WriteOutput{}, err
			}
			if err := op.WriteFile(ctx, target, input.Content); err != nil {
				return WriteOutput{}, err
			}
			patch, patchTruncated := buildWritePatchLimited(previous.Content, input.Content, runtimeToolPatchLimit)
			truncated := previous.Truncated || patchTruncated
			linesRemoved := 0
			if previous.Exists {
				linesRemoved = previous.Lines
			}
			return WriteOutput{
				FilePath:     target,
				Created:      !previous.Exists,
				Bytes:        len(input.Content),
				LinesAdded:   len(splitLines(input.Content)),
				LinesRemoved: linesRemoved,
				Patch:        patch,
				Truncated:    truncated,
			}, nil
		})
	if err != nil {
		return CatalogEntry{}, err
	}
	return runtimeCatalogEntry(RuntimeToolWrite, "Write", "Create or overwrite files.", ToolClassRuntimeFile, false, t, "write_file"), nil
}

func newEditCatalogEntry(op commandline.Operator, workspacePath string, workspaces RuntimeWorkspaceManager) (CatalogEntry, error) {
	t, err := toolutils.InferTool(RuntimeToolEdit,
		"Edit a file by replacing an exact string.",
		func(ctx context.Context, input EditInput) (EditOutput, error) {
			if strings.TrimSpace(input.FilePath) == "" {
				return EditOutput{}, fmt.Errorf("file_path is required")
			}
			if input.OldString == "" {
				return EditOutput{}, fmt.Errorf("old_string is required")
			}
			target, err := workspaceFilePath(ctx, input.FilePath, workspacePath, workspaces)
			if err != nil {
				return EditOutput{}, err
			}
			content, err := op.ReadFile(ctx, target)
			if err != nil {
				return EditOutput{}, err
			}
			count := strings.Count(content, input.OldString)
			if count == 0 {
				return EditOutput{}, fmt.Errorf("old_string not found in %s", target)
			}
			replacements := 1
			if input.ReplaceAll {
				replacements = count
			} else if count > 1 {
				return EditOutput{}, fmt.Errorf("old_string appears %d times in %s; set replace_all=true or provide a more specific string", count, target)
			}
			next := strings.Replace(content, input.OldString, input.NewString, replacements)
			if err := op.WriteFile(ctx, target, next); err != nil {
				return EditOutput{}, err
			}
			patch, truncated := buildSimplePatchLimited(input.OldString, input.NewString, runtimeToolPatchLimit)
			return EditOutput{
				FilePath:     target,
				Replacements: replacements,
				LinesAdded:   countLinesDelta(input.NewString, input.OldString, true) * replacements,
				LinesRemoved: countLinesDelta(input.NewString, input.OldString, false) * replacements,
				Patch:        patch,
				Truncated:    truncated,
			}, nil
		})
	if err != nil {
		return CatalogEntry{}, err
	}
	return runtimeCatalogEntry(RuntimeToolEdit, "Edit", "Patch files using exact string replacement.", ToolClassRuntimeFile, false, t, "str_replace_editor"), nil
}

func newGlobCatalogEntry(op commandline.Operator, workspacePath string, workspaces RuntimeWorkspaceManager) (CatalogEntry, error) {
	t, err := toolutils.InferTool(RuntimeToolGlob,
		"Find files by glob pattern inside the sandbox workspace.",
		func(ctx context.Context, input GlobInput) (GlobOutput, error) {
			if strings.TrimSpace(input.Pattern) == "" {
				return GlobOutput{}, fmt.Errorf("pattern is required")
			}
			searchPath, err := safeSearchPath(input.Path)
			if err != nil {
				return GlobOutput{}, err
			}
			limit := input.Limit
			if limit <= 0 {
				limit = 100
			}
			offset := max(0, input.Offset)
			start := time.Now()
			cmd := "cd " + shellQuote(currentWorkspace(ctx, workspacePath, workspaces)) + " && find " + shellQuote(searchPath) + " -path " + shellQuote(input.Pattern) + " -type f | sort"
			output, err := op.RunCommand(ctx, []string{"sh", "-lc", cmd})
			if err != nil {
				return GlobOutput{}, err
			}
			files := nonEmptyLines(output.Stdout)
			total := len(files)
			if offset > len(files) {
				offset = len(files)
			}
			end := offset + limit
			if end > len(files) {
				end = len(files)
			}
			return GlobOutput{
				DurationMs: int(time.Since(start).Milliseconds()),
				NumFiles:   total,
				Filenames:  files[offset:end],
				Truncated:  end < total,
			}, nil
		})
	if err != nil {
		return CatalogEntry{}, err
	}
	return runtimeCatalogEntry(RuntimeToolGlob, "Glob", "Find files by name pattern.", ToolClassRuntimeFile, true, t, "list_files"), nil
}

func newGrepCatalogEntry(op commandline.Operator, workspacePath string, workspaces RuntimeWorkspaceManager) (CatalogEntry, error) {
	t, err := toolutils.InferTool(RuntimeToolGrep,
		"Search file contents with ripgrep inside the sandbox workspace.",
		func(ctx context.Context, input GrepInput) (GrepOutput, error) {
			if strings.TrimSpace(input.Pattern) == "" {
				return GrepOutput{}, fmt.Errorf("pattern is required")
			}
			searchPath, err := safeSearchPath(input.Path)
			if err != nil {
				return GrepOutput{}, err
			}
			mode := input.OutputMode
			if mode == "" {
				mode = "files_with_matches"
			}
			limit := input.HeadLimit
			if limit <= 0 {
				limit = 250
			}
			offset := max(0, input.Offset)
			args := []string{"rg", "--color", "never"}
			switch mode {
			case "content":
				args = append(args, "--line-number", "--no-heading")
				if input.Context > 0 {
					args = append(args, "-C", strconv.Itoa(input.Context))
				}
			case "count":
				args = append(args, "--count")
			case "files_with_matches":
				args = append(args, "--files-with-matches")
			default:
				return GrepOutput{}, fmt.Errorf("unsupported output_mode %q", mode)
			}
			if input.IgnoreCase {
				args = append(args, "-i")
			}
			if input.Glob != "" {
				args = append(args, "--glob", input.Glob)
			}
			args = append(args, input.Pattern, searchPath)
			cmd := "cd " + shellQuote(currentWorkspace(ctx, workspacePath, workspaces)) + " && " + shellJoin(args) + " || test $? -eq 1"
			output, err := op.RunCommand(ctx, []string{"sh", "-lc", cmd})
			if err != nil {
				return GrepOutput{}, err
			}
			lines := nonEmptyLines(output.Stdout)
			total := len(lines)
			if offset > len(lines) {
				offset = len(lines)
			}
			end := offset + limit
			if end > len(lines) {
				end = len(lines)
			}
			selected := lines[offset:end]
			filenames := grepFilenames(mode, selected)
			sort.Strings(filenames)
			return GrepOutput{
				Mode:          mode,
				NumFiles:      len(filenames),
				Filenames:     filenames,
				Content:       strings.Join(selected, "\n"),
				NumLines:      len(selected),
				NumMatches:    total,
				AppliedLimit:  limit,
				AppliedOffset: offset,
			}, nil
		})
	if err != nil {
		return CatalogEntry{}, err
	}
	return runtimeCatalogEntry(RuntimeToolGrep, "Grep", "Search file contents with ripgrep.", ToolClassRuntimeFile, true, t), nil
}

func newTaskOutputCatalogEntry(tasks RuntimeTaskManager) (CatalogEntry, error) {
	t, err := toolutils.InferTool(RuntimeToolTaskOutput,
		"Read output from a background runtime task.",
		func(ctx context.Context, input TaskOutputInput) (RuntimeTaskOutput, error) {
			if tasks == nil {
				return RuntimeTaskOutput{}, fmt.Errorf("runtime tasks are not available")
			}
			return tasks.ReadTaskOutput(ctx, input.TaskID, input.Offset, input.Limit)
		})
	if err != nil {
		return CatalogEntry{}, err
	}
	return runtimeCatalogEntry(RuntimeToolTaskOutput, "Task Output", "Read background task output.", ToolClassRuntimeTask, true, t), nil
}

func newTaskStopCatalogEntry(tasks RuntimeTaskManager) (CatalogEntry, error) {
	t, err := toolutils.InferTool(RuntimeToolTaskStop,
		"Stop a running background runtime task.",
		func(ctx context.Context, input TaskStopInput) (RuntimeTaskSnapshot, error) {
			if tasks == nil {
				return RuntimeTaskSnapshot{}, fmt.Errorf("runtime tasks are not available")
			}
			return tasks.StopTask(ctx, input.TaskID)
		})
	if err != nil {
		return CatalogEntry{}, err
	}
	return runtimeCatalogEntry(RuntimeToolTaskStop, "Task Stop", "Stop background tasks.", ToolClassRuntimeTask, false, t), nil
}

func newExitPlanModeCatalogEntry() (CatalogEntry, error) {
	t, err := toolutils.InferTool(RuntimeToolExitPlanMode,
		"Request approval to leave plan mode and execute the proposed implementation plan.",
		func(ctx context.Context, input ExitPlanModeInput) (ExitPlanModeOutput, error) {
			return ExitPlanModeOutput{
				Plan:    input.Plan,
				Message: "Plan approval is handled by Starxo plan mode UI; continue only after the user approves the plan.",
			}, nil
		})
	if err != nil {
		return CatalogEntry{}, err
	}
	return runtimeCatalogEntry(RuntimeToolExitPlanMode, "Exit Plan Mode", "Request plan approval before write or shell execution.", ToolClassRuntime, true, t), nil
}

func newEnterWorktreeCatalogEntry(op commandline.Operator, workspacePath string, workspaces RuntimeWorkspaceManager) (CatalogEntry, error) {
	t, err := toolutils.InferTool(RuntimeToolEnterWorktree,
		"Create an isolated git worktree inside the sandbox workspace and switch this session into it.",
		func(ctx context.Context, input EnterWorktreeInput) (WorktreeOutput, error) {
			if workspaces == nil {
				return WorktreeOutput{}, fmt.Errorf("runtime worktree manager is not available")
			}
			return workspaces.EnterWorktree(ctx, op, workspacePath, input.Name)
		})
	if err != nil {
		return CatalogEntry{}, err
	}
	return deferredRuntimeCatalogEntry(RuntimeToolEnterWorktree, "Enter Worktree", "Create and enter an isolated git worktree.", ToolClassRuntime, false, t), nil
}

func newExitWorktreeCatalogEntry(op commandline.Operator, workspacePath string, workspaces RuntimeWorkspaceManager) (CatalogEntry, error) {
	t, err := toolutils.InferTool(RuntimeToolExitWorktree,
		"Leave the current runtime worktree, optionally keeping or removing it.",
		func(ctx context.Context, input ExitWorktreeInput) (WorktreeOutput, error) {
			if workspaces == nil {
				return WorktreeOutput{}, fmt.Errorf("runtime worktree manager is not available")
			}
			return workspaces.ExitWorktree(ctx, op, workspacePath, input.Action, input.DiscardChanges)
		})
	if err != nil {
		return CatalogEntry{}, err
	}
	return deferredRuntimeCatalogEntry(RuntimeToolExitWorktree, "Exit Worktree", "Exit a worktree session created by EnterWorktree.", ToolClassRuntime, false, t), nil
}

func newWorktreeDiffCatalogEntry(op commandline.Operator, workspacePath string, workspaces RuntimeWorkspaceManager) (CatalogEntry, error) {
	t, err := toolutils.InferTool(RuntimeToolWorktreeDiff,
		"Review the current runtime worktree status, diff stat, and optional patch.",
		func(ctx context.Context, input WorktreeDiffInput) (WorktreeDiffOutput, error) {
			if workspaces == nil {
				return WorktreeDiffOutput{}, fmt.Errorf("runtime worktree manager is not available")
			}
			return workspaces.DiffWorktree(ctx, op, workspacePath, input.IncludePatch, input.MaxBytes)
		})
	if err != nil {
		return CatalogEntry{}, err
	}
	entry := deferredRuntimeCatalogEntry(RuntimeToolWorktreeDiff, "Worktree Diff", "Review current runtime worktree changes.", ToolClassRuntimeFile, true, t)
	entry.SearchHint = "worktree diff review status patch changes"
	return entry, nil
}

func newWorktreeMergeCatalogEntry(op commandline.Operator, workspacePath string, workspaces RuntimeWorkspaceManager) (CatalogEntry, error) {
	t, err := toolutils.InferTool(RuntimeToolWorktreeMerge,
		"Commit and merge the active runtime worktree back into the original sandbox workspace.",
		func(ctx context.Context, input WorktreeMergeInput) (WorktreeMergeOutput, error) {
			if workspaces == nil {
				return WorktreeMergeOutput{}, fmt.Errorf("runtime worktree manager is not available")
			}
			return workspaces.MergeWorktree(ctx, op, workspacePath, input.CommitMessage, input.RemoveWorktree)
		})
	if err != nil {
		return CatalogEntry{}, err
	}
	entry := deferredRuntimeCatalogEntry(RuntimeToolWorktreeMerge, "Worktree Merge", "Merge current runtime worktree changes back to the original workspace.", ToolClassRuntimeFile, false, t)
	entry.SearchHint = "worktree merge commit apply changes original workspace"
	return entry, nil
}

func currentWorkspace(ctx context.Context, defaultWorkspace string, workspaces RuntimeWorkspaceManager) string {
	if workspaces == nil {
		return defaultWorkspace
	}
	workspace := workspaces.CurrentWorkspace(ctx, defaultWorkspace)
	if strings.TrimSpace(workspace) == "" {
		return defaultWorkspace
	}
	return cleanRemotePath(workspace)
}

func workspaceFilePath(ctx context.Context, filePath string, workspaceRoot string, workspaces RuntimeWorkspaceManager) (string, error) {
	root := cleanRemotePath(workspaceRoot)
	if root == "" || root == "." {
		return "", fmt.Errorf("sandbox workspace is not active")
	}
	base := currentWorkspace(ctx, root, workspaces)
	if base == "" || base == "." {
		base = root
	}
	base = cleanRemotePath(base)

	p := strings.TrimSpace(filePath)
	if p == "" {
		return "", fmt.Errorf("file_path is required")
	}
	switch {
	case p == "/workspace":
		p = base
	case strings.HasPrefix(p, "/"):
		cleaned := cleanRemotePath(p)
		if isRemotePathInside(cleaned, base) {
			p = cleaned
		} else if strings.HasPrefix(p, "/workspace/") {
			p = path.Join(base, strings.TrimPrefix(p, "/workspace/"))
		} else {
			p = cleaned
		}
	default:
		p = path.Join(base, p)
	}
	p = cleanRemotePath(p)
	if !isRemotePathInside(p, base) {
		return "", fmt.Errorf("path %s is outside sandbox workspace %s", filePath, base)
	}
	return p, nil
}

func isRemotePathInside(p, root string) bool {
	p = cleanRemotePath(p)
	root = cleanRemotePath(root)
	if p == "" || root == "" {
		return false
	}
	if p == root {
		return true
	}
	if root == "/" {
		return strings.HasPrefix(p, "/")
	}
	return strings.HasPrefix(p, root+"/")
}

func cleanRemotePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	cleaned := path.Clean(p)
	if strings.HasPrefix(p, "/") && !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}
	return cleaned
}

type existingFilePreview struct {
	Exists    bool
	Content   string
	Bytes     int64
	Lines     int
	Truncated bool
}

func readExistingFilePreview(ctx context.Context, op commandline.Operator, target string, maxBytes int) (existingFilePreview, error) {
	if previewer, ok := op.(runtimeFilePreviewReader); ok {
		content, bytes, lines, exists, truncated, err := previewer.ReadFilePreview(ctx, target, maxBytes)
		if err != nil {
			return existingFilePreview{}, err
		}
		return existingFilePreview{
			Exists:    exists,
			Content:   content,
			Bytes:     bytes,
			Lines:     lines,
			Truncated: truncated,
		}, nil
	}

	// Non-remote test or legacy operators may not support bounded previews.
	// Production RemoteOperator implements ReadFilePreview so large overwrites
	// do not require reading the whole old file just to render a diff summary.
	content, err := op.ReadFile(ctx, target)
	if err != nil {
		return existingFilePreview{Exists: false}, nil
	}
	return existingFilePreview{
		Exists:  true,
		Content: content,
		Bytes:   int64(len(content)),
		Lines:   len(splitLines(content)),
	}, nil
}

func safeSearchPath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return ".", nil
	}
	if strings.HasPrefix(p, "/workspace/") {
		p = strings.TrimPrefix(p, "/workspace/")
	}
	if strings.HasPrefix(p, "/") {
		return "", fmt.Errorf("absolute search paths are not allowed; use a workspace-relative path")
	}
	cleaned := path.Clean(p)
	if cleaned == "." {
		return ".", nil
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("search path must stay inside the workspace")
	}
	return cleaned, nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func shellJoin(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, shellQuote(arg))
	}
	return strings.Join(quoted, " ")
}

func sessionIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value("sessionID").(string); ok {
		return v
	}
	return ""
}

func splitLines(content string) []string {
	if content == "" {
		return []string{}
	}
	content = strings.TrimSuffix(content, "\n")
	if content == "" {
		return []string{}
	}
	return strings.Split(content, "\n")
}

func nonEmptyLines(content string) []string {
	lines := splitLines(content)
	out := lines[:0]
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	return out
}

func runtimeFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func truncateEnd(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "\n[truncated; full output persisted]"
}

func countLinesDelta(newString, oldString string, added bool) int {
	newLines := len(splitLines(newString))
	oldLines := len(splitLines(oldString))
	if added {
		if newLines > oldLines {
			return newLines - oldLines
		}
		return 0
	}
	if oldLines > newLines {
		return oldLines - newLines
	}
	return 0
}

func buildSimplePatch(oldString, newString string) string {
	patch, _ := buildSimplePatchLimited(oldString, newString, 0)
	return patch
}

func buildSimplePatchLimited(oldString, newString string, limit int) (string, bool) {
	var b strings.Builder
	truncated := !appendPatchContentLimited(&b, "-", oldString, limit)
	if !truncated {
		truncated = !appendPatchContentLimited(&b, "+", newString, limit)
	}
	return finishLimitedPatch(b.String(), truncated, limit)
}

func buildWritePatchLimited(oldString, newString string, limit int) (string, bool) {
	if limit <= 0 {
		return buildSimplePatchLimited(oldString, newString, limit)
	}
	oldBudget := limit / 2
	if oldBudget < 1 {
		oldBudget = 1
	}
	var b strings.Builder
	truncated := !appendPatchContentLimited(&b, "-", oldString, oldBudget)
	if !appendPatchContentLimited(&b, "+", newString, limit) {
		truncated = true
	}
	return finishLimitedPatch(b.String(), truncated, limit)
}

func finishLimitedPatch(content string, truncated bool, limit int) (string, bool) {
	patch := strings.TrimSuffix(content, "\n")
	if !truncated {
		return patch, false
	}
	marker := "\n... (patch truncated)"
	if limit <= 0 {
		return patch + marker, true
	}
	if limit <= len(marker) {
		return "", true
	}
	if len(patch)+len(marker) <= limit {
		return patch + marker, true
	}
	return patch[:limit-len(marker)] + marker, true
}

func appendPatchContentLimited(b *strings.Builder, prefix, content string, limit int) bool {
	if content == "" {
		return true
	}
	content = strings.TrimSuffix(content, "\n")
	if content == "" {
		return true
	}
	for {
		line, rest, found := strings.Cut(content, "\n")
		if !appendPatchLineLimited(b, prefix, line, limit) {
			return false
		}
		if !found {
			return true
		}
		content = rest
	}
}

func appendPatchLineLimited(b *strings.Builder, prefix, line string, limit int) bool {
	nextLen := b.Len() + len(prefix) + len(line) + 1
	if limit > 0 && nextLen > limit {
		remaining := limit - b.Len()
		lineBudget := remaining - len(prefix) - 1
		if lineBudget > 0 {
			if lineBudget > len(line) {
				lineBudget = len(line)
			}
			b.WriteString(prefix)
			b.WriteString(line[:lineBudget])
			b.WriteString("\n")
		}
		return false
	}
	b.WriteString(prefix)
	b.WriteString(line)
	b.WriteString("\n")
	return true
}

func grepFilenames(mode string, lines []string) []string {
	seen := make(map[string]struct{})
	for _, line := range lines {
		name := line
		if mode == "content" || mode == "count" {
			if idx := strings.Index(line, ":"); idx >= 0 {
				name = line[:idx]
			}
		}
		if name == "" {
			continue
		}
		seen[name] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func clampOutputMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case "files_with_matches", "count":
		return strings.TrimSpace(mode)
	default:
		return "content"
	}
}

func appendMaybe(args []string, flag, value string) []string {
	if strings.TrimSpace(value) == "" {
		return args
	}
	return append(args, flag, value)
}

func truncateLines(lines []string, offset, limit int) ([]string, bool) {
	if offset > len(lines) {
		return []string{}, false
	}
	lines = lines[offset:]
	if limit <= 0 || len(lines) <= limit {
		return lines, false
	}
	return lines[:limit], true
}
