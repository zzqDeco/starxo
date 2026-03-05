# 文件级文档索引

> 所属项目: Starxo | 文档类型: 文件级文档索引

---

## 一、映射规则

1. 每个源文件对应一个同名 `.plan.md` 文档文件
2. 文档文件保持与源文件相同的目录层级，统一放在 `doc/src/` 下
3. 映射示例:
   - `main.go` -> `doc/src/main.plan.md`
   - `internal/agent/deep_agent.go` -> `doc/src/internal/agent/deep_agent.plan.md`
   - `frontend/src/stores/chatStore.ts` -> `doc/src/frontend/src/stores/chatStore.plan.md`

---

## 二、完整映射表

### Go 根文件

| 源文件 | 文档文件 | 文件类型 | 所属模块 |
|--------|----------|----------|----------|
| `main.go` | `doc/src/main.plan.md` | Go | 入口 |
| `app.go` | `doc/src/app.plan.md` | Go | 入口 |

### internal/agent/ — 智能体模块

| 源文件 | 文档文件 | 文件类型 | 所属模块 |
|--------|----------|----------|----------|
| `internal/agent/deep_agent.go` | `doc/src/internal/agent/deep_agent.plan.md` | Go | agent |
| `internal/agent/codewriter.go` | `doc/src/internal/agent/codewriter.plan.md` | Go | agent |
| `internal/agent/codeexecutor.go` | `doc/src/internal/agent/codeexecutor.plan.md` | Go | agent |
| `internal/agent/filemanager.go` | `doc/src/internal/agent/filemanager.plan.md` | Go | agent |
| `internal/agent/runner.go` | `doc/src/internal/agent/runner.plan.md` | Go | agent |
| `internal/agent/plan.go` | `doc/src/internal/agent/plan.plan.md` | Go | agent |
| `internal/agent/plan_wrapper.go` | `doc/src/internal/agent/plan_wrapper.plan.md` | Go | agent |
| `internal/agent/prompts.go` | `doc/src/internal/agent/prompts.plan.md` | Go | agent |
| `internal/agent/context.go` | `doc/src/internal/agent/context.plan.md` | Go | agent |
| `internal/agent/tool_wrapper.go` | `doc/src/internal/agent/tool_wrapper.plan.md` | Go | agent |

### internal/service/ — 服务模块

| 源文件 | 文档文件 | 文件类型 | 所属模块 |
|--------|----------|----------|----------|
| `internal/service/chat.go` | `doc/src/internal/service/chat.plan.md` | Go | service |
| `internal/service/sandbox_svc.go` | `doc/src/internal/service/sandbox_svc.plan.md` | Go | service |
| `internal/service/session_svc.go` | `doc/src/internal/service/session_svc.plan.md` | Go | service |
| `internal/service/settings_svc.go` | `doc/src/internal/service/settings_svc.plan.md` | Go | service |
| `internal/service/file_svc.go` | `doc/src/internal/service/file_svc.plan.md` | Go | service |
| `internal/service/container_svc.go` | `doc/src/internal/service/container_svc.plan.md` | Go | service |
| `internal/service/events.go` | `doc/src/internal/service/events.plan.md` | Go | service |

### internal/sandbox/ — 沙箱模块

| 源文件 | 文档文件 | 文件类型 | 所属模块 |
|--------|----------|----------|----------|
| `internal/sandbox/ssh.go` | `doc/src/internal/sandbox/ssh.plan.md` | Go | sandbox |
| `internal/sandbox/docker.go` | `doc/src/internal/sandbox/docker.plan.md` | Go | sandbox |
| `internal/sandbox/manager.go` | `doc/src/internal/sandbox/manager.plan.md` | Go | sandbox |
| `internal/sandbox/operator.go` | `doc/src/internal/sandbox/operator.plan.md` | Go | sandbox |
| `internal/sandbox/transfer.go` | `doc/src/internal/sandbox/transfer.plan.md` | Go | sandbox |
| `internal/sandbox/setup.go` | `doc/src/internal/sandbox/setup.plan.md` | Go | sandbox |

### internal/tools/ — 工具模块

