# runtime_compact.go 技术说明

## 1. 文件定位
- 源文件: `internal/model/runtime_compact.go`
- 文档文件: `doc/src/internal/model/runtime_compact.go.plan.md`
- 所属模块: model

## 2. 核心职责
- 定义 Runtime V2 长会话 compact 的持久化 schema。
- `RuntimeContextCompact` 保存 prompt 压缩后仍必须保留的运行时上下文。

## 3. 关键结构
- `RuntimeContextCompact`:
  - 消息统计: `OriginalMessageCount`、`OmittedMessageCount`、`TokenEstimate`
  - `Summary`: deterministic compact summary
  - `ToolSearch`: discovered tools / deferred announcement / MCP instructions delta state
  - `PermissionGrants`
  - `Tasks`
  - `FileReadState`
  - `DiffSummaries`
  - `Todos`
  - `PlanDocument`
  - `Workspace`
  - `ActiveObjective`: 当前 user turn 的 objective sidecar，保存 runID、scope、acceptance 和 standalone 历史边界
- `CloneRuntimeContextCompact(...)` 深拷贝所有 slice / nested state。

## 4. 维护边界
- model 层不依赖 tools/service，`RuntimeTodoItem` 和 task/workspace compact 类型保持纯数据结构。
- 新增 compact 字段时必须同步 `CloneRuntimeContextCompact` 和 `NormalizeSessionData`。
- `ActiveObjective` 是行为层隔离边界；compact 后必须保留，避免长会话恢复时把旧任务当成当前目标。
