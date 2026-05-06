# runtime_web_tools_test.go 技术说明

## 1. 文件定位
- 源文件: `internal/service/runtime_web_tools_test.go`
- 文档文件: `doc/src/internal/service/runtime_web_tools_test.go.plan.md`
- 所属模块: service

## 2. 核心职责
- 覆盖 Runtime V2 `WebSearch` 自定义 provider 配置。

## 3. 输入与输出
- 输入来源: `httptest.Server`、`config.WebSearchConfig`、`webSearchInput`。
- 输出结果: `webSearchOutput`、HTTP query/body 断言。

## 4. 关键测试覆盖
- TinyFish/http provider 可通过 query/limit 参数调用 JSON endpoint，并按配置路径提取 title/url/snippet。
- POST provider 可使用 body template 和 `resultsPath` 提取数组结果。

## 5. 维护建议
- 增加 provider 类型或 JSON path 语法时同步补解析测试。
