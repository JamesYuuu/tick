# Align Views + Center Empty States Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make Today/Upcoming/History top-aligned for content, while centering only Today/Upcoming empty-state messages, and change History empty label to `None`.

**Architecture:** Reuse the existing ANSI-aware `centerInBox` helper to center empty-state strings within the sheet inner box. Remove History centering so all primary content blocks are top-aligned.

**Tech Stack:** Go, Bubble Tea, Bubbles list, Lipgloss, charmbracelet/x/ansi

---

### Task 1: Update History empty label and remove History centering

**Files:**
- Modify: `internal/ui/views.go`
- Test: `internal/ui/model_history_test.go`

**Step 1: Update the failing test expectation**

In `internal/ui/model_history_test.go`, update the test that currently expects `(none)` to expect `None`.

Run: `go test ./internal/ui -run TestRenderHistoryBody_Empty -count=1`
Expected: FAIL (until implementation is updated)

**Step 2: Implement History changes**

In `internal/ui/views.go`:
- Change the empty rows label from `(none)` to `None`.
- Remove `centerInBox(...)` usage from `renderHistoryBody`; return the `cols` block directly.

**Step 3: Run the UI tests**

Run: `go test ./internal/ui -count=1`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/ui/views.go internal/ui/model_history_test.go
git commit -m "ui: top-align history and rename empty label"
```

---

### Task 2: Center Today/Upcoming empty-state messages inside workspace

**Files:**
- Modify: `internal/ui/views.go`
- Test: `internal/ui/model_test.go`

**Step 1: Write failing centering tests**

Add two tests in `internal/ui/model_test.go`:

- `TestRenderTodayBody_EmptyCenteredInWorkspace`
- `TestRenderUpcomingBody_EmptyCenteredInWorkspace`

Test setup guidelines:
- Create a model with an empty task set.
- Set `m.width`/`m.height` (or apply a `tea.WindowSizeMsg`) to a fixed size, e.g. 80x24.
- Compute the inner box:
  - `innerW := sheetInnerWidth(m.width)`
  - `workspaceH := m.height - (1 + 1 + 1 + 2)`
  - `innerH := max(0, workspaceH - sheetVertMargin)`
- Call `renderTodayBody(m)` / `renderUpcomingBody(m)`.

Assertions:
- The returned body has exactly `innerH` lines.
- The message appears at `topPad := (innerH - 1) / 2` (0-based).
- The message line has the expected left padding inside the inner box:
  - `leftPad := (innerW - ansi.StringWidth(msg)) / 2` (clamped to >= 0)
  - `strings.TrimLeft(line, " ") == msg`

Run: `go test ./internal/ui -run TestRenderTodayBody_EmptyCenteredInWorkspace -count=1`
Expected: FAIL

**Step 2: Implement empty-state centering**

In `internal/ui/views.go`:

- Today empty case:

```go
if len(m.todayList.Items()) == 0 {
    msg := "Nothing due today."
    innerW := sheetInnerWidth(m.width)
    workspaceH := m.height - (1 + 1 + 1 + 2)
    innerH := workspaceH - sheetVertMargin
    if innerH < 0 { innerH = 0 }
    return centerInBox(msg, innerW, innerH)
}
```

- Upcoming empty case: same pattern with `"No upcoming tasks."`.

Keep `adding` behavior unchanged (do not center add input).

**Step 3: Run UI tests**

Run: `go test ./internal/ui -count=1`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/ui/views.go internal/ui/model_test.go
git commit -m "ui: center empty states in today and upcoming"
```

---

### Task 3: Full verification

**Step 1: Run full test suite**

Run: `go test ./... -count=1`
Expected: PASS

**Step 2: Manual smoke (optional)**

Run: `go run ./cmd/tick`

Check:
- Today/Upcoming empty messages appear centered in the workspace.
- Today add input view unchanged.
- History content is top-aligned; empty right column reads `None`.
