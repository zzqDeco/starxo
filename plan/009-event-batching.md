# 009 - 事件流批处理

## 目标

合并高频 timeline events，减少前端渲染压力，提升 UI 流畅度。

## 范围

- `internal/service/chat.go`（事件发射侧）

## 方案

1. 在 ChatService 中添加事件缓冲器
2. 50ms 窗口内的事件聚合后批量发射
3. 使用新事件通道 `agent:timeline_batch`（[]TimelineEvent）
4. 前端 App.vue 监听批量事件并逐条分发到 chatStore
5. 保留单条 `agent:timeline` 作为 fallback，确保向后兼容

## 具体任务

- [ ] 创建 `internal/service/event_batcher.go`: EventBatcher 结构体，包含缓冲区、50ms 定时器、Flush 方法
- [ ] 实现 Add(event) 方法: 将事件加入缓冲区，首个事件启动定时器
- [ ] 实现 Flush() 方法: 批量发射 `agent:timeline_batch` 事件并清空缓冲区
- [ ] 修改 `chat.go`: 事件通过 batcher.Add() 发射而非直接 EventsEmit
- [ ] 前端 App.vue: 添加 `agent:timeline_batch` 事件监听器
- [ ] chatStore: 添加 `addTimelineEvents(events[])` 批量处理方法，一次性更新状态

## 涉及文件

- `internal/service/event_batcher.go`（新建）
- `internal/service/event_batcher_test.go`（新建）
- `internal/service/chat.go`（修改：事件通过 batcher 发射）
- `frontend/src/App.vue`（修改：监听批量事件）
- `frontend/src/stores/chatStore.ts`（修改：添加批量处理方法）

## 预估时间

1 天

## 状态

待实施
