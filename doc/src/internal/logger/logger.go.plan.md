# logger.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/logger/logger.go
- 文档文件: doc/src/internal/logger/logger.go.plan.md
- 文件类型: Go 源码
- 所属模块: logger

## 2. 核心职责
- 该文件实现了 starxo 应用的全局日志系统，基于 Go 标准库 `slog` 构建。日志同时输出到 stderr（Wails 开发控制台）和按日轮转的文件（`<projectRoot>/logs/agent-YYYY-MM-DD.log`）。提供了一组领域特定的便捷日志函数，涵盖 Agent 生命周期事件、Agent 间转移、工具调用/结果/错误、LLM 模型调用/结果、Token 使用量、会话事件和 Runner 事件等，形成结构化的可观测性日志体系。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `Init` 接收项目根目录路径；各日志函数接收事件名称、Agent 名称、工具名称等上下文信息
- 输出结果: 日志输出到 stderr 和日志文件；`L()` 返回全局 `*slog.Logger` 实例

## 4. 关键实现细节
- 结构体/接口定义: 无导出结构体（使用包级全局变量）
- 导出函数/方法:
  - `Init(projectRoot string) error` — 初始化日志系统（创建日志目录和文件）
  - `Close()` — 刷新并关闭日志文件
  - `L() *slog.Logger` — 获取全局日志实例
  - `AgentEvent(event, agent string, attrs ...any)` — 记录 Agent 生命周期事件
  - `Transfer(from, to string, attrs ...any)` — 记录 Agent 间转移
  - `ToolCall(agent, tool, args string)` — 记录工具调用（参数截断到 500 字符）
  - `ToolResult(agent, tool, result string, duration time.Duration)` — 记录工具结果（结果截断到 800 字符）
  - `ToolError(agent, tool string, err error, duration time.Duration)` — 记录工具错误
  - `ModelCall(agent string, messageCount int, attrs ...any)` — 记录 LLM 调用
  - `ModelResult(agent string, hasToolCalls bool, contentLen int, attrs ...any)` — 记录 LLM 响应
  - `TokenUsage(agent string, promptTokens, completionTokens, totalTokens int64)` — 记录 Token 使用量
  - `SessionEvent(event, sessionID string, attrs ...any)` — 记录会话事件
  - `RunnerEvent(event string, attrs ...any)` — 记录 Runner 事件
  - `Error(msg string, err error, attrs ...any)` — 记录通用错误
  - `Debug(msg string, attrs ...any)` — 记录调试信息
  - `Info(msg string, attrs ...any)` — 记录信息日志
  - `Warn(msg string, attrs ...any)` — 记录警告日志
- 未导出函数:
  - `truncate(s string, maxLen int) string` — 截断过长字符串
- Wails 绑定方法: 无
- 事件发射: 无

## 5. 依赖关系
- 内部依赖: 无
- 外部依赖:
  - `fmt`, `io`, `log/slog`, `os`, `path/filepath`, `sync`, `time` (Go 标准库)
- 关键配置:
  - 日志级别: Debug
  - 日志文件模式: `agent-YYYY-MM-DD.log`（按日轮转）
  - 日志目录: `<projectRoot>/logs/`
  - 工具调用参数截断: 500 字符
  - 工具结果截断: 800 字符

## 6. 变更影响面
- `Init` 函数被 `app.go` 的 `startup` 方法调用，变更签名会影响应用启动流程
- 日志格式变更会影响日志解析工具和运维监控
- 领域日志函数被 `callbacks.go` 中的全局回调和各服务组件调用
- 日志文件路径变更会影响日志收集和排查流程

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 当前日志轮转仅按日期区分文件名，不包含文件大小限制或历史文件清理，长期运行需配合外部日志管理。
- `Init` 可被多次调用（会关闭旧文件句柄），但全局变量的使用限制了测试的隔离性。
- 领域日志函数使用 `[TAG]` 前缀格式（如 `[AGENT]`, `[TOOL_CALL]`），便于日志检索，新增领域日志时应遵循此约定。
- `truncate` 函数使用字节长度而非字符长度，对多字节 UTF-8 字符可能在中间截断。
