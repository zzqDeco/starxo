# ConnectionStatus.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/status/ConnectionStatus.vue
- 文档文件: doc/src/frontend/src/components/status/ConnectionStatus.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/status (状态指示模块)

## 2. 核心职责
- 连接状态指示器组件，以药丸形状 (pill) 显示 SSH 和 Docker 的连接状态，并在连接中时显示进度文本。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: connectionStore 状态 (sshConnected, dockerRunning, connecting, initStep, containerID)
- 输出结果: 渲染状态指示器 UI

## 4. 关键实现细节
- **Pinia Store 交互**: connectionStore — sshConnected, dockerRunning, connecting, initStep, containerID
- **模板结构**:
  - SSH 状态药丸: 绿色/红色状态点 + "SSH" 标签，NTooltip 显示详细状态
  - Docker 状态药丸: 绿色/红色状态点 + "Docker" 标签，NTooltip 显示详细状态和容器 ID (前12位)
  - 连接中文本: connecting 时显示 initStep，带 pulse 动画
- **状态点样式**: dot-green (绿色发光) / dot-red (红色发光)

## 5. 依赖关系
- 内部依赖: `@/stores/connectionStore`
- 外部依赖: `naive-ui` (NTooltip)、`vue-i18n` (useI18n)

## 6. 变更影响面
- 被 Header 组件包含，位于顶部导航栏中央
- 状态点样式与 Sidebar 中的 dot 系统共享

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 状态点 CSS 类 (dot-green, dot-red) 在组件内定义，与 Sidebar 中的 dot 系统有重复，可考虑提取到全局样式。
