# choice.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/tools/choice.go
- 文档文件: doc/src/internal/tools/choice.go.plan.md
- 文件类型: Go 源码
- 所属模块: tools

## 2. 核心职责
- 实现 `ask_choice` 工具，利用 Eino 框架的中断/恢复（interrupt/resume）模式让 AI Agent 向用户展示结构化选项列表并获取用户选择。与 `ask_user`（自由文本回答）不同，`ask_choice` 提供预定义的选项供用户选择，适用于需要用户在具体方案间做决策的场景。首次调用触发中断并展示选项，用户选择后恢复执行并返回所选项的标签和描述。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `ChoiceToolInput`（包含 `Question string` 和 `Options []Option`）、`context.Context`（携带 Eino 中断/恢复状态）
- 输出结果: 首次调用触发 `tool.StatefulInterrupt` 中断；恢复后返回格式化字符串 `"User selected: <Label> — <Description>"`；选择索引超出范围时返回 error

## 4. 关键实现细节
- 结构体/接口定义:
  - `Option` — 单个选项，包含 `Label string` 和 `Description string`
  - `ChoiceInfo` — 中断信息，包含 `Question`、`Options` 和 `Selected int`（0-indexed，由用户填充），实现 `String()` 方法
  - `ChoiceState` — 中断状态，包含 `Question` 和 `Options`
  - `ChoiceToolInput` — 工具输入 Schema，包含 `Question` 和 `Options`（带 jsonschema 描述标签）
- 导出函数/方法:
  - `NewChoiceTool() tool.BaseTool` — 创建 `ask_choice` 工具
- 私有函数:
  - `choice(ctx, input) (string, error)` — 核心中断/恢复逻辑
    1. 首次调用: 触发中断，携带问题和选项
    2. 恢复但非目标: 重新中断（保持状态）
    3. 恢复且为目标: 验证选择索引范围，返回所选项信息
- init 函数:
  - 注册 `ChoiceInfo` 和 `ChoiceState` 到 Eino schema 系统
- Wails 绑定方法: 无
- 事件发射: 通过 Eino 中断机制间接触发前端交互

## 5. 依赖关系
- 内部依赖: 无
- 外部依赖:
  - `github.com/cloudwego/eino/components/tool` — `BaseTool`、`GetInterruptState`、`GetResumeContext`、`StatefulInterrupt`
  - `github.com/cloudwego/eino/components/tool/utils` — `InferTool`
  - `github.com/cloudwego/eino/schema` — `Register`
  - `context`、`fmt`（标准库）
- 关键配置: 无

## 6. 变更影响面
- `internal/tools/registry.go` — 通过 `RegisterBuiltin` 注册到工具注册表
- `internal/agent/` — Agent 工具调用时触发中断
- 前端聊天界面 — 需渲染选项列表 UI 组件，收集用户选择索引（0-indexed），填充 `ChoiceInfo.Selected` 后恢复
- `internal/tools/followup.go` — 共享相同的中断/恢复模式

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 用户选择使用 0-indexed 整数索引（`Selected` 字段），前端展示时通常从 1 开始编号，需注意转换。
- 选项验证仅检查索引范围（`selected < 0 || selected >= len(options)`），不检查选项内容。
- 与 `followup.go` 共享相同的中断/恢复代码模式，如需重构可考虑提取公共的中断/恢复辅助函数。
- `ChoiceInfo` 和 `ChoiceState` 必须在 `init()` 中注册，与 `followup.go` 保持一致。
