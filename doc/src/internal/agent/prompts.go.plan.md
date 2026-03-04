# prompts.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/agent/prompts.go
- 文档文件: doc/src/internal/agent/prompts.go.plan.md
- 文件类型: Go 源码
- 所属模块: agent

## 2. 核心职责
- 该文件定义了所有代理的系统提示词（system prompt）。包括核心深度代理（DeepAgent）、代码编写代理（CodeWriter）、代码执行代理（CodeExecutor）和文件管理代理（FileManager）四个角色的提示词。每个提示词通过 `AgentContext` 动态注入运行时环境信息（SSH、容器、工作区路径等），使代理能感知其所处的沙箱环境。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `AgentContext` 结构体，包含 WorkspacePath、ContainerName、ContainerID、SSHHost、SSHPort、SSHUser 等运行时环境信息
- 输出结果: 返回格式化后的系统提示词字符串，直接用于代理初始化配置

## 4. 关键实现细节
- 结构体/接口定义: 无
- 导出函数/方法:
  - `DeepAgentPrompt(ac AgentContext) string`: 生成核心代理提示词，定义了直接工具（ask_user、ask_choice、write_todos、update_todo、notify_user）和子代理（code_writer、code_executor、file_manager）的使用规则及决策逻辑
  - `CodeWriterPrompt(ac AgentContext) string`: 生成代码编写代理提示词，强调使用 str_replace_editor、read_file、list_files 工具进行代码相关操作；包含 reasoning 指导（"Before each tool call, briefly explain what you are about to do and why"）
  - `CodeExecutorPrompt(ac AgentContext) string`: 生成代码执行代理提示词，定义 python_execute、shell_execute、read_file 工具的使用方式；包含 reasoning 指导
  - `FileManagerPrompt(ac AgentContext) string`: 生成文件管理代理提示词，使用 list_files、read_file、write_file 工具处理非代码文件和批量操作；包含 reasoning 指导
- Wails 绑定方法: 无
- 事件发射: 无

## 5. 依赖关系
- 内部依赖:
  - 同包 `agent`: `AgentContext` 结构体
- 外部依赖:
  - `fmt`: 字符串格式化
- 关键配置: 无

## 6. 变更影响面
- 修改提示词内容直接影响 AI 代理的行为模式和决策逻辑
- DeepAgentPrompt 的变更影响代理委派策略和用户交互方式
- 子代理提示词的变更影响各专用代理的工具使用方式和工作流程
- 新增/移除工具时需同步更新对应代理的提示词描述
- 影响 `deep_agent.go`、`codewriter.go`、`codeexecutor.go`、`filemanager.go` 的代理配置

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 修改提示词时应通过实际对话测试验证代理行为变化，避免引入意外的行为回归。reasoning 指导行影响前端 reasoning 事件的内容质量。
- 新增子代理时需在 `DeepAgentPrompt` 的 SUB-AGENTS 部分添加描述，并创建对应的 `*Prompt` 函数。
- 新增工具时需在对应代理的 YOUR TOOLS 部分添加工具说明，包括名称和使用场景。
- 提示词中的环境变量（SSH、容器信息）来自 `AgentContext`，确保 `context.go` 中的字段与提示词模板匹配。
