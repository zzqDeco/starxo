# LLMConfig.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: `frontend/src/components/settings/LLMConfig.vue`
- 文档文件: `doc/src/frontend/src/components/settings/LLMConfig.vue.plan.md`
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/settings

## 2. 核心职责
- 提供 LLM provider、base URL、API Key、model 和自定义 headers 的配置界面，并支持连通性测试。

## 3. 输入与输出
- 输入来源: `settingsStore.settings.llm`
- 输出结果:
  - 表单双向写入 store
  - 调用 `SettingsService.TestLLMConnection`

## 4. 关键实现细节
- provider 切换时会按预设 provider 自动填充默认 `baseURL`
- headers 支持增、改、删三种操作，内部以对象 map 形式存储
- 测试按钮有 `loading / success / error` 三态反馈
- provider 选项和按钮文案全部走 i18n

## 5. 依赖关系
- 内部依赖: `@/stores/settingsStore`、Wails `SettingsService`
- 外部依赖: `vue`、`naive-ui`、`@vicons/ionicons5`、`vue-i18n`

## 6. 变更影响面
- 影响模型配置编辑体验以及前端对 provider 默认值的处理逻辑。

## 7. 维护建议
- 若新增 provider，需同步更新 provider options、默认 URL 映射和语言包。
