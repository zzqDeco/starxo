# connectionStore.ts 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/stores/connectionStore.ts
- 文档文件: doc/src/frontend/src/stores/connectionStore.ts.plan.md
- 文件类型: TypeScript 源码
- 所属模块: frontend/src/stores (Pinia 状态管理)

## 2. 核心职责
- 管理沙盒连接状态（SSH + Docker），封装连接/断开操作与 Go 后端 SandboxService 的交互。
- 跟踪连接进度、错误信息和就绪状态。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: Sidebar.vue 用户操作（连接/断开按钮）、App.vue 的 Wails 事件（sandbox:progress, sandbox:ready, sandbox:disconnected）
- 输出结果: 响应式连接状态供 Sidebar、Header (ConnectionStatus)、ChatPanel 等组件消费

## 4. 关键实现细节
- **State 属性**:
  - `sshConnected: boolean` — SSH 连接状态
  - `dockerRunning: boolean` — Docker 容器运行状态
  - `containerID: string` — 当前容器 ID
  - `initProgress: number` — 初始化进度百分比
  - `initStep: string` — 当前初始化步骤描述
  - `connecting: boolean` — 是否正在连接中
  - `error: string` — 错误信息
- **Getters**:
  - `isReady` — SSH 已连接且 Docker 运行中
  - `statusText` — 状态文本描述（Connecting / Ready / Disconnected 等）
- **Actions**:
  - `connect()` — 连接沙盒：先自动保存设置（调用 settingsStore.saveSettings），再调用 SandboxConnect，最后刷新状态
  - `disconnect()` — 断开沙盒并重置所有状态
  - `refreshStatus()` — 从后端获取最新状态（SSH/Docker/容器ID）
  - `updateProgress(step, percent)` — 更新初始化进度（由 sandbox:progress 事件触发）
  - `setReady()` — 标记沙盒就绪状态（由 sandbox:ready 事件触发）

## 5. 依赖关系
- 内部依赖: `./settingsStore` (useSettingsStore)
- 外部依赖: `pinia` (defineStore)、`vue` (ref, computed)
- Wails 绑定: `wailsjs/go/service/SandboxService` (Connect, Disconnect, GetStatus)

## 6. 变更影响面
- 连接状态影响 Sidebar 底部状态条、Header 的 ConnectionStatus 组件
- `isReady` 状态影响 ChatPanel 空状态提示文案
- 连接流程变更需同步 Go 后端 SandboxService

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- `connect()` 方法依赖 settingsStore，修改设置保存逻辑时注意连接流程的影响。
- 新增连接状态字段时需同步更新 Sidebar 和 ConnectionStatus 的展示逻辑。
