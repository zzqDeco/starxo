# registry.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/tools/registry.go
- 文档文件: doc/src/internal/tools/registry.go.plan.md
- 文件类型: Go 源码
- 所属模块: tools

## 2. 核心职责
- `ToolRegistry` 是工具的中心注册表，线程安全地管理来自三个来源的工具：内置工具（builtins）、MCP 服务器工具（mcpTools，按服务器名分组）和自定义工具（custom）。提供注册（`RegisterBuiltin`/`RegisterMCPTools`/`RegisterCustom`）、查询（`GetAll`/`GetByNames`）和删除（`Remove`）操作。查询时跨所有来源聚合结果。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `tool.BaseTool` 实例（Eino 框架工具接口）、工具名称字符串、MCP 服务器名称
- 输出结果: `RegisterBuiltin` 返回 error（获取工具信息失败时）；`GetAll` 返回所有工具切片 `[]tool.BaseTool`；`GetByNames` 返回按名称匹配的工具子集；`Remove` 无返回值

## 4. 关键实现细节
- 结构体/接口定义:
  - `ToolRegistry` — 工具注册表，持有 `sync.RWMutex`、`builtins map[string]tool.BaseTool`、`mcpTools map[string][]tool.BaseTool`、`custom map[string]tool.BaseTool`
- 导出函数/方法:
  - `NewToolRegistry() *ToolRegistry` — 创建空注册表
  - `RegisterBuiltin(t tool.BaseTool) error` — 注册内置工具（自动从 `Info()` 提取名称）
  - `RegisterMCPTools(name string, tools []tool.BaseTool)` — 注册 MCP 服务器工具集（同名替换）
  - `RegisterCustom(name string, t tool.BaseTool)` — 注册自定义工具
  - `GetAll() []tool.BaseTool` — 返回所有来源的全部工具
  - `GetByNames(names ...string) []tool.BaseTool` — 按名称集合查询工具（跨所有来源，未匹配名称静默忽略）
  - `Remove(name string)` — 从所有来源中删除指定名称的工具
- Wails 绑定方法: 无
- 事件发射: 无

## 5. 依赖关系
- 内部依赖: 无
- 外部依赖:
  - `github.com/cloudwego/eino/components/tool` — 使用 `BaseTool` 接口和 `Info()` 方法
  - `fmt`、`sync`（标准库）
- 关键配置: 无

## 6. 变更影响面
- `internal/tools/builtin.go` — `RegisterBuiltinTools` 函数向注册表注册内置工具
- `internal/tools/mcp.go` — MCP 工具加载后通过 `RegisterMCPTools` 注册
- `internal/tools/custom.go` — 自定义工具通过 `RegisterCustom` 注册
- `internal/agent/` — Agent 层通过 `GetAll()` 或 `GetByNames()` 获取工具列表传递给 Eino 框架
- `app.go` — 应用层创建和管理 ToolRegistry 实例

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- `GetByNames` 对 MCP 工具需调用 `Info()` 获取名称，而 builtins 和 custom 直接用 map key 查找，保持这两种查找方式的一致性。
- `Remove` 方法同时清理三个来源，MCP 工具的删除使用原地过滤（`tools[:0]` 复用底层数组），注意切片引用语义。
- 如需支持工具优先级或覆盖策略，需在 `GetAll`/`GetByNames` 中增加去重逻辑。
