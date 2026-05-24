# release.yml 技术说明

## 文件定位
- 源文件: `.github/workflows/release.yml`
- GitHub Release 打包发布 workflow。

## 核心职责
- 仅在 `v*.*.*` tag 指向 `master` 可达 commit 时构建并发布三平台产物。
- macOS 构建后运行 `scripts/sign-darwin-app.sh`，确保 `.app` 使用 `com.starxo.app` 且签名绑定 `Info.plist`。
- Windows 构建 exe 和 NSIS installer；Linux 构建 tarball；release job 生成 `SHA256SUMS.txt` 并上传资产。

## 维护要点
- macOS zip 前必须保留 bundle id、codesign、Info.plist bound 和 sealed resources 校验。
- 如果后续启用 Developer ID signing/notarization，应替换签名 identity，但不能移除现有 bundle identity guard。
