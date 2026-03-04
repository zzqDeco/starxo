# 业务流程

> 所属项目: Starxo | 文档类型: 业务流程

---

## 一、主流程

### 1.1 应用启动流程

```
main.go: NewApp()
  ├── config.NewStore()           -- 加载/创建 ~/.starxo/config.json
  ├── storage.NewSessionStore()   -- 加载/创建 ~/.starxo/sessions/
  ├── storage.NewContainerStore() -- 加载/创建 ~/.starxo/containers.json
  ├── 创建 6 个 Service 实例
  └── wails.Run()
        └── app.startup(ctx)
              ├── logger.Init()                      -- 初始化文件日志
              ├── logger.RegisterGlobalCallbacks()    -- 注册 Eino 全局回调
              ├── 所有 Service.SetContext(ctx)        -- 分发 Wails context
              ├── 连接 Service 间依赖 (回调注册)
              │     ├── chatService.SetDependencies(sbx, nil)  -- ctxEngine 传 nil（per-session 管理）
              │     ├── chatService.SetSessionService()
              │     ├── sessionService.SetChatService()        -- 注入 ChatService 用于 per-session 状态访问
              │     ├── sandboxService.SetOnConnect()       -> chatService.UpdateSandbox()
              │     ├── sandboxService.SetOnContainerBound() -> sessionService.BindContainer()
              │     ├── sandboxService.SetOnContainerDeactivated() -> chatService.UpdateSandbox(nil)
              │     ├── sessionService.SetOnSessionSwitch()  -> sandboxService.ActivateContainer/DeactivateContainer
              │     ├── sessionService.SetOnDestroyContainer() -> containerService.DestroyContainer
              │     ├── settingsService.SetOnSettingsSave()   -> chatService.InvalidateRunner()
              │     └── chatService.SetOnAgentDone(fn(sessionID))  -> sessionService.SaveCurrentSession()
              ├── sessionService.EnsureDefaultSession()  -- 加载或创建默认会话，恢复数据到 per-session run
              ├── chatService.SetActiveSessionID()       -- 同步活跃会话到 ChatService
              └── sandboxService.StartHealthMonitor()    -- 启动后台健康检查
```

### 1.2 沙箱连接流程

```
用户点击"连接"
  └── SandboxService.ConnectSSH() + CreateAndActivateContainer()
        ├── [SSH 阶段]
        │     ├── 断开已有连接 (保留容器)
        │     ├── emit ssh:progress {step:"Connecting SSH...", percent:10}
        │     ├── SSHClient.Connect()           -- 建立 SSH 连接
        │     ├── emit ssh:progress {step:"SSH connected", percent:50}
        │     └── EnsureDocker()                -- 确认 Docker 可用
        ├── emit ssh:connected
        ├── [容器阶段]
        │     ├── emit container:progress {step:"Creating container...", percent:10}
        │     ├── CreateNewContainer()          -- 创建 Docker 容器
        │     ├── 注册容器到 ContainerStore
        │     ├── onConnect(manager)            -> ChatService.UpdateSandbox()
        │     ├── onContainerBound(regID, wsPath) -> SessionService.BindContainer()
        │     └── emit container:ready {containerID: regID}
        └── setupOutputForwarding()             -- 设置终端输出转发
```

### 1.3 用户消息处理流程 (Per-Session)

