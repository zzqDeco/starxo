# 004 - SSH 连接保活

## 目标

防止 SSH 空闲断连，提高连接稳定性。

## 范围

- `internal/sandbox/ssh.go`

## 方案

1. 在 SSH 连接建立后启动后台 goroutine
2. 每 30 秒发送 `SendRequest("keepalive@openssh.com", true, nil)`
3. 连续 3 次失败触发自动重连
4. 使用 `context.Context` 控制 goroutine 生命周期

## 具体任务

- [ ] 在 `SSHClient` 结构体中添加 `cancelKeepalive context.CancelFunc` 字段
- [ ] 实现 `startKeepalive(ctx context.Context)` 方法：30 秒间隔发送 keepalive 请求
- [ ] 在 `Connect()` 成功后启动 keepalive goroutine
- [ ] 在 `Disconnect()` 中通过 context cancel 停止 keepalive
- [ ] 连续 3 次 keepalive 失败时调用重连回调（通过 Wails 事件通知前端）

## 涉及文件

- `internal/sandbox/ssh.go`（修改 SSHClient 结构体、Connect、Disconnect 方法）

## 预估时间

0.5 天

## 状态

待实施
