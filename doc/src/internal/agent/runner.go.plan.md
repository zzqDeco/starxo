# runner.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/agent/runner.go
- 文档文件: doc/src/internal/agent/runner.go.plan.md
- 文件类型: Go 源码
- 所属模块: agent

## 2. 核心职责
- 该文件负责构建两种模式的 ADK Runner：默认模式（default）和计划模式（plan）。默认模式下，deep agent 直接处理所有任务；计划模式下，使用 Eino ADK 的 planexecute 模式，由 planner 生成计划、deep agent 执行步骤、replanner 动态调整计划。两种 runner 均支持流式输出和 checkpoint 持久化。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源:
  - `BuildDefaultRunner`: `context.Context`、`adk.Agent`（deep agent）、`compose.CheckPointStore`（检查点存储）
  - `BuildPlanRunner`: 额外接收 `model.ToolCallingChatModel`（用于 planner/replanner）、`AgentContext`
- 输出结果:
  - `BuildDefaultRunner`: 返回 `*adk.Runner`
  - `BuildPlanRunner`: 返回 `*adk.Runner` 和 error

## 4. 关键实现细节
- 结构体/接口定义: 无自定义结构体
- 导出函数/方法:
  - `BuildDefaultRunner(ctx, deepAgent, checkpointStore) *adk.Runner`: 创建默认模式 runner，启用流式输出，如果 checkpointStore 为 nil 则使用内存存储
  - `BuildPlanRunner(ctx, mdl, deepAgent, ac, checkpointStore) (*adk.Runner, error)`: 创建计划模式 runner，内部构建 planner 和 replanner，将 deep agent 作为执行器包装进 planexecute 代理，最大迭代次数为 20
- Wails 绑定方法: 无
- 事件发射: 无

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/store`: 提供 `NewInMemoryStore()` 作为默认 checkpoint 存储
  - 同包 `agent`: `AgentContext`
- 外部依赖:
  - `github.com/cloudwego/eino/adk`: Runner 和 Agent 接口
  - `github.com/cloudwego/eino/adk/prebuilt/planexecute`: planexecute 模式（Planner、Replanner、Config）
  - `github.com/cloudwego/eino/components/model`: LLM 模型接口
  - `github.com/cloudwego/eino/compose`: CheckPointStore 接口
- 关键配置: planexecute `MaxIterations: 20`

## 6. 变更影响面
- 修改 runner 配置（如关闭流式输出）会影响前端消息接收方式
- 修改 planexecute 参数会影响计划模式的行为和迭代上限
- checkpoint 存储的变更影响会话恢复和中断恢复功能
- 直接影响 `internal/service/chat.go` 中的 `BuildRunners` 方法

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 如需添加新的运行模式，应在此文件中新增对应的 `Build*Runner` 函数，并在 `chat.go` 中增加模式分支。
- planexecute 的 `MaxIterations` 应与 deep agent 的 `MaxIteration` 协调考虑，避免嵌套迭代导致过长运行时间。
- checkpoint 存储目前使用内存实现，如需持久化跨重启的会话状态，需替换为文件/数据库实现。
