# Dev Agent Orphan Tool History Fix

## Summary
- Branch: `fix/dev-agent-orphan-tool-history`.
- Fix orphan assistant `tool_call` history left by interrupted `ask_user` / `ask_choice` turns.
- Keep new standalone user turns from being rejected by OpenAI-compatible providers with `No tool output found for function call ...`.

## Implementation
- Context history repair now enforces provider ordering: each assistant tool-call group is immediately followed by matching tool results.
- Existing non-adjacent tool results are moved next to their assistant call; missing results are synthesized with an explicit interruption reason.
- Prompt preparation defensively repairs the history slice sent to the model, so old or sliced session data cannot leak invalid tool messages.
- Pending interrupts now remember their tool call ids. When a user supersedes, stops, or loses sandbox state before resuming, Starxo repairs the history before clearing the interrupt.
- Session restore logs repaired tool-call ids and schedules a best-effort save so legacy `messages.json` / `session_data.json` data is migrated without user action.

## Verification
- Targeted tests cover orphan insertion, non-adjacent result movement, partial multi-tool groups, sliced prompt repair, interrupt supersede, and stop cleanup.
- Full verification should run:
  - `go test ./...`
  - `cd frontend && npm run build`
  - `/Users/zhaoziqian/go/bin/wails build -skipbindings -trimpath`

## Manual Regression
- Open the legacy session that contains `call_Z8O8xNJHMqO9LAYaxZRenFks` and send a new standalone file write/read request.
- Trigger `ask_user`, send a new standalone task instead of resuming, and confirm the new task starts without provider tool-call pairing errors.
- Verify local LLM and remote sandbox file operations still complete normally.
