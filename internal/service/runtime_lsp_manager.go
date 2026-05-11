package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino-ext/components/tool/commandline"

	"starxo/internal/config"
	"starxo/internal/logger"
	"starxo/internal/sandbox"
	"starxo/internal/tools"
)

const (
	runtimeLSPRequestTimeout = 15 * time.Second
	runtimeLSPMaxResultBytes = 16 * 1024
)

type runtimeLSPProcessStarter interface {
	StartProcess(ctx context.Context, command []string) (sandbox.RuntimeProcess, error)
}

type runtimeLSPManager struct {
	mu      sync.Mutex
	servers map[string]*runtimeLSPServer
	now     func() time.Time
	cfg     config.RuntimeLSPConfig
}

type runtimeLSPServerSpec struct {
	Language   string
	Executable string
	Command    []string
}

func newRuntimeLSPManager(now func() time.Time) *runtimeLSPManager {
	if now == nil {
		now = time.Now
	}
	return &runtimeLSPManager{
		servers: make(map[string]*runtimeLSPServer),
		now:     now,
		cfg:     normalizeRuntimeLSPConfig(config.RuntimeLSPConfig{}),
	}
}

type RuntimeLSPStatus struct {
	Enabled          bool                     `json:"enabled"`
	RequestTimeoutMS int                      `json:"requestTimeoutMs"`
	MaxResultBytes   int                      `json:"maxResultBytes"`
	Configured       []RuntimeLSPConfigured   `json:"configured"`
	Servers          []RuntimeLSPServerStatus `json:"servers"`
}

type RuntimeLSPConfigured struct {
	Language   string   `json:"language"`
	Executable string   `json:"executable"`
	Command    []string `json:"command"`
	Extensions []string `json:"extensions,omitempty"`
	Disabled   bool     `json:"disabled,omitempty"`
}

type RuntimeLSPServerStatus struct {
	SessionID    string `json:"sessionId"`
	Workspace    string `json:"workspace"`
	Language     string `json:"language"`
	Executable   string `json:"executable"`
	Command      string `json:"command"`
	Alive        bool   `json:"alive"`
	OpenDocs     int    `json:"openDocs"`
	RequestCount int64  `json:"requestCount"`
	StartedAt    int64  `json:"startedAt"`
	LastUsedAt   int64  `json:"lastUsedAt,omitempty"`
	LastError    string `json:"lastError,omitempty"`
}

func (m *runtimeLSPManager) SetConfig(cfg config.RuntimeLSPConfig) {
	cfg = normalizeRuntimeLSPConfig(cfg)
	m.mu.Lock()
	m.cfg = cfg
	m.mu.Unlock()
}

func (m *runtimeLSPManager) Status(sessionID string) RuntimeLSPStatus {
	cfg := m.configSnapshot()
	m.mu.Lock()
	servers := make([]RuntimeLSPServerStatus, 0, len(m.servers))
	for _, server := range m.servers {
		if sessionID != "" && server.sessionID != sessionID {
			continue
		}
		servers = append(servers, server.status())
	}
	m.mu.Unlock()
	return RuntimeLSPStatus{
		Enabled:          runtimeLSPEnabled(cfg),
		RequestTimeoutMS: cfg.RequestTimeoutMS,
		MaxResultBytes:   cfg.MaxResultBytes,
		Configured:       runtimeLSPConfiguredStatus(cfg),
		Servers:          servers,
	}
}

func (m *runtimeLSPManager) configSnapshot() config.RuntimeLSPConfig {
	m.mu.Lock()
	defer m.mu.Unlock()
	return cloneRuntimeLSPConfig(m.cfg)
}

