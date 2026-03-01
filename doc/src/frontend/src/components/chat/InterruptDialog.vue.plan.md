# InterruptDialog.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/components/chat/InterruptDialog.vue
- 文档文件: doc/src/frontend/src/components/chat/InterruptDialog.vue.plan.md
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/chat (聊天模块)

## 2. 核心职责
- 中断对话框组件，当 Agent 需要用户输入时显示模态界面。
- 支持两种中断类型：followup（追问 — 自由文本回答）和 choice（选择 — 从选项中选择）。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: chatStore.pendingInterrupt 状态
- 输出结果: 调用 Wails 绑定 ResumeWithAnswer/ResumeWithChoice 恢复 Agent 执行

## 4. 关键实现细节
- **Pinia Store 交互**: chatStore — pendingInterrupt, clearInterrupt, setGenerating
- **Wails 绑定调用**:
  - `ResumeWithAnswer(text)` — 提交追问的文本回答
  - `ResumeWithChoice(index)` — 提交选择的选项索引
  - `StopGeneration()` — 取消中断
- **交互逻辑**:
  - followup: 显示问题列表 + 文本输入框，Enter 提交
  - choice: 显示选项卡片列表，点击即提交
  - 取消: 点击背景或取消按钮，清除中断并停止生成
  - 提交中: isSubmitting 状态锁防止重复提交
- **模板结构**: 固定定位背景遮罩 → NCard 对话框 → 条件渲染 followup 或 choice 模板

## 5. 依赖关系
- 内部依赖: `@/stores/chatStore`
- 外部依赖: `vue` (ref, computed)、`naive-ui` (NButton, NInput, NCard)、`vue-i18n` (useI18n)
- Wails 绑定: `wailsjs/go/service/ChatService` (ResumeWithAnswer, ResumeWithChoice, StopGeneration)

## 6. 变更影响面
- 中断类型扩展需同步 InterruptEvent 类型定义和 Go 后端
- UI 交互修改影响用户与 Agent 的交互流程

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 新增中断类型时需在模板中添加对应的条件渲染分支。
- backdrop 使用 `position: fixed`，需注意 z-index 与其他模态组件的层叠关系。
