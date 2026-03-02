# 010 - 对话硬限制

## 目标

防止长会话内存泄漏，对后端对话历史和前端 timeline 事件设置上限。

## 范围

- `internal/context/history.go`
- `frontend/src/stores/chatStore.ts`

## 方案

1. 后端: ConversationHistory 添加 maxMessages 硬限制（默认 200），超出时丢弃最早消息
2. 前端: chatStore 中 timeline events 添加 maxEvents 限制（默认 500），超出时裁剪
3. 限制值可通过 AgentConfig 配置

## 具体任务

- [ ] 修改 `history.go`: AddMessage 时检查消息总数，超出 maxMessages 时移除头部最早消息
- [ ] 在 ConversationHistory 结构体中添加 `maxMessages int` 字段，构造时从配置读取
- [ ] 修改 `chatStore.ts`: addTimelineEvent 时检查事件总数，超出 maxEvents 时裁剪数组头部
- [ ] 在 AgentConfig 中添加 `maxMessages` 和 `maxTimelineEvents` 配置项（含默认值）
- [ ] 添加单元测试: 验证超出限制时正确裁剪、边界值测试

## 涉及文件

- `internal/context/history.go`（修改：添加上限检查）
- `frontend/src/stores/chatStore.ts`（修改：添加 timeline 上限）
- `internal/config/config.go`（修改：添加 maxMessages/maxTimelineEvents 配置项）
- `internal/context/history_test.go`（新建）

## 预估时间

0.5 天

## 状态

待实施
