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
- 默认 locale 优先读本地缓存，否则回退到 `zh`
- `fallbackLocale` 固定为 `en`
- 语言包一次性注册为 `{ en, zh }`

## 5. 依赖关系
- 内部依赖: `./en`、`./zh`
- 外部依赖: `vue-i18n`

## 6. 变更影响面
- 直接影响应用启动时的语言选择和缺失 key 的回退行为。

## 7. 维护建议
- 新增语言时，应同步扩展 messages 注册和 locale 持久化选择逻辑。
