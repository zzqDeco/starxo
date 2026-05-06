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

	"github.com/cloudwego/eino-ext/components/tool/commandline"

	"starxo/internal/sandbox"
	"starxo/internal/tools"
)

type fakeLSPRuntimeOperator struct {
	files      map[string]string
	startCount int
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
	if strings.Contains(cmd, "command -v 'gopls'") {
		return &commandline.CommandOutput{ExitCode: 0}, nil
	}
	return &commandline.CommandOutput{ExitCode: 0, Stdout: "go\n"}, nil
}

func (o *fakeLSPRuntimeOperator) StartProcess(ctx context.Context, command []string) (sandbox.RuntimeProcess, error) {
	o.startCount++
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
