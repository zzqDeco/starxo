# runtime_deferred_tools.go 技术说明

## 1. 文件定位
- 源文件: `internal/tools/runtime_deferred_tools.go`
- 文档文件: `doc/src/internal/tools/runtime_deferred_tools.go.plan.md`
- 所属模块: tools

## 2. 核心职责
- 定义 Runtime V2 非 MCP deferred tools 的 DTO、catalog entry 和轻量实现。
- 当前包含 `LSP`、`Skill`、`NotebookEdit`，并提供 `WebFetch` / `WebSearch` catalog helper。

## 3. 输入与输出
- 输入来源: remote sandbox operator、workspace path、runtime workspace manager。
- 输出结果: deferred runtime catalog entries 和结构化 tool result。

## 4. 关键实现细节
- `LSP` 是轻量 code-intelligence fallback，不启动持久语言服务器；通过远端 `rg`/`sed` 支持 definition、references、hover、document_symbol、workspace_symbol。
- `Skill` 支持列出和读取 workspace 内 `.starxo/skills` / `.claude/skills` 的 `SKILL.md`。
- `NotebookEdit` 解析 `.ipynb` JSON，支持 view、replace_cell、insert_cell、delete_cell，并强制走 workspace path guard。
- `WebFetch` / `WebSearch` 的实际 HTTP 实现在 service 层，本文件只创建 runtime catalog metadata。
- read-only trusted 工具: `LSP`、`Skill`、`WebFetch`、`WebSearch`；可写工具: `NotebookEdit`。

## 5. 依赖关系
- 内部依赖: `catalog.go`、`runtime_tools.go`
- 外部依赖: `github.com/cloudwego/eino/components/tool/utils`

## 6. 变更影响面
- ToolSearch 可发现更多 runtime-wide deferred tools，不再局限 MCP。
- plan mode 只会暴露 read-only trusted deferred tools；`NotebookEdit` 仍需 default/approved 执行面。

## 7. 维护建议
- 真正接入 LSP server 时应保持当前 DTO 兼容，并把 fallback 作为无 LSP server 时的降级路径。
- Notebook 写入前后若增加 diff UI，应复用现有 structured result，不要在工具内输出大 JSON。