func (m *runtimeLSPManager) Query(ctx context.Context, op commandline.Operator, workspacePath string, workspaces tools.RuntimeWorkspaceManager, input tools.LSPInput) (tools.LSPOutput, bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cfg := m.configSnapshot()
	if !runtimeLSPEnabled(cfg) {
		return tools.LSPOutput{}, false, nil
	}
	opName := runtimeLSPOperation(input.Operation)
	if !runtimeLSPCanHandle(opName, input) {
		return tools.LSPOutput{}, false, nil
	}
	starter, ok := op.(runtimeLSPProcessStarter)
	if !ok {
		return tools.LSPOutput{}, false, fmt.Errorf("sandbox operator does not support persistent processes")
	}
	workspace := cleanRuntimeRemotePath(workspacePath)
	if workspaces != nil {
		workspace = cleanRuntimeRemotePath(workspaces.CurrentWorkspace(ctx, workspacePath))
	}
	if workspace == "" || workspace == "." {
		return tools.LSPOutput{}, false, fmt.Errorf("sandbox workspace is not active")
	}

	spec, languageID, ok, err := m.resolveServerSpec(ctx, op, workspace, input, cfg)
	if err != nil || !ok {
		return tools.LSPOutput{}, false, err
	}

	var targetPath, content string
	if runtimeLSPNeedsDocument(opName) {
		targetPath, err = runtimeLSPWorkspaceFilePath(ctx, input.FilePath, workspacePath, workspaces)
		if err != nil {
			return tools.LSPOutput{}, false, err
		}
		content, err = op.ReadFile(ctx, targetPath)
		if err != nil {
			return tools.LSPOutput{}, false, err
		}
		languageID = runtimeLSPLanguageIDForFile(spec.Language, input.FilePath)
	}

	server, err := m.server(ctx, starter, workspace, spec, cfg)
	if err != nil {
		return tools.LSPOutput{}, false, err
	}
	if targetPath != "" {
		if err := server.openOrChange(ctx, targetPath, languageID, content); err != nil {
			m.dropServer(server.key)
			return tools.LSPOutput{}, false, err
		}
	}
	result, count, err := server.query(ctx, opName, targetPath, input, runtimeLSPTimeout(cfg), runtimeLSPMaxBytes(cfg))
	if err != nil {
		m.dropServer(server.key)
		return tools.LSPOutput{}, false, err
	}
	return tools.LSPOutput{
		Operation:   opName,
		Engine:      "lsp:" + spec.Language,
		Language:    spec.Language,
		Result:      result,
		FilePath:    input.FilePath,
		ResultCount: count,
	}, true, nil
}

func (m *runtimeLSPManager) CloseAll() {
	m.mu.Lock()
	servers := make([]*runtimeLSPServer, 0, len(m.servers))
	for _, server := range m.servers {
		servers = append(servers, server)
	}
	m.servers = make(map[string]*runtimeLSPServer)
	m.mu.Unlock()
	for _, server := range servers {
		server.close()
	}
}

func (m *runtimeLSPManager) dropServer(key string) {
	m.mu.Lock()
	server := m.servers[key]
	delete(m.servers, key)
	m.mu.Unlock()
	if server != nil {
		server.close()
	}
}

func (m *runtimeLSPManager) server(ctx context.Context, starter runtimeLSPProcessStarter, workspace string, spec runtimeLSPServerSpec, cfg config.RuntimeLSPConfig) (*runtimeLSPServer, error) {
	sessionID := workspaceSessionID(ctx)
	key := strings.Join([]string{sessionID, workspace, spec.Language, spec.Executable, strings.Join(spec.Command, "\x00")}, "\x00")
	m.mu.Lock()
	if server := m.servers[key]; server != nil && server.isAlive() {
		m.mu.Unlock()
		return server, nil
	}
	m.mu.Unlock()

	serverCmd := strings.Join(shellQuoteRuntimeArgs(spec.Command), " ")
	startCmd := "cd " + shellQuoteRuntime(workspace) + " && exec " + serverCmd
	proc, err := starter.StartProcess(ctx, []string{"sh", "-lc", startCmd})
	if err != nil {
		return nil, err
	}
	server := newRuntimeLSPServer(key, sessionID, workspace, spec, proc, m.now().UnixMilli())
	if err := server.initialize(ctx, runtimeLSPTimeout(cfg)); err != nil {
		server.close()
		return nil, err
	}

	m.mu.Lock()
	if existing := m.servers[key]; existing != nil && existing.isAlive() {
		m.mu.Unlock()
		server.close()
		return existing, nil
	}
	m.servers[key] = server
	m.mu.Unlock()
	return server, nil
}

