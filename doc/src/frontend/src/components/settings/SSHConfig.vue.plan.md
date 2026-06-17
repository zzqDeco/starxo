# SSHConfig.vue 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: `frontend/src/components/settings/SSHConfig.vue`
- 文档文件: `doc/src/frontend/src/components/settings/SSHConfig.vue.plan.md`
- 文件类型: Vue 单文件组件
- 所属模块: frontend/src/components/settings

## 2. 核心职责
- 提供 SSH 主机、端口、用户名、密码和私钥配置界面，并支持连接测试。

## 3. 输入与输出
- 输入来源: `settingsStore.settings.ssh`
- 输出结果:
  - 表单双向写入 store
  - 调用 `SettingsService.TestSSHConnection`

## 4. 关键实现细节
- 端口使用数字输入并限制到合法范围
- 私钥输入使用 textarea + monospace 样式
- 测试按钮在 success / error / idle 之间切换视觉状态
- 连接测试失败时展示错误详情；macOS 局域网 permission-like dial failure 会提示用户检查 Local Network 权限。
- macOS Local Network 命中时挂载 `MacLocalNetworkFixCard`，用户可重新检测并复制 reset 命令。

## 5. 依赖关系
- 内部依赖: `@/stores/settingsStore`、Wails `SettingsService`
- 外部依赖: `vue`、`naive-ui`、`@vicons/ionicons5`、`vue-i18n`
- 错误格式化依赖: `frontend/src/utils/sshErrorHints.ts`
- 修复提示依赖: `frontend/src/components/settings/MacLocalNetworkFixCard.vue`

## 6. 变更影响面
- 直接影响设置页的 SSH 连接配置体验和测试反馈。

## 7. 维护建议
- 若后端 SSH 配置支持新的认证字段，应同步补到本表单和 i18n 词条。
