# zh.ts 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/locales/zh.ts
- 文档文件: doc/src/frontend/src/locales/zh.ts.plan.md
- 文件类型: TypeScript
- 所属模块: frontend/src/locales

## 2. 核心职责
- 中文语言包，作为默认语言提供前端全部中文文案。

## 3. 输入与输出
- 输入来源: `locales/index.ts` 注册并设为默认 locale
- 输出结果: `export default` 翻译对象供 `t()` 调用

## 4. 关键实现细节
- 键组与英文包同构:
  - `common`, `header`, `sidebar`, `chat`, `input`, `message`, `interrupt`
  - `todo`, `plan`, `status`, `settings`, `files`, `terminal`, `layout`, `containers`
- 本轮新增键组/键:
  - `header.workspaceOpen`, `header.workspaceClose`
  - `header.commandPalette`, `header.commandPlaceholder`
  - `chat.workbenchTitle`, `chat.sandboxReady`, `chat.sandboxRequired`, `chat.capability*`
  - `sidebar.waitingInput`, `sidebar.agentRunning`, `sidebar.mode*Short`
  - `runtime.*`
  - `taskRail.*`
  - `workspace.*`（含 `drawerTitle`, `openFile`）
  - `codePreview.*`
  - `palette.openWorkspace`

## 5. 依赖关系
- 内部依赖: 被 `frontend/src/locales/index.ts` 引用
- 外部依赖: 无（纯数据）

## 6. 变更影响面
- 与 `en.ts` 键路径必须保持一致。
- 缺失键会导致界面回退 key 字符串显示。

## 7. 维护建议
- 新增界面文案时先在中文包落地，再同步英文包。
- 保持格式化占位符与英文包一致（如 `{count}`）。