func (m *runtimeLSPManager) resolveServerSpec(ctx context.Context, op commandline.Operator, workspace string, input tools.LSPInput, cfg config.RuntimeLSPConfig) (runtimeLSPServerSpec, string, bool, error) {
	language := runtimeLSPNormalizeLanguage(input.Language)
	if language == "" {
		language = runtimeLSPLanguageFromPath(input.FilePath, cfg)
	}
	if language == "" {
		language = m.detectWorkspaceLanguage(ctx, op, workspace)
	}
	if language == "" {
		return runtimeLSPServerSpec{}, "", false, fmt.Errorf("could not infer LSP language")
	}
	spec, ok := runtimeLSPServerSpecForLanguage(language, cfg)
	if !ok {
		return runtimeLSPServerSpec{}, "", false, fmt.Errorf("no LSP server mapping for language %q", language)
	}
	available, err := runtimeLSPCommandAvailable(ctx, op, workspace, spec.Executable)
	if err != nil {
		return runtimeLSPServerSpec{}, "", false, err
	}
	if !available {
		return runtimeLSPServerSpec{}, "", false, fmt.Errorf("language server %s is not installed on the remote sandbox", spec.Executable)
	}
	return spec, runtimeLSPLanguageIDForFile(spec.Language, input.FilePath), true, nil
}

func (m *runtimeLSPManager) detectWorkspaceLanguage(ctx context.Context, op commandline.Operator, workspace string) string {
	cmd := "cd " + shellQuoteRuntime(workspace) + " && " +
		"if [ -f go.mod ]; then echo go; " +
		"elif [ -f package.json ] || find . -maxdepth 2 \\( -name '*.ts' -o -name '*.tsx' \\) -print -quit 2>/dev/null | grep -q .; then echo typescript; " +
		"elif [ -f pyproject.toml ] || [ -f requirements.txt ] || find . -maxdepth 2 -name '*.py' -print -quit 2>/dev/null | grep -q .; then echo python; " +
		"elif [ -f Cargo.toml ]; then echo rust; fi"
	output, err := op.RunCommand(ctx, []string{"sh", "-lc", cmd})
	if err != nil || output.ExitCode != 0 {
		return ""
	}
	return runtimeLSPNormalizeLanguage(strings.TrimSpace(output.Stdout))
}

type runtimeLSPServer struct {
	key       string
	sessionID string
	root      string
	spec      runtimeLSPServerSpec
	proc      sandbox.RuntimeProcess
	stdin     io.WriteCloser
	reader    *bufio.Reader
	writeMu   sync.Mutex

	mu           sync.Mutex
	nextID       int64
	pending      map[int64]chan runtimeLSPResponse
	openDocs     map[string]*runtimeLSPOpenDoc
	readErr      error
	readDone     chan struct{}
	closeOnce    sync.Once
	startedAt    int64
	lastUsedAt   int64
	requestCount int64
	lastError    string
}

type runtimeLSPResponse struct {
	Result json.RawMessage
	Error  *runtimeLSPError
}

type runtimeLSPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type runtimeLSPOpenDoc struct {
	Content  string
	Version  int
	Language string
}

func newRuntimeLSPServer(key, sessionID, root string, spec runtimeLSPServerSpec, proc sandbox.RuntimeProcess, startedAt int64) *runtimeLSPServer {
	server := &runtimeLSPServer{
		key:       key,
		sessionID: sessionID,
		root:      root,
		spec:      spec,
		proc:      proc,
		stdin:     proc.Stdin(),
		reader:    bufio.NewReader(proc.Stdout()),
		pending:   make(map[int64]chan runtimeLSPResponse),
		openDocs:  make(map[string]*runtimeLSPOpenDoc),
		readDone:  make(chan struct{}),
		startedAt: startedAt,
	}
	go server.readLoop()
	go io.Copy(io.Discard, proc.Stderr())
	go func() {
		if err := proc.Wait(); err != nil {
			logger.Debug("[LSP] language server exited", "language", spec.Language, "error", err)
		}
	}()
	return server
}

