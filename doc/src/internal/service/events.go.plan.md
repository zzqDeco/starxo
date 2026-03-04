# events.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/service/events.go
- 文档文件: doc/src/internal/service/events.go.plan.md
- 文件类型: Go 源码
- 所属模块: service

## 2. 核心职责
- 该文件定义了所有通过 Wails 事件系统从后端发送到前端的数据传输对象（DTO）和事件结构体。这些类型构成了后端与前端之间的事件契约。
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
  - `SandboxProgressEvent`: 连接/容器进度事件（Step、Percent），用于 `ssh:progress` 和 `container:progress` 事件
  - `FileInfoDTO`: 文件信息 DTO（Name、Path、Size、IsOutput）
  - `SandboxStatusDTO`: 沙箱状态 DTO（SSHConnected、DockerRunning、ContainerID、**ActiveContainerID**、**ActiveContainerName**、**DockerAvailable**）— 扩展了活跃容器信息和 Docker 可用性
  - `SessionSwitchedEvent`: 会话切换事件（Session、ContainerID）
  - `TimelineEvent`: 统一时间线事件（Type 字段支持 "reasoning" 和 "thinking" 事件类型）
  - `InterruptEvent`: 中断事件（followup/choice）
  - `InterruptOption`: 中断选项（Label、Description）
  - `PlanEvent`: 计划事件（Steps []PlanStepDTO）
  - `PlanStepDTO`: 计划步骤 DTO
  - `ModeChangedEvent`: 模式切换事件（Mode）

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/model`: Session 类型
- 外部依赖: 无

## 6. 变更影响面
- `SandboxStatusDTO` 扩展的三个字段（ActiveContainerID、ActiveContainerName、DockerAvailable）被前端 `GetStatus()` 调用消费
- `SandboxProgressEvent` 同时用于 `ssh:progress` 和 `container:progress` 两类事件
- 所有结构体的 JSON 标签是前后端通信的契约，修改会影响前端

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 修改 JSON 标签前必须同步检查前端 TypeScript 类型定义。
- 新增事件类型时应在此文件集中定义。
