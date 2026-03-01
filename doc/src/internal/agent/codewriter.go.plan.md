# codewriter.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/agent/codewriter.go
- 文档文件: doc/src/internal/agent/codewriter.go.plan.md
- 文件类型: Go 源码
- 所属模块: agent

## 2. 核心职责
- 该文件负责创建代码编写子代理（code_writer），这是深度代理的主要工作子代理，负责所有代码相关任务：读取文件、列出目录、创建新文件、编辑现有代码和重构。它使用 Eino 的 `commandline.NewStrReplaceEditor` 工具进行精确的字符串替换编辑，同时配备 read_file 和 list_files 工具使其具备自主文件检查能力。所有工具均通过 `WrapToolsWithEvents` 包装以支持前端时间线事件。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `context.Context`、`model.ToolCallingChatModel`（LLM 模型）、`commandline.Operator`（沙箱操作器）、`AgentContext`（运行时环境上下文）
- 输出结果: 返回 `adk.Agent` 接口实例（code_writer 子代理），出错时返回 error

## 4. 关键实现细节
- 结构体/接口定义: 无自定义结构体（使用 `filemanager.go` 中定义的 `ReadFileInput`/`ReadFileOutput`、`ListFilesInput`/`ListFilesOutput`）
- 导出函数/方法:
  - `NewCodeWriterAgent(ctx, mdl, op, ac) (adk.Agent, error)`: 创建代码编写子代理，内部构建 str_replace_editor、read_file、list_files 工具以及 FollowUp、Choice、NotifyUser 中断工具，最大迭代次数为 30
- Wails 绑定方法: 无
- 事件发射: 通过 `WrapToolsWithEvents("code_writer", ...)` 间接发射 `tool_call` 和 `tool_result` 事件

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/tools` (agenttools): FollowUpTool、ChoiceTool、NotifyUserTool
  - 同包 `agent`: `CodeWriterPrompt`、`AgentContext`、`WrapToolsWithEvents`、`ReadFileInput`/`ReadFileOutput`、`ListFilesInput`/`ListFilesOutput`
- 外部依赖:
  - `github.com/cloudwego/eino-ext/components/tool/commandline`: StrReplaceEditor、Operator
  - `github.com/cloudwego/eino/adk`: ChatModelAgent
  - `github.com/cloudwego/eino/components/model`: LLM 模型接口
  - `github.com/cloudwego/eino/components/tool`: 工具接口
  - `github.com/cloudwego/eino/components/tool/utils` (toolutils): `InferTool` 工具推断创建
  - `github.com/cloudwego/eino/compose`: 工具节点配置
- 关键配置: `MaxIterations: 30`

## 6. 变更影响面
- 修改工具列表影响代码编写代理的能力范围
- read_file 和 list_files 的实现变更影响代理的文件检查能力
- `MaxIterations` 变更影响代理处理复杂编辑任务的能力上限
- 影响 `deep_agent.go` 中的子代理组装
- list_files 使用 `find` 命令，默认深度 3 级，路径为空时使用 `ac.WorkspacePath`

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 新增工具时需同时更新 `prompts.go` 中 `CodeWriterPrompt` 的工具描述。
- `ReadFileInput`/`ReadFileOutput` 等类型定义在 `filemanager.go` 中，修改时需注意跨文件影响。
- 工具事件包装确保前端可见性，新增工具应始终通过 `WrapToolsWithEvents` 包装。
