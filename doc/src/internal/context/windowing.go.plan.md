# windowing.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/context/windowing.go
- 文档文件: doc/src/internal/context/windowing.go.plan.md
- 文件类型: Go 源码
- 所属模块: agentctx

## 2. 核心职责
- 该文件实现了对话消息的窗口化（windowing）策略，用于在消息数量超出 LLM 上下文窗口限制时进行智能裁剪。核心策略为：保留第一条消息（通常是系统提示词）和最近的 N 条消息，中间省略的部分插入摘要占位符。同时提供单条消息内容的智能截断功能，保留头部 60% 和尾部 20% 的内容，中间用截断标记替代。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `[]*schema.Message` 消息切片；`WindowConfig` 配置（最大消息数、最大单条内容长度）
- 输出结果: 经过窗口化和截断处理的 `[]*schema.Message` 新切片

## 4. 关键实现细节
- 结构体/接口定义:
  - `WindowConfig` — 窗口化配置，包含 `MaxMessages` (默认 20) 和 `MaxContentLen` (默认 4000)
- 导出函数/方法:
  - `DefaultWindowConfig() WindowConfig` — 返回默认窗口配置
  - `WindowMessages(messages []*schema.Message, cfg WindowConfig) []*schema.Message` — 对消息列表应用窗口化策略
  - `TruncateContent(content string, maxLen int) string` — 智能截断单条消息内容
- 未导出函数:
  - `truncateAll(messages []*schema.Message, maxContentLen int) []*schema.Message` — 批量截断所有消息
  - `truncateMsg(msg *schema.Message, maxContentLen int) *schema.Message` — 截断单条消息（无需截断时返回原指针）
- Wails 绑定方法: 无
- 事件发射: 无
- 窗口化算法:
  1. 消息总数 <= MaxMessages: 仅截断超长内容
  2. 消息总数 > MaxMessages: 保留第 1 条 + 最后 (MaxMessages-1) 条，中间插入省略占位符
- 内容截断算法: 保留前 60% + `...[truncated]...` 标记 + 后 20%

## 5. 依赖关系
- 内部依赖: 无
- 外部依赖:
  - `fmt` (格式化占位符文本)
  - `github.com/cloudwego/eino/schema` (Message 消息类型)
- 关键配置:
  - 默认最大消息数: 20
  - 默认最大单条内容长度: 4000 字符

## 6. 变更影响面
- 修改窗口化策略会直接影响 Agent 的上下文记忆能力和 Token 消耗
- 修改截断比例（60%/20%）会影响 Agent 对长消息的理解
- `WindowMessages` 被 `Engine.PrepareMessages()` 调用，是消息发送到 LLM 前的最后处理环节
- 占位符文本格式 `[Earlier conversation with N messages omitted for brevity]` 若变更需注意 Agent 是否能正确理解

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 当前窗口化策略基于消息数量而非实际 Token 数，后续可集成 tokenizer 实现更精确的预算控制。
- 截断标记 `...[truncated]...` 为硬编码字符串，Agent 可能无法完全理解其含义，可考虑在系统提示词中说明。
- `truncateMsg` 在无需截断时返回原始指针以避免不必要的内存分配，修改此优化时需注意副作用。
