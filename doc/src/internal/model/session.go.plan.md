# session.go 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: internal/model/session.go
- 文档文件: doc/src/internal/model/session.go.plan.md
- 文件类型: Go 源码
- 所属模块: model

## 2. 核心职责
- 该文件定义了会话的持久化数据模型 `Session`，代表一个用户与 AI Agent 的完整对话会话。每个会话包含唯一标识、标题、关联的容器 ID、工作区路径、创建/更新时间戳以及消息计数。该结构体是会话管理系统的核心数据类型，用于会话列表展示、会话切换和持久化存储。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: 无外部输入（纯数据类型定义）
- 输出结果: 无（通过 JSON 序列化存储到文件系统和传递给前端）

## 4. 关键实现细节
- 结构体/接口定义:
  - `Session` — 会话结构体，包含以下字段:
    - `ID` (string) — 会话唯一标识 (UUID 前 8 位)
    - `Title` (string) — 会话标题
    - `ContainerID` (string) — 关联的容器注册 ID
    - `WorkspacePath` (string, omitempty) — 工作区路径
    - `CreatedAt` (int64) — 创建时间（Unix 毫秒时间戳）
    - `UpdatedAt` (int64) — 最后更新时间（Unix 毫秒时间戳）
    - `MessageCount` (int) — 消息数量
- 导出函数/方法: 无
- Wails 绑定方法: 无
- 事件发射: 无

## 5. 依赖关系
- 内部依赖: 无
- 外部依赖: 无
- 关键配置: 无

## 6. 变更影响面
- 修改字段名或 JSON tag 会破坏已有会话数据的反序列化兼容性
- 该结构体被以下组件使用:
  - `storage.SessionStore` — CRUD 操作和磁盘持久化
  - `service.SessionService` — 会话管理业务逻辑
  - 前端 — 会话列表展示和切换
- 新增字段需同步更新 `SessionStore` 的序列化逻辑和前端 TypeScript 类型

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 时间戳使用 Unix 毫秒格式 (`UnixMilli`)，与前端 JavaScript 的 `Date.now()` 一致，变更格式需同步前端。
- `ContainerID` 关联的是容器注册表 ID（非 Docker 容器 ID），命名可能引起混淆，注释中应明确说明。
- `WorkspacePath` 使用了 `omitempty`，表示可选字段，适用于未绑定容器的会话。
