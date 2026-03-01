# 业务域定义

> 所属项目: Starxo | 文档类型: 业务域

---

## 一、业务边界

Starxo 是一款 **AI 编程智能体桌面应用**，不是服务端应用或 SaaS 产品。

### 边界内 (In Scope)

- 桌面原生应用，通过 Wails v2 打包为单一可执行文件
- 通过 SSH 连接远程服务器，在 Docker 容器内提供隔离的代码执行环境
- 集成多种 LLM 提供商 (OpenAI、DeepSeek、火山引擎 Ark、Ollama)
- Deep Agent 架构：主控智能体编排子智能体完成编程任务
- 支持 MCP (Model Context Protocol) 扩展工具能力
- 会话管理与持久化
- 容器生命周期管理 (create/start/stop/reconnect/destroy)

### 边界外 (Out of Scope)

- 不提供 Web 界面或 HTTP API 服务
- 不支持多用户并发访问
- 不包含本地代码执行能力 (所有执行均在远程容器中)
- 不管理远程服务器的部署或配置
- 不提供代码版本控制集成 (Git 操作由 Agent 通过命令行工具完成)

---

## 二、核心业务对象

### Session (会话)

- 定义: 用户与 AI Agent 的一次完整对话上下文
- 生命周期: 创建 -> 活跃 -> 切换/保存 -> 删除
- 持久化: `~/.starxo/sessions/{id}/` 目录，包含消息历史和时间线事件
- 属性: ID (UUID), 名称, 创建时间, 更新时间, 消息列表, 时间线事件列表, 绑定容器 ID
- 源文件: `internal/model/session.go`, `internal/storage/session_store.go`, `internal/service/session_svc.go`

### Agent (智能体)

- 定义: 基于 Eino ADK 构建的 AI 智能体，具备工具调用和子智能体编排能力
- 层级结构:
  - **Deep Agent (coding_agent)**: 主控智能体，负责任务理解和子智能体调度
  - **Code Writer (code_writer)**: 代码编写子智能体，使用 shell 命令进行文件操作
  - **Code Executor (code_executor)**: 代码执行子智能体，运行命令并返回结果
  - **File Manager (file_manager)**: 文件管理子智能体，处理文件浏览和搜索
- 运行模式:
  - **默认模式 (default)**: Deep Agent 直接作为 Runner 运行，自主决策
  - **计划模式 (plan)**: PlanExecute 模式，先生成计划再逐步执行
- 源文件: `internal/agent/` 目录全部文件

### Sandbox (沙箱)

- 定义: 远程隔离执行环境，由 SSH 连接 + Docker 容器组成
- 组件:
  - **SSH Client**: 与远程服务器建立连接，提供命令执行和文件传输通道
  - **Docker Manager**: 在远程服务器上管理 Docker 容器的创建、启动、停止
  - **Operator**: 封装容器内的命令执行、文件读写操作
  - **File Transfer**: 基于 SFTP 的文件上传/下载，支持本地-远程-容器三级传输
- 生命周期: 连接 SSH -> 创建/启动容器 -> 就绪 -> 执行操作 -> 断开 (容器保活)
- 源文件: `internal/sandbox/` 目录全部文件

### Tool (工具)

- 定义: Agent 可调用的原子能力单元
- 分类:
  - **内置工具 (Builtin)**: shell 命令执行、文件读写，通过 `commandline.Operator` 实现
  - **中断工具 (Interrupt)**: `ask_user` (追问)、`ask_choice` (选择)，触发 StatefulInterrupt 暂停执行
  - **追踪工具**: `write_todos` (写入待办)、`update_todo` (更新待办)、`notify_user` (通知用户)
  - **MCP 工具**: 通过 MCP 协议从外部服务器加载的动态工具
- 源文件: `internal/tools/` 目录全部文件

### Message (消息)

- 定义: 用户与 Agent 之间的交互单元
- 角色: user (用户)、assistant (助手)、tool (工具结果)、system (系统)
- 上下文管理: 通过 `context.Engine` 维护消息历史，`WindowMessages` 控制窗口大小
- 源文件: `internal/model/message.go`, `internal/context/` 目录

### Plan (计划)

- 定义: 计划模式下 Agent 生成的任务分解
- 状态: todo -> doing -> done / failed / skipped
- 属性: taskId, status, desc, execResult
- 由 Eino PlanExecute 框架管理，通过 `agent:plan` 事件推送到前端
- 源文件: `internal/agent/plan.go`, `internal/agent/plan_wrapper.go`

### Container (容器)

- 定义: 远程 Docker 容器的注册记录
- 生命周期: 创建 -> 运行 -> 停止 -> 销毁 (独立于 SSH 连接)
- 持久化: `~/.starxo/containers.json`
- 属性: 注册 ID, Docker ID, 名称, 镜像, SSH Host, 工作目录, 状态, 创建时间
- 状态: running, stopped, unknown
- 源文件: `internal/model/container.go`, `internal/storage/container_store.go`, `internal/service/container_svc.go`

---

## 三、角色与用户

### 目标用户

需要 AI 辅助编程的开发者，具备以下特征:

- 拥有可 SSH 访问的远程服务器 (已安装 Docker)
- 需要在隔离环境中进行代码实验、调试或原型开发
- 希望通过自然语言描述编程需求，由 AI Agent 自动完成代码编写、执行和调试
- 可能使用不同的 LLM 提供商 (OpenAI、DeepSeek、本地 Ollama 等)

### 用户交互方式

1. **对话式**: 在聊天面板中输入自然语言需求
2. **中断响应**: 回答 Agent 的追问或在选项中做出选择
3. **文件操作**: 上传文件到容器、下载容器中的文件、浏览工作区
4. **配置管理**: 设置 SSH 连接、Docker 参数、LLM 提供商、MCP 服务器
5. **会话管理**: 创建、切换、重命名、删除会话
6. **容器管理**: 查看、启动、停止、销毁容器

---

## 四、域约束

### 硬性约束

1. **SSH 可达**: 必须有一台 SSH 可访问的远程服务器作为执行环境
2. **Docker 可用**: 远程服务器上必须已安装并运行 Docker 引擎
3. **网络连通**: 本地桌面到远程服务器的 SSH 端口必须可达
4. **LLM 配置**: 至少配置一个有效的 LLM 提供商 (API Key 或本地 Ollama)

### 软性约束

1. **容器镜像**: 默认使用 `python:3.11-slim`，用户可自定义
2. **资源限制**: 默认内存 2048MB、CPU 1.0 核，用户可调整
3. **工作目录**: 容器内默认 `/workspace`，用户可配置
4. **上下文窗口**: 默认保留最近 20 条消息，单条最大 4000 字符
5. **Agent 迭代**: Deep Agent 最大 50 次迭代，Plan 模式最大 20 次迭代
