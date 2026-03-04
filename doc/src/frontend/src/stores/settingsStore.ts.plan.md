# settingsStore.ts 技术说明

## 1. 文件定位
- 项目: starxo
- 源文件: frontend/src/stores/settingsStore.ts
- 文档文件: doc/src/frontend/src/stores/settingsStore.ts.plan.md
- 文件类型: TypeScript 源码
- 所属模块: frontend/src/stores (Pinia 状态管理)

## 2. 核心职责
- 管理应用配置（SSH、Docker、LLM、MCP、Agent），封装与 Go 后端 SettingsService 的交互。
- 提供默认配置值和部分更新方法。
- 该文件的变更应与项目级规则文档和接口文档保持一致。

## 3. 输入与输出
- 输入来源: SettingsPanel 及其子表单组件、App.vue 的 onMounted 初始化
- 输出结果: 响应式 AppSettings 对象，供 SettingsPanel 和 connectionStore 消费

## 4. 关键实现细节
- **默认配置** (`defaultSettings`):
  - SSH: host=127.0.0.1, port=22, user=root
  - Docker: image=ubuntu:22.04, memoryLimit=2048MB, cpuLimit=2, workDir=/workspace, network=true
  - LLM: type=openai, baseURL=https://api.openai.com/v1, model=gpt-4
  - MCP: servers=[] (空)
  - Agent: maxIterations=30
- **State 属性**:
  - `settings: AppSettings` — 完整配置对象（使用 structuredClone 深拷贝初始化）
  - `loaded: boolean` — 是否已从后端加载
  - `saving: boolean` — 是否正在保存
- **Actions**:
  - `loadSettings()` — 从后端加载配置并与默认值合并（而非直接覆盖），mcp.servers 使用 Array.isArray() 防御性检查；失败则使用默认值
  - `saveSettings()` — 保存配置到后端
  - `updateSSH/updateDocker/updateLLM(partial)` — 使用 Object.assign 部分更新配置
  - `addMCPServer(server)` / `removeMCPServer(index)` — MCP 服务器增删；addMCPServer 在 servers 为 null 时先初始化为空数组
  - `resetToDefaults()` — 重置为默认配置

## 5. 依赖关系
- 内部依赖: `@/types/config` (AppSettings)
- 外部依赖: `pinia` (defineStore)、`vue` (ref)
- Wails 绑定: `wailsjs/go/service/SettingsService` (GetSettings, SaveSettings)

## 6. 变更影响面
- 默认配置修改影响首次启动的初始设置
- 配置结构变更需同步 `@/types/config` 中的 AppSettings 接口和 Go 后端配置结构体
- 保存逻辑被 connectionStore.connect() 依赖

## 7. 维护建议
- 修改该文件后，同步更新项目级 `implementation.plan.md` 与相关规则文档。
- 新增配置项需同步更新 defaultSettings、AppSettings 类型和对应的设置表单组件。
- `saveSettings` 使用 `as any` 类型断言，后续可考虑改善类型安全。
