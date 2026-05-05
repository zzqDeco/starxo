# SandboxConfig.vue 技术说明

## 文件定位
- 源文件: `frontend/src/components/settings/SandboxConfig.vue`
- 替代旧 `DockerConfig.vue`，用于配置远端轻量沙箱运行时。

## 核心职责
- 编辑 `settings.sandbox`：runtime、rootDir、workDirName、网络、内存、命令超时、Python 初始化和包列表。
- 嵌入 `SandboxDiagnosticsPanel`，负责运行诊断和安装普通 runtime 依赖。

## 维护要点
- 该组件不再暴露 Docker image/cpu/container 配置。
- 诊断和安装使用当前表单中的 SSH + sandbox 配置，允许保存前验证。
