# useWailsEvent.ts 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/composables/useWailsEvent.ts
- 文档文件: doc/src/frontend/src/composables/useWailsEvent.ts.plan.md
- 文件类型: TypeScript 源码
- 所属模块: frontend/src/composables (组合式函数)

## 2. 核心职责
- 提供 Vue 组合式函数，封装 Wails 事件的生命周期管理（自动注册和注销）。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: eventName (事件名称), handler (事件处理函数)
- 输出结果: 在 onMounted 时注册事件监听，在 onUnmounted 时自动注销

## 4. 关键实现细节
- **泛型支持**: `useWailsEvent<T>` 允许指定事件数据类型
- **生命周期管理**:
  - `onMounted` → `EventsOn(eventName, handler)` 注册监听
  - `onUnmounted` → `EventsOff(eventName)` 注销监听
- **注意**: EventsOff 只传入事件名，会注销该事件的所有监听器

## 5. 依赖关系
- 内部依赖: 无
- 外部依赖: `vue` (onMounted, onUnmounted)
- Wails 绑定: `wailsjs/runtime/runtime` (EventsOn, EventsOff)

## 6. 变更影响面
- 被 TerminalPanel 和 FileExplorer 使用
- 注销行为修改影响所有使用此 composable 的组件

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 当前 EventsOff 会移除同名事件的所有监听器，如果多个组件监听同一事件可能产生冲突。需要时可改用 EventsOff(eventName, handler) 精确注销。
