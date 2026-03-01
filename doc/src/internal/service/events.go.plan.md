# events.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/service/events.go
- 文档文件: doc/src/internal/service/events.go.plan.md
- 文件类型: Go 源码
- 所属模块: service

## 2. 核心职责
- 该文件定义了所有通过 Wails 事件系统从后端发送到前端的数据传输对象（DTO）和事件结构体。这些类型构成了后端与前端之间的事件契约，涵盖消息、流式输出、代理动作、工具结果、终端输出、沙箱进度、文件信息、沙箱状态、会话切换、时间线事件、中断事件、计划事件和模式切换等所有通信场景。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: 无直接输入，这些类型由 service 层的其他文件实例化
- 输出结果: 通过 `wailsruntime.EventsEmit` 序列化为 JSON 发送到前端

## 4. 关键实现细节
- 结构体/接口定义:
  - `MessageEvent`: 完整消息事件（ID、Agent、Content、Role、Timestamp）
  - `StreamChunkEvent`: 流式输出块事件（Agent、Content、Role）
  - `AgentActionEvent`: 代理动作事件（Type、AgentName、Details、ToolID）
  - `ToolResultEvent`: 工具结果事件（AgentName、ToolCallID、Content）
  - `TerminalOutputEvent`: 终端输出事件（Stdout、Stderr、ExitCode）
  - `SandboxProgressEvent`: 沙箱连接进度事件（Step、Percent）
  - `FileInfoDTO`: 文件信息 DTO（Name、Path、Size、IsOutput）
  - `SandboxStatusDTO`: 沙箱状态 DTO（SSHConnected、DockerRunning、ContainerID）
  - `SessionSwitchedEvent`: 会话切换事件（Session、ContainerID）
  - `TimelineEvent`: 统一时间线事件（ID、Type、Agent、Content、ToolName、ToolArgs、ToolID、Timestamp），Type 可为 "message"/"tool_call"/"tool_result"/"transfer"/"info"/"interrupt"/"plan"
  - `InterruptEvent`: 中断事件（Type、InterruptID、CheckpointID、Questions、Options、Question），Type 为 "followup" 或 "choice"
  - `InterruptOption`: 中断选项（Label、Description）
  - `PlanEvent`: 计划事件（Steps []PlanStepDTO）
  - `PlanStepDTO`: 计划步骤 DTO（TaskID、Status、Desc、ExecResult）
  - `ModeChangedEvent`: 模式切换事件（Mode）
- 导出函数/方法: 无（纯类型定义文件）
- Wails 绑定方法: 无
- 事件发射: 无（类型定义供其他文件使用）

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/model`: Session 类型（用于 SessionSwitchedEvent）
- 外部依赖: 无
- 关键配置: 无

## 6. 变更影响面
- 所有结构体的 JSON 标签是前后端通信的契约，修改 JSON 名称会导致前端解析失败
- `TimelineEvent` 是最核心的事件类型，被 `chat.go` 大量使用
- `InterruptEvent` 的字段变更影响前端中断 UI 的渲染
- `FileInfoDTO` 被 `file_svc.go` 使用
- `SandboxStatusDTO` 和 `SandboxProgressEvent` 被 `sandbox_svc.go` 使用
- `SessionSwitchedEvent` 被 `session_svc.go` 使用
- 新增事件类型时需确保前端有对应的事件监听器

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 修改任何 JSON 标签前必须同步检查前端 TypeScript 类型定义，保持前后端一致。
- `TimelineEvent.Type` 使用字符串枚举，建议在前端也定义对应的类型联合，避免拼写错误。
- 新增事件类型时应在此文件集中定义，避免在其他 service 文件中零散定义。
- 可考虑为频繁使用的事件名称（如 "agent:timeline"）定义常量，减少字符串拼写错误。
