# 业务流程

> 所属项目: Eino Coding Agent (starxo) | 文档类型: 业务流程

---

## 一、主流程

### 1.1 应用启动流程

```
main.go: NewApp()
  ├── config.NewStore()           -- 加载/创建 ~/.eino-agent/config.json
  ├── storage.NewSessionStore()   -- 加载/创建 ~/.eino-agent/sessions/
  ├── storage.NewContainerStore() -- 加载/创建 ~/.eino-agent/containers.json
  ├── 创建 6 个 Service 实例
  └── wails.Run()
        └── app.startup(ctx)
              ├── logger.Init()                      -- 初始化文件日志
              ├── logger.RegisterGlobalCallbacks()    -- 注册 Eino 全局回调
              ├── 所有 Service.SetContext(ctx)        -- 分发 Wails context
              ├── 连接 Service 间依赖 (回调注册)
              │     ├── sessionService.SetCtxEngine()
              │     ├── chatService.SetDependencies()
              │     ├── sandboxService.SetOnConnect()       -> chatService.UpdateSandbox()
              │     ├── sandboxService.SetOnContainerBound() -> sessionService.BindContainer()
              │     ├── sessionService.SetOnSessionSwitch()  -> sandboxService.ConnectExisting()
              │     ├── settingsService.SetOnSettingsSave()   -> chatService.InvalidateRunner()
              │     └── chatService.SetOnAgentDone()          -> sessionService.SaveCurrentSession()
              ├── sessionService.EnsureDefaultSession()  -- 加载或创建默认会话
              └── sandboxService.StartHealthMonitor()    -- 启动后台健康检查
```

### 1.2 沙箱连接流程

```
用户点击"连接"
  └── SandboxService.Connect()
        ├── 断开已有连接 (保留容器)
        ├── emit sandbox:progress {step:"Connecting SSH...", percent:10}
        ├── SSHClient.Connect()           -- 建立 SSH 连接
        ├── emit sandbox:progress {step:"SSH connected", percent:30}
        ├── RemoteDockerManager.Create()  -- 创建 Docker 容器
        ├── emit sandbox:progress {step:"Container created", percent:60}
        ├── SandboxManager 组装 (SSH + Docker + Operator + Transfer)
        ├── 注册容器到 ContainerStore
        ├── emit sandbox:progress {step:"Setting up...", percent:80}
        ├── Setup.InitContainer()         -- 初始化容器环境
        ├── onConnect(manager)            -> ChatService.UpdateSandbox()
        ├── onContainerBound(regID, wsPath) -> SessionService.BindContainer()
        └── emit sandbox:ready {sshConnected:true, dockerRunning:true, containerID:...}
```

### 1.3 用户消息处理流程

```
用户输入消息 -> ChatService.SendMessage(userMessage)
  ├── ctxEngine.AddUserMessage(userMessage)     -- 消息加入上下文
  ├── [首次] BuildRunners()                     -- 构建 Agent 和 Runner
  │     ├── llm.NewChatModel()                  -- 创建 LLM 模型
  │     ├── tools.RegisterBuiltinTools()        -- 注册内置工具
  │     ├── tools.ConnectMCPServer() x N        -- 连接 MCP 服务器
  │     ├── agent.BuildDeepAgent()              -- 构建 Deep Agent + 3 子智能体
  │     ├── agent.BuildDefaultRunner()          -- 构建默认模式 Runner
  │     └── agent.BuildPlanRunner()             -- 构建计划模式 Runner
  ├── ctxEngine.PrepareMessages()               -- 应用上下文窗口化
  │     └── WindowMessages()                    -- 截断/裁剪消息历史
  └── goroutine:
        ├── runner.Run(ctx, messages, checkpointID)
        ├── processEvents(events) 循环:
        │     ├── TransferToAgent -> emit agent:timeline {type:"transfer"}
        │     ├── ToolCall -> emit agent:timeline {type:"tool_call"}
        │     ├── ToolResult -> emit agent:timeline {type:"tool_result"}
        │     ├── MessageStream -> drainStream() -> emit agent:timeline {type:"stream_chunk"} x N
        │     ├── Message -> emit agent:timeline {type:"message"}
        │     └── Interrupt -> handleInterrupt() -> emit agent:interrupt
        ├── ctxEngine.AddAssistantMessage(lastContent)
        ├── emit agent:done
        └── onAgentDone() -> sessionService.SaveCurrentSession()
```

### 1.4 Agent 内部执行流程 (默认模式)

```
Runner.Run(messages)
  └── DeepAgent (coding_agent) 接收消息
        ├── LLM 推理，决定下一步动作
        ├── [工具调用] 直接使用 ask_user/ask_choice/write_todos/update_todo/notify_user/MCP工具
        ├── [子智能体] transfer_to_agent -> code_writer / code_executor / file_manager
        │     ├── code_writer: 使用 shell 命令编写代码文件
        │     ├── code_executor: 使用 shell 命令执行代码
        │     └── file_manager: 使用 shell 命令管理文件
        ├── [中断] ask_user/ask_choice -> StatefulInterrupt -> 暂停等待用户
        └── 循环直至任务完成或达到 MaxIteration(50)
```

### 1.5 Agent 内部执行流程 (计划模式)

```
PlanRunner.Run(messages)
  └── PlanExecute Agent
        ├── Planner: 分析任务，生成步骤计划 -> emit agent:plan
        ├── 逐步执行:
        │     ├── 选取下一个 todo 步骤 -> status: doing
        │     ├── DeepAgent 执行该步骤 (同默认模式)
        │     ├── 步骤完成 -> status: done
        │     └── emit agent:plan (更新计划状态)
        ├── Replanner: 评估进度，必要时调整计划
        └── 循环直至所有步骤完成或达到 MaxIterations(20)
```

