# filemanager.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/agent/filemanager.go
- 文档文件: doc/src/internal/agent/filemanager.go.plan.md
- 文件类型: Go 源码
- 所属模块: agent

## 2. 核心职责
- 该文件负责创建文件管理子代理（file_manager），处理批量非代码文件操作、工作区探索和配置/文本文件写入。它提供 list_files、read_file、write_file 三个核心工具。此外，该文件还定义了多个在其他子代理中复用的工具输入/输出类型（ListFilesInput/Output、ReadFileInput/Output、WriteFileInput/Output）。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `context.Context`、`model.ToolCallingChatModel`（LLM 模型）、`commandline.Operator`（沙箱操作器）、`AgentContext`（运行时环境上下文）
- 输出结果: 返回 `adk.Agent` 接口实例（file_manager 子代理），出错时返回 error

## 4. 关键实现细节
- 结构体/接口定义:
  - `ListFilesInput`: 文件列表输入，包含 `Path` 字段（目录路径）
  - `ListFilesOutput`: 文件列表输出，包含 `Files` 字段（文件路径列表）
  - `ReadFileInput`: 文件读取输入，包含 `Path` 字段（绝对路径）
  - `ReadFileOutput`: 文件读取输出，包含 `Content` 字段（文件内容）
  - `WriteFileInput`: 文件写入输入，包含 `Path`（绝对路径）和 `Content`（写入内容）字段
  - `WriteFileOutput`: 文件写入输出，包含 `Success` 字段（是否成功）
- 导出函数/方法:
  - `NewFileManagerAgent(ctx, mdl, op, ac) (adk.Agent, error)`: 创建文件管理子代理，构建 list_files（使用 find 命令，最深 3 级）、read_file（通过 Operator.ReadFile）、write_file（通过 Operator.WriteFile）工具，最大迭代次数为 30
- Wails 绑定方法: 无
- 事件发射: 通过 `WrapToolsWithEvents("file_manager", ...)` 间接发射事件

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/tools` (agenttools): FollowUpTool、ChoiceTool、NotifyUserTool
  - 同包 `agent`: `FileManagerPrompt`、`AgentContext`、`WrapToolsWithEvents`
- 外部依赖:
  - `github.com/cloudwego/eino-ext/components/tool/commandline`: Operator
  - `github.com/cloudwego/eino/adk`: ChatModelAgent
  - `github.com/cloudwego/eino/components/model`: LLM 模型接口
  - `github.com/cloudwego/eino/components/tool`: 工具接口
  - `github.com/cloudwego/eino/components/tool/utils` (toolutils): `InferTool`
  - `github.com/cloudwego/eino/compose`: 工具节点配置
- 关键配置: `MaxIterations: 30`，list_files 默认最深 3 级

## 6. 变更影响面
- `ListFilesInput`/`ReadFileInput`/`WriteFileInput` 等类型被 `codewriter.go` 和 `codeexecutor.go` 复用，修改结构需检查所有引用
- 修改 write_file 工具行为可能影响文件管理代理的文件创建/覆盖逻辑
- list_files 的深度限制（maxdepth 3）变更影响工作区文件发现能力
- 影响 `deep_agent.go` 中的子代理组装

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 该文件中定义的工具 I/O 类型被多个子代理文件引用，修改字段名/类型时需全局搜索确认影响范围。
- 新增工具时需同步更新 `prompts.go` 中 `FileManagerPrompt` 的工具描述。
- write_file 当前会直接覆盖已有文件，如需添加备份/确认机制应在此处实现。
