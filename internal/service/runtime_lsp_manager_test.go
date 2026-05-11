package service

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cloudwego/eino-ext/components/tool/commandline"

	"starxo/internal/config"
	"starxo/internal/sandbox"
	"starxo/internal/tools"
)

type fakeLSPRuntimeOperator struct {
	files          map[string]string
	failWriteOnce  map[string]error
	startCount     int
	startCommands  [][]string
	processFactory func() sandbox.RuntimeProcess
}

type fakeLSPWorkspaceManager struct {
	current string
}

func (m fakeLSPWorkspaceManager) CurrentWorkspace(ctx context.Context, defaultWorkspace string) string {
	if m.current == "" {
		return defaultWorkspace
	}
	return m.current
}

func (m fakeLSPWorkspaceManager) EnterWorktree(ctx context.Context, op commandline.Operator, defaultWorkspace, name string) (tools.WorktreeOutput, error) {
	return tools.WorktreeOutput{}, fmt.Errorf("not implemented")
}

func (m fakeLSPWorkspaceManager) ExitWorktree(ctx context.Context, op commandline.Operator, defaultWorkspace, action string, discardChanges bool) (tools.WorktreeOutput, error) {
	return tools.WorktreeOutput{}, fmt.Errorf("not implemented")
}

func (m fakeLSPWorkspaceManager) DiffWorktree(ctx context.Context, op commandline.Operator, defaultWorkspace string, includePatch bool, maxBytes int) (tools.WorktreeDiffOutput, error) {
	return tools.WorktreeDiffOutput{}, fmt.Errorf("not implemented")
}

func (m fakeLSPWorkspaceManager) MergeWorktree(ctx context.Context, op commandline.Operator, defaultWorkspace string, commitMessage string, removeWorktree bool) (tools.WorktreeMergeOutput, error) {
	return tools.WorktreeMergeOutput{}, fmt.Errorf("not implemented")
}

func (o *fakeLSPRuntimeOperator) ReadFile(ctx context.Context, filePath string) (string, error) {
	content, ok := o.files[filePath]
	if !ok {
		return "", fmt.Errorf("not found")
	}
	return content, nil
}

func (o *fakeLSPRuntimeOperator) WriteFile(ctx context.Context, filePath string, content string) error {
	if err, ok := o.failWriteOnce[filePath]; ok {
		delete(o.failWriteOnce, filePath)
		return err
	}
	o.files[filePath] = content
	return nil
}

func (o *fakeLSPRuntimeOperator) IsDirectory(ctx context.Context, filePath string) (bool, error) {
	return false, nil
}

func (o *fakeLSPRuntimeOperator) Exists(ctx context.Context, filePath string) (bool, error) {
	_, ok := o.files[filePath]
	return ok, nil
}

func (o *fakeLSPRuntimeOperator) RunCommand(ctx context.Context, command []string) (*commandline.CommandOutput, error) {
	cmd := strings.Join(command, " ")
	if strings.Contains(cmd, "command -v 'gopls'") || strings.Contains(cmd, "command -v 'custom-ls'") {
		return &commandline.CommandOutput{ExitCode: 0}, nil
	}
	return &commandline.CommandOutput{ExitCode: 0, Stdout: "go\n"}, nil
}

func (o *fakeLSPRuntimeOperator) StartProcess(ctx context.Context, command []string) (sandbox.RuntimeProcess, error) {
	o.startCount++
	o.startCommands = append(o.startCommands, append([]string(nil), command...))
	if o.processFactory != nil {
		return o.processFactory(), nil
	}
	return newFakeLanguageServerProcess(), nil
}

type fakeLanguageServerProcess struct {
	clientInW       *io.PipeWriter
	clientOutR      *io.PipeReader
	killOnce        sync.Once
	done            chan struct{}
	formatResult    any
	formatResultSet bool
}

func newFakeLanguageServerProcess(options ...func(*fakeLanguageServerProcess)) *fakeLanguageServerProcess {
	clientInR, clientInW := io.Pipe()
	clientOutR, clientOutW := io.Pipe()
	p := &fakeLanguageServerProcess{
		clientInW:  clientInW,
		clientOutR: clientOutR,
		done:       make(chan struct{}),
	}
	for _, option := range options {
		option(p)
	}
	go p.serve(clientInR, clientOutW)
	return p
}

func fakeLanguageServerFormatResult(result any) func(*fakeLanguageServerProcess) {
	return func(p *fakeLanguageServerProcess) {
		p.formatResult = result
		p.formatResultSet = true
	}
}

func (p *fakeLanguageServerProcess) Stdin() io.WriteCloser { return p.clientInW }

