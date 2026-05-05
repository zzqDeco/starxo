# file_svc.go 技术说明

## 文件定位
- 源文件: `internal/service/file_svc.go`
- Wails 绑定的本地文件上传、下载、预览服务。

## 核心职责
- 通过 `SandboxService.Manager()` 获取当前 sandbox manager。
- 上传/下载使用 `FileTransfer` 直接 SFTP 到当前 sandbox workspace。
- 文件列表和预览通过 `RemoteOperator` 在 sandbox runtime 内执行。

## 维护要点
- 旧变量名 `docker` 仅是兼容过渡，本质是 runtime manager。
- `workspacePath` 来自当前 session，旧 `/workspace` 会由 transfer/operator 映射到真实 workspace。
