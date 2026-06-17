# i18nFormat.ts 技术说明

## 文件定位
- 源文件: `frontend/src/utils/i18nFormat.ts`
- 前端 i18n 命名占位符安全格式化工具。

## 核心职责
- `formatNamedMessage(t, key, named)` 先调用 `vue-i18n`，再对仍残留的 `{name}` 做显示层 fallback 替换。
- 防止 task progress、code preview 行数、request count 等用户可见文案显示原始占位符。

## 维护要点
- 该工具只处理显示层 fallback，不替代 locale 文案管理。
- 新增带命名占位符的关键 UI 文案时，优先使用该工具或等价的安全插值路径。
