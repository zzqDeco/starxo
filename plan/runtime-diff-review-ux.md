# Runtime Diff Review UX

## Summary
- Branch: `feature/runtime-diff-review-ux`.
- Goal: make Runtime V2 file write/edit results reviewable without forcing users to read raw JSON.
- Scope:
  - `Write` returns line removals, bounded patch content, and truncation metadata for create/overwrite operations.
  - `Edit` returns bounded patch content and truncation metadata.
  - Runtime compact stores write patches and removed-line counts alongside edit summaries.
  - Timeline renders `Write` / `Edit` / legacy `write_file` / `str_replace_editor` results as structured diff metadata and patch blocks.

## Backend
- Extended `tools.WriteOutput` with `linesRemoved`, `patch`, and `truncated`.
- Extended `tools.EditOutput` with `truncated`.
- Added bounded simple patch generation so large edit/write payloads are truncated while building the preview instead of after constructing the full patch.
- Updated runtime compact diff summaries to preserve write patch and removed-line metadata.

## Frontend
- `TimelineEventItem.vue` now recognizes Runtime V2 `Write` and `Edit` names in addition to legacy aliases.
- File/edit tool summaries show created/updated state, replacements, bytes, and `+/-` line counts.
- Expanded results render patch content in a dedicated diff block when structured JSON is available.

## Verification
- Targeted Go tests cover write overwrite patch metadata and existing edit patch behavior.
- Full `go test ./...`.
- Frontend `npm run build`.
- GitHub Codex review before merge.