func (s *runtimeLSPServer) isAlive() bool {
	select {
	case <-s.readDone:
		return false
	default:
		return true
	}
}

func (s *runtimeLSPServer) close() {
	s.closeOnce.Do(func() {
		_ = s.proc.Kill()
	})
}

func (s *runtimeLSPServer) initialize(ctx context.Context, timeout time.Duration) error {
	initCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	_, err := s.call(initCtx, "initialize", map[string]any{
		"processId": nil,
		"rootUri":   runtimeLSPFileURI(s.root),
		"workspaceFolders": []map[string]any{{
			"uri":  runtimeLSPFileURI(s.root),
			"name": path.Base(s.root),
		}},
		"capabilities": map[string]any{
			"textDocument": map[string]any{
				"definition":     map[string]any{},
				"references":     map[string]any{},
				"hover":          map[string]any{},
				"documentSymbol": map[string]any{},
				"synchronization": map[string]any{
					"didSave": true,
				},
			},
			"workspace": map[string]any{
				"symbol":        map[string]any{},
				"configuration": true,
			},
		},
	})
	if err != nil {
		return err
	}
	return s.notify(context.Background(), "initialized", map[string]any{})
}

func (s *runtimeLSPServer) openOrChange(ctx context.Context, filePath, languageID, content string) error {
	uri := runtimeLSPFileURI(filePath)
	s.mu.Lock()
	doc := s.openDocs[uri]
	if doc == nil {
		s.openDocs[uri] = &runtimeLSPOpenDoc{Content: content, Version: 1, Language: languageID}
		s.mu.Unlock()
		return s.notify(ctx, "textDocument/didOpen", map[string]any{
			"textDocument": map[string]any{
				"uri":        uri,
				"languageId": languageID,
				"version":    1,
				"text":       content,
			},
		})
	}
	if doc.Content == content && doc.Language == languageID {
		s.mu.Unlock()
		return nil
	}
	doc.Content = content
	doc.Language = languageID
	doc.Version++
	version := doc.Version
	s.mu.Unlock()
	return s.notify(ctx, "textDocument/didChange", map[string]any{
		"textDocument": map[string]any{
			"uri":     uri,
			"version": version,
		},
		"contentChanges": []map[string]any{{"text": content}},
	})
}

func (s *runtimeLSPServer) query(ctx context.Context, operation, filePath string, input tools.LSPInput, timeout time.Duration, maxBytes int) (string, int, error) {
	method, params := runtimeLSPMethodAndParams(operation, filePath, input)
	s.markRequestStart()
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	result, err := s.call(callCtx, method, params)
	if err != nil {
		s.markRequestError(err.Error())
		return "", 0, err
	}
	formatted, count := runtimeLSPFormatResult(result, maxBytes)
	return formatted, count, nil
}

func (s *runtimeLSPServer) markRequestStart() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastUsedAt = time.Now().UnixMilli()
	s.requestCount++
	s.lastError = ""
}

func (s *runtimeLSPServer) markRequestError(lastErr string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastUsedAt = time.Now().UnixMilli()
	s.lastError = lastErr
}

func (s *runtimeLSPServer) status() RuntimeLSPServerStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	status := RuntimeLSPServerStatus{
		SessionID:    s.sessionID,
		Workspace:    s.root,
		Language:     s.spec.Language,
		Executable:   s.spec.Executable,
		Command:      strings.Join(s.spec.Command, " "),
		Alive:        s.isAlive(),
		OpenDocs:     len(s.openDocs),
		RequestCount: s.requestCount,
		StartedAt:    s.startedAt,
		LastUsedAt:   s.lastUsedAt,
		LastError:    s.lastError,
	}
	if status.LastError == "" && s.readErr != nil {
		status.LastError = s.readErr.Error()
	}
	return status
}

