# 业务规则目录

> 所属项目: Eino Coding Agent (starxo) | 文档类型: 业务规则

---

## 优先级矩阵

| 优先级 | 定义 | 规则数 |
|--------|------|--------|
| **P0** | 系统不可用/数据丢失风险，必须满足 | 3 |
| **P1** | 核心功能正确性，应当满足 | 5 |
| **P2** | 辅助功能/运维，建议满足 | 2 |

---

## P0 — 系统关键规则

### BR-001: Agent 必须在沙箱连接后才能处理消息

- **优先级**: P0
- **描述**: `ChatService.SendMessage()` 在构建 Runner 时检查沙箱连接状态。若 `sandbox == nil` 或 `!sandbox.IsConnected()`，返回错误 `"sandbox is not connected"`，拒绝处理用户消息。
- **原因**: Agent 的所有工具 (shell 命令、文件读写) 依赖沙箱中的 `commandline.Operator`。无沙箱时工具不可用，Agent 无法执行任何有意义的操作。
- **影响范围**: `ChatService.BuildRunners()`
- **实现位置**: `internal/service/chat.go` L633-635
- **验证方式**: 未连接沙箱时发送消息，应收到明确错误提示

### BR-002: Deep Agent 通过 transfer_to_agent 编排子 Agent

- **优先级**: P0
- **描述**: Deep Agent (coding_agent) 是唯一的顶层智能体。它通过 Eino ADK 的 `transfer_to_agent` 机制将任务委派给 code_writer、code_executor、file_manager 三个子智能体。子智能体没有 Exit 工具，由 Deep Agent 管理其生命周期。
- **原因**: 这是整个 Agent 系统的核心编排模式。Deep Agent 决定何时使用哪个子智能体，子智能体专注于各自领域的工具调用。
- **影响范围**: Agent 所有编程操作
- **实现位置**: `internal/agent/deep_agent.go` (BuildDeepAgent), `internal/agent/codewriter.go`, `internal/agent/codeexecutor.go`, `internal/agent/filemanager.go`
- **验证方式**: 观察 agent:timeline 事件中的 transfer 类型事件

### BR-003: 中断工具暂停执行并等待用户响应

- **优先级**: P0
- **描述**: `ask_user` (FollowUp) 和 `ask_choice` (Choice) 工具通过 Eino ADK 的 `StatefulInterrupt` 机制暂停 Agent 执行。暂停时保存 checkpoint，前端收到 `agent:interrupt` 事件后显示对话框。用户响应后通过 `ResumeWithAnswer` 或 `ResumeWithChoice` 恢复执行。
- **原因**: Agent 在执行过程中可能需要用户澄清需求或做出选择。中断机制保证 Agent 状态完整保存，恢复后可继续执行。
- **影响范围**: Agent 与用户的交互循环
- **实现位置**: `internal/tools/followup.go`, `internal/tools/choice.go`, `internal/service/chat.go` (handleInterrupt, ResumeWithAnswer, ResumeWithChoice)
- **验证方式**: Agent 调用 ask_user 后，前端应出现追问对话框；用户回答后 Agent 继续执行

---

## P1 — 核心功能规则

### BR-004: 配置变更使 Agent Runner 失效

- **优先级**: P1
- **描述**: 当用户通过 `SettingsService.Save()` 保存设置时，触发 `onSettingsSave` 回调，调用 `ChatService.InvalidateRunner()`。这将 `deepAgent`、`defaultRunner`、`planRunner` 全部置为 nil。下次用户发送消息时，`SendMessage` 检测到 Runner 为空，自动触发 `BuildRunners()` 用新配置重建。
- **优先级**: P1
- **原因**: LLM 提供商、模型、MCP 服务器等配置变更后，必须重建 Agent 以使用新参数。
- **影响范围**: 配置保存 -> Agent 重建
- **实现位置**: `internal/service/chat.go` (InvalidateRunner, invalidateRunners), `app.go` L107-109
- **验证方式**: 修改 LLM 配置后发送消息，应能看到"Building agent runner..."提示

### BR-005: 会话持久化包含消息历史和时间线事件

