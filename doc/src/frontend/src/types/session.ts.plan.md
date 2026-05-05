# session.ts 技术说明

## 文件定位
- 源文件: `frontend/src/types/session.ts`
- 会话和 sandbox registry 前端类型。

## 核心职责
- `ContainerInfo` 新增 `runtimeID`、`runtime`、`workspacePath`。
- 状态联合类型新增 `unavailable`。
- 命名保留 Container/activeContainerID 是为了兼容后端 Wails 绑定和既有 store 结构。
