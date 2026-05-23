# build/darwin/Info.plist 技术说明

## 文件定位
- 源文件: `build/darwin/Info.plist`
- Wails macOS production build 使用的应用 bundle plist 模板。

## 核心职责
- 定义 macOS app bundle 元数据：bundle identifier、名称、版本、图标和最低系统版本。
- 声明本地网络访问用途，允许打包后的 Starxo 通过 SSH 访问局域网内的远端 sandbox 主机。
- 配置 `NSAppTransportSecurity.NSAllowsLocalNetworking`，避免生产包访问 `192.168.x.x` / `.local` 等本地地址时被系统网络策略阻断。

## 维护要点
- 修改 bundle identifier 或版本字段时需要同步 release 打包验证。
- 删除 `NSLocalNetworkUsageDescription` 或 `NSAllowsLocalNetworking` 会影响 macOS production 包连接局域网 SSH 主机。
