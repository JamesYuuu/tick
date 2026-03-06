# History Date Table + Scrollable Details Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Redesign History to use a horizontally centered 7-day ASCII table selector, navigate dates with left/right (auto-rolling window by 1 day, clamped at today), and scroll the task detail area with up/down.

**Architecture:** Keep History state in `Model` (`historyFrom`/`historyTo`/`historyIndex`) and add `historyScroll` for the detail viewport. Render History body as: date selector table, divider, and a sliced task viewport. Reuse existing history refresh commands; only key handling and rendering change.

**Tech Stack:** Go, Bubble Tea, Lipgloss, charmbracelet/x/ansi

---

### Task 1: Add History scroll state and tests (up/down scrolls details)

**Files:**
- Modify: `internal/ui/model.go`
- Test: `internal/ui/model_history_test.go`

**Step 1: Write failing tests for scroll behavior**

Add tests in `internal/ui/model_history_test.go`:

- `TestModel_History_UpDownScrollsDetails_NotDate`
  - Put model in `viewHistory` with a fixed window.
  - Set a selected day and populate enough history rows to exceed the detail viewport height.
  - Press `down/j`: expect `historyScroll` increases and `historyIndex` unchanged.
  - Press `up/k`: expect `historyScroll` decreases and `historyIndex` unchanged.

- `TestModel_History_ScrollResetsOnDateChange`
  - Set `historyScroll > 0`.
  - Press `right`: expect `historyScroll` resets to 0.

Run: `go test ./internal/ui -run TestModel_History_UpDownScrollsDetails_NotDate -count=1`
Expected: FAIL

**Step 2: Implement `historyScroll` field**

In `internal/ui/model.go`:
- Add `historyScroll int` to `Model`.
- Ensure it is reset to 0 when:
  - entering History view
  - changing selected date (`left/right` or window roll)
  - receiving a history refresh message (safe reset)

**Step 3: Wire up `up/down` to scroll**

In `tea.KeyMsg` handling when `m.view == viewHistory`:
- Stop using `HistoryUp/HistoryDown` to change dates.
- Instead, adjust `historyScroll` (clamped to >=0; max handled in rendering).

Run: `go test ./internal/ui -count=1`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/ui/model.go internal/ui/model_history_test.go
git commit -m "ui: add history detail scrolling"
```

---

### Task 2: Re-introduce History left/right navigation for date selection

**Files:**
- Modify: `internal/ui/keys.go`
- Modify: `internal/ui/styles.go`
- Modify: `internal/ui/model.go`
- Test: `internal/ui/model_history_test.go`
- Modify: `README.md`

**Step 1: Write failing tests for left/right date navigation**

Add tests:
- `TestModel_History_LeftRightMovesSelectedDate`
- `TestModel_History_LeftAtEdgeRollsWindowBackOneDay`
- `TestModel_History_RightAtEdgeRollsWindowForwardOneDay_ClampedToToday`

Run: `go test ./internal/ui -run TestModel_History_LeftRightMovesSelectedDate -count=1`
Expected: FAIL

**Step 2: Re-add key bindings**

In `internal/ui/keys.go`:
- Add `HistoryLeft` and `HistoryRight` back.
- Bind `left`/`h` and `right`/`l`.

In `internal/ui/styles.go`:
- Update help for History:
  - `left/h:right/l` to move date
  - `up/k:down/j` to scroll tasks

**Step 3: Implement navigation behavior**

In `internal/ui/model.go` under `viewHistory` key handling:
- `HistoryLeft`:
  - If `historyIndex>0`: `historyIndex--`, reset scroll, refresh selected day.
  - Else: roll window back by 1 day, keep index 0, reset scroll, refresh with stats.

- `HistoryRight`:
  - If `historyIndex<6`: `historyIndex++`, reset scroll, refresh selected day.
  - Else: if `historyTo==today` no-op; otherwise roll forward by 1 day (clamp to today), keep index 6, reset scroll, refresh with stats.

**Step 4: Update README**

In `README.md`, update History keys to reflect the new behavior.

**Step 5: Run UI tests**

Run: `go test ./internal/ui -count=1`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/ui/keys.go internal/ui/styles.go internal/ui/model.go internal/ui/model_history_test.go README.md
git commit -m "ui: navigate history dates with left/right"
```

---

### Task 3: Render date selector as centered ASCII table + divider + scrollable details

**Files:**
- Modify: `internal/ui/views.go`
- Modify: `internal/ui/styles.go`
- Test: `internal/ui/model_history_test.go`

**Step 1: Write failing rendering tests**

Add tests validating:
- The date selector renders 7 dates `MM-DD` in a table-like layout.
- The selected date is reverse-video (verify ANSI SGR 7 or check for style output under ANSI256 profile).
- The divider line is `innerW` dashes.
- Scrolling shows different slices of task rows.

Run: `go test ./internal/ui -run TestRenderHistoryBody_DateTable -count=1`
Expected: FAIL

**Step 2: Implement rendering helpers**

In `internal/ui/styles.go`:
- Add a `Reverse` style (or reuse existing RowSel style) for the selected date cell.
- Add helper `centerLinesInWidth(block string, w int) string` (ANSI-aware) to prefix each line with the same left padding.

In `internal/ui/views.go`:
- Replace current left/right column join with:
  - `renderHistoryDateTable(m)` returning 3-line block (or 1-line fallback if too narrow)
  - `divider := strings.Repeat("-", innerW)`
  - `renderHistoryDetailsViewport(m, detailH)` slicing rows using `m.historyScroll`

Compute:
- `innerW := sheetInnerWidth(m.width)`
- `workspaceH := m.height - (1 + 1 + 1 + 2)`
- `innerH := max(0, workspaceH - sheetVertMargin)`
- `detailH := max(0, innerH - 4)` (3 date lines + 1 divider)

**Step 3: Run UI tests**

Run: `go test ./internal/ui -count=1`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/ui/views.go internal/ui/styles.go internal/ui/model_history_test.go
git commit -m "ui: render history date table and scrollable details"
```

---

### Task 4: Full verification

**Step 1: Run full test suite**

Run: `go test ./... -count=1`
Expected: PASS

**Step 2: Manual smoke (optional)**

Run: `go run ./cmd/tick`

Check:
- History shows top date table; selected date is reverse-video.
- `left/right` move selection and roll window at edges; forward roll clamps at today.
- `up/down` scroll task details only.