```
用户输入消息 -> ChatService.SendMessage(userMessage)
  ├── 检查 activeSessionID 非空
  ├── 获取 per-session SessionRun (getOrCreateRun)
  ├── per-session 运行守卫: run.running == false
  ├── run.ctxEngine.AddUserMessage(userMessage)     -- 消息加入 per-session 上下文
  ├── run.timeline.AddUserTurn()                    -- 记录到 per-session 时间线
  ├── [首次] buildRunnersLocked() (在锁内执行)      -- 构建 Agent 和 Runner
  │     ├── llm.NewChatModel()                      -- 创建 LLM 模型
  │     ├── tools.RegisterBuiltinTools()            -- 注册内置工具
  │     ├── tools.ConnectMCPServer() x N            -- 连接 MCP 服务器
  │     ├── agent.BuildDeepAgent()                  -- 构建 Deep Agent + 3 子智能体
  │     │     └── buildAgentContext()               -- OnToolEvent 通过 ctx 传播 sessionID
  │     ├── agent.BuildDefaultRunner()              -- 构建默认模式 Runner
  │     └── agent.BuildPlanRunner()                 -- 构建计划模式 Runner
  ├── 创建带 sessionID 的 cancellable context
  │     └── contextWithSessionID(runCtx, sessionID)
  ├── run.ctxEngine.PrepareMessages()               -- 应用上下文窗口化
  └── goroutine:
        ├── runner.Run(runCtx, messages, checkpointID)
        ├── processEventsForRun(events, checkpointID, run) 循环:
        │     ├── TransferToAgent -> emitTimelineForRun {type:"transfer"} + {type:"thinking"}
        │     ├── ToolCall -> emitTimelineForRun {type:"reasoning"} + {type:"tool_call"}
        │     ├── ToolResult -> emitTimelineForRun {type:"tool_result"} + {type:"thinking"}
        │     ├── MessageStream -> drainStreamForRun() -> emitTimelineForRun {type:"stream_chunk"} x N
        │     ├── Message -> emitTimelineForRun {type:"message"}
        │     ├── Interrupt -> handleInterruptForRun() -> emit agent:interrupt {sessionId}
        │     └── 孤立 tool_call 检测 -> 注入合成错误响应
        ├── run.ctxEngine.AddAssistantMessage(lastContent)
        ├── emit agent:done {sessionId}
        └── onAgentDone(sessionID) -> sessionService.SaveCurrentSession()
```

### 1.4 Agent 内部执行流程 (默认模式)

```
Runner.Run(messages)
  └── DeepAgent (coding_agent) 接收消息
        ├── LLM 推理，决定下一步动作
        ├── [工具调用] 直接使用 ask_user/ask_choice/write_todos/update_todo/notify_user/MCP工具
        ├── [子智能体] transfer_to_agent -> code_writer / code_executor / file_manager
        │     ├── code_writer: 使用 shell 命令编写代码文件
        │     │     └── eventEmittingTool 通过 ctx 传播 sessionID -> emitTimelineForSession
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
  ├── ChatService.handleInterruptForRun(run)
  │     ├── 保存 PendingInterrupt {checkpointID, interruptID, info} 到 run
  │     └── emit agent:interrupt {type:"followup", questions:[...], sessionId}
  ├── 前端 isActiveSession() 过滤后显示追问对话框
  ├── 用户输入回答
  └── ChatService.ResumeWithAnswer(answer)
        ├── 获取活跃会话的 run 和 pendingInterrupt
        ├── 创建带 sessionID 的 context
        ├── runner.ResumeWithParams(checkpointID, targets)
        └── 继续 processEventsForRun 循环
```

### 2.2 中断/恢复流程 (ask_choice)

```
Agent 调用 ask_choice 工具
  ├── StatefulInterrupt 触发
  ├── ChatService.handleInterruptForRun(run)
  │     ├── 保存 PendingInterrupt 到 run
  │     └── emit agent:interrupt {type:"choice", question, options:[{label,description}], sessionId}
  ├── 前端 isActiveSession() 过滤后显示选择对话框
  ├── 用户选择选项
  └── ChatService.ResumeWithChoice(selectedIndex)
        ├── 获取活跃会话的 run 和 pendingInterrupt
        ├── 创建带 sessionID 的 context
        ├── runner.ResumeWithParams(checkpointID, targets)
        └── 继续 processEventsForRun 循环
```

### 2.3 会话切换流程 (Per-Session)

```
用户选择另一个会话
  └── SessionService.SwitchSession(sessionID)
        ├── 保存当前会话 (per-session 消息 + 时间线 + 流式状态)
        │     └── saveCurrentLocked() -> chatService.SessionCtxEngine/SessionTimeline/SessionStreamingState
        ├── chatService.SetActiveSessionID(sessionID)
        ├── [若会话未运行中] 加载目标会话数据到 per-session run
        │     ├── chatService.GetOrCreateRun(sessionID)
        │     ├── run.ctxEngine.ImportMessages(sessionData.Messages)
        │     └── run.timeline.Import(sessionData.Display)
        ├── [若会话已运行中] 保持内存中的 run 不变（最新状态）
        ├── tools.ClearTodos()
        ├── 构建 SessionSwitchedEvent (含完整状态快照)
        │     └── chatService.GetSessionRunSnapshot(sessionID)
        │           -> running, currentAgent, mode, interrupt
        ├── emit session:switched {session, containerID, agentRunning, currentAgent, mode, hasInterrupt, interrupt}
        └── [若目标会话绑定了容器]
              └── onSessionSwitch(containerRegID)
                    └── go sandboxService.ActivateContainer(containerRegID)
```

