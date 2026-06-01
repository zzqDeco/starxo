# useNativeTextInputSync.ts 技术说明

## 文件定位
- 源文件: `frontend/src/composables/useNativeTextInputSync.ts`
- 为表单外壳内的真实 `input/textarea` 提供 focus 和 DOM 值同步。

## 核心职责
- 暴露 `inputRef` 与 `rootRef`，用于定位组件内层真实文本控件。
- 在 focus、input/change/key/paste 和短轮询期间把 DOM value 同步回 Vue ref。
- 支持 Computer Use、辅助技术和真实键盘/粘贴路径，避免 DOM 值变化但 `v-model` 未更新导致按钮仍 disabled。

## 维护要点
- 仅用于文本输入控件，不承担表单校验职责。
- 轮询在组件挂载期间启用，组件卸载时必须清理。
