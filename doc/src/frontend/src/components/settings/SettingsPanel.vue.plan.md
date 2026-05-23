# SettingsPanel.vue 技术说明

## 文件定位
- 源文件: `frontend/src/components/settings/SettingsPanel.vue`
- 设置弹窗入口。

## 核心职责
- 设置分区包含 SSH、Sandbox、LLM、WebSearch、LSP、Permissions、MCP。
- Sandbox 分区加载 `SandboxConfig.vue`，替代旧 Docker 设置页。
- Permissions 分区加载 `RuntimePermissionsPanel.vue`，用于查看 pending tool approvals 和当前 session 的持久授权。
- WebSearch 分区加载 `WebSearchConfig.vue`，用于配置 TinyFish/Search provider 并运行静态诊断。
- LSP 分区加载 `LSPConfig.vue`，用于配置常驻 language server 和查看当前 session 的 LSP 状态。
- 保存时通过 `settingsStore.saveSettings` 写回后端并触发 runner 缓存失效。
