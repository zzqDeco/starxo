# sandbox_health_test.go 技术说明

## 文件定位
- 源文件: `internal/service/sandbox_health_test.go`
- 覆盖 `SandboxService` health monitor 的分层断连逻辑。

## 覆盖范围
- 单次 raw SSH probe 失败不会清空 manager 或 active sandbox。
- 连续 SSH probe 失败达到阈值后才触发断连清理和 deactivation callback。
- SSH 仍可用但 sandbox workspace 缺失时，只停用 active sandbox，不断开 SSH manager。
- stale generation 的 health goroutine 不会清理当前新连接。
- 同一 SSH manager 下 active sandbox 已切换时，旧 sandbox probe 返回失败不会停用新的 active sandbox。

## 维护要点
- 测试通过注入 `healthSSHProbe` / `healthSandboxProbe` 避免真实 SSH 依赖。
- 新增 health cleanup 分支时，应同步增加 generation/race 相关用例。
