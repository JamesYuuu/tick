---
title: Tick UI Fullscreen 3-Zone Layout (Design)
date: 2026-03-05
branch: feat/ui-refresh-calm-paper
---

# Goal

Improve the tick TUI layout (rendering only) so it:

- Fills the terminal (full width and full height)
- Has clearly separated zones: header / workspace / footer
- Centers all zones as a single column with `maxWidth = 96`
- Uses a small text logo `[tick]` instead of the plain title

No behavioral changes: keep all keybindings, commands, data flows, and view switching exactly the same.

# Constraints

- Rendering/layout changes only; do not change interactions or business logic.
- ASCII-only output.
- Works on narrow terminals; gracefully clamps width/height.
- Deterministic output suitable for tests.

# Approach

Move responsibility for overall screen layout into `Model.View()` (or a dedicated render function called by it):

- `renderToday` / `renderUpcoming` / `renderHistory` produce workspace content only.
- A top-level layout renderer composes:
  - Header (1 line)
  - Separator (1 line)
  - Workspace (variable height, fills remainder)
  - Separator (1 line)
  - Footer (2 lines)

This guarantees full-screen height by padding/truncating to exactly `m.height` lines.

# Layout Spec

## Column centering

Given `windowWidth = m.width`:

- `contentWidth = min(windowWidth, 96)`
- `padLeft = max((windowWidth-contentWidth)/2, 0)`

All rendered lines are left-padded by `padLeft` spaces. Content inside the column is limited to `contentWidth` (truncate or clip where needed).

## Heights

Fixed heights:

- `headerH = 1`
- `sepH = 1` (used twice)
- `footerH = 2`

Workspace height:

- `workspaceH = max(windowHeight - headerH - 2*sepH - footerH, 0)`

The final `View()` output contains exactly `windowHeight` lines (or 0 if `windowHeight <= 0`).

## Header

- Single line.
- Left side logo: `[tick]` (styled like existing app title).
- Right side tabs: Today / Upcoming / History (existing styling).
- Entire line fits within `contentWidth`.

## Separators

ASCII line:

- `strings.Repeat("-", contentWidth)`

## Workspace

- Uses existing sheet/frame styling (border + padding) for the main content.
- Body content is whatever the active view renders (list, add input, history columns, empty state messages).
- The sheet should fit within `contentWidth` and `workspaceH`.

## Footer

Exactly two lines:

1) Status line (or history ratios line when applicable). If nothing to show, output an empty line.
2) Help/key hints line (existing content).

Each line fits within `contentWidth`.

# Component Sizing

On `tea.WindowSizeMsg`, size list/input components based on the workspace's inner rectangle:

- Width derived from `contentWidth` minus sheet border/padding.
- Height derived from `workspaceH` minus sheet border/padding.

This prevents overflow when centered or when `windowWidth > 96`.

# Testing

Add/adjust UI rendering tests to assert:

- `View()` returns exactly `Height` lines after a `tea.WindowSizeMsg`.
- Header includes `[tick]`.
- When `Width > 96`, rendered lines start with left padding (centering).
- Footer always consumes 2 lines (status + help), even when status is empty.

Keep existing behavior tests unchanged.

# Out of Scope

- Changing keybindings, command handling, or app logic.
- Any data migrations or persistence changes.
- New themes, colors, or non-ASCII decorations.
