# runtime_toolsearch_eino.go 技术说明

## 1. 文件定位
- 源文件: `internal/service/runtime_toolsearch_eino.go`
- 文档文件: `doc/src/internal/service/runtime_toolsearch_eino.go.plan.md`
- 所属模块: service

## 2. 核心职责
- 封装 Eino v0.9 dynamic ToolSearch middleware 适配。
- 将 Starxo `ToolCatalog` 的 deferred entries 转换为 Eino dynamic tools，同时保留 Starxo 的 mode/permission/discovery gate。

## 3. 关键实现细节
- `newEinoV09ToolSearchHandler(...)` 面向完整 runner catalog。
- `newEinoV09ToolSearchHandlerForCatalog(...)` 支持传入 `allowEntry`，用于 subagent 按 `allowedTools` 收窄可搜索 deferred 工具。
- `einoV09PotentiallySearchable(...)` 只做 policy-level superset 过滤；真实当前 session 可见性仍由 `ToolSearchState` 和 dynamic surface 在运行期判断。
- `useEinoV09ModelToolSearch(...)` 当前固定 false，避免 Eino model-native deferred retrieval 绕过 Starxo discovery 记录。

## 4. 维护边界
- Eino API 变动应集中改这里，不要把 ToolSearch middleware 细节散回 `ChatService` 或 `Agent` tool。
