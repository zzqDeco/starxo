# en.ts 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/locales/en.ts
- 文档文件: doc/src/frontend/src/locales/en.ts.plan.md
- 文件类型: TypeScript
- 所属模块: frontend/src/locales

## 2. 核心职责
- 英文语言包，集中维护前端全部 UI 文案键值。

## 3. 输入与输出
- 输入来源: `locales/index.ts` 注册
- 输出结果: `export default` 翻译对象供 `t()` 调用

## 4. 关键实现细节
- 主要键组:
  - `common`, `header`, `sidebar`, `chat`, `input`, `message`, `interrupt`
  - `todo`, `plan`, `status`, `settings`, `files`, `terminal`, `layout`, `containers`
- 本轮新增键组/键:
  - `header.workspaceOpen`, `header.workspaceClose`
  - `header.commandPalette`, `header.commandPlaceholder`
  - `chat.workbenchTitle`, `chat.sandboxReady`, `chat.sandboxRequired`, `chat.capability*`
  - `sidebar.waitingInput`, `sidebar.agentRunning`, `sidebar.mode*Short`
  - `runtime.*`
  - `taskRail.*`（任务轨文案）
  - `workspace.*`（工作区抽屉/面板文案，含 `openFile`）
  - `codePreview.*`（代码预览文案）
  - `palette.openWorkspace`

## 5. 依赖关系
- 内部依赖: 被 `frontend/src/locales/index.ts` 引用
- 外部依赖: 无（纯数据）

## 6. 变更影响面
- 组件引用新增键时，必须同步 `zh.ts` 保持同构。
- 键路径变更会直接影响运行时 `t()` 查找。

## 7. 维护建议
- 严格保持与 `zh.ts` 键结构一致。
- 对动态模板字符串（如 `{count}`）保持同名占位符。
