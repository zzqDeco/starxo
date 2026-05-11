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
- 输出结果: `tools.LSPOutput` 或 `tools.LSPEditOutput`，包含 `engine=lsp:<language>`、language、格式化 JSON result / apply summary 和 result/edit count。

## 4. 关键实现细节
- server key: `sessionID + workspacePath + language + executable + command`，同一 session/workspace/language/command 复用同一个常驻进程。
- `agent.lsp` 支持启用/禁用、请求超时、最大结果字节数、自定义 language server command 与扩展名映射。
- 支持语言与命令：
  - Go: `gopls serve`
  - TypeScript/JavaScript: `typescript-language-server --stdio`
  - Python: `pyright-langserver --stdio`
  - Rust: `rust-analyzer`
- `definition`、`references`、`hover` 使用 file/line/character position。
- `document_symbol` 使用当前文件。
- `workspace_symbol` 使用 query string。
- `rename` / `format` 通过独立 writable `LSPEdit` tool 调用，解析 LSP `WorkspaceEdit` / `TextEdit[]`，校验目标 URI 仍在当前 workspace 内，再经 operator 写回文件。
- LSP text edit 使用 0-based line + UTF-16 character position，写回前按倒序 range 应用，避免后续 edit 影响前面 offset。
- `format` 返回 `null` 或空 `TextEdit[]` 时按成功 no-op 处理，不丢弃常驻 server。
- 多文件 edit 写入失败时会尽力回滚已写入文件，避免返回错误后留下明显的半应用状态。
- Worktree 模式下，LSP 返回 active workspace 内的绝对 `file://` URI 时直接接受；`/workspace/...` alias 仍映射到当前 active workspace。
- 每次文件查询前读取当前文件内容，首次发送 `textDocument/didOpen`，内容变化后发送 full-sync `textDocument/didChange`。
- 收到 server request 时返回空结果，避免 `workspace/configuration` 等请求阻塞 server。
- `ChatService.UpdateSandbox` 和 `InvalidateRunner` 会关闭所有常驻 LSP server，避免 SSH/sandbox 切换后保留旧进程。
- `ChatService.GetRuntimeLSPStatus(sessionID)` 返回当前配置、活动 server、open docs、request count、last error，供设置页诊断使用。

## 5. 依赖关系
- 内部依赖: `internal/sandbox`、`internal/tools`、`runtime_workspaces.go`
- 外部依赖: Go stdlib JSON/IO。

## 6. 变更影响面
- `LSP` 从纯 `rg` fallback 升级为 persistent language-server-backed tool；`LSPEdit` 提供需要审批的 rename/format 写入操作。
- 远端需要安装对应 language server；未安装时仍可使用 fallback。
- Worktree 模式下会按当前 active workspace 启动独立 server。

## 7. 维护建议
- 新增常规语言时可扩展内置 language -> server command 映射；项目级特殊语言优先通过 `agent.lsp.servers` 自定义配置。
- 后续若继续扩展 writable LSP 操作（如 codeAction apply），应继续放在 `LSPEdit` 或另建 writable tool，并复用 workspace edit 路径守卫。
