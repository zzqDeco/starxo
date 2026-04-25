# style.css 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/style.css
- 文档文件: doc/src/frontend/src/style.css.plan.md
- 文件类型: CSS 样式
- 所属模块: frontend/src (全局样式)

## 2. 核心职责
- 定义全局 CSS 变量（设计令牌）和基础样式重置，为整个前端应用提供统一的视觉规范。
- 包含自定义字体加载（Nunito）、滚动条样式、选区样式、代码块样式、Markdown 渲染样式、动画关键帧和无障碍支持。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: 通过 main.ts 全局导入
- 输出结果: 应用于整个前端的全局样式规则和 CSS 变量

## 4. 关键实现细节
- **CSS 变量体系**:
  - 背景层级: `--bg-deepest` (#020617) → `--bg-base` (#07111f) → `--bg-surface` (#0f172a) → `--bg-elevated` (#172033) → `--bg-raised` (#1e293b) → `--bg-hover` (#243044)
  - 文本层级: `--text-primary` → `--text-secondary` → `--text-muted` → `--text-faint`
  - 强调色: cyan (#22d3ee)、emerald (#22c55e)、amber (#f59e0b)、rose (#fb7185)、violet (#a78bfa)、blue (#60a5fa)
  - Agent 颜色系统: orchestrator (cyan)、code-writer (blue)、code-executor (purple)、file-manager (green)、default (amber)
  - 间距系统: xs (4px) → sm (8px) → md (16px) → lg (24px) → xl (32px)
  - 圆角系统: sm (6px) → md (8px) → lg (12px) → xl (16px)
  - 字体: sans (Nunito)、mono (JetBrains Mono)
- **代码块样式**: `.hljs-code-block` 包含代码头部（语言标签 + 复制按钮）和 highlight.js 语法高亮色覆盖
- **Markdown 样式**: `.markdown-body` 类定义了段落、链接、列表、行内代码、引用、标题、表格、分割线的样式
- **动画**: fadeIn、pulse、blink、slideInLeft、slideInRight、shimmer（骨架屏）
- **工作台视觉**: `--gradient-workbench` 提供低干扰深色工作区背景，组件以边框、层级阴影和状态色承载信息密度。
- **无障碍**: 全局 `:focus-visible` 焦点环、按钮 cursor 规则、禁用态 cursor、`prefers-reduced-motion` 减少动画支持。
- **Wails 特性**: `.wails-drag` 类启用窗口拖拽区域

## 5. 依赖关系
- 内部依赖: `assets/fonts/nunito-v16-latin-regular.woff2` 字体文件
- 外部依赖: 无（纯 CSS）

## 6. 变更影响面
- CSS 变量修改会影响所有使用这些变量的组件
- 代码块和 Markdown 样式修改影响 MessageBubble 和 TimelineEventItem 的消息渲染
- 需与 App.vue 中的 Naive UI themeOverrides 保持色值一致

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 新增 CSS 变量时遵循现有的命名约定和层级体系。
- 修改颜色值时需同步 App.vue 的 themeOverrides 配置。
- Agent 颜色系统的修改需同步 MessageBubble.vue 和 TimelineEventItem.vue 中的 agentColor 函数。