---

## 二、分支流程

### 2.1 中断/恢复流程 (ask_user)

```
Agent 调用 ask_user 工具
  ├── StatefulInterrupt 触发
  ├── ChatService.handleInterrupt()
  │     ├── 保存 PendingInterrupt {checkpointID, interruptID, info}
  │     └── emit agent:interrupt {type:"followup", questions:[...]}
  ├── 前端显示追问对话框
  ├── 用户输入回答
  └── ChatService.ResumeWithAnswer(answer)
        ├── runner.ResumeWithParams(checkpointID, targets)
        └── 继续 processEvents 循环
```

### 2.2 中断/恢复流程 (ask_choice)

```
Agent 调用 ask_choice 工具
  ├── StatefulInterrupt 触发
  ├── ChatService.handleInterrupt()
  │     ├── 保存 PendingInterrupt
  │     └── emit agent:interrupt {type:"choice", question, options:[{label,description}]}
  ├── 前端显示选择对话框
  ├── 用户选择选项
  └── ChatService.ResumeWithChoice(selectedIndex)
        ├── runner.ResumeWithParams(checkpointID, targets)
        └── 继续 processEvents 循环
```

### 2.3 会话切换流程

```
用户选择另一个会话
  └── SessionService.SwitchSession(sessionID)
        ├── 保存当前会话 (消息 + 时间线)
        ├── 加载目标会话数据
        ├── 重置 ctxEngine (加载历史消息)
        ├── ChatService.InvalidateRunner()  -- 使 Runner 失效
        ├── emit session:switched {session, containerID}
        └── [若目标会话绑定了容器]
              └── onSessionSwitch(containerRegID)
                    └── SandboxService.ConnectExisting(containerRegID)
```

### 2.4 容器重连流程

```
SandboxService.ConnectExisting(containerRegID)
  ├── 从 ContainerStore 获取容器信息 (dockerID, sshHost, wsPath)
  ├── SSHClient.Connect() -- 建立新 SSH 连接
  ├── RemoteDockerManager -- 关联已存在的容器
  ├── 组装 SandboxManager
  ├── onConnect(manager) -> ChatService.UpdateSandbox()
  └── emit sandbox:ready
```

### 2.5 模式切换流程

```
用户切换模式 (default <-> plan)
  └── ChatService.SetMode(mode)
        ├── 校验 mode 值 ("default" 或 "plan")
        ├── 更新内部 mode 字段
        └── emit agent:mode_changed {mode}
```

### 2.6 文件上传流程

```
用户点击上传
  └── FileService.SelectAndUploadFile()
        ├── wailsruntime.OpenFileDialog()     -- 原生文件选择对话框
        └── FileService.UploadFile(localPath)
              ├── transfer.UploadToContainer(localPath, containerPath, docker)
              │     ├── SFTP 上传到远程 /tmp/eino-upload-xxx
              │     ├── docker cp 从远程 host 到容器内
              │     └── 清理远程临时文件
              └── 返回 FileInfoDTO {name, path, size}
```

### 2.7 文件下载流程

```
用户选择文件下载
  └── FileService.DownloadFile(containerPath)
        ├── wailsruntime.SaveFileDialog()     -- 原生保存对话框
        └── transfer.DownloadFromContainer(containerPath, localPath, docker)
              ├── docker cp 从容器到远程 /tmp/eino-download-xxx
              ├── SFTP 下载到本地
              └── 清理远程临时文件
```

---

## 三、异常流程

### 3.1 SSH 断连

```
SSH 连接中断
  ├── HealthMonitor 检测到断连
  ├── emit sandbox:disconnected
  ├── ChatService.UpdateSandbox(nil)  -- 清空沙箱引用
  ├── ChatService.InvalidateRunner()  -- 使 Runner 失效
  └── 用户需手动重连
        └── SandboxService.Connect() 或 SandboxService.Reconnect()
```

### 3.2 容器崩溃/停止

```
容器异常退出
  ├── HealthMonitor 或 Operator 命令失败检测到
  ├── ContainerStore 更新容器状态为 stopped
  └── 用户可选择:
        ├── 重启容器: ContainerService.StartContainer(regID)
        └── 新建容器: SandboxService.Connect()
```

### 3.3 LLM 请求失败

```
LLM API 调用返回错误
  ├── Agent event.Err 非空
  ├── processEvents 捕获错误
  ├── emit agent:timeline {type:"info", content:"Error: ..."}
  └── 事件循环继续 (不中断，后续事件可能恢复)
```

### 3.4 工具执行失败

```
工具调用返回错误
  ├── 工具返回错误信息作为 tool result
  ├── Agent 接收错误信息，决定:
  │     ├── 重试 (修改参数后再次调用)
  │     ├── 换用其他工具
  │     ├── 向用户报告错误
  │     └── 使用 ask_user 向用户求助
  └── emit agent:timeline {type:"tool_result", content:"Error: ..."}
```

### 3.5 MCP 服务器连接失败

```
MCP 服务器不可达
  ├── tools.ConnectMCPServer() 返回错误
  ├── emit agent:error "MCP server {name} connection failed: {err}"
  ├── 跳过该 MCP 服务器，继续处理下一个
  └── Agent 正常运行，但缺少该 MCP 服务器提供的工具
      (非致命错误，记录为警告)
```

### 3.6 应用关闭流程

```
用户关闭窗口
  └── app.shutdown(ctx)
        ├── sessionService.SaveCurrentSession()  -- 保存当前会话
        ├── sandboxService.Disconnect()           -- 断开 SSH (保留容器)
        └── logger.Close()                        -- 关闭日志
```
