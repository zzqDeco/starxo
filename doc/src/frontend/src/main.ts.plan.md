# main.ts 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/main.ts
- 文档文件: doc/src/frontend/src/main.ts.plan.md
- 文件类型: TypeScript 源码
- 所属模块: frontend/src (应用入口)

## 2. 核心职责
- 前端应用的入口文件，负责创建 Vue 3 应用实例并挂载到 DOM。
- 注册全局插件：Pinia 状态管理和 vue-i18n 国际化。
- 导入全局样式文件 `style.css`。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: App.vue 根组件、locales 国际化配置、style.css 全局样式
- 输出结果: 将 Vue 应用实例挂载到 `#app` DOM 节点

## 4. 关键实现细节
- 使用 `createApp(App)` 创建 Vue 3 应用实例
- 通过 `app.use(createPinia())` 注册 Pinia 状态管理
- 通过 `app.use(i18n)` 注册 vue-i18n 国际化插件
- 应用挂载目标为 `#app` 元素

## 5. 依赖关系
- 内部依赖: `./App.vue`、`./locales`（i18n 配置）、`./style.css`
- 外部依赖: `vue` (createApp)、`pinia` (createPinia)

## 6. 变更影响面
- 修改插件注册会影响全局功能可用性（状态管理、国际化）
- 修改挂载目标需同步更新 `index.html` 中的 DOM 节点

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 新增全局插件时在此文件注册，保持入口文件职责清晰。
- 避免在入口文件中放置业务逻辑。
