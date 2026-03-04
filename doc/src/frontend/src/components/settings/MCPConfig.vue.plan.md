# MCPConfig.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/settings/MCPConfig.vue
- 文档文件: doc/src/frontend/src/components/settings/MCPConfig.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/settings (设置模块)

## 2. 核心职责
- MCP 服务器配置子表单组件，提供 MCP 服务器的添加、编辑和删除功能。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: settingsStore.settings.mcp 配置数据
- 输出结果: 通过 settingsStore 的 addMCPServer/removeMCPServer 方法修改 MCP 服务器列表

## 4. 关键实现细节
- **Pinia Store 交互**: settingsStore — addMCPServer, removeMCPServer, settings.mcp.servers
- **删除确认**: 使用内联确认按钮（confirmingDelete ref 状态）替代 NPopconfirm，因 NPopconfirm 的 teleported popover 在 Wails WebView2 中导致点击阻塞问题
- **防御性编程**: mcp.servers 访问使用 `|| []` 空数组回退，防止 null/undefined 导致渲染错误

## 5. 依赖关系
- 内部依赖: `@/stores/settingsStore`
- 外部依赖: `vue` (ref)、`naive-ui` (表单组件)、`vue-i18n` (useI18n)

## 6. 变更影响面
- MCP 服务器列表的增删操作影响 settingsStore 持久化
- 被 SettingsPanel.vue 作为 MCP Tab 的子表单组件渲染
- 删除确认的交互方式变更影响用户体验

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 避免使用 NPopconfirm 等 teleport 到 body 的 Naive UI 组件（WebView2 兼容性问题）。
- mcp.servers 的 null 防御检查不可移除，后端可能返回 null 而非空数组。