func (s *runtimeLSPServer) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id := s.nextRequestID()
	ch := make(chan runtimeLSPResponse, 1)
	s.mu.Lock()
	s.pending[id] = ch
	s.mu.Unlock()

	if err := s.writeMessage(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}); err != nil {
		s.mu.Lock()
		delete(s.pending, id)
		s.mu.Unlock()
		return nil, err
	}

	select {
	case resp := <-ch:
		if resp.Error != nil {
			return nil, fmt.Errorf("LSP %s failed: %s", method, resp.Error.Message)
		}
		return resp.Result, nil
	case <-s.readDone:
		return nil, fmt.Errorf("LSP server stopped: %v", s.readErr)
	case <-ctx.Done():
		s.mu.Lock()
		delete(s.pending, id)
		s.mu.Unlock()
		return nil, ctx.Err()
	}
}

func (s *runtimeLSPServer) notify(ctx context.Context, method string, params any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.writeMessage(map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	})
}

func (s *runtimeLSPServer) nextRequestID() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	return s.nextID
}

func (s *runtimeLSPServer) readLoop() {
	defer close(s.readDone)
	for {
		payload, err := readRuntimeLSPPayload(s.reader)
		if err != nil {
			s.mu.Lock()
			s.readErr = err
			for id, ch := range s.pending {
				delete(s.pending, id)
				ch <- runtimeLSPResponse{Error: &runtimeLSPError{Message: err.Error()}}
			}
			s.mu.Unlock()
			return
		}
		var msg map[string]json.RawMessage
		if err := json.Unmarshal(payload, &msg); err != nil {
			continue
		}
		idRaw, hasID := msg["id"]
		if !hasID {
			continue
		}
		if _, isServerRequest := msg["method"]; isServerRequest {
			s.replyToServerRequest(idRaw, msg)
			continue
		}
		id, ok := runtimeLSPNumericID(idRaw)
		if !ok {
			continue
		}
		var resp runtimeLSPResponse
		_ = json.Unmarshal(payload, &resp)
		s.mu.Lock()
		ch := s.pending[id]
		delete(s.pending, id)
		s.mu.Unlock()
		if ch != nil {
			ch <- resp
		}
	}
}

func (s *runtimeLSPServer) replyToServerRequest(id json.RawMessage, msg map[string]json.RawMessage) {
	var method string
	_ = json.Unmarshal(msg["method"], &method)
	var result any
	if method == "workspace/configuration" {
		result = []any{}
	}
	_ = s.writeMessage(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	})
}

func (s *runtimeLSPServer) writeMessage(msg any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if _, err := fmt.Fprintf(s.stdin, "Content-Length: %d\r\n\r\n", len(data)); err != nil {
		return err
	}
	_, err = s.stdin.Write(data)
	return err
}

func readRuntimeLSPPayload(reader *bufio.Reader) ([]byte, error) {
	for {
		contentLength := 0
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return nil, err
			}
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				break
			}
			lower := strings.ToLower(line)
			if strings.HasPrefix(lower, "content-length:") {
				idx := strings.Index(line, ":")
				raw := ""
				if idx >= 0 {
					raw = strings.TrimSpace(line[idx+1:])
				}
				contentLength, _ = strconv.Atoi(raw)
			}
		}
		if contentLength <= 0 {
			continue
		}
		payload := make([]byte, contentLength)
		_, err := io.ReadFull(reader, payload)
		return payload, err
	}
}

func runtimeLSPNumericID(raw json.RawMessage) (int64, bool) {
	var n int64
	if err := json.Unmarshal(raw, &n); err == nil {
		return n, true
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return int64(f), true
	}
	return 0, false
}

