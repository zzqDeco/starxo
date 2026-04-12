# mcp_runtime.go 技术说明

## 1. 文件定位
- 源文件: `internal/tools/mcp_runtime.go`
- 文档文件: `doc/src/internal/tools/mcp_runtime.go.plan.md`
- 所属模块: tools

## 2. 核心职责
- 管理 MCP server 连接、状态、metadata 缓存和 canonical action adapter。

## 3. 输入与输出
- 输入来源: `config.MCPServerConfig`
- 输出结果:
  - `MCPServerHandle`
  - `BuildMCPActionCatalog(...)`

## 4. 关键实现细节
- server 状态至少包含 `disabled / pending / connected / failed / needs_auth`
- metadata 缓存包括 `Tools / Resources / ResourceTemplates`
- action tool adapter 对外暴露 canonical name，对内调用 remote name
- service 层复用 cached metadata 时必须额外校验 server config identity：
  - 只有 `server name + ConfigIdentityDigest` 同时匹配当前 config 时才可信
  - 同名但 command / url / env / transport 等 identity 变化时，旧 cache 不能复用

## 5. 依赖关系
- 外部依赖:
  - `github.com/modelcontextprotocol/go-sdk/mcp`

## 6. 变更影响面
- 直接影响 deferred MCP searchable/loadable 池和 runner rebuild 生命周期

## 7. 维护建议
- 若增加重连或刷新策略，优先扩展 handle 状态与缓存失效逻辑，不要在 service 层散落判断
