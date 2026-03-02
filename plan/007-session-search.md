# 007 - 会话搜索

## 目标

在侧边栏添加会话名称搜索功能，方便用户快速定位历史会话。

## 范围

- `frontend/src/components/layout/Sidebar.vue`
- `SessionService`

## 方案

1. Sidebar 顶部添加搜索输入框
2. 前端过滤: 按会话标题模糊匹配（使用 computed 过滤 sessionList）
3. 支持清空搜索恢复全部列表

## 具体任务

- [ ] 在 Sidebar.vue 添加 NInput 搜索框（位于会话列表上方）
- [ ] 添加 `searchQuery` ref 和 `filteredSessions` computed，按标题模糊匹配过滤
- [ ] 搜索为空时显示全部会话，有输入时实时过滤
- [ ] 添加 i18n key `sidebar.search` 到 `zh.ts` 和 `en.ts`

## 涉及文件

- `frontend/src/components/layout/Sidebar.vue`（修改：添加搜索框和过滤逻辑）
- `frontend/src/i18n/locales/zh.ts`（添加 sidebar.search）
- `frontend/src/i18n/locales/en.ts`（添加 sidebar.search）

## 预估时间

0.5 天

## 状态

待实施
