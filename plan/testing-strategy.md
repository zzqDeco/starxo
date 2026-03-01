# 测试策略路线图

## 现状分析

当前项目零测试基础设施。所有功能依赖手动验证，没有任何自动化测试覆盖。主要风险：

- Go 后端（`internal/` 下 16+ 个包）无任何 `_test.go` 文件
- Vue 3 前端（`frontend/src/` 下 30+ 个组件/store/composable）无测试配置
- 没有 CI/CD 集成，代码变更无法自动验证
- 核心逻辑（上下文窗口管理、工具注册、配置加载）缺乏回归保护

---

## Phase 1: Go 单元测试（优先级 P1）

**目标**：为核心纯逻辑包建立单元测试，使用 `testing` 标准库 + `testify`。

### 优先测试目标

| 包 | 关键测试点 | 复杂度 |
|----|-----------|--------|
| `internal/config` | `DefaultConfig()` 默认值、`config.json` 加载/保存、字段校验 | 低 |
| `internal/context/windowing` | `WindowMessages()` 消息裁剪逻辑、`TruncateContent()` 截断边界 | 中 |
| `internal/tools/registry` | `RegisterBuiltin/RegisterCustom/RegisterMCPTools` 注册与检索、`Remove` 删除、并发安全 | 中 |
| `internal/model` | `Session`/`Message`/`Container` 数据结构序列化 | 低 |
| `internal/store/checkpoint` | `InMemoryStore` 的 Set/Get 操作、并发读写 | 低 |
| `internal/context/engine` | `PrepareMessages()` 消息组装、`ExportMessages`/`ImportMessages` 序列化往返 | 中 |

### 具体示例

```go
// internal/context/windowing_test.go
func TestWindowMessages_WithinBudget(t *testing.T) {
    // 消息数 <= MaxMessages 时，只截断长内容，不丢弃消息
}

func TestWindowMessages_ExceedsBudget(t *testing.T) {
    // 消息数 > MaxMessages 时，保留首尾消息，中间插入占位符
}

func TestTruncateContent_PreservesHeadAndTail(t *testing.T) {
    // 截断后保留前 60% 和后 20%
}
```

### 依赖

- `go get github.com/stretchr/testify`
- 无需 mock 框架，这些包都是纯逻辑

---

## Phase 2: Go 集成测试（优先级 P1）

**目标**：测试涉及外部依赖的核心业务流程，使用 mock/stub 隔离外部系统。

### Mock 策略

| 外部依赖 | Mock 方式 | 测试目标 |
|----------|----------|----------|
| SSH 连接 (`sandbox.SSHClient`) | 接口抽象 + mock 实现 | `SandboxManager` 连接/断开/重连流程 |
| Docker 命令 (`RemoteDockerManager`) | mock `RunCommand` 返回预设输出 | 容器创建/启动/停止/删除 |
| LLM 调用 (`model.ToolCallingChatModel`) | mock Eino `ChatModel` 接口 | `BuildDeepAgent` 构建流程、Agent 工具调用 |
| `commandline.Operator` | mock 实现 | 子 Agent（code_writer/code_executor/file_manager）工具执行 |
| SFTP (`FileTransfer`) | mock SFTP client | 文件上传/下载/容器传输 |

### 关键集成测试场景

- **Agent 构建流程**：`BuildDeepAgent()` 能正确创建 deep agent 及 3 个子 Agent
- **Sandbox 生命周期**：`Connect -> IsConnected -> Disconnect` 完整流程
- **工具注册与发现**：`ToolRegistry` 多源工具（builtin + MCP + custom）的完整 CRUD
- **上下文引擎**：`Engine.PrepareMessages()` 在不同 token 预算下的消息窗口行为

### 构建标签

```go
//go:build integration
```

集成测试使用 `go test -tags=integration` 运行，默认 `go test` 不触发。

---

## Phase 3: 前端单元测试 Vitest（优先级 P2）

**目标**：为 Vue 3 前端建立 Vitest 测试环境。

### 环境搭建

```bash
cd frontend
npm install -D vitest @vue/test-utils happy-dom
```

### 优先测试目标

| 模块 | 测试点 |
|------|--------|
| `stores/chatStore.ts` | 消息添加/清除、流式消息更新、会话切换 |
| `stores/sessionStore.ts` | 会话 CRUD、当前会话切换 |
| `stores/settingsStore.ts` | 配置加载/保存、默认值回退 |
| `stores/connectionStore.ts` | 连接状态管理、重连逻辑 |
| `composables/useHelpers.ts` | 工具函数 |
| `composables/useWailsEvent.ts` | Wails 事件监听/取消（mock `runtime`） |

### Mock 策略

- Wails runtime 方法（`EventsOn`, `EventsEmit` 等）需要 mock
- Go 绑定方法（`Chat`, `Connect` 等）需要 mock
- 使用 `vi.mock()` 替换 `@wailsapp/runtime` 模块

---

## Phase 4: E2E 测试 Playwright（优先级 P3）

**目标**：对完整 Wails 应用进行端到端测试。

### 挑战

- Wails 应用打包为桌面程序，需要特殊的启动方式
- 需要真实或模拟的 SSH 服务器和 Docker 环境
- LLM 调用需要 mock 以保证测试确定性

### 方案

1. **开发模式测试**：使用 `wails dev` 启动应用，Playwright 连接到 localhost
2. **Mock 后端**：Go 测试中启动 mock SSH/Docker 服务器
3. **关键流程覆盖**：
   - 设置页面配置 SSH/Docker/LLM
   - 建立沙箱连接
   - 发送消息并收到 Agent 响应
   - 文件浏览和传输
   - 会话管理（创建/切换/删除）

---

## 覆盖率目标

| 阶段 | 目标覆盖率 | 时间预估 |
|------|-----------|----------|
| Phase 1 完成 | Go 核心包 80%+ | 1-2 周 |
| Phase 2 完成 | Go 整体 60%+ | 2-3 周 |
| Phase 3 完成 | 前端 store/composable 70%+ | 1-2 周 |
| Phase 4 完成 | 关键用户流程 100% | 2-4 周 |

## CI 集成计划

- **GitHub Actions**：每次 PR 自动运行 Phase 1-3 测试
- **覆盖率门控**：新代码必须达到 70%+ 覆盖率
- **E2E 定时运行**：每日或每次 release 前运行 Phase 4
- **测试报告**：集成 codecov 或类似工具进行覆盖率趋势跟踪
