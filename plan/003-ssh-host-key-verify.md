# 003 - SSH Host Key 验证

## 目标

替换 `InsecureIgnoreHostKey()`，实现 Trust on First Use (TOFU) 模式的 SSH 主机密钥验证。

## 范围

- `internal/sandbox/ssh.go`

## 方案

1. 首次连接时保存主机指纹到 `~/.starxo/known_hosts`
2. 后续连接时验证指纹匹配
3. 指纹变更时中断连接并通过前端警告用户

## 具体任务

- [ ] 创建 `internal/sandbox/hostkeys.go`: 实现 LoadKnownHosts/SaveHostKey/VerifyHostKey 函数
- [ ] 修改 `ssh.go` 的 `Connect()`: 将 `InsecureIgnoreHostKey()` 替换为自定义 HostKeyCallback
- [ ] 新增 Wails 事件 `sandbox:host_key_changed` 用于前端警告
- [ ] 前端: 在 ConnectionStatus 组件中显示 host key 变更警告弹窗
- [ ] 测试: 首次连接保存、匹配通过、指纹变更拒绝

## 涉及文件

- `internal/sandbox/hostkeys.go`（新建）
- `internal/sandbox/ssh.go`（修改 Connect 方法）
- `frontend/src/components/sandbox/ConnectionStatus.vue`（添加警告 UI）
- `internal/sandbox/hostkeys_test.go`（新建）

## 预估时间

1 天

## 状态

待实施
