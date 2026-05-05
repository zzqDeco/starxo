# container_svc.go 技术说明

## 文件定位
- 源文件: `internal/service/container_svc.go`
- Wails 绑定的 sandbox registry 操作服务。服务名保留 ContainerService 以兼容现有前端绑定。

## 核心职责
- `ListContainers` 返回已注册 sandbox 列表。
- `RefreshContainerStatus` 使用 runtime workspace 检查状态。
- `StartContainer`/`ActivateContainer` 激活已有 sandbox。
- `StopContainer`/`DeactivateContainer` 停用当前 sandbox。
- `DestroyContainer` 删除 registry，当前连接可访问时同步删除远端 workspace。

## 维护要点
- 旧 Docker 不再启动/停止/销毁，只作为 `unavailable` 记录展示和删除。
