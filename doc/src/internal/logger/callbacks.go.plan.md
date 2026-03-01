# callbacks.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/logger/callbacks.go
- 文档文件: doc/src/internal/logger/callbacks.go.plan.md
- 文件类型: Go 源码
- 所属模块: logger

## 2. 核心职责
- 该文件实现了 Eino 框架的全局回调处理器注册，通过 `RegisterGlobalCallbacks` 函数将日志记录挂钩到框架的模型调用和工具调用生命周期中。对于每次 LLM 模型调用，记录输入消息数量、输出内容长度、工具调用详情、Token 使用量和调用耗时；对于每次工具调用，记录调用参数、执行结果和耗时。错误事件也被完整记录。通过 context 传递开始时间实现精确的耗时测量。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: Eino 框架回调系统提供的 `callbacks.RunInfo`、`model.CallbackInput/Output`、`tool.CallbackInput/Output` 和错误信息
- 输出结果: 通过 `logger.go` 中的领域日志函数将结构化日志输出到 stderr 和日志文件

## 4. 关键实现细节
- 结构体/接口定义:
  - `contextKey` (string 类型别名) — 用于在 context 中存储计时信息的键类型
- 导出函数/方法:
  - `RegisterGlobalCallbacks()` — 注册全局回调处理器（应用启动时调用一次）
- 未导出函数:
  - `extractAgentName(info *callbacks.RunInfo) string` — 从 RunInfo 中提取 Agent/组件名称
- 未导出常量:
  - `toolStartTimeKey` — 工具调用开始时间的 context key
  - `modelStartTimeKey` — 模型调用开始时间的 context key
- 回调注册:
  - `ChatModel` 回调:
    - `OnStart`: 记录模型调用开始（消息数量），在 context 中存储开始时间
    - `OnEnd`: 记录模型响应（工具调用详情、Token 使用量、内容长度、耗时）
    - `OnError`: 记录模型调用错误
  - `Tool` 回调:
    - `OnStart`: 记录工具调用（参数 JSON），在 context 中存储开始时间
    - `OnEnd`: 记录工具结果（响应内容、耗时）
    - `OnError`: 记录工具错误（错误信息、耗时）
- Wails 绑定方法: 无
- 事件发射: 无

## 5. 依赖关系
- 内部依赖:
  - 同包 `logger.go` 中的日志函数 (ModelCall, ModelResult, TokenUsage, ToolCall, ToolResult, ToolError, Error, Info, L, truncate)
- 外部依赖:
  - `context` (Go 标准库，传递计时信息)
  - `time` (Go 标准库，耗时测量)
  - `github.com/cloudwego/eino/callbacks` (Eino 回调框架，RunInfo, AppendGlobalHandlers)
  - `github.com/cloudwego/eino/components/model` (模型回调输入/输出类型)
  - `github.com/cloudwego/eino/components/tool` (工具回调输入/输出类型)
  - `github.com/cloudwego/eino/utils/callbacks` (HandlerHelper 模板工具，别名 `template`)
- 关键配置: 无

## 6. 变更影响面
- `RegisterGlobalCallbacks` 被 `app.go` 的 `startup` 方法调用，注册一次后影响所有后续的模型和工具调用
- 回调中的日志输出依赖 `logger.go` 中的领域日志函数，函数签名变更需同步更新
- 修改 context key 或计时逻辑会影响耗时测量的准确性
- 回调处理逻辑变更会影响所有 Agent 的可观测性数据

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 回调函数中应避免执行耗时操作或阻塞调用，以免影响 Agent 执行性能。
- `extractAgentName` 当前仅使用 `RunInfo.Name`，若 Eino 框架后续提供更丰富的上下文信息（如组件类型、调用链路），可扩展此函数。
- 工具调用参数在 `OnStart` 回调中通过 `ToolCall` 函数截断到 500 字符，模型响应中的工具调用参数截断到 300 字符，截断阈值可考虑统一或可配置化。
- `OnEnd` 回调中 `output` 可能为 nil（如超时或异常），已做 nil 检查，新增字段访问时需保持此防御性编程习惯。