| 源文件 | 文档文件 | 文件类型 | 所属模块 |
|--------|----------|----------|----------|
| `internal/tools/builtin.go` | `doc/src/internal/tools/builtin.plan.md` | Go | tools |
| `internal/tools/followup.go` | `doc/src/internal/tools/followup.plan.md` | Go | tools |
| `internal/tools/choice.go` | `doc/src/internal/tools/choice.plan.md` | Go | tools |
| `internal/tools/todos.go` | `doc/src/internal/tools/todos.plan.md` | Go | tools |
| `internal/tools/notify.go` | `doc/src/internal/tools/notify.plan.md` | Go | tools |
| `internal/tools/registry.go` | `doc/src/internal/tools/registry.plan.md` | Go | tools |
| `internal/tools/mcp.go` | `doc/src/internal/tools/mcp.plan.md` | Go | tools |
| `internal/tools/custom.go` | `doc/src/internal/tools/custom.plan.md` | Go | tools |
| `internal/tools/error_policy.go` | `doc/src/internal/tools/error_policy.go.plan.md` | Go | tools |

### internal/config/ — 配置模块

| 源文件 | 文档文件 | 文件类型 | 所属模块 |
|--------|----------|----------|----------|
| `internal/config/config.go` | `doc/src/internal/config/config.plan.md` | Go | config |
| `internal/config/store.go` | `doc/src/internal/config/store.plan.md` | Go | config |

### internal/context/ — 上下文模块

| 源文件 | 文档文件 | 文件类型 | 所属模块 |
|--------|----------|----------|----------|
| `internal/context/engine.go` | `doc/src/internal/context/engine.plan.md` | Go | context |
| `internal/context/windowing.go` | `doc/src/internal/context/windowing.plan.md` | Go | context |
| `internal/context/history.go` | `doc/src/internal/context/history.plan.md` | Go | context |
| `internal/context/filecontext.go` | `doc/src/internal/context/filecontext.plan.md` | Go | context |
| `internal/context/timeline.go` | `doc/src/internal/context/timeline.plan.md` | Go | context |

### internal/llm/ — LLM 适配模块

| 源文件 | 文档文件 | 文件类型 | 所属模块 |
|--------|----------|----------|----------|
| `internal/llm/provider.go` | `doc/src/internal/llm/provider.plan.md` | Go | llm |

### internal/model/ — 数据模型模块

| 源文件 | 文档文件 | 文件类型 | 所属模块 |
|--------|----------|----------|----------|
| `internal/model/session.go` | `doc/src/internal/model/session.plan.md` | Go | model |
| `internal/model/message.go` | `doc/src/internal/model/message.plan.md` | Go | model |
| `internal/model/container.go` | `doc/src/internal/model/container.plan.md` | Go | model |
| `internal/model/session_data.go` | `doc/src/internal/model/session_data.plan.md` | Go | model |

### internal/storage/ — 持久化模块

| 源文件 | 文档文件 | 文件类型 | 所属模块 |
|--------|----------|----------|----------|
| `internal/storage/session_store.go` | `doc/src/internal/storage/session_store.plan.md` | Go | storage |
| `internal/storage/container_store.go` | `doc/src/internal/storage/container_store.plan.md` | Go | storage |

### internal/store/ — 检查点模块

| 源文件 | 文档文件 | 文件类型 | 所属模块 |
|--------|----------|----------|----------|
| `internal/store/checkpoint.go` | `doc/src/internal/store/checkpoint.plan.md` | Go | store |

### internal/logger/ — 日志模块

| 源文件 | 文档文件 | 文件类型 | 所属模块 |
|--------|----------|----------|----------|
| `internal/logger/logger.go` | `doc/src/internal/logger/logger.plan.md` | Go | logger |
| `internal/logger/callbacks.go` | `doc/src/internal/logger/callbacks.plan.md` | Go | logger |

### 前端入口文件

| 源文件 | 文档文件 | 文件类型 | 所属模块 |
|--------|----------|----------|----------|
| `frontend/src/main.ts` | `doc/src/frontend/src/main.plan.md` | TypeScript | 前端入口 |
| `frontend/src/App.vue` | `doc/src/frontend/src/App.plan.md` | Vue | 前端入口 |
| `frontend/src/style.css` | `doc/src/frontend/src/style.plan.md` | CSS | 前端样式 |

