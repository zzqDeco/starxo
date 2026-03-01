# notify.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/tools/notify.go
- 文档文件: doc/src/internal/tools/notify.go.plan.md
- 文件类型: Go 源码
- 所属模块: tools

## 2. 核心职责
- 实现 `notify_user` 工具，让 AI Agent 在不中断执行的情况下向用户发送简短的状态更新消息。与 `ask_user`（中断等待回答）不同，`notify_user` 是非阻塞的——Agent 发送通知后立即继续工作。消息以 `[Status]` 前缀格式返回，前端将其渲染为聊天时间线中的内联信息横幅。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `NotifyInput`（包含 `Message string` 状态消息）、`context.Context`
- 输出结果: 返回格式化字符串 `"[Status] <message>"`；消息为空时返回 `"No message provided."`

## 4. 关键实现细节
- 结构体/接口定义:
  - `NotifyInput` — 工具输入，包含 `Message string`（带 jsonschema 描述标签）
- 导出函数/方法:
  - `NewNotifyUserTool() tool.BaseTool` — 创建 `notify_user` 工具（工具名 `notify_user`）
- Wails 绑定方法: 无
- 事件发射: 无（通过工具返回值传递消息，前端解析 `[Status]` 前缀渲染）

## 5. 依赖关系
- 内部依赖: 无
- 外部依赖:
  - `github.com/cloudwego/eino/components/tool` — `BaseTool` 接口
  - `github.com/cloudwego/eino/components/tool/utils` — `InferTool`
  - `context`、`fmt`（标准库）
- 关键配置: 无

## 6. 变更影响面
- `internal/tools/registry.go` — 通过 `RegisterBuiltin` 注册到工具注册表
- 前端聊天界面 — 依赖 `[Status]` 前缀识别通知消息并渲染为信息横幅
- `internal/agent/` — Agent 在长时间任务中调用此工具保持用户知情

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- `[Status]` 前缀是前端识别通知消息的约定，修改时需同步前端解析逻辑。
- 该工具实现极简（无状态、无副作用），是新增类似非阻塞工具的良好参考模板。
- 如需支持通知级别（info/warning/error），可扩展 `NotifyInput` 添加 `Level` 字段。
