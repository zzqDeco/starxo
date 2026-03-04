# 接口说明

> 所属项目: Starxo | 文档类型: 接口说明

---

## 一、外部能力概述

Starxo 的所有外部接口均通过 **Wails v2 绑定机制** 暴露。前端 (Vue 3) 通过 Wails 自动生成的 TypeScript 绑定调用 Go 后端方法，后端通过 Wails 事件系统向前端推送实时数据。

接口分为两类:
1. **绑定服务方法** (前端主动调用): 前端通过 `wailsjs/go/service.XXXService.Method()` 调用
2. **事件通道** (后端主动推送): 后端通过 `wailsruntime.EventsEmit()` 推送，前端通过 `wailsruntime.EventsOn()` 监听

无 HTTP API、无 WebSocket、无 gRPC。所有通信均在进程内通过 Wails IPC 完成。

---

## 二、Wails 绑定服务方法

### 2.1 ChatService — 对话管理

| 方法 | 签名 | 说明 |
|------|------|------|
| `SendMessage` | `(userMessage string) error` | 发送用户消息，触发 Agent 处理。Per-session 运行守卫，异步执行。前置条件: 沙箱已连接、活跃会话已设置 |
| `StopGeneration` | `() error` | 取消活跃会话当前正在运行的 Agent 生成 |
| `ResumeWithAnswer` | `(answer string) error` | 用户回答追问后恢复活跃会话的 Agent 执行 |
| `ResumeWithChoice` | `(selectedIndex int) error` | 用户选择选项后恢复活跃会话的 Agent 执行 |
| `SetMode` | `(mode string) error` | 切换活跃会话的 Agent 模式。值: `"default"` 或 `"plan"` |
| `GetMode` | `() string` | 获取活跃会话当前 Agent 模式 |
| `ClearHistory` | `() error` | 清除活跃会话对话历史。重置 per-session ctxEngine、timeline、streamingState、todos |
| `BuildRunners` | `() error` | 构建 deep agent 和两种 runner |

### 2.2 SandboxService — 沙箱管理

| 方法 | 签名 | 说明 |
|------|------|------|
| `ConnectSSH` | `() error` | 建立 SSH 连接 + 确认 Docker 可用。不创建容器 |
| `DisconnectSSH` | `() error` | 断开 SSH 连接，分离活跃容器 |
| `CreateAndActivateContainer` | `() error` | 创建新容器 + 注册 + 激活 |
| `ActivateContainer` | `(containerRegID string) error` | 切换到已注册容器（验证 SSH Host 匹配） |
| `DeactivateContainer` | `() error` | 分离活跃容器（不停止），SSH 保持连接 |
| `GetStatus` | `() SandboxStatusDTO` | 获取当前沙箱连接状态 |
| `Connect` | `() error` | Legacy: ConnectSSH + CreateAndActivateContainer |
| `ConnectExisting` | `(containerRegID string) error` | Legacy: 如 SSH 未连接先连接，再 ActivateContainer |
| `Disconnect` | `() error` | Legacy: 委托 DisconnectSSH |
| `DisconnectAndDestroy` | `() error` | 停止并移除活跃容器 |

### 2.3 SessionService — 会话管理

| 方法 | 签名 | 说明 |
|------|------|------|
| `CreateSession` | `(name string) (*model.Session, error)` | 创建新会话，自动保存当前会话，设置 ChatService 活跃会话，清除 todos |
| `ListSessions` | `() ([]model.Session, error)` | 列出所有会话 |
| `ListSessionsEnriched` | `() ([]EnrichedSession, error)` | 列出会话（含容器状态信息） |
| `SwitchSession` | `(sessionID string) error` | 切换到指定会话。保存当前、加载目标到 per-session run、发射含完整状态快照的 session:switched 事件 |
| `DeleteSession` | `(sessionID string) error` | 删除会话。先停止运行中代理，清理 per-session 状态，级联销毁子容器 |
| `RenameSession` | `(sessionID, name string) error` | 重命名会话 |
| `GetActiveSession` | `() *model.Session` | 获取当前活跃会话（返回副本） |
| `GetActiveSessionMessages` | `() ([]model.PersistedMessage, error)` | 获取活动会话消息 |
| `SaveChatDisplay` | `(data string) error` | 保存前端显示数据（旧版接口） |
| `LoadChatDisplay` | `() (string, error)` | 加载前端显示数据（旧版接口） |
| `SaveCurrentSession` | `() error` | 持久化当前会话（per-session 数据） |
| `LoadSessionData` | `() (*model.SessionData, error)` | 加载统一会话数据（推荐接口） |