func (p *fakeLanguageServerProcess) Stdout() io.Reader { return p.clientOutR }

func (p *fakeLanguageServerProcess) Stderr() io.Reader { return strings.NewReader("") }

func (p *fakeLanguageServerProcess) Wait() error {
	<-p.done
	return nil
}

func (p *fakeLanguageServerProcess) Kill() error {
	p.killOnce.Do(func() {
		_ = p.clientInW.Close()
		_ = p.clientOutR.Close()
	})
	return nil
}

func (p *fakeLanguageServerProcess) serve(r io.Reader, w io.WriteCloser) {
	defer close(p.done)
	defer w.Close()
	reader := bufio.NewReader(r)
	for {
		payload, err := readRuntimeLSPPayload(reader)
		if err != nil {
			return
		}
		var msg map[string]any
		if err := json.Unmarshal(payload, &msg); err != nil {
			return
		}
		method, _ := msg["method"].(string)
		id, hasID := msg["id"]
		if !hasID {
			continue
		}
		var result any
		switch method {
		case "initialize":
			result = map[string]any{"capabilities": map[string]any{}}
		case "textDocument/definition":
			result = []map[string]any{{
				"uri": "file:///workspace/main.go",
				"range": map[string]any{
					"start": map[string]any{"line": 0, "character": 0},
					"end":   map[string]any{"line": 0, "character": 4},
				},
			}}
		case "textDocument/rename":
			result = map[string]any{
				"changes": map[string]any{
					"file:///workspace/main.go": []map[string]any{{
						"range": map[string]any{
							"start": map[string]any{"line": 1, "character": 5},
							"end":   map[string]any{"line": 1, "character": 8},
						},
						"newText": "newName",
					}},
				},
			}
		case "textDocument/formatting":
			if p.formatResultSet {
				result = p.formatResult
			} else {
				result = []map[string]any{{
					"range": map[string]any{
						"start": map[string]any{"line": 0, "character": 0},
						"end":   map[string]any{"line": 2, "character": 0},
					},
					"newText": "package main\n\nfunc old() {}\n",
				}}
			}
		default:
			result = nil
		}
		_ = writeFakeLSPMessage(w, map[string]any{
			"jsonrpc": "2.0",
			"id":      id,
			"result":  result,
		})
	}
}

func writeFakeLSPMessage(w io.Writer, msg any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(data)); err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func TestRuntimeLSPManagerReusesPersistentServer(t *testing.T) {
	op := &fakeLSPRuntimeOperator{files: map[string]string{
		"/workspace/main.go": "package main\nfunc main() {}\n",
	}}
	manager := newRuntimeLSPManager(nil)
	defer manager.CloseAll()

	input := tools.LSPInput{
		Operation: "definition",
		FilePath:  "main.go",
		Line:      2,
		Character: 6,
	}
	for i := 0; i < 2; i++ {
		out, handled, err := manager.Query(context.Background(), op, "/workspace", nil, input)
		if err != nil {
			t.Fatalf("query %d: %v", i, err)
		}
		if !handled {
			t.Fatalf("query %d was not handled by LSP", i)
		}
		if out.Engine != "lsp:go" || out.ResultCount != 1 || !strings.Contains(out.Result, "main.go") {
			t.Fatalf("unexpected LSP output: %#v", out)
		}
	}
	if op.startCount != 1 {
		t.Fatalf("expected one persistent server, got %d starts", op.startCount)
	}
}

func TestRuntimeLSPManagerAppliesRenameEdit(t *testing.T) {
	op := &fakeLSPRuntimeOperator{files: map[string]string{
		"/workspace/main.go": "package main\nfunc old() {}\n",
	}}
	manager := newRuntimeLSPManager(nil)
	defer manager.CloseAll()

	out, handled, err := manager.Edit(context.Background(), op, "/workspace", nil, tools.LSPEditInput{
		Operation: "rename",
		FilePath:  "main.go",
		Line:      2,
		Character: 6,
		NewName:   "newName",
	})
	if err != nil {
		t.Fatalf("edit rename: %v", err)
	}
	if !handled || out.Engine != "lsp:go" || out.EditCount != 1 {
		t.Fatalf("unexpected edit output: handled=%v out=%#v", handled, out)
	}
	if got := op.files["/workspace/main.go"]; got != "package main\nfunc newName() {}\n" {
		t.Fatalf("unexpected renamed content: %q", got)
	}
}

