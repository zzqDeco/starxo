# DockerConfig.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: `frontend/src/components/settings/DockerConfig.vue`
- 文档文件: `doc/src/frontend/src/components/settings/DockerConfig.vue.plan.md`
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/settings

## 2. 核心职责
- 提供 Docker 配置子表单，编辑镜像、资源限制、工作目录和网络开关。

## 3. 输入与输出
- 输入来源: `settingsStore.settings.docker`
- 输出结果: 通过双向绑定直接更新 settings store 中的 Docker 配置

## 4. 关键实现细节
- 使用 `NForm`、`NInput`、`NInputNumber`、`NSwitch` 进行轻量配置输入
- 所有文案通过 `vue-i18n` 获取
- 镜像与工作目录输入使用 monospace 样式
- 底部信息框只承担说明职责，不参与配置保存

## 5. 依赖关系
- 内部依赖: `@/stores/settingsStore`
- 外部依赖: `naive-ui`、`vue-i18n`

## 6. 变更影响面
- 直接影响设置页的 Docker 配置编辑体验与默认约束。

## 7. 维护建议
- 若新增 Docker 配置字段，应同步更新后端配置结构与本表单。