func runtimeLSPMethodAndParams(operation, filePath string, input tools.LSPInput) (string, any) {
	switch operation {
	case "definition":
		return "textDocument/definition", runtimeLSPTextDocumentPosition(filePath, input)
	case "references":
		params := runtimeLSPTextDocumentPosition(filePath, input)
		params["context"] = map[string]any{"includeDeclaration": true}
		return "textDocument/references", params
	case "hover":
		return "textDocument/hover", runtimeLSPTextDocumentPosition(filePath, input)
	case "document_symbol":
		return "textDocument/documentSymbol", map[string]any{
			"textDocument": map[string]any{"uri": runtimeLSPFileURI(filePath)},
		}
	case "workspace_symbol":
		return "workspace/symbol", map[string]any{"query": input.Symbol}
	default:
		return "workspace/symbol", map[string]any{"query": input.Symbol}
	}
}

func runtimeLSPTextDocumentPosition(filePath string, input tools.LSPInput) map[string]any {
	line := input.Line - 1
	if line < 0 {
		line = 0
	}
	character := input.Character - 1
	if character < 0 {
		character = 0
	}
	return map[string]any{
		"textDocument": map[string]any{"uri": runtimeLSPFileURI(filePath)},
		"position": map[string]any{
			"line":      line,
			"character": character,
		},
	}
}

func runtimeLSPFormatResult(raw json.RawMessage, maxBytes int) (string, int) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return "No LSP result.", 0
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(trimmed), 1
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		data = trimmed
	}
	count := runtimeLSPResultCount(v)
	if maxBytes <= 0 {
		maxBytes = runtimeLSPMaxResultBytes
	}
	if len(data) > maxBytes {
		data = append(data[:maxBytes], []byte("\n[truncated]")...)
	}
	return string(data), count
}

func runtimeLSPResultCount(v any) int {
	switch x := v.(type) {
	case []any:
		return len(x)
	case map[string]any:
		return 1
	case nil:
		return 0
	default:
		return 1
	}
}

func runtimeLSPOperation(operation string) string {
	operation = strings.TrimSpace(operation)
	if operation == "" {
		return "workspace_symbol"
	}
	return operation
}

func runtimeLSPCanHandle(operation string, input tools.LSPInput) bool {
	switch operation {
	case "definition", "references", "hover":
		return strings.TrimSpace(input.FilePath) != "" && input.Line > 0
	case "document_symbol":
		return strings.TrimSpace(input.FilePath) != ""
	case "workspace_symbol":
		return strings.TrimSpace(input.Symbol) != ""
	default:
		return false
	}
}

func runtimeLSPNeedsDocument(operation string) bool {
	switch operation {
	case "definition", "references", "hover", "document_symbol":
		return true
	default:
		return false
	}
}

func runtimeLSPServerSpecForLanguage(language string, cfg config.RuntimeLSPConfig) (runtimeLSPServerSpec, bool) {
	language = runtimeLSPNormalizeLanguage(language)
	for _, server := range cfg.Servers {
		if server.Disabled {
			continue
		}
		if runtimeLSPNormalizeLanguage(server.Language) != language {
			continue
		}
		spec, ok := runtimeLSPServerSpecFromConfig(server)
		if ok {
			return spec, true
		}
	}
	switch language {
	case "go":
		return runtimeLSPServerSpec{Language: "go", Executable: "gopls", Command: []string{"gopls", "serve"}}, true
	case "typescript":
		return runtimeLSPServerSpec{Language: "typescript", Executable: "typescript-language-server", Command: []string{"typescript-language-server", "--stdio"}}, true
	case "javascript":
		return runtimeLSPServerSpec{Language: "javascript", Executable: "typescript-language-server", Command: []string{"typescript-language-server", "--stdio"}}, true
	case "python":
		return runtimeLSPServerSpec{Language: "python", Executable: "pyright-langserver", Command: []string{"pyright-langserver", "--stdio"}}, true
	case "rust":
		return runtimeLSPServerSpec{Language: "rust", Executable: "rust-analyzer", Command: []string{"rust-analyzer"}}, true
	default:
		return runtimeLSPServerSpec{}, false
	}
}

