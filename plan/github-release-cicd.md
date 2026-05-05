# GitHub Release 打包 CI/CD

## Summary
- 新增 GitHub Actions release workflow，在推送 `v*.*.*` tag 时触发。
- tag 必须指向 `origin/master` 可达的 commit，否则停止发布。
- 构建未签名的 macOS、Windows、Linux Wails 桌面包，并创建 GitHub Release。

## Implementation
- Workflow 文件: `.github/workflows/release.yml`
- 预检阶段校验 tag 形态、解析 annotated tag 到 commit、检查 master 可达性。
- 构建阶段使用 Go 1.24.x、Node 22.x、Wails CLI v2.11.0。
- Windows 使用 NSIS 生成安装包；Linux 使用 Ubuntu 24.04 + WebKit 4.1 build tag。
- 发布阶段下载三平台 artifacts，生成 `SHA256SUMS.txt`，通过 `softprops/action-gh-release` 创建/更新 Release。

## Verification
- 本地回归: `go test ./...`、`cd frontend && npm run build`。
- GitHub 验证: 在 master commit 上推送测试 tag，确认 Release assets 和 checksums；再验证非 master tag 在 preflight 失败。
- 发布后检查: Release 页面公开，三平台产物与 `SHA256SUMS.txt` 齐全；下载后校验哈希；三平台至少启动一次；设置页、SSH 测试、sandbox runtime 检测、Linux sandbox workspace 写入和 `network=false` 断网行为通过。

## Notes
- v1 不做 macOS notarization、Apple signing 或 Windows code signing。
- 版本号来自 tag，不自动改写 `wails.json` 的 `productVersion`。