### 2.4 SettingsService — 配置管理

| 方法 | 签名 | 说明 |
|------|------|------|
| `GetSettings` | `() *config.AppConfig` | 获取当前配置 |
| `SaveSettings` | `(cfg *config.AppConfig) error` | 保存配置到文件。触发 onSettingsSave 回调使 Runner 失效 |
| `TestSSHConnection` | `() error` | 使用当前 SSH 配置测试连接 |
| `TestLLMConnection` | `() (string, error)` | 使用当前 LLM 配置测试连接，返回模型响应 |

### 2.5 FileService — 文件管理

| 方法 | 签名 | 说明 |
|------|------|------|
| `SelectAndUploadFile` | `() (FileInfoDTO, error)` | 打开原生文件对话框选择文件并上传到容器 |
| `UploadFile` | `(localPath string) (FileInfoDTO, error)` | 上传指定本地文件到容器 /workspace 目录 |
| `DownloadFile` | `(containerPath string) error` | 从容器下载文件，打开原生保存对话框 |
| `ListWorkspaceFiles` | `() ([]FileInfoDTO, error)` | 列出容器 /workspace 目录下的文件 (最深 3 层) |
| `ReadFilePreview` | `(containerPath string) (string, error)` | 读取容器文件内容预览 (最大 4KB) |

### 2.6 ContainerService — 容器管理

| 方法 | 签名 | 说明 |
|------|------|------|
| `ListContainers` | `() ([]model.Container, error)` | 列出所有注册的容器 |
| `RefreshContainerStatus` | `(containerRegID string) (*model.Container, error)` | 刷新容器实际运行状态 |
| `StartContainer` | `(containerRegID string) error` | 启动已停止的容器 |
| `StopContainer` | `(containerRegID string) error` | 停止运行中的容器 |
| `DestroyContainer` | `(containerRegID string) error` | 销毁容器并从注册表删除 |

---

## 三、Wails 事件通道

### 3.1 Agent 事件

| 事件名 | 载荷类型 | 字段 | 触发时机 |
|--------|----------|------|----------|
| `agent:timeline` | `TimelineEvent` | `id`, `type`, `agent`, `content`, `toolName?`, `toolArgs?`, `toolId?`, `timestamp`, **`sessionId?`** | Agent 产生任何可展示事件时。type 值: `message`, `tool_call`, `tool_result`, `transfer`, `info`, `interrupt`, `stream_chunk`, `stream_end`, `reasoning`, `thinking` |
| `agent:done` | `map[string]string` | **`sessionId`** | Agent 完成一次完整处理 (非中断退出) |
| `agent:error` | `map[string]interface{}` | **`sessionId`**, `error` | Agent 运行出错、Runner 构建失败、MCP 连接失败等 |
| `agent:interrupt` | `InterruptEvent` | `type` (`followup`/`choice`), `interruptId`, `checkpointId`, `questions?[]`, `options?[]`, `question?`, **`sessionId?`** | Agent 调用 ask_user 或 ask_choice 工具暂停执行 |
| `agent:mode_changed` | `ModeChangedEvent` | `mode` (`default`/`plan`), **`sessionId?`** | 用户切换 Agent 模式 |

### 3.2 SSH/容器事件

