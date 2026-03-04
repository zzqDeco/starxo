# engine.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/context/engine.go
- 文档文件: doc/src/internal/context/engine.go.plan.md
- 文件类型: Go 源码
- 所属模块: agentctx

## 2. 核心职责
- 该文件实现了 AI Agent 的上下文引擎 `Engine`，协调对话历史（`ConversationHistory`）和文件上下文（`FileContext`）两大上下文来源，为 LLM 调用准备完整的消息列表。核心功能包括：添加用户/助手/工具消息到历史、组装系统提示词（含文件上下文注入）、应用窗口化策略控制消息数量、导出/导入消息用于会话持久化，以及提供会话元数据供 ADK Runner 使用。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: 用户消息内容 (string)、助手消息内容 (string)、工具调用结果 (toolCallID + content)、系统提示词 (string)、Token 预算 (int)、持久化消息 (`[]model.PersistedMessage`)
- 输出结果: `PrepareMessages()` 返回经过窗口化处理的 `[]*schema.Message` 列表；`ExportMessages()` 返回可序列化的 `[]model.PersistedMessage`；`SessionValues()` 返回 `map[string]any` 元数据

## 4. 关键实现细节
- 结构体/接口定义:
  - `Engine` — 上下文引擎，持有读写锁 `mu`、`ConversationHistory`、`FileContext`、`maxTokens`、`systemPrompt`
- 导出函数/方法:
  - `NewEngine(systemPrompt string, maxTokens int) *Engine` — 创建上下文引擎
  - `(e *Engine) AddUserMessage(content string)` — 添加用户消息
  - `(e *Engine) AddAssistantMessage(content string)` — 添加助手消息
  - `(e *Engine) AddToolResult(toolCallID, content string)` — 添加工具结果消息
  - `(e *Engine) AddMessage(msg *schema.Message)` — 添加完整消息（含 ToolCalls），用于持久化带工具调用的 assistant 消息
  - `(e *Engine) PrepareMessages() []*schema.Message` — 构建完整消息列表
  - `(e *Engine) FileContext() *FileContext` — 获取文件上下文管理器
  - `(e *Engine) History() *ConversationHistory` — 获取对话历史管理器
  - `(e *Engine) ClearHistory()` — 清空对话历史
  - `(e *Engine) SessionValues() map[string]any` — 获取会话元数据
  - `(e *Engine) ExportMessages() []model.PersistedMessage` — 导出消息用于持久化
  - `(e *Engine) ImportMessages(messages []model.PersistedMessage)` — 从持久化数据恢复消息，包含孤儿 tool_call 自动修复
  - `(e *Engine) MessageCount() int` — 获取消息数量
- 未导出函数:
  - `repairOrphanToolCalls(msgs []*schema.Message) []*schema.Message` — 扫描消息列表，为没有对应 tool result 的孤儿 tool_call 注入合成错误响应（`"Error: tool execution was interrupted"`），防止 LLM API 因缺失 tool result 返回 400 错误
- Wails 绑定方法: 无（通过 ChatService 和 SessionService 间接使用）
- 事件发射: 无

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/model` (PersistedMessage 类型)
  - 同包 `ConversationHistory`, `FileContext`, `WindowConfig`, `DefaultWindowConfig`, `WindowMessages`
- 外部依赖:
  - `sync` (读写锁)
  - `github.com/cloudwego/eino/schema` (Message, RoleType 等 Eino 消息类型)
- 关键配置:
  - Token 预算到消息数量的换算: 约 200 tokens/消息

## 6. 变更影响面
- `PrepareMessages()` 的变更会直接影响 LLM 收到的消息内容和格式
- 修改消息窗口化策略会影响上下文长度和 Agent 的对话连贯性
- `ExportMessages`/`ImportMessages` 格式变更会影响会话持久化兼容性
- `ImportMessages` 中的 `repairOrphanToolCalls` 确保从持久化数据恢复时不会因孤儿 tool_call 导致 LLM API 400 错误
- 该引擎被 `ChatService` 和 `SessionService` 共同使用，变更需同步验证两方

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- Token 预算到消息数量的换算采用粗略估算（200 tokens/消息），后续可引入实际 tokenizer 提高精度。
- `ExportMessages`/`ImportMessages` 当前已支持 `ToolCalls` 字段的完整持久化和恢复，确保工具调用链在会话恢复时不丢失。
- `repairOrphanToolCalls` 与 `chat.go` 中 `processEvents` 的运行时孤儿修复逻辑互补：前者修复加载时的历史数据，后者修复运行时的实时事件。修改任一方时需同步检查另一方。
- 系统提示词的读取使用了读锁，但 `systemPrompt` 在 `NewEngine` 后未提供修改方法，锁可简化。