### frontend/src/stores/ — 状态管理

| 源文件 | 文档文件 | 文件类型 | 所属模块 |
|--------|----------|----------|----------|
| `frontend/src/stores/chatStore.ts` | `doc/src/frontend/src/stores/chatStore.plan.md` | TypeScript | stores |
| `frontend/src/stores/sessionStore.ts` | `doc/src/frontend/src/stores/sessionStore.plan.md` | TypeScript | stores |
| `frontend/src/stores/connectionStore.ts` | `doc/src/frontend/src/stores/connectionStore.plan.md` | TypeScript | stores |
| `frontend/src/stores/settingsStore.ts` | `doc/src/frontend/src/stores/settingsStore.plan.md` | TypeScript | stores |
| `frontend/src/stores/containerStore.ts` | `doc/src/frontend/src/stores/containerStore.plan.md` | TypeScript | stores |

### frontend/src/types/ — 类型定义

| 源文件 | 文档文件 | 文件类型 | 所属模块 |
|--------|----------|----------|----------|
| `frontend/src/types/config.ts` | `doc/src/frontend/src/types/config.plan.md` | TypeScript | types |
| `frontend/src/types/message.ts` | `doc/src/frontend/src/types/message.plan.md` | TypeScript | types |
| `frontend/src/types/session.ts` | `doc/src/frontend/src/types/session.plan.md` | TypeScript | types |

### frontend/src/components/chat/ — 聊天组件

| 源文件 | 文档文件 | 文件类型 | 所属模块 |
|--------|----------|----------|----------|
| `frontend/src/components/chat/ChatPanel.vue` | `doc/src/frontend/src/components/chat/ChatPanel.plan.md` | Vue | chat |
| `frontend/src/components/chat/InputArea.vue` | `doc/src/frontend/src/components/chat/InputArea.plan.md` | Vue | chat |
| `frontend/src/components/chat/InterruptDialog.vue` | `doc/src/frontend/src/components/chat/InterruptDialog.plan.md` | Vue | chat |
| `frontend/src/components/chat/MessageBubble.vue` | `doc/src/frontend/src/components/chat/MessageBubble.plan.md` | Vue | chat |
| `frontend/src/components/chat/PlanPanel.vue` | `doc/src/frontend/src/components/chat/PlanPanel.plan.md` | Vue | chat |
| `frontend/src/components/chat/TimelineEventItem.vue` | `doc/src/frontend/src/components/chat/TimelineEventItem.plan.md` | Vue | chat |
| `frontend/src/components/chat/TodoBoard.vue` | `doc/src/frontend/src/components/chat/TodoBoard.plan.md` | Vue | chat |

### frontend/src/components/layout/ — 布局组件

| 源文件 | 文档文件 | 文件类型 | 所属模块 |
|--------|----------|----------|----------|
| `frontend/src/components/layout/Header.vue` | `doc/src/frontend/src/components/layout/Header.plan.md` | Vue | layout |
| `frontend/src/components/layout/MainLayout.vue` | `doc/src/frontend/src/components/layout/MainLayout.plan.md` | Vue | layout |
| `frontend/src/components/layout/Sidebar.vue` | `doc/src/frontend/src/components/layout/Sidebar.plan.md` | Vue | layout |

### frontend/src/components/settings/ — 设置组件

| 源文件 | 文档文件 | 文件类型 | 所属模块 |
|--------|----------|----------|----------|
| `frontend/src/components/settings/SettingsPanel.vue` | `doc/src/frontend/src/components/settings/SettingsPanel.plan.md` | Vue | settings |
| `frontend/src/components/settings/SSHConfig.vue` | `doc/src/frontend/src/components/settings/SSHConfig.plan.md` | Vue | settings |
| `frontend/src/components/settings/DockerConfig.vue` | `doc/src/frontend/src/components/settings/DockerConfig.plan.md` | Vue | settings |
| `frontend/src/components/settings/LLMConfig.vue` | `doc/src/frontend/src/components/settings/LLMConfig.plan.md` | Vue | settings |
| `frontend/src/components/settings/MCPConfig.vue` | `doc/src/frontend/src/components/settings/MCPConfig.plan.md` | Vue | settings |

