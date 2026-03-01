# 覆盖统计

> 所属项目: Eino Coding Agent (starxo) | 文档类型: 覆盖统计

---

## 一、总体统计

| 指标 | 值 |
|------|------|
| 源文件总数 | 80 |
| 已覆盖 (项目级文档) | 10/10 |
| 已覆盖 (文件级文档) | 80/80 |
| 缺失 | 0 |
| 项目级覆盖率 | 100% |
| 文件级覆盖率 | 100% |

---

## 二、按模块统计

| 模块 | 文件数 | 文件级文档 | 状态 |
|------|--------|-----------|------|
| Go 根文件 (main.go, app.go) | 2 | 2/2 | 已完成 |
| internal/agent/ | 10 | 10/10 | 已完成 |
| internal/service/ | 7 | 7/7 | 已完成 |
| internal/sandbox/ | 6 | 6/6 | 已完成 |
| internal/tools/ | 8 | 8/8 | 已完成 |
| internal/config/ | 2 | 2/2 | 已完成 |
| internal/context/ | 4 | 4/4 | 已完成 |
| internal/llm/ | 1 | 1/1 | 已完成 |
| internal/model/ | 3 | 3/3 | 已完成 |
| internal/storage/ | 2 | 2/2 | 已完成 |
| internal/store/ | 1 | 1/1 | 已完成 |
| internal/logger/ | 2 | 2/2 | 已完成 |
| 前端入口文件 | 3 | 3/3 | 已完成 |
| frontend/src/stores/ | 4 | 4/4 | 已完成 |
| frontend/src/types/ | 3 | 3/3 | 已完成 |
| frontend/src/components/chat/ | 7 | 7/7 | 已完成 |
| frontend/src/components/layout/ | 3 | 3/3 | 已完成 |
| frontend/src/components/settings/ | 1 | 1/1 | 已完成 |
| frontend/src/components/files/ | 2 | 2/2 | 已完成 |
| frontend/src/components/status/ | 2 | 2/2 | 已完成 |
| frontend/src/components/terminal/ | 1 | 1/1 | 已完成 |
| frontend/src/composables/ | 2 | 2/2 | 已完成 |
| frontend/src/locales/ | 1 | 1/1 | 已完成 |
| 配置文件 | 3 | 3/3 | 已完成 |

---

## 三、包含文件类型

| 文件类型 | 后缀 | 文件数 | 说明 |
|----------|------|--------|------|
| Go | `.go` | 48 | 后端源代码 |
| Vue 单文件组件 | `.vue` | 22 | 前端组件 |
| TypeScript | `.ts` | 7 | 前端逻辑 (stores, types, composables, config) |
| CSS | `.css` | 1 | 全局样式 |
| JSON | `.json` | 2 | 构建配置 (wails.json, tsconfig.json) |
| **总计** | - | **80** | - |

---

## 四、排除目录

以下目录不纳入文档覆盖范围:

| 排除目录 | 原因 |
|----------|------|
| `node_modules/` | 第三方依赖 |
| `.git/` | 版本控制元数据 |
| `frontend/wailsjs/` | Wails 自动生成的 TypeScript 绑定代码 |
| `build/` | 构建输出目录 |
| `dist/`, `frontend/dist/` | 前端构建输出 |
| `logs/` | 运行时日志文件 |
| `go.mod` / `go.sum` | Go 模块依赖声明，已在项目级文档中描述 |
| `frontend/package.json` | npm 依赖声明，已在项目级文档中描述 |
| `frontend/src/vite-env.d.ts` | Vite 类型声明文件 |

---

## 五、更新记录

| 事件 | 日期 |
|------|------|
| 项目级文档创建 | 2026-03-01 |
| 文件级文档创建 | 2026-03-01 |
| 覆盖率达成 100% | 2026-03-01 |
| 上次更新 | 2026-03-01 |
