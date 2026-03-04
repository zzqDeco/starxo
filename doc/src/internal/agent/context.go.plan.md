# context.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/agent/context.go
- 文档文件: doc/src/internal/agent/context.go.plan.md
- 文件类型: Go 源码
- 所属模块: agent

## 2. 核心职责
- 该文件定义了 `AgentContext` 结构体，用于向所有代理和工具注入运行时环境信息。它消除了代码中对 "/workspace" 等硬编码路径的依赖，使代理能够感知其所处的 Docker 容器、SSH 连接和工作区路径。同时提供 `OnToolEvent` 回调函数，使子代理的工具调用可以被前端时间线追踪。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: 由 `internal/service/chat.go` 的 `buildAgentContext()` 方法构建，数据来源于 `config.Store`（SSH 配置）、`sandbox.SandboxManager`（容器信息）和 `SessionService`（工作区路径）
- 输出结果: `AgentContext` 实例被传递给所有代理构建函数和提示词生成函数

## 4. 关键实现细节
- 结构体/接口定义:
  - `AgentContext`: 运行时环境上下文结构体
    - `WorkspacePath string`: 容器内工作区路径，如 "/workspace"
    - `ContainerName string`: Docker 容器名称
    - `ContainerID string`: Docker 容器短 ID
    - `SSHHost string`: SSH 主机地址
    - `SSHPort int`: SSH 端口
    - `SSHUser string`: SSH 用户名
    - `OnToolEvent func(ctx context.Context, agentName, eventType, toolName, toolArgs, toolID, result string)`: 工具事件回调。**签名变更**: 新增 `ctx context.Context` 作为首参数，ctx 中携带 session 身份信息（使用 `SessionIDFromContext` 提取）。参数依次为: ctx, agentName, eventType ("tool_call"/"tool_result"), toolName, toolArgs, toolID, result
- 导出函数/方法:
  - `DefaultAgentContext() AgentContext`: 返回默认上下文（WorkspacePath="/workspace"、SSHPort=22、SSHUser="root"、ContainerName="unknown"），用于无会话绑定时的回退
- Wails 绑定方法: 无
- 事件发射: 无直接发射，`OnToolEvent` 回调由 `tool_wrapper.go` 中的 `eventEmittingTool` 调用
- 依赖: 导入 `context` 标准库包

## 5. 依赖关系
- 内部依赖: 无
- 外部依赖: `context`（标准库）
- 关键配置: 无

## 6. 变更影响面
- `AgentContext` 被整个 agent 包的所有文件引用：`deep_agent.go`、`prompts.go`、`codewriter.go`、`codeexecutor.go`、`filemanager.go`、`tool_wrapper.go`
- 新增字段需在 `internal/service/chat.go` 的 `buildAgentContext()` 中赋值
- `OnToolEvent` 回调签名变更（新增 `ctx context.Context`）影响 `tool_wrapper.go`（调用方）和 `chat.go`（回调注册方）
- `DefaultAgentContext()` 的默认值变更影响无绑定会话时的代理行为
- `ctx` 参数通过 `SessionIDFromContext(ctx)` 提取 sessionID，使事件路由到正确的会话

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 新增环境信息字段时需同时更新 `DefaultAgentContext()` 的默认值和 `chat.go` 中的构建逻辑。
- `OnToolEvent` 回调通过 `context.Context` 传播 session 身份，使调用方无需额外维护 sessionID 参数。修改回调签名时需同步更新 `tool_wrapper.go` 和 `chat.go`。
- 该结构体是代理层与服务层的核心桥梁，保持其稳定性对整体架构至关重要。
