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
- `DiscoveredToolRecord` 固定字段：
  - `CanonicalName`
  - `Server`
  - `Kind`
  - `DiscoveredAt`
- 旧 payload 缺失 `DiscoveredTools` 时按空集合兼容

## 5. 依赖关系
- 被 `chat.go`、`session_svc.go`、`tool_search.go` 共同消费

## 6. 变更影响面
- 成为 deferred discovery 的唯一持久化状态源

## 7. 维护建议
- discovery 状态不要迁回 `PersistedMessage`；`SessionData` 是唯一权威落盘位置
