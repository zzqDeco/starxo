# history.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/context/history.go
- 文档文件: doc/src/internal/context/history.go.plan.md
- 文件类型: Go 源码
- 所属模块: agentctx

## 2. 核心职责
- 该文件实现了线程安全的对话历史管理器 `ConversationHistory`，负责存储和管理 AI Agent 与用户之间的对话消息序列。提供消息的追加、全量获取、最近 N 条获取、清空和全量替换等操作，所有操作均通过读写锁保证并发安全。返回的消息列表均为副本，防止外部修改影响内部状态。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `*schema.Message` 消息对象；`SetAll` 接收消息切片用于会话恢复
- 输出结果: `GetAll()`/`GetRecent(n)` 返回消息切片的副本；`Len()` 返回消息数量

## 4. 关键实现细节
- 结构体/接口定义:
  - `ConversationHistory` — 对话历史管理器，持有读写锁 `mu` 和消息切片 `messages []*schema.Message`
- 导出函数/方法:
  - `NewConversationHistory() *ConversationHistory` — 创建空的对话历史
  - `(h *ConversationHistory) Add(msg *schema.Message)` — 追加消息
  - `(h *ConversationHistory) GetAll() []*schema.Message` — 获取全部消息副本
  - `(h *ConversationHistory) GetRecent(n int) []*schema.Message` — 获取最近 n 条消息副本
  - `(h *ConversationHistory) Len() int` — 获取消息数量
  - `(h *ConversationHistory) Clear()` — 清空所有消息
  - `(h *ConversationHistory) SetAll(msgs []*schema.Message)` — 全量替换消息（用于会话恢复）
- Wails 绑定方法: 无
- 事件发射: 无

## 5. 依赖关系
- 内部依赖: 无
- 外部依赖:
  - `sync` (读写锁)
  - `github.com/cloudwego/eino/schema` (Message 消息类型)
- 关键配置: 无

## 6. 变更影响面
- 该类被 `Engine` 直接使用，修改接口签名会影响 `engine.go`
- `GetAll` 和 `GetRecent` 返回副本的行为是线程安全的关键保证，不应改为返回引用
- `SetAll` 用于会话恢复（`Engine.ImportMessages`），修改其行为会影响会话加载

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 当前消息存储为无上限的切片，长时间运行的会话可能导致内存增长，建议配合窗口化策略（`windowing.go`）使用。
- `Add` 方法仅追加消息指针，不做深拷贝；若调用方后续修改了传入的 Message 对象，可能导致历史数据被意外修改。