func runtimeLSPServerSpecFromConfig(server config.RuntimeLSPServerConfig) (runtimeLSPServerSpec, bool) {
	language := runtimeLSPNormalizeLanguage(server.Language)
	if language == "" {
		return runtimeLSPServerSpec{}, false
	}
	command := append([]string(nil), server.Command...)
	executable := strings.TrimSpace(server.Executable)
	if len(command) == 0 && executable != "" {
		command = []string{executable}
	}
	if len(command) == 0 {
		return runtimeLSPServerSpec{}, false
	}
	if executable == "" {
		executable = command[0]
	}
	return runtimeLSPServerSpec{Language: language, Executable: executable, Command: command}, true
}

func runtimeLSPNormalizeLanguage(language string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "go", "golang":
		return "go"
	case "ts", "tsx", "typescript", "typescriptreact":
		return "typescript"
	case "js", "jsx", "javascript", "javascriptreact":
		return "javascript"
	case "py", "python":
		return "python"
	case "rs", "rust":
		return "rust"
	default:
		normalized := strings.ToLower(strings.TrimSpace(language))
		if isRuntimeLSPSafeIdentifier(normalized) {
			return normalized
		}
		return ""
	}
}

func runtimeLSPLanguageFromPath(filePath string, cfg config.RuntimeLSPConfig) string {
	ext := strings.ToLower(path.Ext(filePath))
	for _, server := range cfg.Servers {
		if server.Disabled {
			continue
		}
		for _, candidate := range server.Extensions {
			candidate = strings.ToLower(strings.TrimSpace(candidate))
			if candidate != "" && !strings.HasPrefix(candidate, ".") {
				candidate = "." + candidate
			}
			if candidate != "" && candidate == ext {
				return runtimeLSPNormalizeLanguage(server.Language)
			}
		}
	}
	switch ext {
	case ".go":
		return "go"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx", ".mjs", ".cjs":
		return "javascript"
	case ".py", ".pyi":
		return "python"
	case ".rs":
		return "rust"
	default:
		return ""
	}
}

func runtimeLSPLanguageIDForFile(language, filePath string) string {
	ext := strings.ToLower(path.Ext(filePath))
	switch runtimeLSPNormalizeLanguage(language) {
	case "go":
		return "go"
	case "typescript":
		if ext == ".tsx" {
			return "typescriptreact"
		}
		return "typescript"
	case "javascript":
		if ext == ".jsx" {
			return "javascriptreact"
		}
		return "javascript"
	case "python":
		return "python"
	case "rust":
		return "rust"
	default:
		return language
	}
}

func normalizeRuntimeLSPConfig(cfg config.RuntimeLSPConfig) config.RuntimeLSPConfig {
	cfg = cloneRuntimeLSPConfig(cfg)
	if cfg.Enabled == nil {
		enabled := true
		cfg.Enabled = &enabled
	}
	if cfg.RequestTimeoutMS <= 0 {
		cfg.RequestTimeoutMS = int(runtimeLSPRequestTimeout / time.Millisecond)
	}
	if cfg.MaxResultBytes <= 0 {
		cfg.MaxResultBytes = runtimeLSPMaxResultBytes
	}
	return cfg
}

func cloneRuntimeLSPConfig(cfg config.RuntimeLSPConfig) config.RuntimeLSPConfig {
	out := cfg
	if cfg.Enabled != nil {
		enabled := *cfg.Enabled
		out.Enabled = &enabled
	}
	out.Servers = append([]config.RuntimeLSPServerConfig(nil), cfg.Servers...)
	for i := range out.Servers {
		out.Servers[i].Command = append([]string(nil), out.Servers[i].Command...)
		out.Servers[i].Extensions = append([]string(nil), out.Servers[i].Extensions...)
	}
	return out
}

func runtimeLSPEnabled(cfg config.RuntimeLSPConfig) bool {
	return cfg.Enabled == nil || *cfg.Enabled
}

func runtimeLSPTimeout(cfg config.RuntimeLSPConfig) time.Duration {
	if cfg.RequestTimeoutMS <= 0 {
		return runtimeLSPRequestTimeout
	}
	return time.Duration(cfg.RequestTimeoutMS) * time.Millisecond
}

