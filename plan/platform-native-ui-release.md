# Platform Native UI + Release

## Summary

- Add Wails native shell configuration for macOS, Windows, and Linux.
- Add a frontend platform bridge so the Vue UI can select platform-aware tokens.
- Refresh the app from the previous neon-dark dashboard look toward macOS, Windows, and Linux native-feeling surfaces.
- Keep the existing tag-driven GitHub Release flow and add platform artifact sanity checks.

## Implementation

- Wails shell:
  - macOS uses hidden inset titlebar, system appearance, transparent webview/window, and About metadata.
  - Windows follows the system theme, uses Mica where supported, and defines light/dark titlebar colors.
  - Linux sets icon, program name, and WebKit GPU policy without default translucency.
- Platform bridge:
  - `PlatformService.GetPlatformUIInfo` returns platform, GOOS, system appearance mode, translucency support, and Mica support.
  - The frontend writes `data-platform`, `data-theme`, and `data-translucent` on the root element.
- Frontend:
  - Global CSS tokens now default to system light and switch to system dark.
  - Existing component variables are kept as aliases so current components can migrate safely.
  - Main layout, toolbar, sidebar, chat, settings, workspace, sandbox inspector, terminal, and command palette use lower-saturation platform surfaces.
- Release:
  - The release workflow keeps the existing `v*.*.*` tag trigger and master reachability preflight.
  - Packaging now verifies basic macOS bundle metadata, Windows icon/manifest resources, and Linux binary shape before uploading assets.

## Verification

- `go test ./...`
- `cd frontend && npm run build`
- `wails build` on macOS
- Manual visual regression in light and dark system appearance.
- Release dry run with a test tag after merge to `master`.
