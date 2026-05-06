# runtime_deferred_tools.go 技术说明

## 1. 文件定位
- 源文件: `internal/tools/runtime_deferred_tools.go`
- 文档文件: `doc/src/internal/tools/runtime_deferred_tools.go.plan.md`
- 所属模块: tools

## 2. 核心职责
- 定义 Runtime V2 非 MCP deferred tools 的 DTO、catalog entry 和 fallback 实现。
- 当前包含 `LSP`、`Skill`、`NotebookEdit`，并提供 `WebFetch` / `WebSearch` catalog helper。

## 3. 输入与输出
- 输入来源: remote sandbox operator、workspace path、runtime workspace manager、runtime LSP manager。
- 输出结果: deferred runtime catalog entries 和结构化 tool result。

## 4. 关键实现细节
- `LSP` 优先调用 service 层常驻 language server manager；当 server 不可用、未安装或请求缺少 position 信息时，降级到远端 `rg`/`sed` fallback。
- `LSPInput` 支持 `language` override 和 1-based `character`，`LSPOutput` 返回 `engine`、`language` 和 `fallbackReason`。
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
- 扩展 LSP server 能力时保持当前 DTO 兼容，并继续保留 fallback 作为无 server 环境下的降级路径。
- Notebook 写入前后若增加 diff UI，应复用现有 structured result，不要在工具内输出大 JSON。
