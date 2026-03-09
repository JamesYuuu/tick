# Task Selection Unify Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make Today and Upcoming selected task rows use the same selection treatment as tabs and History dates, while keeping delayed task text red even when selected.

**Architecture:** Keep the existing list delegates and only change row rendering logic plus focused tests. Selected rows will stop using the `> ` marker as the primary cue and instead rely on the same selected background system, with delayed rows preserving their red foreground.

**Tech Stack:** Go, Bubble Tea, Lipgloss, Go test.

---

### Task 1: Unify Today and Upcoming selected row rendering

**Files:**
- Modify: `internal/ui/model.go`
- Modify: `internal/ui/model_test.go`
- Test: `internal/ui/model_test.go`

**Step 1: Write the failing test**

Add focused delegate-render tests in `internal/ui/model_test.go`:

```go
func TestTodayItemDelegate_SelectedDelayed_KeepsRedTextAndSelectedBackground(t *testing.T) {
    lipgloss.SetColorProfile(termenv.ANSI256)
    t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

    day := domain.MustParseDay("2026-03-04")
    d := todayItemDelegate{styles: defaultStyles(), currentDay: day}
    l := newTaskList(d)
    l.SetItems([]list.Item{taskItem{task: domain.Task{ID: 1, Title: "late", Status: domain.StatusActive, CreatedDay: day, DueDay: addDays(day, -1)}}})
    l.Select(0)

    var buf bytes.Buffer
    d.Render(&buf, l, 0, l.Items()[0])
    got := buf.String()
    if strings.Contains(got, "> ") {
        t.Fatalf("expected selected row to not use > marker, got %q", got)
    }
    if !strings.Contains(got, "\x1b[41") && !strings.Contains(got, "\x1b[48;") {
        t.Fatalf("expected selected background styling, got %q", got)
    }
    if !strings.Contains(got, "\x1b[31") && !strings.Contains(got, "\x1b[38;5;1m") {
        t.Fatalf("expected delayed red foreground to remain, got %q", got)
    }
}
```

Add a parallel test for `simpleItemDelegate` selected rows showing no `> ` marker and shared selected background.

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ui -run 'TestTodayItemDelegate_SelectedDelayed_KeepsRedTextAndSelectedBackground|TestSimpleItemDelegate_Selected_UsesSharedSelectedStyle' -count=1`
Expected: FAIL because current delegates still use `> ` and `RowSelDl` / `RowSel` paths.

**Step 3: Write minimal implementation**

Update `internal/ui/model.go`:

- Remove the `> ` selected prefix from `todayItemDelegate.Render` and `simpleItemDelegate.Render`.
- Keep unselected rows unchanged.
- For selected normal rows, apply the shared selected background treatment.
- For selected delayed rows, use the selected background but keep the delayed red foreground.

Acceptable shape:

```go
line := it.task.Title
if it.task.IsDelayed(d.currentDay) {
    if selected {
        line = d.styles.DelayedSelected.Render(line)
    } else {
        line = d.styles.Delayed.Render(line)
    }
} else if selected {
    line = d.styles.SelectedRow.Render(line)
}
```

If you do not add new named styles, reuse existing background and foreground pieces cleanly.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/ui -run 'TestTodayItemDelegate_SelectedDelayed_KeepsRedTextAndSelectedBackground|TestSimpleItemDelegate_Selected_UsesSharedSelectedStyle' -count=1`
Expected: PASS

**Step 5: Run package verification and commit**

Run: `go test ./internal/ui -count=1`
Expected: PASS

```bash
git add internal/ui/model.go internal/ui/model_test.go
git commit -m "fix(ui): unify task row selection styling"
```

---

### Task 2: Full verification

**Files:**
- Modify: `internal/ui/model_test.go` (only if needed)

**Step 1: Run full test suite**

Run: `go test ./... -count=1`
Expected: PASS

**Step 2: Review UI diff**

Run: `git diff HEAD~1..HEAD -- internal/ui`
Expected: only task-row rendering and test updates

**Step 3: Commit any final test-only adjustment if needed**

```bash
git add internal/ui/model_test.go
git commit -m "test(ui): cover unified task row selection"
```

Only create this second commit if there are real follow-up test edits after Task 1.

**Step 4: Report status**

Report:
- exact tests run
- whether delayed selected rows keep red foreground
- whether branch is ready to push
