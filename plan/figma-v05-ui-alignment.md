# Starxo Figma v0.5 UI Alignment

## Summary
- Branch: `feature/figma-v05-ui-alignment`.
- Design source: Figma `Starxo Desktop Redesign v0.5 Complete Prototype`.
- Goal: move the frontend toward the v0.5 semantic token system, reusable UI component contract, and one-heavy-inspector workbench layout.

## Implemented Scope
- Added `lucide-vue-next` as the new icon source for the v0.5 UI component layer.
- Added `frontend/src/components/ui/` primitives matching the Figma component API matrix:
  - button, icon button, text field, status badge, source row, file row, timeline item, permission card, inspector segmented control, message bubble, toolbar item, terminal row, settings row, runtime task row, modal surface, empty state.
- Added `--sx-*` semantic tokens to `frontend/src/style.css` and kept existing `--platform-*` and legacy aliases.
- Updated Naive UI theme overrides to read the same semantic token layer.
- Updated app shell to a single inspector model: `runtime`, `workspace`, `terminal`, and `tasks`.
- Split Terminal and Runtime Tasks into first-class inspector modes instead of nesting them under the runtime panel.
- Replaced the floating task rail near the composer with a fixed inline task status bar.
- Reused v0.5 primitives in header toolbar, terminal fallback rows, workspace empty state, and the inspector switcher.

## Verification
- `cd frontend && npm run build`

## Follow-up Scope
- Continue migrating old Naive/Ionicons-heavy business panels to the `Sx*` primitive layer.
- Use visual screenshot regression against the Figma breakpoints before merging.
- Keep backend/runtime behavior unchanged in this UI-only branch.
