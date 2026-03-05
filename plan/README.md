# Starxo 开发计划索引

## 状态图例

| 标记 | 含义 |
|------|------|
| 待实施 | 尚未开始 |
| 进行中 | 正在实施 |
| 已完成 | 已完成并验证 |

## 计划文档

| 编号 | 文件 | 描述 | 预估 | 状态 |
|------|------|------|------|------|
| 001 | [unit-tests-core](001-unit-tests-core.md) | Go 核心包单元测试，建立测试基础 | 1 天 | 待实施 |
| 002 | [credential-encryption](002-credential-encryption.md) | 凭证加密存储，替换明文敏感字段 | 1-2 天 | 待实施 |
| 003 | [ssh-host-key-verify](003-ssh-host-key-verify.md) | SSH Host Key 验证，实现 TOFU | 1 天 | 待实施 |
| 004 | [ssh-keepalive](004-ssh-keepalive.md) | SSH 连接保活，防止空闲断连 | 0.5 天 | 待实施 |
| 005 | [checkpoint-persistence](005-checkpoint-persistence.md) | 检查点持久化，防止重启丢失 | 1 天 | 待实施 |
| 006 | [keyboard-shortcuts](006-keyboard-shortcuts.md) | 键盘快捷键，提升操作效率 | 0.5 天 | 待实施 |
| 007 | [session-search](007-session-search.md) | 会话搜索，侧边栏搜索过滤 | 0.5 天 | 待实施 |
| 008 | [audit-log](008-audit-log.md) | 操作审计日志，记录关键操作 | 1 天 | 待实施 |
| 009 | [event-batching](009-event-batching.md) | 事件流批处理，减少渲染压力 | 1 天 | 待实施 |
| 010 | [conversation-limits](010-conversation-limits.md) | 对话硬限制，防止内存泄漏 | 0.5 天 | 待实施 |
| 011 | [session-container-hierarchy](011-session-container-hierarchy.md) | 会话与容器父子关系重构 | 1-2 天 | 已完成 |
| 012 | [agent-mode-and-tool-error-recovery](012-agent-mode-and-tool-error-recovery.md) | Plan 模式编排约束与工具错误可恢复执行 | 1-2 天 | 已完成 |
