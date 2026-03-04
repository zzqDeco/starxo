# timeline.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/context/timeline.go
- 文档文件: doc/src/internal/context/timeline.go.plan.md
- 文件类型: Go 源码
- 所属模块: agentctx

## 2. 核心职责
- 实现 `TimelineCollector`，在后端内存中累积 display turns（对话轮次和时间线事件），使后端成为唯一的持久化生产者。它镜像了前端 `chatStore` 的事件收集逻辑，但位于服务端，可以在 shutdown、crash recovery、session switch 时可靠保存。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `ChatService.emitTimeline()` 在每次发射 `agent:timeline` 事件时同步写入；`ChatService.SendMessage()` 添加用户轮次
- 输出结果: `Export()` 返回 `[]model.DisplayTurn` 快照，由 `SessionService.saveCurrentLocked()` 写入 `session_data.json`

## 4. 关键实现细节
- 结构体/接口定义:
  - `TimelineCollector` — 线程安全的 display turns 收集器，持有读写锁 `mu` 和 `turns` 切片
- 导出函数/方法:
  - `NewTimelineCollector() *TimelineCollector` — 创建空收集器
  - `(tc) StartTurn(id, role, agent, timestamp)` — 开始新轮次
  - `(tc) AddUserTurn(id, content, timestamp)` — 添加完整用户轮次
  - `(tc) AddEvent(evt, agent)` — 添加事件到当前助手轮次，自动处理：
    - `stream_chunk` — 累积到现有 streaming message event
    - `stream_end` — 终结 streaming message，设置轮次 content
    - `tool_result` — 附加到匹配的 tool_call event
    - 其他类型 — 直接追加
  - `(tc) SetTurnContent(content)` — 设置当前轮次最终内容
  - `(tc) Export() []DisplayTurn` — 导出快照（深拷贝）
  - `(tc) Import(turns)` — 导入数据（会话恢复时使用）
  - `(tc) Clear()` — 重置收集器
  - `(tc) Len() int` — 返回轮次数量
- Wails 绑定方法: 无（通过 ChatService 间接使用）
- 事件发射: 无

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/model` (DisplayTurn, DisplayEvent 类型)
- 外部依赖:
  - `sync` (读写锁)
- 关键配置: 无

## 6. 变更影响面
- `AddEvent` 的事件分类逻辑必须与前端 `chatStore.addTimelineEvent()` 保持一致
- `Export`/`Import` 的数据格式直接影响 `session_data.json` 的 `display` 字段
- 被 `ChatService` 持有并在 `processEvents`、`drainStream`、`SendMessage` 中调用
- 被 `SessionService` 的 `saveCurrentLocked`、`SwitchSession`、`EnsureDefaultSession` 间接使用

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- `AddEvent` 中的 stream_chunk/stream_end/tool_result 特殊处理逻辑与前端 `chatStore.addTimelineEvent()` 镜像，修改任一方时需同步检查另一方。
- `Export` 使用浅拷贝（`copy`），对于 `Events` 切片内的元素是引用共享的。如果需要在 export 后继续修改 turns，考虑深拷贝。
- 线程安全通过 `sync.RWMutex` 保证，所有公开方法都正确获取锁。
