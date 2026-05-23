package agent

import "testing"

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
