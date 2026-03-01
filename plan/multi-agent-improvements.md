# 多 Agent 架构增强

## 现状分析

当前架构（`internal/agent/deep_agent.go`）：

- **固定 3 个子 Agent**：`code_writer`、`code_executor`、`file_manager`，在 `BuildDeepAgent()` 中硬编码构建
- **串行执行**：子 Agent 通过 Eino ADK 的 `transfer` 机制调用，同一时刻只有一个子 Agent 在工作
- **统一模型**：所有子 Agent 共享同一个 `model.ToolCallingChatModel` 实例
- **固定工具集**：每个子 Agent 的工具在构建时确定，运行时不可变
- **50 轮迭代上限**：`MaxIteration: 50`，适用于所有任务
- **无记忆持久化**：Agent 运行结束后上下文丢失，下次对话需从头开始

---

## 改进方向

### 1. 动态子 Agent 配置（优先级 P2）

**目标**：允许用户自定义子 Agent 角色和能力。

#### 设计

```go
type SubAgentConfig struct {
    Name        string   `json:"name"`
    Description string   `json:"description"`
    Instruction string   `json:"instruction"`
    Tools       []string `json:"tools"`       // 工具名称列表
    Model       string   `json:"model"`       // 可选独立模型
    Enabled     bool     `json:"enabled"`
}
```

- 在 `config.json` 的 `agent` 段中新增 `subAgents []SubAgentConfig`
- `BuildDeepAgent()` 根据配置动态构建子 Agent 列表
- 保留默认 3 个子 Agent 作为内置预设，用户可增删改
- 前端 Settings 中新增 Agent 配置面板

#### 预设扩展

| Agent | 职责 | 核心工具 |
|-------|------|---------|
| `debugger` | 调试和错误分析 | bash, read_file, grep |
| `reviewer` | 代码审查 | read_file, grep, write_file |
| `architect` | 架构设计 | read_file, list_files, bash |

---

### 2. Agent 记忆和上下文共享（优先级 P2）

**目标**：跨会话保留 Agent 学到的知识和偏好。

#### 短期记忆（会话内）

- 当前由 `context.Engine` 管理，`WindowMessages()` 负责裁剪
- 改进：为每个子 Agent 维护独立的工作记忆摘要
- 子 Agent 完成任务后生成摘要，存入共享上下文

#### 长期记忆（跨会话）

- 新增 `internal/memory` 包
- 存储项目级知识：文件结构理解、编码风格偏好、常见错误模式
- 基于向量检索的相关记忆召回（可选，依赖外部向量数据库）
- 简单实现：JSON 文件存储关键事实列表，每次会话开始注入 system prompt

#### 子 Agent 间上下文共享

```go
type SharedContext struct {
    mu       sync.RWMutex
    findings map[string]string // agent_name -> 最新发现/状态
    files    map[string]string // 已修改文件路径 -> 修改摘要
}
```

- deep agent 在 transfer 子 Agent 时传递 `SharedContext`
- 子 Agent 完成任务后更新共享上下文
- 后续子 Agent 可以看到前序 Agent 的工作成果

---

### 3. 并行子 Agent 执行（优先级 P1）

**目标**：当多个子 Agent 的任务互不依赖时，并行执行以提升效率。

#### 当前瓶颈

`deep.New()` 使用 Eino ADK 的 transfer 模式，本质上是 deep agent 调用一个子 Agent，等其完成后再继续。对于"同时读多个文件再合并"这类场景效率低下。

#### 方案

1. **任务依赖分析**：deep agent 在 planning 阶段识别可并行的子任务
2. **并行调度器**：新增 `ParallelExecutor` 组件
   ```go
   type ParallelExecutor struct {
       maxConcurrency int
       results        chan AgentResult
   }
   ```
3. **结果合并**：并行任务完成后，deep agent 汇总结果并继续推理
4. **冲突检测**：当两个子 Agent 修改同一文件时，需要冲突解决策略

#### 限制

- 并行执行消耗更多 LLM API 调用
- 需要确保 `commandline.Operator`（`RemoteOperator`）支持并发命令执行
- 当前 `RemoteOperator` 通过 `docker exec` 执行命令，需验证并发安全性

---

### 4. Agent 间直接通信（优先级 P3）

**目标**：允许子 Agent 之间直接交换信息，而不是全部通过 deep agent 中转。

#### 当前流程

```
用户 -> deep_agent -> transfer(code_writer) -> 结果返回 deep_agent -> transfer(code_executor)
```

#### 改进后流程

```
用户 -> deep_agent -> code_writer --直接通知--> code_executor
                                   \-> 结果返回 deep_agent
```

#### 实现方式

- 基于事件总线的消息传递
- 子 Agent 可发布事件（如"文件已修改"），其他 Agent 订阅
- deep agent 保留最终决策权，直接通信仅用于辅助信息传递

---

### 5. Agent 性能监控和指标（优先级 P2）

**目标**：可视化 Agent 行为，帮助用户理解和优化 Agent 表现。

#### 指标采集

| 指标 | 说明 |
|------|------|
| 每个子 Agent 的调用次数 | 判断任务分配是否合理 |
| 每次工具调用耗时 | 识别性能瓶颈 |
| LLM token 消耗 | 成本统计 |
| 任务成功/失败率 | 可靠性评估 |
| 迭代次数 vs 任务复杂度 | 效率分析 |

#### 展示方式

- 在前端 `AgentStatus.vue` 中增加详细指标面板
- Timeline 中展示每个子 Agent 的执行时间线
- 会话结束后生成成本和效率报告

---

## 实施优先级总结

| 改进 | 优先级 | 依赖 | 预估工作量 |
|------|--------|------|-----------|
| 并行子 Agent 执行 | P1 | Eino ADK 支持 | 大 |
| 动态子 Agent 配置 | P2 | 配置系统扩展 | 中 |
| Agent 记忆和上下文共享 | P2 | 存储方案 | 中 |
| 性能监控和指标 | P2 | 前端面板 | 中 |
| Agent 间直接通信 | P3 | 事件总线 | 大 |
