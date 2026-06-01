# Dev UI Functional Regression Fixes

## Summary
- Fix post-redesign functional regressions found by Computer Use regression: inaccessible text inputs, terminal command submission, workspace file refresh, and destroy confirmation.
- Keep the visual system unchanged and limit the change to interaction reliability and workspace file enumeration.

## Key Changes
- Add a small native input synchronization helper so keyboard, paste, and accessibility value updates reach Vue state.
- Use explicit native `textarea`/`input` controls for chat composer, terminal command input, and sandbox destroy confirmation input while keeping the existing shell styling.
- Keep terminal as a non-PTY command runner; submitted commands echo output and continue to emit `workspace:changed`.
- List workspace files through SFTP against the active workspace path instead of a bwrap-wrapped Python script, matching upload/download behavior.
- Show workspace list errors in the inspector empty state instead of silently rendering an empty file tree.

## Verification
- `go test ./...`
- `cd frontend && npm run build`
- `/Users/zhaoziqian/go/bin/wails build -skipbindings -trimpath`
- Computer Use regression against `192.168.31.59`: SSH connect, activate sandbox, terminal write/read, workspace refresh/preview, composer input/send, and destroy confirmation input/button enablement.
