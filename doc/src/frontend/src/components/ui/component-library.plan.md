# frontend/src/components/ui component library 技术说明

## 1. 文件定位
- 项目: starxo
- 源目录: frontend/src/components/ui
- 文档文件: doc/src/frontend/src/components/ui/component-library.plan.md
- 文件类型: Vue 基础 UI 组件库
- 所属模块: frontend/src/components/ui

## 2. 核心职责
- 承接 Figma v0.5 `02 Components / Component API Matrix` 的基础组件契约。
- 统一按钮、图标按钮、状态徽标、source list row、file row、timeline row、permission card、message bubble、terminal row、settings row、runtime task row、modal surface、empty state 等可复用 UI 原语。
- 所有组件优先读取 `--sx-*` semantic tokens，不直接硬编码页面级视觉。

## 3. 输入与输出
- 输入来源: 业务组件通过 props/slots 传入文案、状态、图标名和操作区域。
- 输出结果: 平台统一但可被 macOS/Windows/Linux token 覆盖的 UI 元素。

## 4. 关键实现细节
- `icons.ts` 统一把产品语义图标名映射到 `lucide-vue-next`，避免业务页面继续散落 Ionicons 依赖。
- `SxInspectorSegmented` 是主布局 inspector 模式切换控件，当前模式固定为 `runtime | workspace | terminal | tasks`。
- `SxStatusBadge` 只表达语义状态，不承载业务文案推导。
- `SxEmptyState` 支持 left/center 对齐，避免不同页面空状态自行设计。
- `SxRuntimeTaskRow`、`SxTerminalRow` 分别服务任务 inspector 和 terminal fallback 输出。

## 5. 依赖关系
- 内部依赖: `frontend/src/style.css` 的 `--sx-*` tokens。
- 外部依赖: `lucide-vue-next`。

## 6. 变更影响面
- 新 UI 应优先使用该组件库；迁移期允许旧 Naive UI 和 `@vicons/ionicons5` 继续存在。
- 若 Figma v0.5 component API 变更，应先更新本目录组件，再迁移业务页面。

## 7. 维护建议
- 不在业务页面重新定义同类控件的 hover/active/focus 样式。
- 新增图标先进入 `icons.ts` 的语义映射，再在组件中引用。
- 组件 props 应保持业务中立，避免出现内部 tool canonical name 或后端字段名。
