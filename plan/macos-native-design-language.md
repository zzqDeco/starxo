# macOS Native Design Language Pass

## Summary
- Branch: `fix/macos-native-design-language`.
- Goal: make the current Starxo UI read as a clean macOS desktop app instead of a mixed web dashboard.
- References used for visual direction: Finder source list, System Settings grouped forms, Notes three-pane layout, Xcode-style inspectors/dialogs, VS Code workbench density, and Chrome toolbar/tab hierarchy.

## Design Principles
- Use a split-view foundation: translucent source list, single chat plane, optional right inspector.
- Prefer grouped rows and hairline separators over nested cards, heavy borders, and colored panels.
- Keep color quiet: blue is reserved for primary focus/actions, green/yellow/red only for status dots or destructive/error states.
- Use native typography rhythm: system font, 11/12/13/15px control scale, medium weight for labels, no oversized hero copy in tool surfaces.
- Make empty states feel like native app states, not landing pages.
- Use icon toolbars and concise labels; avoid variable-like or implementation-driven text in visible UI.
- Borrow VS Code's low-noise workbench density for file/runtime panes, but keep macOS surfaces and spacing.
- Borrow Chrome's toolbar treatment: one subtle toolbar plane, selected controls on a white/raised surface, no competing accent fills.
- Keep macOS double-click title/toolbar behavior from the platform-native work, and make the visual chrome match that behavior.

## Scope
- Frontend design tokens and Naive UI theme overrides.
- Main layout, header toolbar, session source list.
- Chat empty state, transcript, tool timeline, composer.
- Runtime and workspace inspector styling.
- High-impact Chinese and English copy that currently reads like implementation names.

## Out Of Scope
- New backend behavior.
- Full product IA rewrite.
- Platform-specific native widgets beyond the Wails/native-window behavior already added.

## Verification
- `cd frontend && npm run build`
- `wails build -skipbindings -trimpath`
- Launch app and inspect at desktop width, checking that the UI has no large blank left area, no dashboard-like card clutter, no competing accent colors, and no titlebar behavior regression.
