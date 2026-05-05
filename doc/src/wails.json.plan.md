# wails.json 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: wails.json
- 文档文件: doc/src/wails.json.plan.md
- 文件类型: 配置文件
- 所属模块: 项目根目录 (Wails 配置)

## 2. 核心职责
- Wails v2 项目配置文件，定义应用名称、前端构建命令和产品信息。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: Wails CLI 读取
- 输出结果: 控制 Wails 构建和开发行为

## 4. 关键配置项
- **$schema**: `https://wails.io/schemas/config.v2.json` — Wails v2 配置模式
- **name**: `starxo` — 项目名称
- **outputfilename**: `starxo` — 构建输出可执行文件名
- **frontend:install**: `npm install` — 前端依赖安装命令
- **frontend:build**: `npm run build` — 前端生产构建命令
- **frontend:dev:watcher**: `npm run dev` — 开发模式前端热重载命令
- **frontend:dev:serverUrl**: `auto` — 自动检测开发服务器 URL
- **info**:
  - productName: `Starxo`
  - productVersion: `0.1.0`
  - comments: `Starxo Coding Agent Desktop Application`
- **author**: zhaoziqian (zhaoziqian@corp.netease.com)

## 5. 依赖关系
- 内部依赖: 引用 frontend/ 目录的 npm 脚本
- 外部依赖: Wails CLI v2
- Release CI 依赖该配置中的 `outputfilename`、`frontend:build` 和 `build/` 平台资源生成 GitHub Release 产物。

## 6. 变更影响面
- 修改 frontend 命令影响构建和开发流程
- 修改 outputfilename 影响打包产物命名
- 修改 productVersion 影响应用版本标识
- 修改 Wails 构建输出路径或文件名时，需同步 `.github/workflows/release.yml` 的产物打包步骤。

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 版本号 (productVersion) 应随发布更新。
- 如需更改前端包管理器（如 pnpm），需同步修改 frontend:install 和相关命令。