func TestRuntimeLSPManagerAppliesFormattingEdit(t *testing.T) {
	op := &fakeLSPRuntimeOperator{files: map[string]string{
		"/workspace/main.go": "package main\nfunc old() {}\n",
	}}
	manager := newRuntimeLSPManager(nil)
	defer manager.CloseAll()

	out, handled, err := manager.Edit(context.Background(), op, "/workspace", nil, tools.LSPEditInput{
		Operation: "format",
		FilePath:  "main.go",
	})
	if err != nil {
		t.Fatalf("edit format: %v", err)
	}
	if !handled || out.EditCount != 1 {
		t.Fatalf("unexpected format output: handled=%v out=%#v", handled, out)
	}
	if got := op.files["/workspace/main.go"]; got != "package main\n\nfunc old() {}\n" {
		t.Fatalf("unexpected formatted content: %q", got)
	}
}

func TestRuntimeLSPManagerHandlesNoopFormattingEdit(t *testing.T) {
	op := &fakeLSPRuntimeOperator{
		files: map[string]string{
			"/workspace/main.go": "package main\n\nfunc old() {}\n",
		},
		processFactory: func() sandbox.RuntimeProcess {
			return newFakeLanguageServerProcess(fakeLanguageServerFormatResult(nil))
		},
	}
	manager := newRuntimeLSPManager(nil)
	defer manager.CloseAll()

	out, handled, err := manager.Edit(context.Background(), op, "/workspace", nil, tools.LSPEditInput{
		Operation: "format",
		FilePath:  "main.go",
	})
	if err != nil {
		t.Fatalf("noop format: %v", err)
	}
	if !handled || out.EditCount != 0 || len(out.ChangedFiles) != 0 {
		t.Fatalf("unexpected noop format output: handled=%v out=%#v", handled, out)
	}
	if got := op.files["/workspace/main.go"]; got != "package main\n\nfunc old() {}\n" {
		t.Fatalf("noop format should not modify content: %q", got)
	}
}

func TestRuntimeLSPApplyWorkspaceEditRejectsUnsupportedDocumentChanges(t *testing.T) {
	op := &fakeLSPRuntimeOperator{files: map[string]string{
		"/workspace/main.go": "package main\n",
	}}
	raw := json.RawMessage(`{"documentChanges":[{"kind":"rename","oldUri":"file:///workspace/main.go","newUri":"file:///workspace/renamed.go"}]}`)

	_, err := runtimeLSPApplyWorkspaceEdit(context.Background(), op, "/workspace", "/workspace", nil, raw)
	if err == nil || !strings.Contains(err.Error(), "unsupported LSP document change") {
		t.Fatalf("expected unsupported document change error, got %v", err)
	}
	if got := op.files["/workspace/main.go"]; got != "package main\n" {
		t.Fatalf("unsupported document change should not modify files, got %q", got)
	}
}

func TestRuntimeLSPApplyWorkspaceEditDoesNotPartiallyWrite(t *testing.T) {
	op := &fakeLSPRuntimeOperator{files: map[string]string{
		"/workspace/a.go": "old\n",
	}}
	raw := json.RawMessage(`{"changes":{
		"file:///workspace/a.go":[{"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":3}},"newText":"new"}],
		"file:///workspace/b.go":[{"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":0}},"newText":"missing"}]
	}}`)

	_, err := runtimeLSPApplyWorkspaceEdit(context.Background(), op, "/workspace", "/workspace", nil, raw)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected missing second file error, got %v", err)
	}
	if got := op.files["/workspace/a.go"]; got != "old\n" {
		t.Fatalf("first file should not be written before all edits validate, got %q", got)
	}
}

func TestRuntimeLSPApplyWorkspaceEditRollsBackFailedWrites(t *testing.T) {
	op := &fakeLSPRuntimeOperator{
		files: map[string]string{
			"/workspace/a.go": "oldA\n",
			"/workspace/b.go": "oldB\n",
		},
		failWriteOnce: map[string]error{"/workspace/b.go": fmt.Errorf("disk full")},
	}
	raw := json.RawMessage(`{"changes":{
		"file:///workspace/a.go":[{"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":4}},"newText":"newA"}],
		"file:///workspace/b.go":[{"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":4}},"newText":"newB"}]
	}}`)

	_, err := runtimeLSPApplyWorkspaceEdit(context.Background(), op, "/workspace", "/workspace", nil, raw)
	if err == nil || !strings.Contains(err.Error(), "rolled back") {
		t.Fatalf("expected rollback error, got %v", err)
	}
	if got := op.files["/workspace/a.go"]; got != "oldA\n" {
		t.Fatalf("first file should be rolled back, got %q", got)
	}
	if got := op.files["/workspace/b.go"]; got != "oldB\n" {
		t.Fatalf("failed file should be restored, got %q", got)
	}
}

