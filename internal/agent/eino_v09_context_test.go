package agent

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/cloudwego/eino-ext/components/tool/commandline"
)

func TestRemoteWorkspaceBackendResolveGuardsWorkspace(t *testing.T) {
	backend := &remoteWorkspaceBackend{workspace: "/workspace"}

	got, err := backend.resolve("src/main.go")
	if err != nil {
		t.Fatalf("resolve relative path: %v", err)
	}
	if got != "/workspace/src/main.go" {
		t.Fatalf("unexpected relative path: %q", got)
	}

	got, err = backend.resolve("/workspace/AGENTS.md")
	if err != nil {
		t.Fatalf("resolve workspace absolute path: %v", err)
	}
	if got != "/workspace/AGENTS.md" {
		t.Fatalf("unexpected workspace absolute path: %q", got)
	}

	if _, err := backend.resolve("../outside"); err == nil {
		t.Fatalf("expected parent traversal to be rejected")
	}
	if _, err := backend.resolve("/etc/passwd"); err == nil {
		t.Fatalf("expected outside absolute path to be rejected")
	}
}

func TestRemoteSkillBackendListIgnoresMissingSkillRoots(t *testing.T) {
	op := &fakeContextOperator{
		files: map[string]string{
			"/workspace/.starxo/skills/demo/SKILL.md": "---\nname: demo\ndescription: Demo skill\n---\nBody",
		},
		run: func(command []string) (*commandline.CommandOutput, error) {
			joined := strings.Join(command, " ")
			if strings.Contains(joined, "find .starxo/skills .claude/skills") {
				return nil, errors.New("find failed because one root is missing")
			}
			return &commandline.CommandOutput{Stdout: ".starxo/skills/demo/SKILL.md\n"}, nil
		},
	}
	backend := &remoteSkillBackend{backend: &remoteWorkspaceBackend{op: op, workspace: "/workspace"}}

	skills, err := backend.List(context.Background())
	if err != nil {
		t.Fatalf("list skills: %v", err)
	}
	if len(skills) != 1 || skills[0].Name != "demo" {
		t.Fatalf("expected discovered demo skill, got %#v", skills)
	}
}

func TestRemoteSkillBackendGetRejectsPathTraversalName(t *testing.T) {
	backend := &remoteSkillBackend{backend: &remoteWorkspaceBackend{workspace: "/workspace"}}
	if _, err := backend.Get(context.Background(), "../.."); err == nil {
		t.Fatalf("expected traversal skill name to be rejected")
	}
	if _, err := backend.Get(context.Background(), "nested/name"); err == nil {
		t.Fatalf("expected slash skill name to be rejected")
	}
}

type fakeContextOperator struct {
	files map[string]string
	run   func([]string) (*commandline.CommandOutput, error)
}

func (o *fakeContextOperator) ReadFile(_ context.Context, filePath string) (string, error) {
	if o != nil && o.files != nil {
		if content, ok := o.files[filePath]; ok {
			return content, nil
		}
	}
	return "", os.ErrNotExist
}

func (o *fakeContextOperator) WriteFile(_ context.Context, filePath string, content string) error {
	if o.files == nil {
		o.files = map[string]string{}
	}
	o.files[filePath] = content
	return nil
}

func (o *fakeContextOperator) IsDirectory(context.Context, string) (bool, error) {
	return false, nil
}

func (o *fakeContextOperator) Exists(_ context.Context, filePath string) (bool, error) {
	if o != nil && o.files != nil {
		_, ok := o.files[filePath]
		return ok, nil
	}
	return false, nil
}

func (o *fakeContextOperator) RunCommand(_ context.Context, command []string) (*commandline.CommandOutput, error) {
	if o.run != nil {
		return o.run(command)
	}
	return &commandline.CommandOutput{}, nil
}
