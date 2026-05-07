# todos.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/tools/todos.go
- 文档文件: doc/src/internal/tools/todos.go.plan.md
- 文件类型: Go 源码
- 所属模块: tools

## 2. 核心职责
- 实现任务追踪工具 `write_todos` 和 `update_todo`，支持 AI Agent 以 DAG（有向无环图）形式声明和管理多步骤任务进度。`write_todos` 用于声明或更新完整任务列表（含依赖关系），`update_todo` 用于更新单个任务的状态。任务数据按 `sessionID` 存储在内存中的 session bucket，缺少 `ctx.Value("sessionID")` 时回退到全局兼容 bucket；工具返回完整的 JSON 序列化任务列表供前端渲染可视化 DAG 组件。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `WriteTodosInput`（包含 `Todos []TodoItem` 完整任务列表）、`UpdateTodoInput`（包含 `ID`、`Status`、可选 `Title`）、`context.Context`
- 输出结果: 返回格式化的摘要字符串（含各状态计数）和完整 JSON 任务列表（以 `---` 分隔）；错误时返回 error（如未知依赖 ID、无效状态）；`update_todo` ID 未找到时返回软提示（非 error），避免中断 Agent 管道

## 4. 关键实现细节
- 结构体/接口定义:
  - `TodoItem` — 任务项，包含 `ID`、`Title`、`Status`（pending/in_progress/done/failed/blocked）、`DependsOn []string`
  - `WriteTodosInput` — `write_todos` 工具输入
  - `UpdateTodoInput` — `update_todo` 工具输入，包含 `ID`、`Status`、可选 `Title`
  - `todoStore` — 包级内存任务存储（`sync.Mutex` + 全局兼容 `[]TodoItem` + `map[sessionID][]TodoItem`）
- 导出函数/方法:
  - `ClearTodosForSession(sessionID)` — 重置指定 session 的内存 todo 存储
  - `SnapshotTodosForSession(sessionID)` — 返回指定 session 的 `model.RuntimeTodoItem` 深拷贝，供 Runtime context compact 持久化
  - `RestoreTodosForSession(sessionID, ...)` — 从 `SessionData.RuntimeContextCompact.Todos` 恢复指定 session 的内存 todo 状态
  - `ClearTodos()` / `SnapshotTodos()` / `RestoreTodos(...)` — 全局兼容 wrappers，仅操作缺少 `sessionID` 时使用的全局 bucket
  - `NewWriteTodosTool() tool.BaseTool` — 创建 `write_todos` 工具
    - 验证 DAG 有效性：检查所有 `DependsOn` 引用的 ID 是否存在
    - 全量替换当前 context session bucket 中的任务列表；没有 `sessionID` 时替换全局兼容 bucket
    - 返回状态计数摘要和完整 JSON
  - `NewUpdateTodoTool() tool.BaseTool` — 创建 `update_todo` 工具
    - 验证 ID 非空和状态值合法性
    - 在当前 context session bucket 内按 ID 查找并更新状态，可选更新标题；没有 `sessionID` 时使用全局兼容 bucket
    - ID 未找到时返回 Warning 软提示（非 error），防止 NodeRunError 中断 Agent 管道
    - 返回更新后的状态计数摘要和完整 JSON
- 全局状态:
  - `todoStore` — 包级 `var`，通过 `sync.Mutex` 保护的 session bucket + 全局兼容 bucket 内存存储
- Wails 绑定方法: 无
- 事件发射: 无（前端通过解析工具返回的 JSON 渲染 DAG）

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/logger` — 诊断日志（`write_todos` 存储确认、`update_todo` 查找调试）
  - `starxo/internal/model` — Runtime compact todo DTO
- 外部依赖:
  - `github.com/cloudwego/eino/components/tool` — `BaseTool` 接口
  - `github.com/cloudwego/eino/components/tool/utils` — `InferTool`
  - `context`、`encoding/json`、`fmt`、`strings`、`sync`（标准库）
- 关键配置: 无

## 6. 变更影响面
- `ClearTodosForSession(...)` 被 `internal/service/session_svc.go`（CreateSession、SwitchSession/启动兜底）和 `internal/service/chat.go`（ClearHistory、RemoveSession）调用
- `SnapshotTodosForSession(...)` / `RestoreTodosForSession(...)` 被 Runtime context compact 和 session restore 调用
- `internal/tools/registry.go` — 通过 `RegisterBuiltin` 注册到工具注册表
- 前端 DAG 组件 — 依赖工具返回的 JSON 格式（`TodoItem` 结构），字段变更需同步前端解析逻辑
- `internal/agent/` — Agent 在多步骤任务中调用这两个工具追踪进度

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- `todoStore` 仍是包级内存状态，但已按 `sessionID` 分 bucket；新增调用路径应优先使用 session-scoped API，只有兼容旧路径时才使用全局 wrappers。
- `write_todos` 执行全量替换（`copy`），不支持增量更新；如需增量模式可新增工具或参数。
- DAG 验证仅检查依赖 ID 存在性，不检测循环依赖；如任务依赖复杂度增加需增加环检测。
- `TodoItem` 的 JSON 字段变更需同步前端 Vue 组件的解析逻辑。
