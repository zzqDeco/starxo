# SplitHandle.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/layout/SplitHandle.vue
- 文档文件: doc/src/frontend/src/components/layout/SplitHandle.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/layout (布局模块)

## 2. 核心职责
- 通用可拖拽面板分割条组件，用于调整相邻面板的尺寸。
- 支持水平和垂直方向、localStorage 持久化、双击恢复默认尺寸。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: Props (direction, minSize, maxSize, defaultSize, storageKey, reverse)
- 输出结果: 发射 `update:size` 事件通知父组件新尺寸

## 4. 关键实现细节
- **Props**:
  - `direction: 'horizontal' | 'vertical'` — 拖拽方向（默认 horizontal）
  - `minSize: number` — 最小像素值（默认 180）
  - `maxSize: number` — 最大像素值（默认 600）
  - `defaultSize: number` — 默认尺寸（必传）
  - `storageKey?: string` — localStorage 持久化键名
  - `reverse: boolean` — 是否反向计算（默认 false，用于右侧面板）
- **Emits**: `update:size(value: number)` — 尺寸变化事件
- **拖拽实现**:
  - mousedown 记录起始位置和起始尺寸，添加 document 级 mousemove/mouseup 监听
  - mousemove 计算 delta，clamp 到 [minSize, maxSize] 范围
  - mouseup 移除监听，持久化到 localStorage
  - 拖拽时 body 设置 `cursor: col-resize/row-resize` + `user-select: none`
- **双击**: 恢复 defaultSize 并持久化
- **初始化**: 从 localStorage 恢复已保存的尺寸，范围校验后发射 update:size
- **样式**:
  - 4px 宽透明触发区域（通过 CSS 变量 `--splitter-width`）
  - hover/dragging 时显示背景色（`--splitter-hover`）和 2px 居中高亮线（`--splitter-active`）
  - z-index 使用 `--z-splitter`（默认 10）

## 5. 依赖关系
- 内部依赖: 无
- 外部依赖: `vue` (ref, computed, onUnmounted)

## 6. 变更影响面
- 被 MainLayout.vue 使用（左侧和右侧分割条）
- CSS 变量来自全局 style.css（--splitter-width, --splitter-hover, --splitter-active, --z-splitter）

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- reverse 属性用于右侧面板（鼠标右移时面板缩小），确保新增使用场景时方向逻辑正确。
- onUnmounted 清理 document 事件监听和 body 样式，防止内存泄漏。
