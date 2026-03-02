# container.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/model/container.go
- 文档文件: doc/src/internal/model/container.go.plan.md
- 文件类型: Go 源码
- 所属模块: model

## 2. 核心职责
- 该文件定义了容器注册表的数据模型，包括容器状态枚举 `ContainerStatus` 和容器实体 `Container`。每个 Container 记录了 Docker 容器的元信息（Docker ID、名称、镜像）、SSH 连接信息（主机、端口）、状态和生命周期时间戳。该模型是容器管理和沙箱复用的基础，支持容器的注册、状态追踪和跨会话重连。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: 无外部输入（纯数据类型定义）
- 输出结果: 无（通过 JSON 序列化存储到文件系统和传递给前端）

## 4. 关键实现细节
- 结构体/接口定义:
  - `ContainerStatus` (string 类型别名) — 容器状态枚举:
    - `ContainerRunning` = "running"
    - `ContainerStopped` = "stopped"
    - `ContainerUnknown` = "unknown"
    - `ContainerDestroyed` = "destroyed"
  - `Container` — 容器实体结构体，包含以下字段:
    - `ID` (string) — 注册表唯一标识
    - `DockerID` (string) — Docker 容器 ID
    - `Name` (string) — 容器名称
    - `Image` (string) — Docker 镜像名称
    - `SSHHost` (string) — SSH 主机地址
    - `SSHPort` (int) — SSH 端口
    - `Status` (ContainerStatus) — 当前状态
    - `SetupComplete` (bool) — 初始化是否完成
    - `SessionID` (string) — 所属会话 ID（与 Session.Containers 构成双向引用）
    - `CreatedAt` (int64) — 创建时间戳
    - `LastUsedAt` (int64) — 最后使用时间戳
- 导出函数/方法: 无
- Wails 绑定方法: 无
- 事件发射: 无

## 5. 依赖关系
- 内部依赖: 无
- 外部依赖: 无
- 关键配置: 无

## 6. 变更影响面
- 修改字段名或 JSON tag 会破坏已有容器注册数据（`~/.starxo/containers.json`）的反序列化兼容性
- `ContainerStatus` 枚举值变更会影响状态判断逻辑（如 `ContainerDestroyed` 在 `ContainerStore.RegisteredDockerIDs` 中用于过滤）
- 该结构体被以下组件使用:
  - `storage.ContainerStore` — 容器注册表 CRUD 和持久化
  - `service.ContainerService` — 容器管理业务逻辑
  - `service.SandboxService` — 沙箱连接和容器生命周期管理
  - 前端 — 容器列表展示和管理

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 新增容器状态值时需同步更新所有状态判断逻辑，特别是 `ContainerStore.RegisteredDockerIDs` 中的过滤条件。
- `ID` 与 `DockerID` 的区分很重要：`ID` 是应用内部的注册标识，`DockerID` 是实际 Docker 容器的标识，两者不可混淆。
- `SessionID` 记录容器归属的会话，与 `Session.Containers` 构成双向引用，确保父子关系一致性。容器创建时由 `SandboxService` 设置。
- 时间戳字段使用 Unix 毫秒格式，与 `Session` 模型保持一致。
