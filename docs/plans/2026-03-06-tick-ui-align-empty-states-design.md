## Background

We currently have inconsistent vertical alignment across views:
- History content was centered within the workspace.
- Today / Upcoming are list-based and top-aligned.

This iteration makes the layout feel coherent:
- All views are top-aligned by default.
- Only the empty-state text for Today/Upcoming is centered within the workspace.

---

## Goals

- Make Today / Upcoming / History all top-aligned for content rendering.
- Center Today/Upcoming empty-state copy within the sheet workspace (both horizontally and vertically).
- Keep Today add-input view unchanged (not centered).
- Update History empty-state label from `(none)` to `None`.

Non-goals:
- Do not change list sizing, scrolling, selection behavior.
- Do not introduce new keybindings.
- Do not center History; keep it top-aligned.

---

## UX / Copy

- Today empty: `Nothing due today.` (centered)
- Upcoming empty: `No upcoming tasks.` (centered)
- History right column empty: `None` (top-aligned)

---

## Layout Rules

Definitions (as implemented in `Model.View()`):
- `workspaceHeight = windowHeight - (header + separators + footer)`
- `innerHeight = workspaceHeight - sheetVertMargin`
- `innerWidth = sheetInnerWidth(windowWidth)`

Rules:
- When a view returns a list view (`m.todayList.View()` / `m.upcomingList.View()`), it is rendered as-is and remains top-aligned.
- For Today/Upcoming empty-state strings, wrap them with `centerInBox(msg, innerWidth, innerHeight)`.
- For History, remove centering and return the joined columns block directly.

---

## Implementation Notes

- Reuse the existing ANSI-aware `centerInBox` helper in `internal/ui/styles.go`.
- Update `internal/ui/views.go`:
  - Today: if no items -> return centered empty message.
  - Upcoming: if no items -> return centered empty message.
  - History: when no rows -> use `None`; remove `centerInBox` call for History.

---

## Test Strategy

- Add UI tests asserting Today/Upcoming empty-state centering:
  - Given a fixed window size, `renderTodayBody(m)` and `renderUpcomingBody(m)` place the message at the expected vertical center line and with expected left padding inside the inner box.
- Update existing History empty-state test to expect `None`.
