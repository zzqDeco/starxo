# 008 - 操作审计日志

## 目标

记录 Agent 关键操作的结构化审计日志，用于安全审计和问题排查。

## 范围

- `internal/service/chat.go`
- `internal/sandbox/operator.go`

## 方案

1. 创建 `internal/logger/audit.go`，独立的审计日志写入器
2. 输出 JSON Lines 格式到 `~/.starxo/audit/audit-YYYY-MM-DD.jsonl`
3. 记录事件类型: `command.exec`、`file.write`、`file.read`、`agent.transfer`、`llm.request`、`ssh.connect`、`container.create`
4. 每条记录包含: timestamp、event_type、session_id、details、duration_ms
5. 30 天自动轮转（启动时清理过期文件）

## 具体任务

- [ ] 创建 `internal/logger/audit.go`: AuditLogger 结构体、LogEvent 方法、日志文件按日期轮转
- [ ] 定义 AuditEvent 结构体: Timestamp、EventType、SessionID、Details(map)、DurationMs
- [ ] 在 `chat.go` 的 SendMessage 中记录 `llm.request` 事件（模型名、token 数）
- [ ] 在 `operator.go` 的 Execute 中记录 `command.exec` 事件（命令内容、执行时长）
- [ ] 在 `transfer.go` 的上传/下载中记录 `file.write`/`file.read` 事件（文件路径、大小）
- [ ] 启动时调用 CleanOldLogs() 清理 30 天前的审计日志文件

## 涉及文件

- `internal/logger/audit.go`（新建）
- `internal/logger/audit_test.go`（新建）
- `internal/service/chat.go`（修改：添加审计日志调用）
- `internal/sandbox/operator.go`（修改：添加审计日志调用）
- `internal/sandbox/transfer.go`（修改：添加审计日志调用）
- `app.go`（修改：初始化 AuditLogger，启动时清理旧日志）

## 预估时间

1 天

## 状态

待实施
