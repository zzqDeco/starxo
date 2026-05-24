package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/filesystem"
	"github.com/cloudwego/eino/adk/middlewares/agentsmd"
	"github.com/cloudwego/eino/adk/middlewares/reduction"
	"github.com/cloudwego/eino/adk/middlewares/skill"
	"github.com/cloudwego/eino/adk/middlewares/summarization"
	einomodel "github.com/cloudwego/eino/components/model"
	"gopkg.in/yaml.v3"
)

func NewEinoV09ContextMiddlewares(ctx context.Context, mdl einomodel.ToolCallingChatModel, op commandline.Operator, ac AgentContext) ([]adk.ChatModelAgentMiddleware, error) {
	backend := &remoteWorkspaceBackend{op: op, workspace: ac.WorkspacePath}
	var handlers []adk.ChatModelAgentMiddleware

	summary, err := summarizationMiddleware(ctx, mdl, ac)
	if err != nil {
		return nil, err
	}
	if summary != nil {
		handlers = append(handlers, summary)
	}

	reducer, err := reduction.New(ctx, &reduction.Config{
		Backend:                   backend,
		ReadFileToolName:          "Read",
		RootDir:                   ".starxo/tool-results",
		MaxLengthForTrunc:         32 * 1024,
		MaxTokensForClear:         120000,
		ClearAtLeastTokens:        8000,
		ClearRetentionSuffixLimit: 4,
	})
	if err != nil {
		return nil, fmt.Errorf("create Eino v0.9 reduction middleware: %w", err)
	}
	handlers = append(handlers, reducer)

	skillMW, err := skill.NewMiddleware(ctx, &skill.Config{
		Backend: &remoteSkillBackend{backend: backend},
	})
	if err != nil {
		return nil, fmt.Errorf("create Eino v0.9 skill middleware: %w", err)
	}
	handlers = append(handlers, skillMW)

	agentsMD, err := agentsmd.New(ctx, &agentsmd.Config{
		Backend:             backend,
		AgentsMDFiles:       []string{path.Join(ac.WorkspacePath, "AGENTS.md"), path.Join(ac.WorkspacePath, ".starxo/AGENTS.md")},
		AllAgentsMDMaxBytes: 64 * 1024,
	})
	if err != nil {
		return nil, fmt.Errorf("create Eino v0.9 agentsmd middleware: %w", err)
	}
	handlers = append(handlers, agentsMD)
	return handlers, nil
}

func summarizationMiddleware(ctx context.Context, mdl einomodel.ToolCallingChatModel, ac AgentContext) (adk.ChatModelAgentMiddleware, error) {
	if mdl == nil {
		return nil, nil
	}
	mw, err := summarizationNew(ctx, mdl, ac)
	if err != nil {
		return nil, fmt.Errorf("create Eino v0.9 summarization middleware: %w", err)
	}
	return mw, nil
}

func summarizationNew(ctx context.Context, mdl einomodel.ToolCallingChatModel, ac AgentContext) (adk.ChatModelAgentMiddleware, error) {
	return summarization.New(ctx, &summarization.Config{
		Model: mdl,
		Trigger: &summarization.TriggerCondition{
			ContextTokens:   120000,
			ContextMessages: 180,
		},
		TranscriptFilePath: path.Join(ac.WorkspacePath, ".starxo/transcript.md"),
	})
}

type remoteWorkspaceBackend struct {
	op        commandline.Operator
	workspace string
}

func (b *remoteWorkspaceBackend) Read(ctx context.Context, req *filesystem.ReadRequest) (*filesystem.FileContent, error) {
	if b == nil || b.op == nil || req == nil {
		return nil, os.ErrNotExist
	}
	filePath, err := b.resolve(req.FilePath)
	if err != nil {
		return nil, err
	}
	exists, err := b.op.Exists(ctx, filePath)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("%w: %s", os.ErrNotExist, filePath)
	}
	content, err := b.op.ReadFile(ctx, filePath)
	if err != nil {
		return nil, err
	}
	return &filesystem.FileContent{Content: content}, nil
}

func (b *remoteWorkspaceBackend) Write(ctx context.Context, req *filesystem.WriteRequest) error {
	if b == nil || b.op == nil || req == nil {
		return errors.New("workspace backend is not available")
	}
	filePath, err := b.resolve(req.FilePath)
	if err != nil {
		return err
	}
	dir := path.Dir(filePath)
	if _, err := b.op.RunCommand(ctx, []string{"mkdir", "-p", dir}); err != nil {
		return err
	}
	return b.op.WriteFile(ctx, filePath, req.Content)
}

