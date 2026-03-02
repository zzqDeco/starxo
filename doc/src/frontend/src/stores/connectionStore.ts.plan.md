# connectionStore.ts 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/stores/connectionStore.ts
- 文档文件: doc/src/frontend/src/stores/connectionStore.ts.plan.md
- 文件类型: TypeScript 源码
- 所属模块: frontend/src/stores (Pinia 状态管理)

## 2. 核心职责
- **仅管理 SSH 连接状态**，封装 SSH 连接/断开操作与 Go 后端 SandboxService 的交互。容器管理已移至 containerStore。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: Sidebar.vue 用户操作（连接/断开按钮）、App.vue 的 Wails 事件（`ssh:progress`, `ssh:connected`, `ssh:disconnected`）
- 输出结果: 响应式 SSH 连接状态供 Sidebar、Header (ConnectionStatus)、ChatPanel、ContainerPanel 等组件消费

## 4. 关键实现细节
- **State 属性**:
  - `sshConnected: boolean` — SSH 连接状态
  - `initProgress: number` — 初始化进度百分比
  - `initStep: string` — 当前初始化步骤描述
  - `connecting: boolean` — 是否正在连接中
  - `error: string` — 错误信息
- **Getters**:
  - `isReady` — 等同于 `sshConnected`（SSH 连接即就绪）
  - `statusText` — 状态文本描述（Connecting / SSH Connected / Disconnected）
- **Actions**:
  - `connect()` — 连接 SSH：先自动保存设置，再调用 `ConnectSSH()`，最后刷新状态
  - `disconnect()` — 断开 SSH 并重置状态，调用 `DisconnectSSH()`
  - `refreshStatus()` — 从后端获取最新 SSH 状态
  - `updateProgress(step, percent)` — 更新连接进度（由 `ssh:progress` 事件触发）
  - `setSSHConnected()` — 标记 SSH 已连接（由 `ssh:connected` 事件触发）
  - `setSSHDisconnected()` — 标记 SSH 已断开（由 `ssh:disconnected` 事件触发）

## 5. 依赖关系
- 内部依赖: `./settingsStore` (useSettingsStore)
- 外部依赖: `pinia` (defineStore)、`vue` (ref, computed)
- Wails 绑定: `wailsjs/go/service/SandboxService` (ConnectSSH, DisconnectSSH, GetStatus)

## 6. 变更影响面
- SSH 状态影响 Sidebar 底部状态条、Header 的 ConnectionStatus 组件
- `sshConnected` 状态影响 ContainerPanel 的"新建容器"按钮启用状态
- `isReady` 状态影响 ChatPanel 空状态提示文案

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 已移除 `dockerRunning` 和 `containerID` 状态，容器相关状态在 `containerStore` 管理。
- 事件名从 `sandbox:*` 迁移到 `ssh:*`，修改时需与后端 SandboxService 保持一致。