func runtimeLSPMaxBytes(cfg config.RuntimeLSPConfig) int {
	if cfg.MaxResultBytes <= 0 {
		return runtimeLSPMaxResultBytes
	}
	return cfg.MaxResultBytes
}

func runtimeLSPConfiguredStatus(cfg config.RuntimeLSPConfig) []RuntimeLSPConfigured {
	defaults := []config.RuntimeLSPServerConfig{
		{Language: "go", Executable: "gopls", Command: []string{"gopls", "serve"}, Extensions: []string{".go"}},
		{Language: "typescript", Executable: "typescript-language-server", Command: []string{"typescript-language-server", "--stdio"}, Extensions: []string{".ts", ".tsx"}},
		{Language: "javascript", Executable: "typescript-language-server", Command: []string{"typescript-language-server", "--stdio"}, Extensions: []string{".js", ".jsx", ".mjs", ".cjs"}},
		{Language: "python", Executable: "pyright-langserver", Command: []string{"pyright-langserver", "--stdio"}, Extensions: []string{".py", ".pyi"}},
		{Language: "rust", Executable: "rust-analyzer", Command: []string{"rust-analyzer"}, Extensions: []string{".rs"}},
	}
	merged := append(defaults, cfg.Servers...)
	out := make([]RuntimeLSPConfigured, 0, len(merged))
	for _, item := range merged {
		language := runtimeLSPNormalizeLanguage(item.Language)
		if language == "" {
			continue
		}
		spec, ok := runtimeLSPServerSpecFromConfig(item)
		if !ok {
			spec, ok = runtimeLSPServerSpecForLanguage(language, config.RuntimeLSPConfig{})
		}
		if !ok {
			continue
		}
		out = append(out, RuntimeLSPConfigured{
			Language:   language,
			Executable: spec.Executable,
			Command:    append([]string(nil), spec.Command...),
			Extensions: append([]string(nil), item.Extensions...),
			Disabled:   item.Disabled,
		})
	}
	return out
}

func isRuntimeLSPSafeIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}

func runtimeLSPCommandAvailable(ctx context.Context, op commandline.Operator, workspace, executable string) (bool, error) {
	cmd := "cd " + shellQuoteRuntime(workspace) + " && command -v " + shellQuoteRuntime(executable) + " >/dev/null 2>&1"
	output, err := op.RunCommand(ctx, []string{"sh", "-lc", cmd})
	if err != nil {
		return false, err
	}
	return output.ExitCode == 0, nil
}

func runtimeLSPWorkspaceFilePath(ctx context.Context, filePath, workspacePath string, workspaces tools.RuntimeWorkspaceManager) (string, error) {
	workspace := cleanRuntimeRemotePath(workspacePath)
	if workspaces != nil {
		workspace = cleanRuntimeRemotePath(workspaces.CurrentWorkspace(ctx, workspacePath))
	}
	if workspace == "" || workspace == "." {
		return "", fmt.Errorf("sandbox workspace is not active")
	}
	p := strings.TrimSpace(filePath)
	if p == "" {
		return "", fmt.Errorf("file_path is required")
	}
	if p == "/workspace" {
		p = workspace
	} else if strings.HasPrefix(p, "/workspace/") {
		p = path.Join(workspace, strings.TrimPrefix(p, "/workspace/"))
	} else if !strings.HasPrefix(p, "/") {
		p = path.Join(workspace, p)
	}
	p = cleanRuntimeRemotePath(p)
	if p != workspace && !strings.HasPrefix(p, workspace+"/") {
		return "", fmt.Errorf("path %s is outside sandbox workspace %s", filePath, workspace)
	}
	return p, nil
}

func runtimeLSPFileURI(filePath string) string {
	return (&url.URL{Scheme: "file", Path: filePath}).String()
}

func shellQuoteRuntimeArgs(args []string) []string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		quoted[i] = shellQuoteRuntime(arg)
	}
	return quoted
}
