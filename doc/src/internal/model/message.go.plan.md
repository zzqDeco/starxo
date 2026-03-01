# message.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/model/message.go
- 文档文件: doc/src/internal/model/message.go.plan.md
- 文件类型: Go 源码
- 所属模块: model

## 2. 核心职责
- 该文件定义了对话消息的持久化数据模型 `PersistedMessage`，用于将 Eino 框架的 `schema.Message` 转换为可序列化的精简格式存储到磁盘。该结构体故意与 Eino 内部类型解耦，采用简单的字符串字段表示角色、内容、名称和工具调用 ID，确保存储格式的稳定性和前向兼容性。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: 无外部输入（纯数据类型定义）
- 输出结果: 无（通过 JSON 序列化存储到文件系统）

## 4. 关键实现细节
- 结构体/接口定义:
  - `PersistedMessage` — 持久化消息结构体，包含以下字段:
    - `Role` (string) — 消息角色 (user/assistant/system/tool)
    - `Content` (string) — 消息内容
    - `Name` (string, omitempty) — 可选的发送者名称
    - `ToolCallID` (string, omitempty) — 可选的工具调用 ID（仅 tool 角色消息使用）
- 导出函数/方法: 无
- Wails 绑定方法: 无
- 事件发射: 无

## 5. 依赖关系
- 内部依赖: 无
- 外部依赖: 无
- 关键配置: 无

## 6. 变更影响面
- 修改字段名或 JSON tag 会破坏已有会话数据的反序列化兼容性
- 该结构体被以下组件使用:
  - `agentctx.Engine` — `ExportMessages`/`ImportMessages` 进行序列化转换
  - `storage.SessionStore` — `SaveMessages`/`LoadMessages` 进行磁盘读写
- 新增字段需同步更新 `Engine.ExportMessages` 和 `Engine.ImportMessages` 的转换逻辑

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 当前不包含 `ToolCalls` 字段（即 assistant 消息发起的工具调用列表），如需支持完整的工具调用链持久化需扩展此结构体。
- JSON tag 使用了驼峰命名 (`toolCallId`)，与前端 JavaScript 命名惯例一致，变更时需同步前端。
- 字段变更应考虑向后兼容，建议新增字段使用 `omitempty` 标签。
