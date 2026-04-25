# CodePreview.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/files/CodePreview.vue
- 文档文件: doc/src/frontend/src/components/files/CodePreview.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/files

## 2. 核心职责
- 文件代码预览组件，展示路径/类型/大小/行数，并进行语法高亮和逐行渲染。

## 3. 输入与输出
- 输入来源: Props (`path`, `content`, `loading`, `fileSize`)
- 输出结果: 渲染代码表格，支持一键复制全文

## 4. 关键实现细节
- 语言识别:
  - 按文件扩展名映射到 highlight.js 语言
  - 通过 `getHighlighter()` 复用 highlight.js core 与常用语言注册
- 渲染策略:
  - 先高亮（或 HTML 转义兜底）再按行切分
  - 使用双列表格渲染：左侧行号，右侧代码
- 交互:
  - `copyAll()` 将全文写入剪贴板
- 文案:
  - 全部走 i18n（`codePreview.*`）

## 5. 依赖关系
- 外部依赖:
  - `vue`
  - `naive-ui`
  - `@/composables/highlight`
  - `highlight.js/lib/core`
  - `@vicons/ionicons5`
  - `vue-i18n`

## 6. 变更影响面
- 作为 WorkspacePanel 的右侧预览区，直接影响文件可读性与定位效率。

## 7. 维护建议
- 若后续支持超大文件，建议限制最大渲染行数并增加提示。
- 剪贴板 API 在受限环境可能失败，可补充失败提示。
