# ContainerPanel.vue 技术说明

## 文件定位
- 源文件: `frontend/src/components/containers/ContainerPanel.vue`
- 右侧 runtime dock 的 sandbox 列表面板。组件名保留 ContainerPanel 以兼容现有结构。

## 核心职责
- 展示当前会话和其他会话的已注册 sandbox。
- 通过现有 Wails `ContainerService` 绑定创建、激活、停用、刷新和销毁 sandbox。
- UI 文案改为沙箱，状态支持 `unavailable` 以展示旧 Docker 记录。
- macOS native 设计语言下使用 inspector 列表行：透明背景、hairline 分隔、系统灰 hover/selected，不使用彩色卡片或大面积状态色。

## 维护要点
- 后端服务名和事件名仍有 container 兼容命名，前端用户文案应保持 sandbox。
