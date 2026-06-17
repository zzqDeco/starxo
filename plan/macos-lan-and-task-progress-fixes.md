# macOS Local Network 与任务进度文案修复

## Summary
- 修复 macOS app bundle 启动路径访问局域网 SSH 时的 Local Network 诊断缺口。
- 修复 inline task status 在部分运行路径下显示 `{done}/{total}` 占位符的问题。
- 保持 bundle identifier 为 `com.starxo.app`，不使用 `com.wails.starxo`。

## Implementation
- `SettingsService.CheckMacLocalNetworkAccess` 只做 host/port 级 TCP 诊断，不读取或输出 SSH 密码、私钥。
- macOS + 私网 host + app dial 出现 `no route to host`、`network is unreachable`、`operation not permitted` 或 `permission denied` 时，返回 Local Network 权限/旧 bundle identity 缓存的可读诊断。
- 前端 SSH 错误区和 SSH 设置页显示本地化修复卡片，提供系统设置、重开 app、新用户/VM snapshot 复测建议，不提供不可靠的 `tccutil` reset 命令。
- `scripts/sign-darwin-app.sh` 和 release workflow 校验渲染后的 app plist 必须包含 `NSLocalNetworkUsageDescription` 与 `NSAppTransportSecurity.NSAllowsLocalNetworking=true`。
- 用户可见的 task/line/count 文案走 `formatNamedMessage`，当 vue-i18n 未插值时做显示层 fallback。

## Validation
- `go test ./...`
- `cd frontend && npm run build`
- `/Users/zhaoziqian/go/bin/wails build -skipbindings -trimpath`
- `scripts/sign-darwin-app.sh`
- macOS app 通过 `open build/bin/starxo.app` 启动后验证局域网 SSH 错误能展示 Local Network 修复卡片。
