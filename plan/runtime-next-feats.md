# Runtime Next Features

## Summary
- Branch: `feature/runtime-next-feats`.
- Scope: continue the Runtime V2 feature batch with persistent LSP configurability, WebSearch/TinyFish settings diagnostics, and runtime task observability.

## Changes
- LSP runtime:
  - Added `agent.lsp` settings for enable/disable, request timeout, max result bytes, and custom server mappings.
  - Kept built-in mappings for Go, TypeScript, JavaScript, Python, and Rust.
  - Added `ChatService.GetRuntimeLSPStatus(sessionID)` for running server visibility.
- WebSearch/TinyFish:
  - Added `SettingsService.DiagnoseWebSearch(cfg)`.
  - Added settings UI for enabling WebSearch, selecting default provider, configuring TinyFish, and running diagnostics.
  - TinyFish alignment: `GET https://api.search.tinyfish.ai`, `X-API-Key`, `TINYFISH_API_KEY`, `query/location/language/page`.
- Runtime task observability:
  - Task snapshots now include output size and duration.
  - Runtime Tasks panel displays output size in the task detail metadata.

## Verification
- `go test ./...`
- `cd frontend && npm run build`
- `git diff --check`

## Follow-Ups
- Add writable LSP actions behind permission approval.
- Add live WebSearch smoke tests with explicit user action and clearer network error reporting.
- Add richer Agent/worktree merge and diff review flows.
