# highlight.ts 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/composables/highlight.ts
- 文档文件: doc/src/frontend/src/composables/highlight.ts.plan.md
- 文件类型: TypeScript 源码
- 所属模块: frontend/src/composables

## 2. 核心职责
- 提供共享的 highlight.js core 单例，避免 Markdown 渲染和代码预览分别引入完整 highlight.js 包。
- 统一注册前端常用代码语言和别名。
- 提供 `escapeHtml()` 作为无高亮时的安全 HTML 转义工具。

## 3. 关键实现细节
- `getHighlighter()` 首次调用时注册 bash、css、go、javascript、json、markdown、python、typescript、xml。
- 注册 sh/shell/zsh、js/jsx、ts/tsx、html/vue 等别名，覆盖当前工作区预览和消息代码块最常见语言。
- `registered` 标记保证语言注册只执行一次。

## 4. 变更影响面
- 影响 `useHelpers.ts` 的 Markdown 代码块高亮。
- 影响 `CodePreview.vue` 的工作区文件预览高亮。
- 新增语言支持时应只添加必要语言，避免重新拉大全量语言包。
