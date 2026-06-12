# mac_local_network_diagnostics.go 技术说明

## 文件定位
- 源文件: `internal/service/mac_local_network_diagnostics.go`
- Wails `SettingsService` 的 macOS Local Network 诊断辅助。

## 核心职责
- `CheckMacLocalNetworkAccess(sshCfg)` 对当前 SSH host/port 做 app 进程 TCP dial 诊断。
- 仅在 macOS + 私网/localhost/mDNS host 场景下分类 Local Network 权限风险。
- 可选调用本机 `nc` 做 CLI reachability 对比；CLI 可达但 app 不可达时标记为权限或 bundle identity 缓存问题。

## 维护要点
- 不读取、不记录、不返回 SSH 密码或私钥。
- 不自动运行 `tccutil`，只返回 copy-only reset command。
- 该诊断只解释 app 进程到 host/port 的可达性，不替代 SSH 认证测试。