| 事件名 | 载荷类型 | 字段 | 触发时机 |
|--------|----------|------|----------|
| `ssh:progress` | `SandboxProgressEvent` | `step`, `percent` (0-100) | SSH 连接和 Docker 检查进度 |
| `ssh:connected` | `nil` | - | SSH 连接成功并 Docker 就绪 |
| `ssh:disconnected` | `nil` | - | SSH 断连（健康检查失败或主动断开） |
| `container:progress` | `SandboxProgressEvent` | `step`, `percent` (0-100) | 容器创建/激活进度 |
| `container:ready` | `map[string]string` | `containerID` | 新容器创建并激活就绪 |
| `container:activated` | `map[string]string` | `containerID` | 切换到已有容器就绪 |
| `container:deactivated` | `nil` | - | 容器分离 |

### 3.3 Session 事件

| 事件名 | 载荷类型 | 字段 | 触发时机 |
|--------|----------|------|----------|
| `session:switched` | `SessionSwitchedEvent` | `session`, `containerID?`, **`agentRunning`**, **`currentAgent?`**, **`mode`**, **`hasInterrupt`**, **`interrupt?`** | 活跃会话切换完成，包含完整的 per-session 状态快照 |

---

## 四、配置契约

配置文件路径: `~/.starxo/config.json`

### 4.1 SSH 配置 (`ssh`)

```json
{
  "host": "192.168.1.100",
  "port": 22,
  "user": "root",
  "password": "xxx",
  "privateKey": "/path/to/id_rsa"
}
```

| 字段 | 类型 | 默认值 | 必填 | 说明 |
|------|------|--------|------|------|
| `host` | string | `""` | 是 | SSH 服务器地址 |
| `port` | int | `22` | 否 | SSH 端口 |
| `user` | string | `"root"` | 否 | SSH 用户名 |
| `password` | string | `""` | 条件 | 密码认证 (与 privateKey 二选一) |
| `privateKey` | string | `""` | 条件 | 私钥文件路径 (与 password 二选一) |

### 4.2 Docker 配置 (`docker`)

```json
{
  "image": "python:3.11-slim",
  "memoryLimit": 2048,
  "cpuLimit": 1.0,
  "workDir": "/workspace",
  "network": true
}
```

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `image` | string | `"python:3.11-slim"` | Docker 镜像名 |
| `memoryLimit` | int64 | `2048` | 内存限制 (MB) |
| `cpuLimit` | float64 | `1.0` | CPU 核数限制 |
| `workDir` | string | `"/workspace"` | 容器工作目录 |
| `network` | bool | `true` | 是否允许网络访问 |

### 4.3 LLM 配置 (`llm`)

```json
{
  "type": "openai",
  "baseURL": "https://api.openai.com/v1",
  "apiKey": "sk-xxx",
  "model": "gpt-4o",
  "headers": {}
}
```

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `type` | string | `"openai"` | 提供商类型: `openai`, `ark`, `ollama` |
| `baseURL` | string | `""` | API 基础 URL (用于 OpenAI 兼容接口如 DeepSeek) |
| `apiKey` | string | `""` | API 密钥 |
| `model` | string | `"gpt-4o"` | 模型名称 |
| `headers` | map[string]string | `null` | 额外 HTTP 请求头 |

### 4.4 MCP 配置 (`mcp`)

```json
{
  "servers": [
    {
      "name": "filesystem",
      "transport": "stdio",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/workspace"],
      "env": {},
      "enabled": true
    },
    {
      "name": "remote-server",
      "transport": "sse",
      "url": "http://localhost:3001/sse",
      "enabled": true
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `servers[].name` | string | 服务器名称 (用于日志和错误提示) |
| `servers[].transport` | string | 传输类型: `stdio` (本地进程) 或 `sse` (远程HTTP) |
| `servers[].command` | string | stdio 模式: 启动命令 |
| `servers[].args` | []string | stdio 模式: 命令参数 |
| `servers[].url` | string | sse 模式: 服务器 URL |
| `servers[].env` | map[string]string | 环境变量 |
| `servers[].enabled` | bool | 是否启用 |

### 4.5 Agent 配置 (`agent`)

```json
{
  "maxIterations": 30
}
```

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `maxIterations` | int | `30` | Agent 配置的最大迭代次数 (实际受 Deep Agent MaxIteration=50 和 PlanExecute MaxIterations=20 约束) |
