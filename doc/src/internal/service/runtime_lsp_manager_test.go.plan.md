# runtime_lsp_manager_test.go 技术说明

## 1. 文件定位
- 源文件: `internal/service/runtime_lsp_manager_test.go`
- 文档文件: `doc/src/internal/service/runtime_lsp_manager_test.go.plan.md`
- 所属模块: service

## 2. 核心职责
- 覆盖 Runtime V2 常驻 LSP manager 的进程复用能力。

## 3. 输入与输出
- 输入来源: fake commandline operator、fake sandbox long-running process、`tools.LSPInput`。
- 输出结果: `tools.LSPOutput` 和 fake process start count。

## 4. 关键测试覆盖
- `definition` 查询可通过 fake language server 返回结果。
- 连续两次同 workspace/language 查询只启动一个常驻 server。
- 输出包含 `engine=lsp:go` 和 LSP result count。
- 自定义 `agent.lsp.servers` 可覆盖 language server command，并在 status API 中显示 open docs/request count。
- `agent.lsp.enabled=false` 时 manager 不处理查询，让 tools 层 fallback。

## 5. 维护建议
- 新增 LSP lifecycle 行为时优先补 fake JSON-RPC server 测试，避免依赖真实远端 language server。
