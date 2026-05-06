# runtime_lsp_manager.go 技术说明

## 1. 文件定位
- 源文件: `internal/service/runtime_lsp_manager.go`
- 文档文件: `doc/src/internal/service/runtime_lsp_manager.go.plan.md`
- 所属模块: service

## 2. 核心职责
- 为 Runtime V2 `LSP` 工具提供常驻 language server 管理。
- 通过远端 sandbox 长进程 stdio 运行 LSP server，并用 JSON-RPC 2.0 执行 initialize、didOpen/didChange 和查询请求。
- 在 language server 不可用、未安装、操作不适合 LSP 时，让 tools 层降级到 `rg`/`sed` fallback。

## 3. 输入与输出
- 输入来源: `tools.RuntimeLSPManager.Query`、当前 sandbox operator、workspace path、session context、`tools.LSPInput`。
- 输出结果: `tools.LSPOutput`，包含 `engine=lsp:<language>`、language、格式化 JSON result 和 result count。

## 4. 关键实现细节
- server key: `sessionID + workspacePath + language`，同一 session/workspace/language 复用同一个常驻进程。
- 支持语言与命令：
  - Go: `gopls serve`
  - TypeScript/JavaScript: `typescript-language-server --stdio`
  - Python: `pyright-langserver --stdio`
  - Rust: `rust-analyzer`
- `definition`、`references`、`hover` 使用 file/line/character position。
- `document_symbol` 使用当前文件。
- `workspace_symbol` 使用 query string。
- 每次文件查询前读取当前文件内容，首次发送 `textDocument/didOpen`，内容变化后发送 full-sync `textDocument/didChange`。
- 收到 server request 时返回空结果，避免 `workspace/configuration` 等请求阻塞 server。
- `ChatService.UpdateSandbox` 和 `InvalidateRunner` 会关闭所有常驻 LSP server，避免 SSH/sandbox 切换后保留旧进程。

## 5. 依赖关系
- 内部依赖: `internal/sandbox`、`internal/tools`、`runtime_workspaces.go`
- 外部依赖: Go stdlib JSON/IO。

## 6. 变更影响面
- `LSP` 从纯 `rg` fallback 升级为 persistent language-server-backed tool。
- 远端需要安装对应 language server；未安装时仍可使用 fallback。
- Worktree 模式下会按当前 active workspace 启动独立 server。

## 7. 维护建议
- 新增语言时只扩展 language -> server command 映射和 languageId 映射。
- 后续若支持 writable LSP 操作（rename/codeAction apply/format），必须拆成可写 tool 或在 permission spec 中单独走审批。
