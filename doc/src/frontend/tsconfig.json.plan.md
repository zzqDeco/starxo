# tsconfig.json 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/tsconfig.json
- 文档文件: doc/src/frontend/tsconfig.json.plan.md
- 文件类型: 配置文件
- 所属模块: frontend/ (TypeScript 编译配置)

## 2. 核心职责
- TypeScript 5.7 编译配置，定义编译目标、模块解析策略和类型检查规则。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: TypeScript 编译器 / IDE 读取
- 输出结果: 控制类型检查行为和代码提示

## 4. 关键配置项
- **target**: `ES2020` — 编译目标
- **module**: `ESNext` — 模块系统
- **lib**: `["ES2023", "DOM", "DOM.Iterable"]` — 可用类型库
- **moduleResolution**: `bundler` — 使用 bundler 模式解析模块（适配 Vite）
- **allowImportingTsExtensions**: `true` — 允许导入 .ts 扩展名
- **resolveJsonModule**: `true` — 允许导入 JSON 文件
- **isolatedModules**: `true` — 每个文件独立编译（Vite 要求）
- **noEmit**: `true` — 不输出编译结果（由 Vite 处理）
- **jsx**: `preserve` — 保留 JSX（交给 Vue 编译器处理）
- **strict**: `true` — 启用严格类型检查
- **noUnusedLocals**: `false` — 不报告未使用的局部变量
- **noUnusedParameters**: `false` — 不报告未使用的参数
- **noFallthroughCasesInSwitch**: `true` — switch 必须 break
- **paths**: `{"@/*": ["./src/*"]}` — 路径别名映射
- **baseUrl**: `.` — 路径解析基准目录
- **include**: `src/**/*.ts`, `src/**/*.d.ts`, `src/**/*.tsx`, `src/**/*.vue`
- **references**: `tsconfig.node.json` — 项目引用（用于 Vite 配置文件的类型检查）

## 5. 依赖关系
- 内部依赖: `tsconfig.node.json` (项目引用)
- 外部依赖: TypeScript 5.7

## 6. 变更影响面
- strict 模式修改影响全局类型检查严格度
- paths 修改需同步 vite.config.ts 的 resolve.alias
- lib 修改影响可用的全局 API 类型

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- `@/*` 路径别名需与 vite.config.ts 中的 resolve.alias 保持一致。
- noUnusedLocals/noUnusedParameters 设为 false 以减少开发阶段的干扰，生产发布前可考虑开启。
