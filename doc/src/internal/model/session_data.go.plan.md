# session_data.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/model/session_data.go
- 文档文件: doc/src/internal/model/session_data.go.plan.md
- 文件类型: Go 源码
- 所属模块: model

## 2. 核心职责
- 定义统一的会话持久化数据模型 `SessionData`，将 LLM 对话历史（messages）和前端显示数据（display timeline turns）合并为一个原子写入的 JSON 文件。消除了之前 `messages.json`（后端生产）和 `display.json`（前端生产）由不同组件在不同时机保存导致的数据不一致问题。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: 由 `SessionStore.SaveSessionData()` 序列化，由 `SessionStore.LoadSessionData()` 反序列化
- 输出结果: JSON 文件 `~/.starxo/sessions/{id}/session_data.json`

## 4. 关键实现细节
- 结构体/接口定义:
  - `SessionData` — 统一存储格式，包含 `Version`（格式版本号）、`Messages`（LLM 对话历史）、`Display`（前端时间线数据）、`Streaming`（流式中途状态，可选）
  - `DisplayTurn` — 一个对话轮次（用户或助手），包含 `ID`、`Role`、`Content`、`Agent`、`Timestamp`、`Events`（子事件列表）
  - `DisplayEvent` — 轮次内的单个时间线事件，包含 `ID`、`Type`（message/tool_call/tool_result/transfer/info/interrupt）、`Agent`、`Content`、`ToolName`、`ToolArgs`、`ToolID`、`ToolResult`、`Timestamp`、`IsStreaming`
  - `StreamingState` — 流式中途状态快照，包含 `PartialContent`（部分内容）和 `AgentName`（当前代理名）
- 导出函数/方法: 无（纯数据模型）
- Wails 绑定方法: 通过 `SessionService.LoadSessionData()` 返回给前端
- 事件发射: 无

## 5. 依赖关系
- 内部依赖: 同包 `PersistedMessage` 类型
- 外部依赖: 无
- 关键配置: 无

## 6. 变更影响面
- 修改 `SessionData` 结构会影响 `session_data.json` 的格式，需考虑向后兼容（通过 `Version` 字段）
- `DisplayTurn` 和 `DisplayEvent` 结构需与前端 `chatStore` 的消息格式和 Wails 绑定类型保持一致
- 新增字段需同步更新 `frontend/wailsjs/go/models.ts` 中的 TypeScript 模型类

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 修改 `SessionData` 结构时递增 `Version` 字段值，并在 `LoadSessionData` 中添加迁移逻辑。
- `DisplayEvent` 的字段设计镜像前端 `TurnEvent` 类型，两者应保持同步。
