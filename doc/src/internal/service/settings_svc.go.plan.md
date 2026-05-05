# settings_svc.go 技术说明

## 文件定位
- 源文件: `internal/service/settings_svc.go`
- Wails 绑定的设置服务。

## 核心职责
- `GetSettings` / `SaveSettings` 管理 AppConfig。
- `TestSSHConnection` 和 `TestLLMConnection` 保持原有职责。
- `CheckSandboxRuntime(cfg)` 用传入的 SSH + sandbox 配置临时连接远端并检测 bwrap/Seatbelt。
- `InstallSandboxRuntime(cfg)` 用传入配置临时连接远端，Linux 上显式安装 bubblewrap/Python 依赖。

## 维护要点
- 检测/安装不依赖当前已连接 SandboxService，适合设置保存前验证。
