## Background

History currently renders as a left-side vertical date list and a right-side task list separated by a vertical divider. We want a clearer information hierarchy and better support for future scrolling of the task details.

---

## Goals

- Replace the left vertical date list with a horizontal 7-day date selector rendered as an ASCII table.
- Keep History content top-aligned inside the workspace.
- Use `left/h` and `right/l` to move the selected date; when at the edges, auto-roll the 7-day window by 1 day and clamp forward movement at today.
- Use `up/k` and `down/j` to scroll the task detail area (not to change dates).
- Selected date uses reverse video (background/foreground swapped) to stand out.

---

## Non-Goals

- No mouse support.
- No modal dialogs.
- No month calendar grid.
- Do not change task grouping rules or delayed highlighting rules.

---

## Layout

Within the sheet inner box (the same `innerW`/`innerH` computed by `Model.View()`):

1) Date selector table (3 lines)

- ASCII-only borders.
- 7 columns, one per day in the current `[historyFrom..historyTo]` window.
- Each cell shows `MM-DD`.
- The selected day cell is rendered in reverse-video.
- The whole table is horizontally centered within `innerW`.

Example (conceptual):

```
+-------+-------+-------+-------+-------+-------+-------+
| 03-01 | 03-02 | 03-03 | 03-04 | 03-05 | 03-06 | 03-07 |
+-------+-------+-------+-------+-------+-------+-------+
```

2) Divider (1 line)

- `-` repeated to `innerW`.

3) Task detail viewport (remaining lines)

- Shows only tasks for the selected day, using existing formatting:
  - Done: `[✓] <title>`
  - Abandoned: `[✗] <title>`
  - Overdue active created on selected day: `[ ] <title>`
  - Delayed rows are red per existing rules
- If there are no rows for the day: show `None`.
- Supports vertical scrolling via `up/down`.
- Changing selected day or rolling window resets scroll to top.

---

## Keybindings / Behavior

- `left/h`:
  - If `historyIndex > 0`: `historyIndex--`, refresh selected day.
  - If `historyIndex == 0`: roll window back by 1 day (`historyFrom--`, `historyTo--`), keep `historyIndex==0`, refresh selected day + stats.

- `right/l`:
  - If `historyIndex < 6`: `historyIndex++`, refresh selected day.
  - If `historyIndex == 6`:
    - If `historyTo == today`: no-op.
    - Else roll window forward by 1 day (clamped so `historyTo` never exceeds today), keep `historyIndex==6`, refresh selected day + stats.

- `up/k` and `down/j`:
  - Adjust `historyScroll` (0..max) to scroll task rows within the detail viewport.
  - If content fits without scrolling: no-op.

---

## Responsive / Narrow Width

If `innerW` is too small for the full ASCII table width:
- Fall back to a single-line date strip (no border) still horizontally centered.
- Selected date still uses reverse-video.
- Dates may be truncated by existing width clipping.

---

## Testing Strategy

- Model navigation tests:
  - `left/right` move `historyIndex` and/or roll the window, including clamp-at-today.
  - `up/down` changes scroll but does not change `historyIndex`.
  - Scroll resets on date change.

- Rendering tests:
  - Date table includes 7 `MM-DD` entries and shows reverse-video for selected day.
  - Divider line length equals `innerW`.
  - Task viewport shows expected slice when scrolled.
