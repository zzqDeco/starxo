# followup.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/tools/followup.go
- 文档文件: doc/src/internal/tools/followup.go.plan.md
- 文件类型: Go 源码
- 所属模块: tools

## 2. 核心职责
- 实现 `ask_user` 工具，利用 Eino 框架的中断/恢复（interrupt/resume）模式让 AI Agent 在执行过程中向用户提出澄清性问题。首次调用时触发中断并携带问题列表，用户回答后恢复执行并返回用户的回答文本。该工具使 Agent 能够在信息不足时主动向用户寻求帮助，而非猜测性地继续执行。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `FollowUpToolInput`（包含 `Questions []string` 问题列表）、`context.Context`（携带 Eino 中断/恢复状态）
- 输出结果: 首次调用触发 `tool.StatefulInterrupt` 中断（返回空字符串和中断 error）；恢复后返回用户回答字符串 `resumeData.UserAnswer`

## 4. 关键实现细节
- 结构体/接口定义:
  - `FollowUpInfo` — 中断信息，包含 `Questions []string` 和 `UserAnswer string`，实现 `String()` 方法用于文本展示
  - `FollowUpState` — 中断状态，包含 `Questions []string`，用于恢复时重建上下文
  - `FollowUpToolInput` — 工具输入 Schema，包含 `Questions []string`（带 jsonschema 描述标签）
- 导出函数/方法:
  - `NewFollowUpTool() tool.BaseTool` — 创建 `ask_user` 工具（工具名 `ask_user`）
- 私有函数:
  - `followUp(ctx, input) (string, error)` — 核心中断/恢复逻辑
    1. 首次调用（未中断）: 调用 `tool.StatefulInterrupt` 触发中断
    2. 恢复但非目标: 重新触发中断（保持状态）
    3. 恢复且为目标: 返回用户回答
- init 函数:
  - 注册 `FollowUpInfo` 和 `FollowUpState` 到 Eino schema 系统（支持序列化/反序列化）
- Wails 绑定方法: 无
- 事件发射: 通过 Eino 中断机制间接触发前端交互

## 5. 依赖关系
- 内部依赖: 无
- 外部依赖:
  - `github.com/cloudwego/eino/components/tool` — `BaseTool`、`GetInterruptState`、`GetResumeContext`、`StatefulInterrupt` 中断/恢复 API
  - `github.com/cloudwego/eino/components/tool/utils` — `InferTool` 泛型工具创建器
  - `github.com/cloudwego/eino/schema` — `Register` 类型注册
  - `context`、`fmt`、`strings`（标准库）
- 关键配置: 无

## 6. 变更影响面
- `internal/tools/registry.go` — 通过 `RegisterBuiltin` 注册到工具注册表
- `internal/agent/` — Agent 在工具调用时触发中断，Agent Runner 负责暂停/恢复流程
- 前端聊天界面 — 中断时前端需展示问题列表并收集用户输入，恢复时将 `FollowUpInfo`（含 UserAnswer）传回
- `internal/tools/choice.go` — 使用相同的中断/恢复模式，修改模式时需同步

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- `FollowUpInfo` 和 `FollowUpState` 必须在 `init()` 中注册到 schema 系统，否则序列化会失败。
- 中断/恢复模式是 Eino 框架的核心模式，修改时需理解 `GetInterruptState`、`GetResumeContext`、`StatefulInterrupt` 三个 API 的交互语义。
- 如果非目标恢复的情况下直接返回（而非重新中断），会导致多工具中断场景下的状态丢失。
