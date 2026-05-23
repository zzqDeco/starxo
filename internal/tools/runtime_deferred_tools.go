package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
)

const (
	RuntimeToolLSP          = "LSP"
	RuntimeToolLSPEdit      = "LSPEdit"
	RuntimeToolSkill        = "Skill"
	RuntimeToolNotebookEdit = "NotebookEdit"
	RuntimeToolWebFetch     = "WebFetch"
	RuntimeToolWebSearch    = "WebSearch"
)

type LSPInput struct {
	Operation string `json:"operation" jsonschema:"description=definition, references, hover, document_symbol, or workspace_symbol"`
	Symbol    string `json:"symbol,omitempty" jsonschema:"description=symbol or text to search for"`
	FilePath  string `json:"file_path,omitempty" jsonschema:"description=file path inside the workspace"`
	Line      int    `json:"line,omitempty" jsonschema:"description=1-based line for position-based operations"`
	Character int    `json:"character,omitempty" jsonschema:"description=1-based character for position-based operations; defaults to 1"`
	Language  string `json:"language,omitempty" jsonschema:"description=optional language override: go, typescript, javascript, python, or rust"`
	Limit     int    `json:"limit,omitempty" jsonschema:"description=max result lines"`
}

type LSPEditInput struct {
	Operation string `json:"operation" jsonschema:"description=rename or format"`
	FilePath  string `json:"file_path" jsonschema:"description=file path inside the workspace"`
	Line      int    `json:"line,omitempty" jsonschema:"description=1-based line for rename"`
	Character int    `json:"character,omitempty" jsonschema:"description=1-based character for rename; defaults to 1"`
	NewName   string `json:"new_name,omitempty" jsonschema:"description=new symbol name for rename"`
	Language  string `json:"language,omitempty" jsonschema:"description=optional language override: go, typescript, javascript, python, or rust"`
}

type LSPOutput struct {
	Operation      string `json:"operation"`
	Engine         string `json:"engine,omitempty"`
	Language       string `json:"language,omitempty"`
	Result         string `json:"result"`
	FilePath       string `json:"filePath,omitempty"`
	ResultCount    int    `json:"resultCount,omitempty"`
	FallbackReason string `json:"fallbackReason,omitempty"`
}

type LSPEditOutput struct {
	Operation    string   `json:"operation"`
	Engine       string   `json:"engine,omitempty"`
	Language     string   `json:"language,omitempty"`
	FilePath     string   `json:"filePath,omitempty"`
	ChangedFiles []string `json:"changedFiles,omitempty"`
	EditCount    int      `json:"editCount"`
	Summary      string   `json:"summary"`
}

type SkillInput struct {
	Action string `json:"action,omitempty" jsonschema:"description=list or read"`
	Name   string `json:"name,omitempty" jsonschema:"description=skill directory name for read"`
}

type SkillOutput struct {
	Action string   `json:"action"`
	Skills []string `json:"skills,omitempty"`
	Name   string   `json:"name,omitempty"`
	Prompt string   `json:"prompt,omitempty"`
}

type NotebookEditInput struct {
	FilePath string `json:"file_path" jsonschema:"description=ipynb path inside the workspace"`
	Command  string `json:"command" jsonschema:"description=view, replace_cell, insert_cell, or delete_cell"`
	Index    int    `json:"index,omitempty" jsonschema:"description=zero-based cell index"`
	CellType string `json:"cell_type,omitempty" jsonschema:"description=code or markdown"`
	Source   string `json:"source,omitempty" jsonschema:"description=cell source for insert or replace"`
}

type NotebookEditOutput struct {
	FilePath string `json:"filePath"`
	Command  string `json:"command"`
	Cells    int    `json:"cells"`
	Summary  string `json:"summary"`
}

type RuntimeLSPManager interface {
	Query(ctx context.Context, op commandline.Operator, workspacePath string, workspaces RuntimeWorkspaceManager, input LSPInput) (LSPOutput, bool, error)
	Edit(ctx context.Context, op commandline.Operator, workspacePath string, workspaces RuntimeWorkspaceManager, input LSPEditInput) (LSPEditOutput, bool, error)
}

