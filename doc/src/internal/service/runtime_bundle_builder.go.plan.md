# runtime_bundle_builder.go 技术说明

## 1. 文件定位
- 源文件: `internal/service/runtime_bundle_builder.go`
- 文档文件: `doc/src/internal/service/runtime_bundle_builder.go.plan.md`
- 所属模块: service

## 2. 核心职责
- 封装 `RunnerBundle` 构建流程，让 `ChatService` 不直接承载 runtime catalog、ToolSearch middleware 和 agent 装配细节。
- 按固定顺序组装 runtime core tools、`Agent` tool、deferred runtime tools、web tools、MCP action/resource tools 和 dev-only sample。
- 构建 default/plan 两套 runtime handlers，并把顶层 agent 交给 `buildTopLevelRuntimeAgents(...)`。

## 3. 关键实现细节
- `Build(...)` 负责创建 chat model、探测 agentic beta protocol、构造 generation-bound `deferredMCPProvider`、设置 LSP config、构造 subagent registry、安装 default/plan `ChatModelAgent`。
- `buildRuntimeCatalog(...)` 只负责 catalog 组装和 permission wrapper，不直接启动 runner。
- `buildRuntimeHandlers(...)` 只负责 dynamic MCP surface 与 Eino v0.9 ToolSearch bridge。
- `registerWrappedCatalogEntries(...)` 统一 catalog 注册错误文案，避免各类 runtime/MCP tool 分散重复注册逻辑。
- `runtimeAlwaysLoadedTools(...)` 从 catalog 中提取 always-load 工具面，deferred 工具只通过 ToolSearch 暴露。

## 4. 维护边界
- 这里仍位于 `service` package，以便复用现有未导出的 task/worktree/LSP/provider 类型；后续可以再把 provider/host 接口外移到 runtime package。
- 新增 runtime tool source 时优先扩展 `buildRuntimeCatalog(...)`，不要把组装逻辑放回 `ChatService.prepareRunnerBundleFromSurface(...)`。
- 新增 Eino middleware 或 runtime engine 时优先扩展 `buildRuntimeHandlers(...)` / `buildTopLevelRuntimeAgents(...)`。
