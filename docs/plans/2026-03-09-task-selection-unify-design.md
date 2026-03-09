## Background

Today and Upcoming task rows still use a separate selected-row style from the tab/header selection and History date selection. The desired behavior is to unify the selected state visually while preserving special emphasis for delayed tasks.

---

## Goals

- Make Today and Upcoming selected task rows use the same selection treatment as tabs and History dates.
- Preserve delayed-task emphasis by keeping delayed text red even when the row is selected.
- Remove the old `> ` selected-row prefix if it conflicts with the unified selected style.

---

## Chosen Design

- Selected task rows use the shared selected background treatment.
- Normal selected rows use the normal selected foreground.
- Delayed selected rows use the same selected background, but keep the delayed red foreground.
- Unselected rows keep current behavior.

This yields four clear states:

- normal unselected: default text
- normal selected: selected background + normal selected foreground
- delayed unselected: red text
- delayed selected: selected background + red text

---

## Implementation Notes

- Update `todayItemDelegate.Render` and `simpleItemDelegate.Render` in `internal/ui/model.go`.
- Remove the `> ` selected prefix so visual selection comes from the row styling itself.
- Reuse the existing selected background color and delayed foreground color instead of inventing a new style system.

---

## Testing

- Add focused tests for Today and Upcoming row rendering.
- Verify selected rows no longer rely on `> `.
- Verify delayed selected rows keep red foreground while gaining selected background.

---

## Acceptance Criteria

- Today and Upcoming selected rows visually align with the tab/date selection system.
- Delayed text stays red when selected.
- No `> ` marker remains as the primary selection cue.
