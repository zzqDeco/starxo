# useNativeWindow.ts 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/composables/useNativeWindow.ts
- 文档文件: doc/src/frontend/src/composables/useNativeWindow.ts.plan.md
- 文件类型: TypeScript composable
- 所属模块: frontend/src/composables

## 2. 核心职责
- 封装自定义 toolbar 或非原生 titlebar 区域的窗口 zoom 行为。
- 优先调用 Wails runtime 的 `WindowToggleMaximise` / `WindowUnmaximise`，与平台窗口管理保持一致。
- 在 runtime toggle 失败时使用窗口尺寸/位置 API 做 best-effort fallback。

## 3. 输入与输出
- 输入来源: 用户双击 Header、Sidebar 顶部命中区或非 macOS 顶部命中区。
- 输出结果: 切换窗口最大化/还原状态。

## 4. 关键实现细节
- `toggleWindowZoom()`:
  - 使用 `WindowIsMaximised()` 判断当前状态。
  - 已最大化时调用 `WindowUnmaximise()`。
  - 未最大化时调用 `WindowToggleMaximise()`。
  - 发生异常时进入 `fallbackToggleWindowZoom()`。
- fallback:
  - 根据 `window.screen.availWidth/availHeight` 计算可用屏幕大小。
  - 记录还原前 `restoreBounds`。
  - 设置位置和尺寸前先调用 `WindowUnmaximise()`，避免在最大化状态下 size/position 被平台忽略。
- `toggleInFlight` 防止快速双击/重复事件导致状态抖动。

## 5. 依赖关系
- 内部依赖: 无
- 外部依赖: `frontend/wailsjs/runtime/runtime`

## 6. 变更影响面
- 影响 Header、Sidebar、MainLayout 中自定义区域的双击 zoom 行为。
- macOS 顶部边框和系统标题栏行为仍由 native titlebar 处理，该 composable 只作为应用内 fallback。

## 7. 维护建议
- 不要把该 composable 当作原生标题栏替代品；优先使用平台 titlebar 提供系统级窗口行为。
- 修改窗口尺寸 fallback 后需用打包后的 macOS app 验证，不只验证浏览器或 Vite dev server。
