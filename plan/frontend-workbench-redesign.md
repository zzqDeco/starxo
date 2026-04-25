# Starxo Frontend Workbench Redesign

## Goal

Turn the current chat-centered UI into a dense developer workbench for AI coding sessions. The redesign keeps the existing Wails event model and multi-session isolation while making session state, sandbox runtime, task progress, files, and agent activity easier to scan.

## Delivery Shape

- `feature/frontend-workbench-foundation`: design tokens, main layout, chat/composer flow, command palette fixes, and frontend chunking.
- `feature/frontend-workbench-runtime`: session run-state event, runtime inspector, session run-state badges, workspace file linking, and interaction/accessibility polish.

## Design Direction

- Dark developer workbench with OLED/slate surfaces and restrained color use.
- Cyan: focus and primary navigation.
- Emerald: connected/running/success.
- Amber: pending/connecting/warning.
- Rose: destructive/error.
- Violet: secondary agent activity.
- Compact 6-8px radii for repeated tool surfaces, clear borders, visible focus rings, and no layout-shifting hover effects.

## Interaction Model

- Left rail: sessions, per-session runtime hints, connection controls.
- Center canvas: chat output, agent timeline, task rail, and composer.
- Header: global app status, command trigger, workspace/settings/language actions.
- Right runtime area: first a responsive container dock, then a full runtime inspector in phase 2.
- Workspace: on-demand drawer first, upgraded to a file studio with path-to-preview linking in phase 2.

## Verification

- `cd frontend && npm run build` for every phase.
- Runtime phase also runs targeted Go tests under `internal/service`, `internal/model`, and `internal/storage`.
- Manual `wails dev` verification should cover SSH states, container lifecycle, mode switching, multi-session isolation, interrupt dialogs, workspace file operations, and responsive widths at 375/768/1024/1440.
