# History View Task 3 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Redo History view body to match final spec: left MM-DD list + divider + right selected-day outcomes with delayed red and overdue-active listing.

**Architecture:** Model refresh for History carries three lists for the selected day: done (done day), abandoned (abandoned day), and active-created (created day). View renders two columns with a vertical divider and formats lines with a uniform `[]` prefix, applying the existing delayed style when `DueDay < shownDay` (done/abandoned) or `DueDay < today` (overdue active).

**Tech Stack:** Go, bubbletea, lipgloss/termenv, internal domain.Day/Task.

---

### Task 1: Update/Write Failing Tests For New History Rendering

**Files:**
- Modify: `internal/ui/model_history_test.go`
- Modify: `internal/ui/model_test.go` (fakeApp)

**Step 1: Write failing tests**
- Assert left column uses `MM-DD` (and does not include `YYYY-MM-DD`).
- Assert right column includes `[✓]`, `[✗]`, `[ ]` formatting and `(none)` empty state.
- Assert delayed lines render in red (ANSI) for done/abandoned when `DueDay < shownDay`.
- Assert overdue-active list only includes tasks where `StatusActive && DueDay < today`.

**Step 2: Run tests to verify they fail**
Run: `go test ./internal/ui -count=1`
Expected: FAIL due to missing fields/methods and old History rendering.

### Task 2: Add HistoryActiveByCreatedDay To UI Model Refresh

**Files:**
- Modify: `internal/ui/model.go`
- Modify: `internal/ui/model_test.go`

**Step 1: Minimal implementation**
- Extend `appClient` with `HistoryActiveByCreatedDay`.
- Add `historyActiveCreated []domain.Task` to `Model`.
- Add `activeCreated []domain.Task` to `historyRefreshMsg`.
- Update `cmdRefreshHistorySelectedDay` and `cmdRefreshHistoryWithStats` to query active-created list.
- Update `Update(historyRefreshMsg)` to store it.
- Update fakeApp to implement `HistoryActiveByCreatedDay` backed by a map keyed by day.

**Step 2: Run tests to move failures to rendering**
Run: `go test ./internal/ui -count=1`

### Task 3: Rewrite renderHistoryBody Layout And Formatting

**Files:**
- Modify: `internal/ui/views.go`

**Step 1: Minimal implementation**
- Left column: 7 days with prefix `> ` for selected and `  ` otherwise; format days as `MM-DD`.
- Middle column: ASCII vertical divider `|` sized to max(left,right) lines.
- Right column: show only selected-day lines, grouped and ordered: done -> abandoned -> overdue-active.
- Formatting:
  - Done: `[✓] <title>`
  - Abandoned: `[✗] <title>`
  - Overdue active: `[ ] <title>` and red.
- Delayed red:
  - Done/abandoned: if `DueDay < selectedDay` then red.
  - Overdue active: filter `StatusActive && DueDay < today` then red.
- Empty: if no lines, show `(none)`.

**Step 2: Run tests to verify green**
Run: `go test ./internal/ui -count=1`

### Task 4: Full Test Pass + Commit

**Step 1: Run full suite**
Run: `go test ./... -count=1`

**Step 2: Commit (no amend)**
Run:
```bash
git add internal/ui/model.go internal/ui/views.go internal/ui/model_test.go internal/ui/model_history_test.go docs/plans/2026-03-05-history-task3.md
git commit -m "feat(ui): redo history layout and delayed highlighting"
```