func TestRuntimeLSPApplyWorkspaceEditAcceptsAbsoluteActiveWorktreeTargets(t *testing.T) {
	activeWorkspace := "/workspace/.starxo/worktrees/feat"
	target := activeWorkspace + "/main.go"
	op := &fakeLSPRuntimeOperator{files: map[string]string{
		target: "old\n",
	}}
	raw := json.RawMessage(`{"changes":{
		"file:///workspace/.starxo/worktrees/feat/main.go":[{"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":3}},"newText":"new"}]
	}}`)

	out, err := runtimeLSPApplyWorkspaceEdit(context.Background(), op, activeWorkspace, "/workspace", fakeLSPWorkspaceManager{current: activeWorkspace}, raw)
	if err != nil {
		t.Fatalf("apply worktree absolute edit: %v", err)
	}
	if out.EditCount != 1 || len(out.ChangedFiles) != 1 || out.ChangedFiles[0] != target {
		t.Fatalf("unexpected worktree edit output: %#v", out)
	}
	if got := op.files[target]; got != "new\n" {
		t.Fatalf("expected active worktree file to be edited, got %q", got)
	}
}

func TestRuntimeLSPManagerUsesConfiguredServerAndStatus(t *testing.T) {
	op := &fakeLSPRuntimeOperator{files: map[string]string{
		"/workspace/app.foo": "symbol demo\n",
	}}
	manager := newRuntimeLSPManager(func() time.Time { return time.UnixMilli(1700000000000) })
	defer manager.CloseAll()
	enabled := true
	manager.SetConfig(config.RuntimeLSPConfig{
		Enabled:          &enabled,
		RequestTimeoutMS: 5000,
		MaxResultBytes:   4096,
		Servers: []config.RuntimeLSPServerConfig{{
			Language:   "foo",
			Executable: "custom-ls",
			Command:    []string{"custom-ls", "--stdio"},
			Extensions: []string{".foo"},
		}},
	})

	ctx := context.WithValue(context.Background(), "sessionID", "sess-1")
	out, handled, err := manager.Query(ctx, op, "/workspace", nil, tools.LSPInput{
		Operation: "definition",
		FilePath:  "app.foo",
		Line:      1,
		Character: 1,
	})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !handled || out.Engine != "lsp:foo" {
		t.Fatalf("expected configured LSP to handle query, handled=%v out=%#v", handled, out)
	}
	startCommand := strings.Join(op.startCommands[0], " ")
	if op.startCount != 1 || !strings.Contains(startCommand, "custom-ls") || !strings.Contains(startCommand, "--stdio") {
		t.Fatalf("expected custom server command, starts=%d commands=%#v", op.startCount, op.startCommands)
	}
	status := manager.Status("sess-1")
	if !status.Enabled || status.RequestTimeoutMS != 5000 || status.MaxResultBytes != 4096 {
		t.Fatalf("unexpected status config: %#v", status)
	}
	if len(status.Servers) != 1 || status.Servers[0].Language != "foo" || status.Servers[0].OpenDocs != 1 || status.Servers[0].RequestCount != 1 {
		t.Fatalf("unexpected status servers: %#v", status.Servers)
	}
}

func TestRuntimeLSPManagerDisabledFallsBack(t *testing.T) {
	enabled := false
	manager := newRuntimeLSPManager(nil)
	manager.SetConfig(config.RuntimeLSPConfig{Enabled: &enabled})
	out, handled, err := manager.Query(context.Background(), &fakeLSPRuntimeOperator{}, "/workspace", nil, tools.LSPInput{
		Operation: "workspace_symbol",
		Symbol:    "main",
		Language:  "go",
	})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if handled || out.Result != "" {
		t.Fatalf("disabled LSP should not handle query, handled=%v out=%#v", handled, out)
	}
}

func TestGetRuntimeLSPStatusRefreshesSavedConfig(t *testing.T) {
	store := newTestConfigStore(t)
	chat := NewChatService(store)
	enabled := false
	if err := store.Update(func(cfg *config.AppConfig) {
		cfg.Agent.LSP.Enabled = &enabled
		cfg.Agent.LSP.RequestTimeoutMS = 4321
		cfg.Agent.LSP.MaxResultBytes = 2048
	}); err != nil {
		t.Fatalf("update config: %v", err)
	}

	status, err := chat.GetRuntimeLSPStatus("sess-1")
	if err != nil {
		t.Fatalf("get LSP status: %v", err)
	}
	if status.Enabled {
		t.Fatalf("expected saved disabled config in status, got %#v", status)
	}
	if status.RequestTimeoutMS != 4321 || status.MaxResultBytes != 2048 {
		t.Fatalf("expected refreshed saved config in status, got %#v", status)
	}
}
