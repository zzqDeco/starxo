# settingsStore.ts 技术说明

## 文件定位
- 源文件: `frontend/src/stores/settingsStore.ts`
- Pinia 设置状态。

## 核心职责
- 默认配置使用 `sandbox` 块，包含 runtime/rootDir/workDirName/network/memory/timeout/Python 包。
- 加载后端设置时合并 `sandbox`，并兼容旧 `docker.memoryLimit/network` 到 sandbox。
- `updateSandbox` 替代旧 `updateDocker`。
