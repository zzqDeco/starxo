# SettingsPanel.vue 技术说明

## 文件定位
- 源文件: `frontend/src/components/settings/SettingsPanel.vue`
- 设置弹窗入口。

## 核心职责
- 设置分区包含 SSH、Sandbox、LLM、MCP。
- Sandbox 分区加载 `SandboxConfig.vue`，替代旧 Docker 设置页。
- 保存时通过 `settingsStore.saveSettings` 写回后端并触发 runner 缓存失效。
