# useUiFeedback.ts 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: `frontend/src/composables/useUiFeedback.ts`
- 文档文件: `doc/src/frontend/src/composables/useUiFeedback.ts.plan.md`
- 文件类型: TypeScript 源码
- 所属模块: frontend/src/composables

## 2. 核心职责
- 对 Naive UI 的 message / dialog 能力做统一封装，给前端提供一致的成功、信息、错误和危险确认反馈。

## 3. 输入与输出
- 输入来源: 调用方传入的动作名、错误对象、确认文案
- 输出结果:
  - 统一格式的 UI message
  - `confirmDanger(...)` 返回布尔 Promise

## 4. 关键实现细节
- `toReadableError(...)` 负责把字符串或 Error-like 对象提取成可展示文本
- `stripTechnicalDetails(...)` 会移除通用 `error:` 前缀、stack 尾迹等技术噪音
- `error(...)` 使用 i18n 文案拼装面向用户的错误提示
- `confirmDanger(...)` 统一使用 warning dialog，并转成 Promise 形式

## 5. 依赖关系
- 外部依赖: `naive-ui`、`vue-i18n`

## 6. 变更影响面
- 影响多个设置页和交互操作的前端反馈一致性。

## 7. 维护建议
- 若需要更复杂的错误脱敏策略，应优先在本 composable 内集中处理，而不是在组件里各自清洗。