func NewRuntimeDeferredCatalogEntries(op commandline.Operator, workspacePath string, workspaces RuntimeWorkspaceManager, lsp RuntimeLSPManager) ([]CatalogEntry, error) {
	builders := []func() (CatalogEntry, error){
		func() (CatalogEntry, error) { return newLSPCatalogEntry(op, workspacePath, workspaces, lsp) },
		func() (CatalogEntry, error) { return newLSPEditCatalogEntry(op, workspacePath, workspaces, lsp) },
		func() (CatalogEntry, error) { return newSkillCatalogEntry(op, workspacePath, workspaces) },
		func() (CatalogEntry, error) { return newNotebookEditCatalogEntry(op, workspacePath, workspaces) },
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

func RuntimeWebFetchCatalogEntry(t tool.BaseTool) CatalogEntry {
	return deferredRuntimeCatalogEntry(RuntimeToolWebFetch, "Web Fetch", "Fetch a URL over HTTP GET and return readable text.", ToolClassRuntime, true, t)
}

func RuntimeWebSearchCatalogEntry(t tool.BaseTool) CatalogEntry {
	return deferredRuntimeCatalogEntry(RuntimeToolWebSearch, "Web Search", "Search the web and return compact result links.", ToolClassRuntime, true, t)
}

func newLSPCatalogEntry(op commandline.Operator, workspacePath string, workspaces RuntimeWorkspaceManager, lsp RuntimeLSPManager) (CatalogEntry, error) {
	t, err := toolutils.InferTool(RuntimeToolLSP,
		"Persistent language-server code intelligence with rg/sed fallback for definitions, references, hover, document symbols, and workspace symbols.",
		func(ctx context.Context, input LSPInput) (LSPOutput, error) {
			var fallbackReason string
			if lsp != nil {
				output, handled, err := lsp.Query(ctx, op, workspacePath, workspaces, input)
				if handled {
					return output, err
				}
				if err != nil {
					fallbackReason = err.Error()
				}
			}
			output, err := runFallbackLSP(ctx, op, workspacePath, workspaces, input)
			if output.FallbackReason == "" {
				output.FallbackReason = fallbackReason
			}
			return output, err
		})
	if err != nil {
		return CatalogEntry{}, err
	}
	entry := deferredRuntimeCatalogEntry(RuntimeToolLSP, "LSP", "Code intelligence operations over the workspace.", ToolClassRuntimeFile, true, t)
	entry.SearchHint = "definition references hover symbols code intelligence language server"
	return entry, nil
}

func newLSPEditCatalogEntry(op commandline.Operator, workspacePath string, workspaces RuntimeWorkspaceManager, lsp RuntimeLSPManager) (CatalogEntry, error) {
	t, err := toolutils.InferTool(RuntimeToolLSPEdit,
		"Apply language-server edits in the workspace. Supports rename and format; requires runtime permission approval.",
		func(ctx context.Context, input LSPEditInput) (LSPEditOutput, error) {
			if lsp == nil {
				return LSPEditOutput{}, fmt.Errorf("LSP edit manager is not available")
			}
			output, handled, err := lsp.Edit(ctx, op, workspacePath, workspaces, input)
			if !handled && err == nil {
				err = fmt.Errorf("LSP edit operation %q is not available", strings.TrimSpace(input.Operation))
			}
			return output, err
		})
	if err != nil {
		return CatalogEntry{}, err
	}
	entry := deferredRuntimeCatalogEntry(RuntimeToolLSPEdit, "LSP Edit", "Writable language-server operations such as rename and format.", ToolClassRuntimeFile, false, t)
	entry.SearchHint = "rename format apply edit code action language server"
	return entry, nil
}

func runFallbackLSP(ctx context.Context, op commandline.Operator, workspacePath string, workspaces RuntimeWorkspaceManager, input LSPInput) (LSPOutput, error) {
	opName := strings.TrimSpace(input.Operation)
	if opName == "" {
		opName = "workspace_symbol"
	}
	limit := input.Limit
	if limit <= 0 {
		limit = 80
	}
	searchPath, err := safeSearchPath(input.FilePath)
	if err != nil {
		return LSPOutput{}, err
	}
	workspace := currentWorkspace(ctx, workspacePath, workspaces)
	var cmd string
	switch opName {
	case "definition":
		if input.Symbol == "" {
			return runFallbackLSPLineContext(ctx, op, workspacePath, workspaces, input, opName)
		}
		pattern := `(func|type|class|def|const|var|let)\s+` + input.Symbol + `\b|` + input.Symbol + `\s*[:=]`
		cmd = "cd " + shellQuote(workspace) + " && rg --color never --line-number --no-heading " + shellQuote(pattern) + " " + shellQuote(searchPath) + " | head -n " + strconv.Itoa(limit)
	case "references", "workspace_symbol":
		if opName == "references" && input.Symbol == "" {
			return runFallbackLSPLineContext(ctx, op, workspacePath, workspaces, input, opName)
		}
		if input.Symbol == "" {
			return LSPOutput{}, fmt.Errorf("symbol is required for %s", opName)
		}
		cmd = "cd " + shellQuote(workspace) + " && rg --color never --line-number --no-heading --fixed-strings " + shellQuote(input.Symbol) + " " + shellQuote(searchPath) + " | head -n " + strconv.Itoa(limit)
	case "document_symbol":
		pattern := `^\s*(func|type|class|def|const|var|let|interface|struct)\s+`
		cmd = "cd " + shellQuote(workspace) + " && rg --color never --line-number --no-heading " + shellQuote(pattern) + " " + shellQuote(searchPath) + " | head -n " + strconv.Itoa(limit)
	case "hover":
		if input.FilePath == "" || input.Line <= 0 {
			return LSPOutput{}, fmt.Errorf("file_path and positive line are required for hover")
		}
		target, err := workspaceFilePath(ctx, input.FilePath, workspacePath, workspaces)
		if err != nil {
			return LSPOutput{}, err
		}
		start := input.Line - 3
		if start < 1 {
			start = 1
		}
		end := input.Line + 3
		cmd = "sed -n " + shellQuote(fmt.Sprintf("%d,%dp", start, end)) + " " + shellQuote(target)
	default:
		return LSPOutput{}, fmt.Errorf("unsupported LSP operation %q", opName)
	}
	output, err := op.RunCommand(ctx, []string{"sh", "-lc", cmd + " || test $? -eq 1"})
	if err != nil {
		return LSPOutput{}, err
	}
	lines := nonEmptyLines(output.Stdout)
	return LSPOutput{
		Operation:   opName,
		Engine:      "fallback:rg",
		Result:      strings.Join(lines, "\n"),
		FilePath:    input.FilePath,
		ResultCount: len(lines),
	}, nil
}

func runFallbackLSPLineContext(ctx context.Context, op commandline.Operator, workspacePath string, workspaces RuntimeWorkspaceManager, input LSPInput, operation string) (LSPOutput, error) {
	if input.FilePath == "" || input.Line <= 0 {
		return LSPOutput{}, fmt.Errorf("symbol is required for %s", operation)
	}
	target, err := workspaceFilePath(ctx, input.FilePath, workspacePath, workspaces)
	if err != nil {
		return LSPOutput{}, err
	}
	start := input.Line - 3
	if start < 1 {
		start = 1
	}
	end := input.Line + 3
	cmd := "sed -n " + shellQuote(fmt.Sprintf("%d,%dp", start, end)) + " " + shellQuote(target)
	output, err := op.RunCommand(ctx, []string{"sh", "-lc", cmd})
	if err != nil {
		return LSPOutput{}, err
	}
	lines := nonEmptyLines(output.Stdout)
	return LSPOutput{
		Operation:   operation,
		Engine:      "fallback:sed",
		Result:      strings.Join(lines, "\n"),
		FilePath:    input.FilePath,
		ResultCount: len(lines),
	}, nil
}

func newSkillCatalogEntry(op commandline.Operator, workspacePath string, workspaces RuntimeWorkspaceManager) (CatalogEntry, error) {
	t, err := toolutils.InferTool(RuntimeToolSkill,
		"List or read workspace skills from .starxo/skills or .claude/skills.",
		func(ctx context.Context, input SkillInput) (SkillOutput, error) {
			action := input.Action
			if action == "" {
				action = "list"
			}
			workspace := currentWorkspace(ctx, workspacePath, workspaces)
			switch action {
			case "list":
				cmd := "cd " + shellQuote(workspace) + " && find .starxo/skills .claude/skills -mindepth 2 -maxdepth 2 -name SKILL.md -print 2>/dev/null | sort"
				output, err := op.RunCommand(ctx, []string{"sh", "-lc", cmd + " || true"})
				if err != nil {
					return SkillOutput{}, err
				}
				return SkillOutput{Action: action, Skills: nonEmptyLines(output.Stdout)}, nil
			case "read":
				if strings.TrimSpace(input.Name) == "" {
					return SkillOutput{}, fmt.Errorf("name is required for read")
				}
				name, err := safeSearchPath(input.Name)
				if err != nil {
					return SkillOutput{}, err
				}
				candidates := []string{
					pathJoin(".starxo/skills", name, "SKILL.md"),
					pathJoin(".claude/skills", name, "SKILL.md"),
				}
				for _, candidate := range candidates {
					target, err := workspaceFilePath(ctx, candidate, workspacePath, workspaces)
					if err != nil {
						return SkillOutput{}, err
					}
					content, err := op.ReadFile(ctx, target)
					if err == nil {
						return SkillOutput{Action: action, Name: input.Name, Prompt: content}, nil
					}
				}
				return SkillOutput{}, fmt.Errorf("skill %s not found", input.Name)
			default:
				return SkillOutput{}, fmt.Errorf("unsupported skill action %q", action)
			}
		})
	if err != nil {
		return CatalogEntry{}, err
	}
	entry := deferredRuntimeCatalogEntry(RuntimeToolSkill, "Skill", "List or read workspace skill prompts.", ToolClassRuntime, true, t)
	entry.SearchHint = "skill prompt capability domain instructions"
	return entry, nil
}

func newNotebookEditCatalogEntry(op commandline.Operator, workspacePath string, workspaces RuntimeWorkspaceManager) (CatalogEntry, error) {
	t, err := toolutils.InferTool(RuntimeToolNotebookEdit,
		"View or edit Jupyter notebook cells in .ipynb files.",
		func(ctx context.Context, input NotebookEditInput) (NotebookEditOutput, error) {
			if strings.TrimSpace(input.FilePath) == "" {
				return NotebookEditOutput{}, fmt.Errorf("file_path is required")
			}
			target, err := workspaceFilePath(ctx, input.FilePath, workspacePath, workspaces)
			if err != nil {
				return NotebookEditOutput{}, err
			}
			content, err := op.ReadFile(ctx, target)
			if err != nil {
				return NotebookEditOutput{}, err
			}
			var nb map[string]any
			if err := json.Unmarshal([]byte(content), &nb); err != nil {
				return NotebookEditOutput{}, fmt.Errorf("parse notebook: %w", err)
			}
			cells, err := notebookCells(nb)
			if err != nil {
				return NotebookEditOutput{}, err
			}
			command := input.Command
			if command == "" {
				command = "view"
			}
			if command == "view" {
				return NotebookEditOutput{FilePath: target, Command: command, Cells: len(cells), Summary: notebookSummary(cells)}, nil
			}
			nextCells, summary, err := editNotebookCells(cells, input)
			if err != nil {
				return NotebookEditOutput{}, err
			}
			nb["cells"] = nextCells
			data, err := json.MarshalIndent(nb, "", "  ")
			if err != nil {
				return NotebookEditOutput{}, err
			}
			if err := op.WriteFile(ctx, target, string(data)+"\n"); err != nil {
				return NotebookEditOutput{}, err
			}
			return NotebookEditOutput{FilePath: target, Command: command, Cells: len(nextCells), Summary: summary}, nil
		})
	if err != nil {
		return CatalogEntry{}, err
	}
	entry := deferredRuntimeCatalogEntry(RuntimeToolNotebookEdit, "Notebook Edit", "View or edit Jupyter notebook cells.", ToolClassRuntimeFile, false, t)
	entry.SearchHint = "jupyter notebook ipynb cell edit"
	return entry, nil
}

func pathJoin(parts ...string) string {
	return strings.Join(parts, "/")
}

func notebookCells(nb map[string]any) ([]any, error) {
	raw, ok := nb["cells"].([]any)
	if !ok {
		return nil, fmt.Errorf("notebook cells must be an array")
	}
	return raw, nil
}

func notebookSummary(cells []any) string {
	lines := make([]string, 0, len(cells))
	for i, cell := range cells {
		m, _ := cell.(map[string]any)
		cellType, _ := m["cell_type"].(string)
		source := sourceString(m["source"])
		lines = append(lines, fmt.Sprintf("%d: %s %q", i, cellType, truncateNotebookSource(source)))
	}
	return strings.Join(lines, "\n")
}

func editNotebookCells(cells []any, input NotebookEditInput) ([]any, string, error) {
	if input.Index < 0 || input.Index > len(cells) || (input.Command != "insert_cell" && input.Index == len(cells)) {
		return nil, "", fmt.Errorf("cell index %d out of range", input.Index)
	}
	next := append([]any(nil), cells...)
	switch input.Command {
	case "replace_cell":
		next[input.Index] = notebookCell(input.CellType, input.Source)
		return next, fmt.Sprintf("Replaced cell %d", input.Index), nil
	case "insert_cell":
		cell := notebookCell(input.CellType, input.Source)
		next = append(next, nil)
		copy(next[input.Index+1:], next[input.Index:])
		next[input.Index] = cell
		return next, fmt.Sprintf("Inserted cell %d", input.Index), nil
	case "delete_cell":
		next = append(next[:input.Index], next[input.Index+1:]...)
		return next, fmt.Sprintf("Deleted cell %d", input.Index), nil
	default:
		return nil, "", fmt.Errorf("unsupported notebook command %q", input.Command)
	}
}

func notebookCell(cellType, source string) map[string]any {
	if cellType == "" {
		cellType = "code"
	}
	cell := map[string]any{
		"cell_type": cellType,
		"metadata":  map[string]any{},
		"source":    strings.SplitAfter(source, "\n"),
	}
	if cellType == "code" {
		cell["execution_count"] = nil
		cell["outputs"] = []any{}
	}
	return cell
}

func sourceString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case []any:
		var b strings.Builder
		for _, part := range x {
			b.WriteString(fmt.Sprint(part))
		}
		return b.String()
	default:
		return ""
	}
}

func truncateNotebookSource(source string) string {
	source = strings.TrimSpace(source)
	if len(source) <= 80 {
		return source
	}
	return source[:80] + "..."
}

func WebSearchURL(query string) string {
	return "https://duckduckgo.com/html/?" + url.Values{"q": []string{query}}.Encode()
}
