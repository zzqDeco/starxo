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
	files         map[string]string
	startCount    int
	startCommands [][]string
}

func (o *fakeLSPRuntimeOperator) ReadFile(ctx context.Context, filePath string) (string, error) {
	content, ok := o.files[filePath]
	if !ok {
		return "", fmt.Errorf("not found")
	}
	return content, nil
}

func (o *fakeLSPRuntimeOperator) WriteFile(ctx context.Context, filePath string, content string) error {
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
	return newFakeLanguageServerProcess(), nil
}

type fakeLanguageServerProcess struct {
	clientInW  *io.PipeWriter
	clientOutR *io.PipeReader
	killOnce   sync.Once
	done       chan struct{}
}

func newFakeLanguageServerProcess() *fakeLanguageServerProcess {
	clientInR, clientInW := io.Pipe()
	clientOutR, clientOutW := io.Pipe()
	p := &fakeLanguageServerProcess{
		clientInW:  clientInW,
		clientOutR: clientOutR,
		done:       make(chan struct{}),
	}
	go p.serve(clientInR, clientOutW)
	return p
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
