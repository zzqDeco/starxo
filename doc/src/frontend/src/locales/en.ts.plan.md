# en.ts 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/locales/en.ts
- 文档文件: doc/src/frontend/src/locales/en.ts.plan.md
- 文件类型: TypeScript 源码
- 所属模块: frontend/src/locales (国际化)

## 2. 核心职责
- 英文语言包，定义应用所有 UI 文本的英文翻译。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: 通过 locales/index.ts 注册到 vue-i18n
- 输出结果: 默认导出翻译对象，供 `t()` 函数使用

## 4. 关键实现细节
- **翻译键分组**:
  - `common` — 通用操作文本 (Connect, Disconnect, Save, Cancel 等)
  - `header` — 顶部导航栏 (标题、面板切换)
  - `sidebar` — 侧边栏 (新对话、会话管理、容器状态)
  - `chat` — 聊天区域 (标题、空状态提示、提示卡片)
  - `input` — 输入区域 (占位符、附件、发送提示)
  - `message` — 消息展示 (复制、参数、结果、展开、thinking="Thinking...")
  - `interrupt` — 中断对话框 (标题、输入提示、选项)
  - `todo` — 任务看板 (taskProgress、collapse、expand)
  - `plan` — 计划面板 (标题)
  - `status` — 连接状态 (SSH/Docker 状态文本)
  - `settings` — 设置面板 (SSH/Docker/LLM/MCP 各子配置)
  - `files` — 文件管理 (工作区、上传下载)
  - `terminal` — 终端 (输出标题)
  - `layout` — 布局 (Terminal/Files/Containers Tab 标签)
  - `containers` — 容器管理 (标题、状态、操作按钮、**createContainer**、**activate**、**deactivate**、**creating**、**sshRequired**、**sshReadyHint**)

## 5. 依赖关系
- 内部依赖: 被 `locales/index.ts` 导入
- 外部依赖: 无（纯数据对象）

## 6. 变更影响面
- 新增/修改翻译键需同步中文语言包 `zh.ts`
- 翻译键被所有使用 `t()` 的组件引用
- 键路径变更需全局搜索替换所有 `t('old.key')` 调用

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 新增 UI 文本时需同时在 en.ts 和 zh.ts 中添加对应翻译。
- 保持翻译键结构与 `zh.ts` 完全一致。
