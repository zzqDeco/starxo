# runtime_web_tools.go 技术说明

## 1. 文件定位
- 源文件: `internal/service/runtime_web_tools.go`
- 文档文件: `doc/src/internal/service/runtime_web_tools.go.plan.md`
- 所属模块: service

## 2. 核心职责
- 提供 Runtime V2 deferred `WebFetch` / `WebSearch` 工具的 service-owned 实现。
- 将 web 访问包装成 catalog entry，统一接入 ToolSearch 和 permission pipeline。
- 支持 `agent.webSearch` 配置自定义 provider，并提供对齐 TinyFish Search API 的专用 provider。

## 3. 输入与输出
- 输入来源: `WebFetch` / `WebSearch` tool call。
- 输出结果:
  - `WebFetch` 返回 HTTP status、compact text、truncated 标记。
  - `WebSearch` 返回 provider、搜索 URL 和压缩后的结果行。

## 4. 关键实现细节
- `WebFetch` 使用 Go `http.Client` 发起 GET，请求超时和最大读取字节数可配置。
- HTML 通过轻量 tag strip 和空白压缩转换为文本，避免把整页 HTML 注入模型。
- `WebSearch` 默认使用 DuckDuckGo HTML endpoint，调用 `WebFetch` 后筛选非空结果行。
- 自定义 provider 支持 `type=http|tinyfish|duckduckgo`。
- `type=tinyfish` 对齐 TinyFish 官方 Search API：`GET https://api.search.tinyfish.ai`、`X-API-Key` 默认来自 `TINYFISH_API_KEY`、请求参数为 `query/location/language/page`，响应从 `results[].title/url/snippet` 提取。
- `type=http` provider 支持 GET/POST、headers、query/limit 参数、body template、结果 JSON path、title/url/snippet path。
- 模板变量支持 `{query}`、`{query_url}`、`{query_json}`、`{limit}` 及双大括号等价写法。
- 两个工具都是 deferred runtime tools；可被 ToolSearch 发现后再注入模型 schema。

## 5. 依赖关系
- 内部依赖: `internal/tools`
- 外部依赖: Go `net/http`

## 6. 变更影响面
- Runtime V2 首次提供非 MCP 的 web deferred tool surface。
- 网络能力来自本地应用进程，不经过远端 sandbox。

## 7. 维护建议
- TinyFish 协议变更时优先更新专用 adapter 和 fixture 测试，不把差异泄漏给 tool schema。
- 不在 WebSearch 中执行本地命令；非 TinyFish 搜索能力通过 `type=http` endpoint 接入。
- 大页面或二进制响应应继续限制读取大小，避免 prompt 膨胀。
