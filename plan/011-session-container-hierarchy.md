# 011 - 会话与容器父子关系重构

## 目标

重构会话（Session）与容器（Container/Docker）的关系模型，建立严格的父子层级结构，解决当前平级设计导致的管理混乱和资源泄漏问题。

## 问题分析

### 现状
- `Session.ContainerID` 单一字段关联一个容器，但无唯一性约束，多个会话可共用同一容器
- 容器生命周期独立于会话：删除会话不清理容器，造成孤儿容器和资源浪费
- 切换到空会话时，前端不清空上一会话的聊天记录，界面残留误导用户
- 全局只有一个 `SandboxManager`，切换到无容器会话时旧连接仍然存活

### 目标
- 会话与容器严格父子：容器归属于创建它的会话，不允许共用
- 一个会话可拥有多个容器，每个容器独立管理状态
- 删除会话时级联清理所有子容器（远程停止+销毁+注册表移除）
- 切换到空会话时前端正确清空消息和连接状态

## 具体任务

### Phase 1: 数据模型重构

- [x] **`internal/model/session.go`**: 将 `ContainerID string` 改为 `Containers []string`（容器注册 ID 列表），新增 `ActiveContainerID string` 标识当前活跃容器
- [x] **`internal/model/container.go`**: 新增 `SessionID string` 字段，记录所属会话 ID，建立双向引用
- [x] **`frontend/src/types/session.ts`**: 同步更新 `Session` 接口，`containerID` → `containers: string[]` + `activeContainerID: string`；扩展富化字段为容器数组

### Phase 2: 存储层适配

- [x] **`internal/storage/session_store.go`**: 适配新字段的序列化/反序列化，兼容旧格式迁移（单 `containerID` → `containers` 数组）
- [x] **`internal/storage/container_store.go`**: 新增 `FindBySessionID(sessionID) []Container` 方法

### Phase 3: 服务层逻辑

- [x] **`internal/service/session_svc.go`**:
  - `BindContainer()`: 将容器追加到 `session.Containers`，设为 `ActiveContainerID`；校验容器未被其他会话绑定
  - `DeleteSession()`: 级联调用 `onDestroyContainer` 清理所有子容器
  - `SwitchSession()`: 切换到无容器会话时，主动断开当前沙箱连接
  - `ListSessionsEnriched()`: 适配多容器富化（基于 ActiveContainerID）
- [x] **`internal/service/sandbox_svc.go`**:
  - `Connect()` 注册容器时设置 `container.SessionID`（从活跃会话获取）
  - `SetSessionService()` 方法注入会话服务引用
- [x] **`internal/service/container_svc.go`**: `DestroyContainer()` 后同步更新所属会话的 `Containers` 列表

### Phase 4: 前端修复

- [x] **`frontend/src/App.vue`**: `session:switched` 事件处理中，无容器会话时同步清空 `connectionStore` 的 SSH/Docker 状态
- [x] **`frontend/src/components/layout/Sidebar.vue`**: 会话列表展示多容器状态（活跃容器 + 额外容器计数 badge）
- [x] **切换空会话**: `onSessionSwitch` 在 `containerRegID == ""` 时调用 `sandboxService.Disconnect()`，前端收到断连事件后更新 `connectionStore`

### Phase 5: 向后兼容

- [x] 数据迁移：`loadSession` 检测旧格式 `containerID`，自动迁移到 `containers` 数组并持久化新格式
- [ ] 孤儿容器清理：启动时扫描无 `sessionID` 的容器，尝试匹配或标记为孤儿（后续优化）

## 涉及文件

**后端修改:**
- `internal/model/session.go`
- `internal/model/container.go`
- `internal/storage/session_store.go`
- `internal/storage/container_store.go`
- `internal/service/session_svc.go`
- `internal/service/sandbox_svc.go`
- `internal/service/container_svc.go`
- `app.go`（回调接线调整）

**前端修改:**
- `frontend/src/types/session.ts`
- `frontend/src/stores/sessionStore.ts`
- `frontend/src/stores/connectionStore.ts`
- `frontend/src/components/layout/Sidebar.vue`
- `frontend/src/App.vue`

**测试:**
- `internal/model/session_test.go`（更新）
- `internal/model/container_test.go`（更新）
- `internal/storage/session_store_test.go`（新建）
- `internal/storage/container_store_test.go`（新建）

**文档:**
- 相关 doc/ 文件同步更新

## 风险与注意事项

- 数据迁移需考虑旧版本用户数据兼容
- 级联删除涉及远程 Docker 操作，需处理网络不可达场景（尽力删除，记录失败）
- 全局 `SandboxManager` 单例限制了同时连接多个容器的可能，本期保持单活跃容器设计，后续可扩展为多连接

## 状态

已完成
