# Runtime LSP Edit Actions

## Summary
- Branch: `feature/runtime-lsp-edit-actions`.
- Add writable language-server edit operations without changing the read-only `LSP` tool.
- New deferred tool: `LSPEdit`, currently supporting `rename` and `format`.

## Design
- `LSP` remains read-only trusted and keeps fallback behavior.
- `LSPEdit` is deferred, runtime-file scoped, and not read-only trusted, so existing Runtime V2 permission queue handles approval.
- `runtimeLSPManager.Edit` reuses the persistent language server lifecycle, opens/syncs the target document, calls `textDocument/rename` or `textDocument/formatting`, validates all returned file URIs against the active workspace, applies text edits, and writes files through the sandbox operator.
- Text edits are applied in reverse range order and map LSP UTF-16 character positions to byte offsets.

## Verification
- `go test ./internal/service -run 'TestRuntimeLSPManager'`
- `go test ./internal/tools -run 'TestRuntimeDeferredEntriesMetadata'`
- `go test ./...`
- `cd frontend && npm run build`
- GitHub Codex review before merge.
