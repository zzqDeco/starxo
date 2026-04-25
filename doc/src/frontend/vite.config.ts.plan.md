# vite.config.ts 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/vite.config.ts
- 文档文件: doc/src/frontend/vite.config.ts.plan.md
- 文件类型: 配置文件
- 所属模块: frontend/ (Vite 构建配置)

## 2. 核心职责
- Vite 6 构建工具配置，定义 Vue 3 插件和路径别名。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: Vite CLI / Wails 开发服务器读取
- 输出结果: 控制 Vite 的构建和开发行为

## 4. 关键配置项
- **plugins**: `vue()` — @vitejs/plugin-vue，启用 Vue 3 SFC 支持
- **resolve.alias**: `@` → `src/` 目录 — 路径别名，使用 `resolve(__dirname, 'src')` 计算绝对路径
- **build.chunkSizeWarningLimit**: 650 — 当前 Naive UI vendor chunk 约 550KB，避免旧版 1.8MB 首包优化后仍被默认阈值误报。
- **build.rollupOptions.output.manualChunks**:
  - `vue`: Vue/Pinia/i18n/VueUse 基础运行时
  - `naive`: Naive UI 与图标库
  - `markdown`: markdown-it 与 highlight.js core

## 5. 依赖关系
- 内部依赖: 无
- 外部依赖: `vite` (defineConfig)、`@vitejs/plugin-vue` (vue)、`path` (resolve)

## 6. 变更影响面
- 新增 Vite 插件影响构建管线
- 修改路径别名需同步 tsconfig.json 中的 paths 配置
- 构建配置变更影响 Wails 的前端构建流程
- 手动分块会改变生产构建产物文件名和加载顺序，需通过 `npm run build` 回归验证

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- `@` 路径别名需与 tsconfig.json 中的 `paths["@/*"]` 保持一致。
- 如需配置代理、环境变量或构建优化，在此文件中添加。