### frontend/src/components/files/ — 文件组件

| 源文件 | 文档文件 | 文件类型 | 所属模块 |
|--------|----------|----------|----------|
| `frontend/src/components/files/FileExplorer.vue` | `doc/src/frontend/src/components/files/FileExplorer.plan.md` | Vue | files |
| `frontend/src/components/files/FileTransfer.vue` | `doc/src/frontend/src/components/files/FileTransfer.plan.md` | Vue | files |

### frontend/src/components/containers/ — 容器组件

| 源文件 | 文档文件 | 文件类型 | 所属模块 |
|--------|----------|----------|----------|
| `frontend/src/components/containers/ContainerPanel.vue` | `doc/src/frontend/src/components/containers/ContainerPanel.plan.md` | Vue | containers |

### frontend/src/components/status/ — 状态组件

| 源文件 | 文档文件 | 文件类型 | 所属模块 |
|--------|----------|----------|----------|
| `frontend/src/components/status/AgentStatus.vue` | `doc/src/frontend/src/components/status/AgentStatus.plan.md` | Vue | status |
| `frontend/src/components/status/ConnectionStatus.vue` | `doc/src/frontend/src/components/status/ConnectionStatus.plan.md` | Vue | status |

### frontend/src/components/terminal/ — 终端组件

| 源文件 | 文档文件 | 文件类型 | 所属模块 |
|--------|----------|----------|----------|
| `frontend/src/components/terminal/TerminalPanel.vue` | `doc/src/frontend/src/components/terminal/TerminalPanel.plan.md` | Vue | terminal |

### frontend/src/composables/ — 组合式函数

| 源文件 | 文档文件 | 文件类型 | 所属模块 |
|--------|----------|----------|----------|
| `frontend/src/composables/useHelpers.ts` | `doc/src/frontend/src/composables/useHelpers.plan.md` | TypeScript | composables |
| `frontend/src/composables/useWailsEvent.ts` | `doc/src/frontend/src/composables/useWailsEvent.plan.md` | TypeScript | composables |

### frontend/src/locales/ — 国际化

| 源文件 | 文档文件 | 文件类型 | 所属模块 |
|--------|----------|----------|----------|
| `frontend/src/locales/zh.ts` | `doc/src/frontend/src/locales/zh.plan.md` | TypeScript | locales |
| `frontend/src/locales/en.ts` | `doc/src/frontend/src/locales/en.plan.md` | TypeScript | locales |
| `frontend/src/locales/index.ts` | `doc/src/frontend/src/locales/index.plan.md` | TypeScript | locales |

### 配置文件

| 源文件 | 文档文件 | 文件类型 | 所属模块 |
|--------|----------|----------|----------|
| `wails.json` | `doc/src/wails.plan.md` | JSON | 构建配置 |
| `frontend/vite.config.ts` | `doc/src/frontend/vite.config.plan.md` | TypeScript | 构建配置 |
| `frontend/tsconfig.json` | `doc/src/frontend/tsconfig.plan.md` | JSON | 构建配置 |

---

## 三、统计

| 分类 | 文件数 |
|------|--------|
| Go 根文件 | 2 |
| internal/agent/ | 10 |
| internal/service/ | 7 |
| internal/sandbox/ | 6 |
| internal/tools/ | 9 |
| internal/config/ | 2 |
| internal/context/ | 5 |
| internal/llm/ | 1 |
| internal/model/ | 4 |
| internal/storage/ | 2 |
| internal/store/ | 1 |
| internal/logger/ | 2 |
| 前端入口 | 3 |
| frontend/src/stores/ | 5 |
| frontend/src/types/ | 3 |
| frontend/src/components/chat/ | 7 |
| frontend/src/components/layout/ | 3 |
| frontend/src/components/settings/ | 5 |
| frontend/src/components/files/ | 2 |
| frontend/src/components/containers/ | 1 |
| frontend/src/components/status/ | 2 |
| frontend/src/components/terminal/ | 1 |
| frontend/src/composables/ | 2 |
| frontend/src/locales/ | 3 |
| 配置文件 | 3 |
| **总计** | **91** |
