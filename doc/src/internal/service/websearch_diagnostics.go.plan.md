# websearch_diagnostics.go 技术说明

## 文件定位
- 源文件: `internal/service/websearch_diagnostics.go`
- 所属模块: service

## 核心职责
- 为设置页提供 `SettingsService.DiagnoseWebSearch` 和显式用户触发的 `SettingsService.TestWebSearch`。
- 对 Runtime V2 `WebSearch` provider 做静态诊断，包括启用状态、默认 provider、endpoint scheme/host、公网目标和 TinyFish API key env。

## 关键实现细节
- DuckDuckGo 作为隐式 provider 支持默认配置。
- TinyFish 默认端点为 `https://api.search.tinyfish.ai`，默认 API key env 为 `TINYFISH_API_KEY`。
- endpoint 校验复用 `runtime_web_tools.go` 的 URL guard，非公网 endpoint 标记为 warn，运行时实际访问仍需 permission queue。
- 诊断不泄露 API key，只返回 env name 和是否存在。
- `TestWebSearch` 会执行真实搜索 smoke test，返回 provider、URL、结果数量、耗时和失败 message；调用失败或响应成功但解析结果为 0 时不会抛给前端 toast，而是返回 `ok=false` 结果供面板展示。

## 维护建议
- 不要在静态诊断中默认发起真实搜索；live smoke 必须保持显式用户动作。
