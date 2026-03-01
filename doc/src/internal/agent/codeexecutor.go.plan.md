# codeexecutor.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/agent/codeexecutor.go
- 文档文件: doc/src/internal/agent/codeexecutor.go.plan.md
- 文件类型: Go 源码
- 所属模块: agent

## 2. 核心职责
- 该文件负责创建代码执行子代理（code_executor），专门用于在沙箱容器中执行 Python 脚本和 Shell 命令。它使用 Eino 的 `commandline.NewPyExecutor` 执行 Python 代码，并通过自定义的 `shell_execute` 工具执行任意 Shell 命令。同时配备 read_file 工具以便在执行前检查脚本内容。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `context.Context`、`model.ToolCallingChatModel`（LLM 模型）、`commandline.Operator`（沙箱操作器）、`AgentContext`（运行时环境上下文）
- 输出结果: 返回 `adk.Agent` 接口实例（code_executor 子代理），出错时返回 error

## 4. 关键实现细节
- 结构体/接口定义:
  - `ShellInput`: Shell 命令执行输入，包含 `Command` 字段
  - `ShellOutput`: Shell 命令执行输出，包含 `Stdout`、`Stderr`、`ExitCode` 字段
- 导出函数/方法:
  - `NewCodeExecutorAgent(ctx, mdl, op, ac) (adk.Agent, error)`: 创建代码执行子代理，内部构建 python_execute（使用 python3）、shell_execute（通过 `sh -c` 执行）、read_file 工具以及中断工具，最大迭代次数为 30
- Wails 绑定方法: 无
- 事件发射: 通过 `WrapToolsWithEvents("code_executor", ...)` 间接发射事件

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/tools` (agenttools): FollowUpTool、ChoiceTool、NotifyUserTool
  - 同包 `agent`: `CodeExecutorPrompt`、`AgentContext`、`WrapToolsWithEvents`、`ReadFileInput`/`ReadFileOutput`
- 外部依赖:
  - `github.com/cloudwego/eino-ext/components/tool/commandline`: PyExecutor、Operator
  - `github.com/cloudwego/eino/adk`: ChatModelAgent
  - `github.com/cloudwego/eino/components/model`: LLM 模型接口
  - `github.com/cloudwego/eino/components/tool`: 工具接口
  - `github.com/cloudwego/eino/components/tool/utils` (toolutils): `InferTool`
  - `github.com/cloudwego/eino/compose`: 工具节点配置
- 关键配置: Python 执行器使用 `python3` 命令，`MaxIterations: 30`

## 6. 变更影响面
- shell_execute 的实现变更直接影响沙箱中命令执行的安全性和行为
- Python 执行器配置变更影响 Python 代码执行环境
- 影响 `deep_agent.go` 中的子代理组装
- `ShellInput`/`ShellOutput` 类型被其他文件引用时需保持稳定

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- shell_execute 通过 `sh -c` 执行命令，应注意命令注入风险（当前依赖沙箱隔离保证安全）。
- 新增工具时需同步更新 `prompts.go` 中 `CodeExecutorPrompt` 的工具描述。
- 如需支持其他编程语言执行器，应参考 Python 执行器模式在此文件中新增。
