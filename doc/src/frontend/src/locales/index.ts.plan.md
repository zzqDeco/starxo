# index.ts 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: `frontend/src/locales/index.ts`
- 文档文件: `doc/src/frontend/src/locales/index.ts.plan.md`
- 文件类型: TypeScript 源码
- 所属模块: frontend/src/locales

## 2. 核心职责
- 初始化并导出全局 `vue-i18n` 实例。

## 3. 输入与输出
- 输入来源:
  - `localStorage.getItem('locale')`
  - `en.ts` / `zh.ts` 语言包
- 输出结果: 默认导出的 `i18n` 实例

## 4. 关键实现细节
- 使用组合式 API 模式：`legacy: false`
- 默认 locale 优先读本地缓存，否则读浏览器语言，最终回退到 `zh`
- `normalizeLocale()` 会把 `zh-CN` / `en-US` 等平台 locale 归一化为 `zh` / `en`，并写回 localStorage，避免 locale key 漂移。
- `fallbackLocale` 采用双向 fallback：中文缺失回退英文，英文缺失回退中文。
- 语言包一次性注册为 `{ en, zh }`
- `missing` handler 不直接展示原始 key；先手动查当前语言包与 fallback 语言包，最后才把 key 的最后一段 humanize，避免 UI 暴露 `header.title` 或 `Mode Label` 等变量名式文案。

## 5. 依赖关系
- 内部依赖: `./en`、`./zh`
- 外部依赖: `vue-i18n`

## 6. 变更影响面
- 直接影响应用启动时的语言选择和缺失 key 的回退行为。

## 7. 维护建议
- 新增语言时，应同步扩展 messages 注册和 locale 持久化选择逻辑。
