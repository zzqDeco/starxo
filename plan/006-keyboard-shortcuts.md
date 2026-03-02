# 006 - 键盘快捷键

## 目标

添加核心键盘快捷键提升操作效率。

## 范围

- 前端全局

## 方案

1. 使用 `@vueuse/core` 的 `useMagicKeys` 或直接 `onKeyDown` 监听键盘事件
2. 在 App.vue 中注册全局快捷键
3. 确保快捷键不与输入框内容编辑冲突

## 快捷键列表

| 快捷键 | 功能 |
|--------|------|
| `Ctrl+Enter` | 发送消息（InputArea 中） |
| `Ctrl+L` | 清空当前对话 |
| `Ctrl+N` | 新建会话 |
| `Ctrl+,` | 打开设置 |
| `Escape` | 关闭设置/对话框 |
| `Ctrl+Shift+T` | 切换终端面板 |

## 具体任务

- [ ] 创建 `frontend/src/composables/useKeyboardShortcuts.ts`: 封装全局快捷键注册逻辑
- [ ] 在 App.vue 中调用 `useKeyboardShortcuts()` 注册全局快捷键
- [ ] 处理焦点冲突: 输入框聚焦时屏蔽 `Ctrl+L` 等可能冲突的快捷键
- [ ] 在设置面板或 Header 中显示快捷键提示（Tooltip 或帮助弹窗）

## 涉及文件

- `frontend/src/composables/useKeyboardShortcuts.ts`（新建）
- `frontend/src/App.vue`（注册快捷键）
- `frontend/src/components/layout/Header.vue`（可选：添加快捷键提示入口）

## 预估时间

0.5 天

## 状态

待实施
