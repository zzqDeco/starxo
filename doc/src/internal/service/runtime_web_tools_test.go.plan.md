# runtime_web_tools_test.go 技术说明

## 1. 文件定位
- 源文件: `internal/service/runtime_web_tools_test.go`
- 文档文件: `doc/src/internal/service/runtime_web_tools_test.go.plan.md`
- 所属模块: service

## 2. 核心职责
- 覆盖 Runtime V2 `WebSearch` 自定义 provider 配置和 TinyFish Search API adapter。

## 3. 输入与输出
- 输入来源: `httptest.Server`、`config.WebSearchConfig`、`webSearchInput`。
- 输出结果: `webSearchOutput`、HTTP query/body 断言。

## 4. 关键测试覆盖
- HTTP provider 可通过自定义 query/limit 参数调用 JSON endpoint，并按配置路径提取 title/url/snippet。
- TinyFish provider 使用 `X-API-Key` header、`query/location/language/page` 参数，不发送非官方 `limit` 参数，并解析官方 `results` 响应结构。
- TinyFish provider 缺少 API key 环境变量时返回明确错误。
- `TINYFISH_API_KEY` 存在时运行真实 TinyFish Search API smoke test；未设置时自动 skip。
- POST provider 可使用 body template 和 `resultsPath` 提取数组结果。

## 5. 维护建议
- 增加 provider 类型或 JSON path 语法时同步补解析测试。
