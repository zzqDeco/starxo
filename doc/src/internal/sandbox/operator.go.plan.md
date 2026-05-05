# operator.go 技术说明

## 文件定位
- 源文件: `internal/sandbox/operator.go`
- 实现 Eino `commandline.Operator`。

## 核心职责
- 将 Agent 文件和命令操作委托到当前轻量沙箱 runtime。
- `RunCommand` 调用 `RemoteRuntimeManager.ExecInSandbox`，并把输出转发给前端终端。
- `ReadFile`、`WriteFile`、`Exists`、`IsDirectory` 都限制在当前 sandbox workspace 内。

## 路径规则
- 相对路径映射到当前 workspace。
- 旧 `/workspace/...` 路径映射到当前真实 workspace。
- workspace 外绝对路径直接拒绝。

## 维护要点
- 写文件使用 base64 经 runtime shell 写入，避免普通 shell 字符串破坏内容。
- 路径守卫是安全关键点，新增文件操作必须复用同一规则。
