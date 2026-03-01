# SettingsPanel.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/settings/SettingsPanel.vue
- 文档文件: doc/src/frontend/src/components/settings/SettingsPanel.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/settings (设置模块)

## 2. 核心职责
- 设置面板模态组件，以 Tab 页形式组织 SSH、Docker、LLM、MCP 四类配置表单。
- 提供保存、取消和恢复默认操作。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: Props (show: boolean)
- 输出结果: Emits (update:show)；调用 settingsStore.saveSettings 持久化配置

## 4. 关键实现细节
- **Props/Emits**: 使用 `v-model:show` 模式控制模态可见性
- **Pinia Store 交互**: settingsStore — saveSettings, resetToDefaults
- **Tab 页结构**: ssh / docker / llm / mcp，各 Tab 渲染对应的子表单组件
- **子表单组件**: SSHConfigForm (SSHConfig.vue)、DockerConfigForm (DockerConfig.vue)、LLMConfigForm (LLMConfig.vue)、MCPConfigForm (MCPConfig.vue)
- **Footer 操作**: 恢复默认 (resetToDefaults) | 取消 (关闭模态) | 保存 (saveSettings + 关闭模态)
- **模态配置**: NModal + NCard，600px 宽度，支持遮罩点击和 ESC 关闭

## 5. 依赖关系
- 内部依赖: `@/stores/settingsStore`、`./SSHConfig.vue`、`./DockerConfig.vue`、`./LLMConfig.vue`、`./MCPConfig.vue`
- 外部依赖: `vue` (ref)、`naive-ui` (NModal, NCard, NTabs, NTabPane, NButton, NIcon)、`@vicons/ionicons5` (Close)、`vue-i18n` (useI18n)

## 6. 变更影响面
- 新增配置 Tab 需添加子表单组件和 Tab 页
- 保存逻辑修改影响 settingsStore 持久化
- 被 MainLayout 通过 v-model:show 控制

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 子表单组件 (SSHConfig, DockerConfig, LLMConfig, MCPConfig) 直接绑定 settingsStore.settings 对象，无需额外的 props 传递。
- 新增设置类别时需同步创建子表单组件和 AppSettings 类型扩展。
