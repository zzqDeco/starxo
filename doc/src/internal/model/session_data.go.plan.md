# session_data.go 技术说明

## 1. 文件定位
- 源文件: `internal/model/session_data.go`
- 文档文件: `doc/src/internal/model/session_data.go.plan.md`
- 所属模块: model

## 2. 核心职责
- 定义统一持久化的 `SessionData` 结构。

## 3. 输入与输出
- 输入来源: `ChatService.ExportSessionSnapshot(...)`
- 输出结果: `session_data.json`

## 4. 关键实现细节
- `SessionData` 新增 `DiscoveredTools []DiscoveredToolRecord`
- `SessionData` 新增 `DeferredAnnouncementState *DeferredAnnouncementState`
- `SessionData` 新增 `MCPInstructionsDeltaState *MCPInstructionsDeltaState`
- `DiscoveredToolRecord` 固定字段：
  - `CanonicalName`
  - `Server`
  - `Kind`
  - `DiscoveredAt`
- `DeferredAnnouncementState` 当前固定只记录：
  - `AnnouncedSearchableCanonicalNames`
- phase-2 空 state 规范：
  - `DeferredAnnouncementState.AnnouncedSearchableCanonicalNames` 使用稳定排序后的空切片，不混用 `nil`
  - `MCPInstructionsDeltaState` 的三组 server 集合也统一使用空切片
  - `MCPInstructionsDeltaState.LastInstructionsFingerprint` 固定为规范化空 summary 的确定性 fingerprint
- 旧 payload 缺失这些新增字段时按空状态兼容，不中断 restore
- `DeferredSurfaceDebug` 不进入 `SessionData` 持久化；它只存在于 `SessionSnapshot` / debug API 的 best-effort runtime 视图中

## 5. 依赖关系
- 被 `chat.go`、`session_svc.go`、`tool_search.go` 共同消费

## 6. 变更影响面
- 成为 deferred discovery 的唯一持久化状态源

## 7. 维护建议
- discovery 状态不要迁回 `PersistedMessage`；`SessionData` 是唯一权威落盘位置
- deferred delta state 与 discovery state 语义不同，不要合并成同一个字段