- **优先级**: P1
- **描述**: 会话保存时 (`SessionService.SaveCurrentSession()`) 将 `ctxEngine` 中的消息历史和前端 `TimelineEvent` 列表序列化到 `~/.eino-agent/sessions/{id}/` 目录。会话切换时从该目录恢复状态。
- **原因**: 用户期望在会话切换后能恢复之前的对话上下文和可视化时间线。
- **影响范围**: 会话保存/加载/切换
- **实现位置**: `internal/service/session_svc.go`, `internal/storage/session_store.go`
- **验证方式**: 创建会话，发送消息，切换到另一会话再切回，消息历史应完整恢复

### BR-006: 文件传输使用 SFTP 通道

- **优先级**: P1
- **描述**: 所有文件传输通过 SFTP 进行。上传流程: 本地文件 -> SFTP 上传到远程 `/tmp/eino-upload-xxx` -> `docker cp` 拷贝到容器内。下载流程相反: `docker cp` 从容器到远程 `/tmp/eino-download-xxx` -> SFTP 下载到本地。传输完成后清理远程临时文件。
- **原因**: Docker 容器运行在远程服务器上，本地无法直接访问容器文件系统。需要通过 SSH 的 SFTP 作为中转通道。
- **影响范围**: 文件上传/下载
- **实现位置**: `internal/sandbox/transfer.go` (UploadToContainer, DownloadFromContainer)
- **验证方式**: 上传文件到容器并下载回来，内容应一致

### BR-007: 上下文窗口化应用于所有消息

- **优先级**: P1
- **描述**: `ctxEngine.PrepareMessages()` 在发送给 LLM 前对消息列表应用窗口化策略。默认保留最近 20 条消息，每条消息内容超过 4000 字符时截断 (保留前 60% + 后 20% + 截断标记)。超出窗口的旧消息用摘要占位符替代。
- **原因**: LLM 有上下文长度限制，过长的消息历史会导致 API 调用失败或质量下降。窗口化策略在保留最新上下文的同时控制总量。
- **影响范围**: 所有 Agent 调用
- **实现位置**: `internal/context/windowing.go` (WindowMessages, TruncateContent), `internal/context/engine.go` (PrepareMessages)
- **验证方式**: 长对话后检查实际发送给 LLM 的消息数量应不超过窗口大小

### BR-008: 所有 Wails 绑定服务方法必须线程安全

- **优先级**: P1
- **描述**: Wails 前端调用 Go 方法时可能并发执行。所有 Service 的公开方法使用 `sync.Mutex` 保护共享状态。`ChatService` 尤其重要，因为 Agent goroutine 和前端调用可能同时操作 `ctxEngine`、`pendingInterrupt` 等字段。
- **原因**: Wails 的前端-后端调用是异步的，多个前端操作可能同时到达后端。不加锁会导致数据竞争和崩溃。
- **影响范围**: 所有 Service 公开方法
- **实现位置**: `internal/service/chat.go` (mu sync.Mutex), 其他 Service 文件
- **验证方式**: 使用 `go run -race` 检测数据竞争

---

## P2 — 辅助功能规则

### BR-009: 容器生命周期独立于应用

- **优先级**: P2
- **描述**: Docker 容器在远程服务器上运行，其生命周期独立于桌面应用。应用关闭时仅断开 SSH 连接，不停止或删除容器。应用重启后可通过 `ConnectExisting` 重新连接到之前的容器。容器生命周期操作: create -> start -> stop -> reconnect -> destroy。
- **原因**: 容器中可能有用户的工作成果和环境配置，不应因应用重启而丢失。
- **影响范围**: 容器管理、应用关闭
- **实现位置**: `app.go` (shutdown 方法), `internal/service/sandbox_svc.go` (ConnectExisting), `internal/service/container_svc.go`
- **验证方式**: 关闭应用后在远程服务器上检查容器仍在运行，重启应用后可重连

### BR-010: MCP 服务器连接失败为非致命错误

- **优先级**: P2
- **描述**: 在 `BuildRunners()` 过程中，如果某个 MCP 服务器连接失败或工具加载失败，仅发出 `agent:error` 事件通知前端，然后跳过该服务器继续处理。Agent 仍可正常运行，只是缺少该 MCP 服务器提供的工具。
- **原因**: MCP 是扩展能力，不是核心功能。单个 MCP 服务器不可用不应阻止整个 Agent 启动。
- **影响范围**: Agent 初始化
- **实现位置**: `internal/service/chat.go` L660-677 (BuildRunners 中的 MCP 循环)
- **验证方式**: 配置一个无效的 MCP 服务器，Agent 应仍能正常启动并处理消息
