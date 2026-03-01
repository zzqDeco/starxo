# mcp.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/tools/mcp.go
- 文档文件: doc/src/internal/tools/mcp.go.plan.md
- 文件类型: Go 源码
- 所属模块: tools

## 2. 核心职责
- 提供 MCP（Model Context Protocol）服务器的连接和工具加载能力。`ConnectMCPServer` 根据配置建立与 MCP 服务器的连接，支持两种传输方式：stdio（子进程通信）和 SSE（HTTP Server-Sent Events）。`LoadMCPTools` 从已连接的 MCP 会话中获取可用工具列表，可选按名称过滤。这使得 starxo 能够动态扩展 AI Agent 的工具集，接入外部 MCP 兼容的工具服务。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: `config.MCPServerConfig`（服务器名称、传输类型、命令/URL、参数、环境变量、工具名称过滤列表）、`context.Context`、`*mcp.ClientSession`（已连接会话）
- 输出结果: `ConnectMCPServer` 返回 `*mcp.ClientSession` 和 error；`LoadMCPTools` 返回 `[]tool.BaseTool` 和 error

## 4. 关键实现细节
- 结构体/接口定义: 无自定义结构体
- 导出函数/方法:
  - `ConnectMCPServer(ctx, cfg) (*mcp.ClientSession, error)` — 建立 MCP 服务器连接
    - stdio 模式: 使用 `exec.CommandContext` 启动子进程，通过 `mcp.CommandTransport` 通信，支持自定义环境变量
    - sse 模式: 使用 `mcp.SSEClientTransport` 连接 HTTP SSE 端点
  - `LoadMCPTools(ctx, session, toolNames) ([]tool.BaseTool, error)` — 从 MCP 会话加载工具（空 toolNames 表示加载全部）
- Wails 绑定方法: 无
- 事件发射: 无

## 5. 依赖关系
- 内部依赖:
  - `starxo/internal/config` — 使用 `MCPServerConfig`
- 外部依赖:
  - `github.com/cloudwego/eino-ext/components/tool/mcp/officialmcp` — `GetTools` 函数获取 MCP 工具
  - `github.com/cloudwego/eino/components/tool` — `BaseTool` 接口
  - `github.com/modelcontextprotocol/go-sdk/mcp` — MCP 协议客户端（`Client`、`ClientSession`、`Transport`、`CommandTransport`、`SSEClientTransport`、`Implementation`）
  - `context`、`fmt`、`os/exec`（标准库）
- 关键配置: `config.MCPServerConfig`（Name、Transport、Command、Args、Env、URL、ToolNames）

## 6. 变更影响面
- `app.go` — 应用层在初始化时调用 `ConnectMCPServer` 和 `LoadMCPTools`，并通过 `ToolRegistry.RegisterMCPTools` 注册
- `internal/tools/registry.go` — 加载的 MCP 工具通过 `RegisterMCPTools` 注册到注册表
- `internal/config/` — `MCPServerConfig` 结构变更需同步
- 前端 MCP 配置界面 — 传输类型和参数变更影响配置选项

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- MCP 连接的 `Implementation.Version` 当前硬编码为 `"1.0.0"`，如需版本协商应提取到配置中。
- stdio 模式的子进程生命周期由 `context` 管理，context 取消时子进程会被终止。
- 如需支持新的 MCP 传输类型（如 WebSocket），在 `switch cfg.Transport` 中添加新 case。
- `LoadMCPTools` 依赖 `officialmcp.GetTools`，该函数的行为变更需关注。
