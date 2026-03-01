# 性能与可靠性

## 现状分析

当前已知的性能和可靠性问题：

- **上下文窗口估算粗糙**：`context/engine.go` 使用 `maxTokens / 200` 启发式估算消息数量上限，200 token/消息 的假设与实际差距较大
- **无 SSH 连接池**：每次 SFTP 操作创建新 client，每次命令创建新 session
- **无流式批处理**：Timeline events 逐条推送到前端，高频事件时可能造成渲染压力
- **内存检查点**：`store/checkpoint.go` 使用 `inMemoryStore`，应用重启后检查点数据全部丢失
- **文件传输无优化**：`transfer.go` 使用 `io.Copy` 直接传输，大文件无进度报告、无分块、无断点续传
- **无 LLM 请求缓存**：相同上下文重复调用 LLM 时无缓存机制

---

## 改进方向

### 1. 上下文窗口优化（优先级 P1）

**当前实现**（`internal/context/engine.go:78-85`）：

```go
// Rough heuristic: ~200 tokens per message on average.
estimated := e.maxTokens / 200
```

#### 改进方案

- **精确 token 计数**：集成 tiktoken-go 或类似库，按模型实际 tokenizer 计算
- **分层预算**：
  - System prompt：固定预算（如 20% 上下文窗口）
  - 文件上下文：动态预算（根据 `FileContext` 内容）
  - 对话历史：剩余预算
- **智能裁剪**：
  - 优先保留包含工具调用结果的消息（信息密度高）
  - 对纯文本消息更积极地截断
  - 基于消息重要性评分的窗口策略
- **消息摘要**：对被裁剪的消息生成摘要，而不是简单丢弃

---

### 2. 流式性能优化（优先级 P2）

**问题**：Agent 执行时产生大量 timeline events（工具调用、子 Agent 切换、中间推理等），逐条通过 Wails events 推送到前端。

#### 改进方案

- **事件批处理**：
  ```go
  type EventBatcher struct {
      buffer   []TimelineEvent
      interval time.Duration // 如 50ms
      flush    func([]TimelineEvent)
  }
  ```
  - 将 50ms 内的事件聚合为一批发送
  - 减少 Wails runtime 的事件桥接开销
- **虚拟滚动**：前端 timeline 列表使用虚拟滚动，仅渲染可见区域
- **事件去重**：相邻重复事件合并（如连续的 "thinking..." 状态）
- **惰性渲染**：折叠的 timeline 节点不渲染详情，展开时才加载

---

### 3. SSH 连接池和 Keepalive 调优（优先级 P1）

**当前问题**：`SSHClient` 维护单个 `*ssh.Client` 连接，无 keepalive 机制。长时间空闲后连接可能静默断开。

#### 改进方案

- **Keepalive 心跳**：
  ```go
  go func() {
      ticker := time.NewTicker(30 * time.Second)
      for range ticker.C {
          _, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
          if err != nil {
              // 触发重连
          }
      }
  }()
  ```
- **自动重连**：检测连接断开后自动重建 SSH 连接和 Docker exec session
- **SFTP 连接复用**：维护 SFTP client 池，避免每次文件操作创建新 client
- **连接健康检查**：定期验证连接可用性，前端展示连接状态

---

### 4. 容器资源监控和限制（优先级 P2）

**当前配置**（`config.go`）：

```go
Docker: DockerConfig{
    MemoryLimit: 2048,  // MB
    CPULimit:    1.0,
    ...
}
```

#### 改进方案

- **实时资源监控**：通过 `docker stats` 采集 CPU/内存/网络使用率
- **前端资源仪表盘**：在 `ConnectionStatus.vue` 中展示容器资源使用图表
- **资源告警**：内存使用超过 80% 时提醒用户
- **动态限制调整**：允许运行时修改容器资源限制（`docker update`）
- **磁盘空间监控**：监控容器内 `/workspace` 磁盘使用量

---

### 5. LLM 请求缓存和去重（优先级 P2）

#### 改进方案

- **语义缓存**：对相同消息序列的 LLM 响应进行缓存
  - 缓存键：消息内容 hash + 模型名称 + temperature
  - 缓存策略：LRU，按 token 数量限制总缓存大小
  - 仅缓存 temperature=0 的确定性请求
- **请求去重**：同一会话中，如果用户快速重复发送相同消息，只发送一次 LLM 请求
- **流式缓存**：缓存流式响应的完整结果，重放时模拟流式输出

---

### 6. 大文件传输优化（优先级 P2）

**当前实现**：`transfer.go` 使用 `io.Copy` 直接传输，容器传输走 `docker cp`（先 SFTP 到宿主机 /tmp 再 docker cp）。

#### 改进方案

- **分块传输**：大文件分块传输，支持进度报告
  ```go
  type TransferProgress struct {
      FileName    string
      TotalBytes  int64
      SentBytes   int64
      SpeedBPS    float64
  }
  ```
- **并行传输**：多文件时并行 SFTP 传输（需 SFTP 连接池支持）
- **压缩传输**：超过阈值的文件先压缩再传输
- **断点续传**：传输中断后从断点继续（记录已传输偏移量）
- **增量同步**：对工作区文件使用 rsync 式增量同步

---

### 7. 长会话内存泄漏预防（优先级 P1）

#### 风险点

| 组件 | 泄漏风险 | 缓解措施 |
|------|---------|---------|
| `ConversationHistory` | 消息无限积累 | 窗口裁剪（已有），增加硬上限 |
| `TimelineEvent` 前端列表 | 事件无限增长 | 虚拟滚动 + 旧事件归档 |
| `inMemoryStore` 检查点 | 检查点数据累积 | 定期清理过期检查点 |
| SSH session/SFTP client | 未正确关闭 | defer 保障 + 连接池管理 |
| Goroutine 泄漏 | context 取消后仍运行 | 统一 context 取消传播 |

- **内存监控**：集成 `runtime.MemStats` 定期采样
- **GC 优化**：长会话定期触发手动 GC
- **资源限制**：设置会话最大消息数、最大 timeline events 数

---

### 8. 检查点存储持久化（优先级 P1）

**当前实现**：`store/checkpoint.go` 使用 `inMemoryStore`（`map[string][]byte`），应用重启后数据丢失。

#### 改进方案

- **SQLite 后端**：使用 SQLite 存储检查点数据
  ```go
  type SQLiteStore struct {
      db *sql.DB
  }
  ```
- **文件系统后端**：每个检查点存为独立文件，适合简单部署
- **过期清理**：检查点设置 TTL（如 7 天），定期清理过期数据
- **与会话关联**：检查点按 session ID 组织，会话删除时级联清理

---

## 实施优先级总结

| 改进 | 优先级 | 影响范围 | 预估工作量 |
|------|--------|---------|-----------|
| 上下文窗口优化 | P1 | Agent 推理质量 | 中 |
| SSH 连接池和 Keepalive | P1 | 连接稳定性 | 中 |
| 长会话内存泄漏预防 | P1 | 应用稳定性 | 中 |
| 检查点存储持久化 | P1 | 中断恢复能力 | 小 |
| 流式性能优化 | P2 | 前端流畅度 | 中 |
| 容器资源监控 | P2 | 运维可见性 | 小 |
| LLM 请求缓存 | P2 | 成本和延迟 | 中 |
| 大文件传输优化 | P2 | 文件操作体验 | 中 |
