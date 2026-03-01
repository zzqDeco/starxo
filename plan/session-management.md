# 会话管理增强

## 现状分析

当前会话管理由以下组件实现：

- **后端**：`internal/storage/session_store.go` 存储会话数据，`internal/service/session_svc.go` 提供 CRUD 服务
- **前端**：`stores/sessionStore.ts` 管理会话状态，`Sidebar.vue` 展示会话列表
- **检查点**：`store/checkpoint.go` 使用 `inMemoryStore`（纯内存 `map[string][]byte`），重启后丢失
- **上下文**：`context/engine.go` 的 `ExportMessages`/`ImportMessages` 支持对话序列化
- **模型**：`model/session.go` 定义 `Session` 数据结构

### 已有能力

- 会话创建、列出、切换、删除
- 对话历史持久化（通过 `PersistedMessage`）
- 容器关联（每个会话绑定一个 Docker 容器）

### 不足之处

- 无会话搜索/过滤功能
- 无会话导出/导入
- 无对话分支能力（无法从某条消息 fork 出新对话）
- 检查点仅内存存储，重启后 interrupt/resume 状态丢失
- 所有会话共享全局 LLM/SSH 配置，无法按会话定制

---

## 改进方向

### 1. 会话导出/导入（优先级 P2）

**目标**：支持将会话导出为可传输格式，在不同设备间共享。

#### 导出格式

```json
{
  "version": "1.0",
  "exportedAt": "2026-03-01T12:00:00Z",
  "session": {
    "id": "...",
    "name": "...",
    "createdAt": "...",
    "messages": [...],
    "metadata": {
      "llmModel": "gpt-4o",
      "containerImage": "python:3.11-slim"
    }
  },
  "checkpoints": [...],
  "files": [...]
}
```

#### 功能点

- **完整导出**：包含对话历史、检查点数据、工作区文件列表
- **选择性导出**：只导出对话历史（轻量），或包含沙箱文件（完整）
- **文件打包**：沙箱内的工作区文件可选择性打包为 tar.gz 附带导出
- **导入冲突处理**：导入时检测 session ID 冲突，支持覆盖或创建副本
- **格式兼容性**：导出文件包含版本号，向后兼容

---

### 2. 会话搜索和过滤（优先级 P1）

#### 搜索能力

- **关键词搜索**：搜索会话名称和对话内容
- **时间范围过滤**：按创建时间、最后活跃时间筛选
- **状态过滤**：按容器状态（运行中/已停止/已销毁）筛选
- **标签系统**：用户可为会话添加自定义标签，按标签过滤

#### 前端实现

- `Sidebar.vue` 顶部增加搜索框
- 支持模糊搜索和高亮匹配
- 过滤器下拉菜单（时间、状态、标签）
- 搜索结果中显示匹配的消息片段预览

#### 后端支持

- `SessionStore` 增加 `Search(query string, filters SessionFilters) []Session`
- 使用 SQLite FTS5 全文搜索（如果后端存储迁移到 SQLite）
- 或简单的内存过滤（当前会话数量不大时足够）

---

### 3. 对话分支（优先级 P3）

**目标**：从对话中的任意消息 fork 出新的对话分支，探索不同的编码路径。

#### 使用场景

- Agent 给出了不满意的方案，想从之前的消息重新开始
- 想对比两种不同的实现方案
- 回到某个检查点，尝试不同的指令

#### 数据模型

```go
type Session struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    ParentID    string    `json:"parentId,omitempty"`    // 新增：父会话 ID
    ForkPointID string    `json:"forkPointId,omitempty"` // 新增：fork 起始消息 ID
    CreatedAt   time.Time `json:"createdAt"`
    // ...
}
```

#### 功能点

- **Fork 操作**：右键消息 -> "从此处创建分支"
- **分支树可视化**：在 Sidebar 中以树状结构展示会话及其分支
- **分支比较**：side-by-side 对比两个分支的对话和结果
- **合并**：将分支中的代码变更合并回主会话（高级功能，远期）

#### 实现注意

- fork 时复制消息历史到 fork 点，之后独立
- 检查点数据也需要从 fork 点复制
- 容器状态：新分支可以复用容器（共享工作区）或创建新容器（隔离工作区）

