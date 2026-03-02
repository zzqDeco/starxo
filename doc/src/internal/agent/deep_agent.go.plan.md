# deep_agent.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/agent/deep_agent.go
- 文档文件: doc/src/internal/agent/deep_agent.go.plan.md
- 文件类型: Go 源码
- 所属模块: agent

## 2. 核心职责
- 该文件负责构建核心深度代理（deep agent），这是整个 AI 编码代理的中枢。它组装三个专用子代理（code_writer、code_executor、file_manager）和一组直接工具（FollowUp、Choice、WriteTodos、UpdateTodo、NotifyUser 及额外 MCP 工具），使用 CloudWeGo Eino ADK 的 `deep.New()` 创建一个具备任务委派能力的自主代理。该代理在默认模式下作为 runner 的直接代理使用，在 plan 模式下作为 planexecute 的执行器使用。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `context.Context`、`model.ToolCallingChatModel`（LLM 模型）、`commandline.Operator`（沙箱操作器）、`[]tool.BaseTool`（额外工具如 MCP 工具）、`AgentContext`（运行时环境上下文）
- 输出结果: 返回 `adk.Agent` 接口实例（deep agent），可被 runner 或 planexecute 直接使用；出错时返回 error

## 4. 关键实现细节
- 结构体/接口定义: 无自定义结构体，依赖 `adk.Agent` 接口
- 导出函数/方法:
  - `BuildDeepAgent(ctx, mdl, op, extraTools, ac) (adk.Agent, error)`: 构建核心深度代理，内部创建三个子代理并组装直接工具列表，配置最大迭代次数为 50
- Wails 绑定方法: 无（由 service 层间接调用）
- 事件发射: 无直接事件发射，子代理工具通过 `AgentContext.OnToolEvent` 回调间接发射事件

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/tools` (agenttools): 提供 FollowUpTool、ChoiceTool、WriteTodosTool、UpdateTodoTool、NotifyUserTool
  - 同包 `agent`: `NewCodeWriterAgent`、`NewCodeExecutorAgent`、`NewFileManagerAgent`、`DeepAgentPrompt`、`AgentContext`
- 外部依赖:
  - `github.com/cloudwego/eino-ext/components/tool/commandline`: 沙箱命令行操作器
  - `github.com/cloudwego/eino/adk`: ADK 代理框架
  - `github.com/cloudwego/eino/adk/prebuilt/deep`: 深度代理预构建模块
  - `github.com/cloudwego/eino/components/model`: LLM 模型接口
  - `github.com/cloudwego/eino/components/tool`: 工具接口
  - `github.com/cloudwego/eino/compose`: 组合配置
- 关键配置:
  - `MaxIteration: 50`（深度代理最大迭代次数）
  - `WithoutWriteTodos: true`（禁用 Eino 框架内置的 write_todos 工具，使用 starxo 自定义实现，含 DAG 验证和前端渲染）

## 6. 变更影响面
- 修改子代理列表会影响代理的任务委派能力
- 修改直接工具列表会影响代理与用户的交互方式（中断、进度跟踪等）
- `MaxIteration` 变更影响代理执行复杂任务的能力上限
- 影响 `runner.go` 中 `BuildDefaultRunner` 和 `BuildPlanRunner` 的行为
- 影响 `internal/service/chat.go` 中 `BuildRunners` 方法

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 新增子代理时需同时在 `prompts.go` 中添加对应的 system prompt，并在 `DeepAgentPrompt` 中更新子代理说明。
- 新增直接工具时需确保在 `internal/tools` 包中实现并遵循 Eino BaseTool 接口。
- 调整 `MaxIteration` 需考虑 LLM 调用成本与超时风险。
