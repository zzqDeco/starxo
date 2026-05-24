# runtime_tool_provider.go 技术说明

## 1. 文件定位
- 源文件: `internal/service/runtime_tool_provider.go`
- 文档文件: `doc/src/internal/service/runtime_tool_provider.go.plan.md`
- 所属模块: service

## 2. 核心职责
- 提供 generation-bound runtime tool provider，把 runner bundle 的 catalog、MCP handles、session discovery 和 permission context 组合成工具运行期视图。
- 实现 dynamic MCP surface、ToolSearch、permission wrapper、MCP resource adapter 所需的 provider 接口。

## 3. 关键实现细节
- `deferredMCPProvider` 绑定一个 `RunnerBundle`，catalog/handles 固定在该 generation，discovery 和 mode 每次从 `SessionRun` 读取。
- 子 agent 可通过 context-scoped runtime mode override 覆盖读取到的 session mode；该 override 用于让 fork/fresh subagent 的 ToolSearch surface 与实际 permission mode 保持一致。
- `PrepareDeferredSyntheticMessages(...)` 只计算当前 session 的 deferred surface delta 和 MCP instructions delta，commit 在模型调用成功后推进。
- `AddDiscoveredTools(...)` 只接受当前 bundle catalog 中仍是 deferred 且非 always-load 的 canonical tool，并触发 session save。
- `newDeferredUnknownToolHandler(...)` 给模型返回 runtime-aware unknown tool 说明，区分已加载、可搜索但未加载和当前模式不可用。

## 4. 维护边界
- 该 provider 仍依赖 `ChatService` 读取 session state；后续如果继续解耦，应先抽 `RuntimeSessionStateStore` 接口。
- 不在这里构建 catalog 或 runner；catalog 由 `runtimeBundleBuilder` 构建。
