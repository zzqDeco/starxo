# sandbox_svc.go 技术说明

## 文件定位
- 源文件: `internal/service/sandbox_svc.go`
- Wails 绑定的沙箱生命周期服务。

## 核心职责
- `ConnectSSH` 建立 SSH 并检测轻量 sandbox runtime。
- `CreateAndActivateContainer` 保留旧方法名，但实际创建并激活 sandbox workspace。
- `ActivateContainer` 通过 runtime ID 和 workspacePath 激活已有 sandbox。
- `DeactivateContainer` 只清除当前 operator/runtime 激活状态，不删除 workspace。
- `destroyActiveSandbox` 删除当前 active sandbox workspace 并停用 sandbox，但保留 SSH manager 和 health monitor。
- `RunTerminalCommand` 在当前 active sandbox workspace 中执行用户提交的非交互式 shell 命令。
  - 命令成功退出后发出带 active sandbox registry ID、source/action 和时间戳的 `workspace:changed`，让 workspace inspector 自动刷新 terminal 产物。
  - 执行前必须校验 active session 绑定的 sandbox 与当前 active sandbox 一致；未绑定会话不能复用旧会话 sandbox。
- `GetStatus` 同时返回新 `runtimeAvailable/sandboxActive/activeSandbox*` 字段和旧 Docker/Container 兼容字段。
- `StartHealthMonitor`/`healthCheck` 负责保守的远端健康监控：
  - 先用 raw SSH `echo ping` 判断 SSH liveness。
  - 单次 SSH probe 失败只记录日志，连续失败达到阈值后才发出 `ssh:disconnected`。
  - SSH 可用但 active sandbox workspace 不存在时，只发出 `container:deactivated`，保留 SSH manager。

## 维护要点
- 旧 Docker 记录状态为 `unavailable` 时禁止激活。
- 事件名暂时保留 `container:*` 以兼容前端监听。
- `DisconnectAndDestroy` 是显式断连销毁语义，会关闭 SSH 并发出断开/停用事件；普通 sandbox 销毁不得复用它。
- active sandbox 远端删除失败时必须保留 active 状态，避免 UI 误以为数据已删除。
- 手动断开、重连、销毁和 health failure 必须走统一 helper，确保 health monitor、active sandbox、Wails 事件和 `onContainerDeactivated` callback 不分叉。
- Health monitor 使用 generation guard；旧 goroutine 返回时不得清理新 manager。
- 停用 active sandbox 时必须在 `SandboxService` 状态锁内调用 `mgr.DetachContainer()`，避免解锁后并发激活新 sandbox 被旧停用流程误拆。
