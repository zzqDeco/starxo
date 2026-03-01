# builtin.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/tools/builtin.go
- 文档文件: doc/src/internal/tools/builtin.go.plan.md
- 文件类型: Go 源码
- 所属模块: tools

## 2. 核心职责
- 定义并注册所有内置工具（`shell_execute`、`list_files`、`read_file`、`write_file`、`str_replace_editor`、`python_execute`），这些工具通过 `commandline.Operator` 接口委托到远程沙箱执行。同时提供 `sanitizedTool` 包装器，用于清理 LLM 生成的 JSON 参数中的控制字符，防止工具调用失败。每个工具使用 Eino 的 `InferTool` 泛型机制，根据 Go 结构体标签自动推导 JSON Schema。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `ToolRegistry`（工具注册表）、`commandline.Operator`（沙箱操作接口）、`workspacePath`（默认工作目录路径）
- 输出结果: `RegisterBuiltinTools` 返回 error；注册的工具被添加到 ToolRegistry 中；各工具的输入/输出通过 JSON Schema 定义

## 4. 关键实现细节
- 结构体/接口定义:
  - `ShellInput` / `ShellOutput` — `shell_execute` 工具的输入/输出
  - `ListFilesInput` / `ListFilesOutput` — `list_files` 工具的输入/输出
  - `ReadFileInput` / `ReadFileOutput` — `read_file` 工具的输入/输出
  - `WriteFileInput` / `WriteFileOutput` — `write_file` 工具的输入/输出
  - `sanitizedTool` — 工具包装器，清理 JSON 参数中的控制字符
- 导出函数/方法:
  - `RegisterBuiltinTools(registry, op, workspacePath) error` — 创建并注册所有内置工具
- 注册的工具:
  - `shell_execute` — 在沙箱中执行 shell 命令（通过 `sh -c`）
  - `list_files` — 列出目录下文件（`find -maxdepth 3 -type f`）
  - `read_file` — 读取文件内容
  - `write_file` — 写入文件内容
  - `str_replace_editor` — Eino 内置的字符串替换编辑器（被 `sanitizedTool` 包装）
  - `python_execute` — Eino 内置的 Python 执行器（使用 `python3`）
- 私有类型/方法:
  - `sanitizedTool` — 实现 `tool.BaseTool`，转发 `Info()` 并在 `InvokableRun` 中清理 JSON 参数
  - `sanitizeJSONStringValues(input) string` — 解析 JSON 并清理所有字符串值中的控制字符
  - `sanitizeMapValues(m)` — 递归清理 map 中的字符串值
  - `stripControlChars(s) string` — 移除控制字符（保留 `\n`、`\t`、`\r`）
- Wails 绑定方法: 无
- 事件发射: 无

## 5. 依赖关系
- 内部依赖:
  - 同包引用: `ToolRegistry`（注册工具）
- 外部依赖:
  - `github.com/cloudwego/eino-ext/components/tool/commandline` — `Operator` 接口、`StrReplaceEditor`、`PyExecutor`、`CommandOutput` 类型
  - `github.com/cloudwego/eino/components/tool` — `BaseTool`、`InvokableTool`、`Option` 接口
  - `github.com/cloudwego/eino/schema` — `ToolInfo` 类型
  - `github.com/cloudwego/eino/components/tool/utils` — `InferTool` 泛型工具创建器
  - `context`、`encoding/json`、`fmt`、`strings`、`unicode`（标准库）
- 关键配置: `workspacePath` 作为 `list_files` 的默认目录

## 6. 变更影响面
- `internal/tools/registry.go` — 工具注册到 ToolRegistry
- `internal/sandbox/operator.go` — 内置工具通过 `commandline.Operator` 接口调用沙箱操作
- `internal/agent/` — Agent 使用注册表中的内置工具
- 前端工具调用展示 — 工具名称和参数 Schema 变更影响前端渲染

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 新增内置工具时遵循现有模式：定义 Input/Output 结构体 -> `InferTool` 创建 -> `RegisterBuiltin` 注册。
- `sanitizedTool` 专为 `str_replace_editor` 设计（LLM 常在编辑操作中产生控制字符），新工具如遇类似问题也可复用此包装器。
- `stripControlChars` 的字符过滤规则变更需谨慎，避免误删有效内容（如制表符和换行符已被保留）。
- `list_files` 的 `maxdepth 3` 限制是硬编码值，如需可配置化应提取为参数。
