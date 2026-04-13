# vite-env.d.ts 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: `frontend/src/vite-env.d.ts`
- 文档文件: `doc/src/frontend/src/vite-env.d.ts.plan.md`
- 文件类型: TypeScript 声明文件
- 所属模块: frontend/src

## 2. 核心职责
- 提供 Vite 和 Vue 单文件组件的类型声明，并补充一组历史遗留的 `window.go` / `window.runtime` 全局类型。

## 3. 输入与输出
- 输入来源: TypeScript 编译阶段
- 输出结果: 前端源码在编译期可见的模块和全局类型声明

## 4. 关键实现细节
- `*.vue` 模块声明保证 SFC 能被 TypeScript 正确导入
- `Window` 扩展声明覆盖若干 Wails 风格的全局调用入口，主要用于兼容旧代码路径
- 这些声明不等于运行时保证；真实接口仍以后端绑定和当前调用方式为准

## 5. 依赖关系
- 外部依赖: `vite/client`、`vue` 类型
- 内部依赖: `frontend/src/types/config`

## 6. 变更影响面
- 影响 TypeScript 编译体验和旧式全局 API 调用的类型提示。

## 7. 维护建议
- 若全局 `window.go` 兼容层继续收缩，应同步精简这里的声明，避免类型与实际运行时漂移。
