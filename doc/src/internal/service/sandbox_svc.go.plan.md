# sandbox_svc.go 技术说明

## 文件定位
- 源文件: `internal/service/sandbox_svc.go`
- Wails 绑定的沙箱生命周期服务。

## 核心职责
- `ConnectSSH` 建立 SSH 并检测轻量 sandbox runtime。
- `CreateAndActivateContainer` 保留旧方法名，但实际创建并激活 sandbox workspace。
- `ActivateContainer` 通过 runtime ID 和 workspacePath 激活已有 sandbox。
- `DeactivateContainer` 只清除当前 operator/runtime 激活状态，不删除 workspace。
- `GetStatus` 同时返回新 `runtimeAvailable/sandboxActive/activeSandbox*` 字段和旧 Docker/Container 兼容字段。

## 维护要点
- 旧 Docker 记录状态为 `unavailable` 时禁止激活。
- 事件名暂时保留 `container:*` 以兼容前端监听。
