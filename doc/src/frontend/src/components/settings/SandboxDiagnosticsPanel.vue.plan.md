# SandboxDiagnosticsPanel.vue 技术说明

## 文件定位
- 源文件: `frontend/src/components/settings/SandboxDiagnosticsPanel.vue`
- 设置页中的沙箱运行时诊断面板。

## 核心职责
- 调用 `SettingsService.DiagnoseSandboxRuntime` 显示远端 runtime 检查项。
- 调用 `SettingsService.InstallSandboxRuntime` 安装普通 Linux 依赖后自动重跑诊断。
- 展示 fix guide；sudo/sysctl/AppArmor 等命令只提供复制按钮，不在前端执行。

## 维护要点
- 面板直接读取当前 settings store，支持保存前诊断。
- 检查项状态使用 `pass/warn/fail/info/skipped`，新增状态时需要同步标签样式。
- 修复建议以 backend DTO 为准，前端不拼接或猜测命令。