func (b *remoteWorkspaceBackend) resolve(filePath string) (string, error) {
	root := path.Clean(strings.TrimSpace(b.workspace))
	if root == "" || root == "." || root == "/" {
		return "", fmt.Errorf("invalid workspace root %q", b.workspace)
	}
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return root, nil
	}
	if strings.HasPrefix(filePath, "/") {
		cleaned := path.Clean(filePath)
		if cleaned == root || strings.HasPrefix(cleaned, root+"/") {
			return cleaned, nil
		}
		return "", fmt.Errorf("workspace backend path outside workspace: %s", filePath)
	}
	cleaned := path.Clean(path.Join(root, filePath))
	if cleaned == root || strings.HasPrefix(cleaned, root+"/") {
		return cleaned, nil
	}
	return "", fmt.Errorf("workspace backend path outside workspace: %s", filePath)
}

type remoteSkillBackend struct {
	backend *remoteWorkspaceBackend
}

func (b *remoteSkillBackend) List(ctx context.Context) ([]skill.FrontMatter, error) {
	if b == nil || b.backend == nil || b.backend.op == nil {
		return nil, nil
	}
	cmd := "cd " + shellQuote(b.backend.workspace) + ` && for d in .starxo/skills .claude/skills; do [ -d "$d" ] && find "$d" -mindepth 2 -maxdepth 2 -name SKILL.md -print; done 2>/dev/null | sort`
	out, err := b.backend.op.RunCommand(ctx, []string{"sh", "-c", cmd})
	if err != nil {
		return nil, nil
	}
	lines := strings.Split(strings.TrimSpace(out.Stdout), "\n")
	matters := make([]skill.FrontMatter, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		s, err := b.readSkill(ctx, line)
		if err != nil {
			continue
		}
		matters = append(matters, s.FrontMatter)
	}
	sort.Slice(matters, func(i, j int) bool { return matters[i].Name < matters[j].Name })
	return matters, nil
}

func (b *remoteSkillBackend) Get(ctx context.Context, name string) (skill.Skill, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return skill.Skill{}, fmt.Errorf("skill name is required")
	}
	if !isSafeSkillName(name) {
		return skill.Skill{}, fmt.Errorf("invalid skill name %q", name)
	}
	candidates := []string{
		path.Join(".starxo/skills", name, "SKILL.md"),
		path.Join(".claude/skills", name, "SKILL.md"),
	}
	for _, candidate := range candidates {
		s, err := b.readSkill(ctx, candidate)
		if err == nil {
			return s, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return skill.Skill{}, err
		}
	}
	return skill.Skill{}, fmt.Errorf("%w: skill %s", os.ErrNotExist, name)
}

func isSafeSkillName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	if strings.ContainsAny(name, `/\`) {
		return false
	}
	return path.Clean(name) == name
}

func (b *remoteSkillBackend) readSkill(ctx context.Context, relPath string) (skill.Skill, error) {
	content, err := b.backend.Read(ctx, &filesystem.ReadRequest{FilePath: relPath})
	if err != nil {
		return skill.Skill{}, err
	}
	fm, body := parseSkillFrontMatter(content.Content)
	if fm.Name == "" {
		fm.Name = path.Base(path.Dir(relPath))
	}
	if fm.Description == "" {
		fm.Description = fm.Name
	}
	return skill.Skill{
		FrontMatter:   fm,
		Content:       strings.TrimSpace(body),
		BaseDirectory: path.Dir(b.backend.mustResolve(relPath)),
	}, nil
}

func (b *remoteWorkspaceBackend) mustResolve(filePath string) string {
	resolved, err := b.resolve(filePath)
	if err != nil {
		return b.workspace
	}
	return resolved
}

func parseSkillFrontMatter(content string) (skill.FrontMatter, string) {
	content = strings.TrimLeft(content, "\ufeff")
	if !strings.HasPrefix(content, "---\n") {
		return skill.FrontMatter{}, content
	}
	rest := strings.TrimPrefix(content, "---\n")
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return skill.FrontMatter{}, content
	}
	raw := rest[:idx]
	body := strings.TrimPrefix(rest[idx:], "\n---")
	body = strings.TrimPrefix(body, "\n")
	var fm skill.FrontMatter
	_ = yaml.Unmarshal([]byte(raw), &fm)
	return fm, body
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
