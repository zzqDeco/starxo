# useHelpers.ts 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/composables/useHelpers.ts
- 文档文件: doc/src/frontend/src/composables/useHelpers.ts.plan.md
- 文件类型: TypeScript 源码
- 所属模块: frontend/src/composables (组合式函数)

## 2. 核心职责
- 提供两个通用组合式函数：`useMarkdown`（Markdown 渲染）和 `useAutoScroll`（自动滚动管理）。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: useMarkdown — Markdown 文本字符串；useAutoScroll — 滚动容器 ref
- 输出结果: useMarkdown — HTML 字符串；useAutoScroll — 滚动状态和控制方法

## 4. 关键实现细节
- **useMarkdown**:
  - 使用单例模式 (`mdInstance`) 初始化 markdown-it 实例
  - 配置: html=false, linkify=true, typographer=true
  - 代码高亮: 集成 highlight.js，生成带语言标签和复制按钮的代码块
  - 代码块 HTML 结构: `.hljs-code-block` > `.code-block-header` (lang label + copy btn) + `code.hljs`
  - 复制按钮使用内联 onclick 调用 `navigator.clipboard.writeText`
  - 返回 `{ renderMarkdown }` 函数
- **useAutoScroll**:
  - 接收 `containerRef` (HTMLElement ref)
  - 维护状态: `isAutoScroll` (是否自动滚动) 和 `isNearBottom` (是否接近底部)
  - `checkScroll()` — 检测是否在底部 80px 阈值内
  - `scrollToBottom(smooth?)` — 滚动到底部，支持平滑/即时模式
  - `onScroll()` — 滚动事件处理器，更新状态
  - 返回 `{ isAutoScroll, isNearBottom, scrollToBottom, onScroll }`

## 5. 依赖关系
- 内部依赖: 无
- 外部依赖: `vue` (ref, computed, onMounted, onUnmounted)、`markdown-it` (MarkdownIt)、`highlight.js` (hljs)

## 6. 变更影响面
- useMarkdown 修改影响 MessageBubble 和 TimelineEventItem 的消息渲染
- useAutoScroll 修改影响 ChatPanel 的滚动行为
- Markdown 渲染的 HTML 结构修改需同步 style.css 中的代码块样式

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- markdown-it 的 highlight 函数生成的 HTML 结构需与 style.css 中的 `.hljs-code-block` 相关样式保持一致。
- 复制按钮使用 onclick 内联脚本，受 CSP 策略限制时可能需要改为事件委托。
- useAutoScroll 的 80px 阈值可根据用户反馈调整。
