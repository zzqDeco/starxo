# session.ts 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/types/session.ts
- 文档文件: doc/src/frontend/src/types/session.ts.plan.md
- 文件类型: TypeScript 源码
- 所属模块: frontend/src/types (类型定义)

## 2. 核心职责
- 定义会话和容器信息相关的 TypeScript 接口类型。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: 无（纯类型定义）
- 输出结果: 导出类型接口供 sessionStore 和 Sidebar 使用

## 4. 关键实现细节
- **导出类型/接口**:
  - `Session` — 会话信息: id, title, containerID, workspacePath?(可选), createdAt, updatedAt, messageCount。富化字段: containerStatus? (running/stopped/unknown/destroyed/''), containerName?, containerSSH?
  - `ContainerInfo` — 容器详情: id, dockerID, name, image, sshHost, sshPort, status (running/stopped/unknown/destroyed), setupComplete, createdAt, lastUsedAt

## 5. 依赖关系
- 内部依赖: 无
- 外部依赖: 无（纯类型定义）

## 6. 变更影响面
- `Session` 修改影响 sessionStore、Sidebar（会话列表渲染）、App.vue（session:switched 事件处理）
- `ContainerInfo` 修改影响容器管理相关功能
- 字段需与 Go 后端 ListSessionsEnriched 返回的数据结构一致

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- Session 的富化字段（containerStatus/containerName/containerSSH）来自 ListSessionsEnriched，修改时需同步后端。
