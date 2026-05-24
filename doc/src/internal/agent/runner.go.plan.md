# runner.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/agent/runner.go
- 文档文件: doc/src/internal/agent/runner.go.plan.md
- 文件类型: Go 源码
- 所属模块: agent

## 2. 核心职责
- 该文件负责构建两种模式的 ADK Runner：默认模式（default）和计划模式（plan）。两种模式现在都运行同一个 ChatModelAgent/ReAct 行为层；计划模式只是权限和工具面的收窄，不再默认构建 Eino PlanExecute planner/replanner 图。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源:
  - `BuildDefaultRunner`: `context.Context`、`adk.Agent`（runtime agent）、`compose.CheckPointStore`（检查点存储）
  - `BuildPlanRunner`: 保留 `model.ToolCallingChatModel` 和 `AgentContext` 参数用于兼容旧调用，实际直接包装传入的 runtime agent
- 输出结果:
  - `BuildDefaultRunner`: 返回 `*adk.Runner`
  - `BuildPlanRunner`: 返回 `*adk.Runner` 和 error

## 4. 关键实现细节
- 结构体/接口定义: 无自定义结构体
- 导出函数/方法:
  - `BuildDefaultRunner(ctx, runtimeAgent, checkpointStore) *adk.Runner`: 创建默认模式 runner，启用流式输出，如果 checkpointStore 为 nil 则使用内存存储
  - `BuildPlanRunner(ctx, mdl, runtimeAgent, ac, checkpointStore) (*adk.Runner, error)`: 创建计划模式 runner，直接包装 runtime agent；plan 行为由 prompt、permission queue 和 `ExitPlanMode` 控制
- Wails 绑定方法: 无
- 事件发射: 无

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/store`: 提供 `NewInMemoryStore()` 作为默认 checkpoint 存储
  - 同包 `agent`: `AgentContext`
- 外部依赖:
  - `github.com/cloudwego/eino/adk`: Runner 和 Agent 接口
  - `github.com/cloudwego/eino/components/model`: LLM 模型接口
  - `github.com/cloudwego/eino/compose`: CheckPointStore 接口
- 关键配置: runner `EnableStreaming: true`

## 6. 变更影响面
- 修改 runner 配置（如关闭流式输出）会影响前端消息接收方式
- 修改 plan runner 的 agent 会影响 plan mode 下的权限面、`ExitPlanMode` 和中断恢复行为
- checkpoint 存储的变更影响会话恢复和中断恢复功能
- 直接影响 `internal/service/chat.go` 中的 `BuildRunners` 方法

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 如需添加新的运行模式，应在此文件中新增对应的 `Build*Runner` 函数，并在 `chat.go` 中增加模式分支。
- plan mode 现在和 default mode 共享 runtime loop；不要重新引入 planner/replanner 图，除非同时重写 checkpoint、permission 和 current-objective 行为。
- checkpoint 存储目前使用内存实现，如需持久化跨重启的会话状态，需替换为文件/数据库实现。