### 2.4 会话删除流程 (Per-Session)

```
用户删除会话
  └── SessionService.DeleteSession(sessionID)
        ├── 检查非活跃会话
        ├── [若会话有运行中代理]
        │     ├── chatService.StopSessionGeneration(sessionID)
        │     └── chatService.WaitForSessionDone(sessionID, 10s)
        ├── chatService.RemoveSession(sessionID) -- 清理 per-session 状态
        ├── 加载会话获取容器列表
        ├── 级联销毁所有子容器 (best-effort)
        │     └── onDestroyContainer(cid) x N
        └── sessionStore.Delete(sessionID)
```

### 2.5 容器重连流程

```
SandboxService.ConnectExisting(containerRegID)
  ├── 从 ContainerStore 获取容器信息 (dockerID, sshHost, wsPath)
  ├── [若 SSH 未连接] ConnectSSH() + EnsureDocker()
  └── ActivateContainer(containerRegID)
        ├── 验证 SSH Host 匹配
        ├── AttachToContainer()
        ├── onConnect(manager) -> ChatService.UpdateSandbox()
        ├── onContainerBound(regID, wsPath) -> SessionService.BindContainer()
        └── emit container:activated {containerID}
```

### 2.6 模式切换流程

```
用户切换模式 (default <-> plan)
  └── ChatService.SetMode(mode)
        ├── 校验 mode 值 ("default" 或 "plan")
        ├── 更新 per-session run.mode 字段
        └── emit agent:mode_changed {mode, sessionId}
```

### 2.7 文件上传流程

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

### 2.8 文件下载流程

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
  ├── HealthMonitor.healthCheck() 检测到断连 (短暂 RLock)
  ├── 清理 manager 和 activeContainerRegID (短暂 Lock)
  ├── emit ssh:disconnected
  └── 用户需手动重连
        └── SandboxService.ConnectSSH() + CreateAndActivateContainer()
```

### 3.2 容器崩溃/停止

```
容器异常退出
  ├── HealthMonitor 或 Operator 命令失败检测到
  ├── ContainerStore 更新容器状态为 stopped
  └── 用户可选择:
        ├── 重启容器: ContainerService.StartContainer(regID)
        └── 新建容器: SandboxService.CreateAndActivateContainer()
```

### 3.3 LLM 请求失败

```
LLM API 调用返回错误
  ├── Agent event.Err 非空
  ├── processEventsForRun 捕获错误
  ├── emitTimelineForRun {type:"info", content:"Error: ...", sessionId}
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
  └── emitTimelineForRun {type:"tool_result", content:"Error: ...", sessionId}
```

### 3.5 MCP 服务器连接失败

```
MCP 服务器不可达
  ├── tools.ConnectMCPServer() 返回错误
  ├── emit agent:error {sessionId, error: "MCP server {name} connection failed: {err}"}
  ├── 跳过该 MCP 服务器，继续处理下一个
  └── Agent 正常运行，但缺少该 MCP 服务器提供的工具
      (非致命错误，记录为警告)
```

### 3.6 应用关闭流程

```
用户关闭窗口
  └── app.shutdown(ctx)
        ├── sessionService.SaveCurrentSession()  -- 保存当前会话 (per-session 持久化)
        ├── sandboxService.Disconnect()           -- 断开 SSH (保留容器)
        └── logger.Close()                        -- 关闭日志
```

### 3.7 后台会话代理运行

```
用户切换到新会话，但旧会话代理仍在运行
  ├── SwitchSession 不取消旧会话代理
  ├── 旧会话 run.running == true, run.runDone channel 未关闭
  ├── 后台代理事件通过 emitTimelineForRun 发射 (携带旧 sessionId)
  ├── 前端 isActiveSession() 过滤掉非活跃会话事件
  ├── 后台代理完成 -> onAgentDone(sessionID) -> SaveCurrentSession
  └── 切换回旧会话时 -> IsSessionRunning() == true -> 不从磁盘重新加载 (使用内存中最新状态)
```
