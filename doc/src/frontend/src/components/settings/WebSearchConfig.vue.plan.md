# WebSearchConfig.vue 技术说明

## 文件定位
- 源文件: `frontend/src/components/settings/WebSearchConfig.vue`
- 设置页 WebSearch/TinyFish 配置面板。

## 核心职责
- 管理 `settings.agent.webSearch` 的启用状态、默认 provider 和 TinyFish provider 配置。
- 提供 TinyFish preset，写入 `type=tinyfish`、`endpoint=https://api.search.tinyfish.ai`、`apiKeyEnv=TINYFISH_API_KEY`、默认 location/language。
- 调用 `SettingsService.DiagnoseWebSearch` 展示 provider 静态诊断结果。
- 调用 `SettingsService.TestWebSearch` 执行用户显式触发的 smoke test，展示 provider、URL 和紧凑结果。

## 维护建议
- 诊断按钮只做配置/endpoint/API key env 检查，不应在用户无感知时发起真实搜索；smoke test 按钮才允许真实网络请求。
- 新增 provider 类型时同步更新后端 `WebSearchProviderConfig`、诊断结果和 provider options。