---

### 4. 持久化检查点存储（优先级 P1）

**当前问题**：`inMemoryStore`（`store/checkpoint.go`）使用 `map[string][]byte`，应用关闭后数据丢失，导致 interrupt/resume 功能无法跨重启工作。

#### 改进方案

**方案 A：SQLite 存储**（推荐）

```go
type SQLiteCheckpointStore struct {
    db *sql.DB
}

// 表结构
// CREATE TABLE checkpoints (
//     key TEXT PRIMARY KEY,
//     value BLOB NOT NULL,
//     session_id TEXT,
//     created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
//     updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
// )
```

**方案 B：文件系统存储**

```go
type FileCheckpointStore struct {
    baseDir string // ~/.starxo/checkpoints/
}
// 每个检查点存为独立文件：{baseDir}/{key}.bin
```

#### 清理策略

- 检查点关联 session ID，会话删除时级联清理
- 设置 TTL（如 7 天），定期清理过期检查点
- 容量限制（如最多 100MB），超出时 LRU 淘汰

#### 兼容性

- `compose.CheckPointStore` 接口不变（`Set`/`Get`），只替换底层实现
- 首次启动自动创建存储结构
- 从内存迁移无需数据迁移（内存数据本身不持久）

---

### 5. 按会话配置（优先级 P2）

**目标**：每个会话可以有独立的 LLM 模型、工作目录等配置。

#### 配置层级

```
全局默认配置 (config.json)
  └── 会话级覆盖 (session config)
```

#### 可覆盖项

| 配置项 | 说明 | 使用场景 |
|--------|------|---------|
| LLM 模型 | 不同会话使用不同模型 | 简单任务用小模型、复杂任务用大模型 |
| 容器镜像 | 不同语言项目使用不同基础镜像 | Python 项目 vs Node.js 项目 |
| 工作目录 | 容器内的默认工作路径 | 多项目并行开发 |
| Agent 迭代上限 | 简单任务减少上限避免浪费 | 按任务复杂度调整 |
| System Prompt 附加 | 会话级 prompt 扩展 | 特定项目的编码规范 |

#### 数据模型扩展

```go
type SessionConfig struct {
    LLMModel      string `json:"llmModel,omitempty"`
    ContainerImage string `json:"containerImage,omitempty"`
    WorkDir       string `json:"workDir,omitempty"`
    MaxIterations int    `json:"maxIterations,omitempty"`
    ExtraPrompt   string `json:"extraPrompt,omitempty"`
}
```

- 为空字段回退到全局配置
- 前端会话设置弹窗，可修改当前会话的配置

---

### 6. 数据迁移和向后兼容策略（优先级 P1）

**目标**：确保版本升级时用户数据不丢失。

#### 版本化存储

```go
const CurrentSchemaVersion = 2

type StorageHeader struct {
    SchemaVersion int    `json:"schemaVersion"`
    AppVersion    string `json:"appVersion"`
    CreatedAt     string `json:"createdAt"`
}
```

#### 迁移框架

```go
type Migration struct {
    FromVersion int
    ToVersion   int
    Migrate     func(data []byte) ([]byte, error)
}

var migrations = []Migration{
    {1, 2, migrateV1ToV2}, // 例如：添加 SessionConfig 字段
}
```

#### 原则

- **只增不删**：新增字段使用 `omitempty`，旧数据自动兼容
- **自动迁移**：应用启动时检测 schema 版本，自动执行迁移链
- **备份优先**：迁移前自动备份原始数据文件
- **迁移日志**：记录每次迁移的详情（版本、时间、变更内容）

---

## 实施优先级总结

| 改进 | 优先级 | 用户价值 | 预估工作量 |
|------|--------|---------|-----------|
| 会话搜索和过滤 | P1 | 高（会话多时必需） | 小 |
| 持久化检查点存储 | P1 | 高（中断恢复能力） | 小 |
| 数据迁移框架 | P1 | 高（升级安全保障） | 中 |
| 会话导出/导入 | P2 | 中（跨设备共享） | 中 |
| 按会话配置 | P2 | 中（灵活性提升） | 中 |
| 对话分支 | P3 | 中（探索性开发） | 大 |
