# Full Repository Simplification Refactor Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Aggressively simplify the whole repository by deleting duplicate/low-value code and tests while keeping all external behavior unchanged.

**Architecture:** Refactor by layer (`domain -> store/sqlite -> app -> ui`) so regressions are isolated. Introduce a small number of shared helpers to remove repeated logic, then prune redundant tests to a compact behavior-focused suite. Keep interfaces and user-visible behavior stable.

**Tech Stack:** Go 1.24, Bubble Tea/Lipgloss (TUI), SQLite (modernc.org/sqlite), Go test tooling.

---

### Task 1: Add Shared UI Layout Metrics Helper

**Files:**
- Create: `internal/ui/layout.go`
- Modify: `internal/ui/model_test.go`
- Modify: `internal/ui/model.go`
- Test: `internal/ui/model_test.go`

**Step 1: Write the failing test**

```go
func TestLayoutMetrics_Consistency(t *testing.T) {
    g := calcLayoutMetrics(80, 24)
    if g.contentW != contentWidth(80) {
        t.Fatalf("content width mismatch: got %d want %d", g.contentW, contentWidth(80))
    }
    if g.innerW != sheetInnerWidth(80) {
        t.Fatalf("inner width mismatch: got %d want %d", g.innerW, sheetInnerWidth(80))
    }
    if g.workspaceH != 19 { // 24 - (header1 + sep1 + sep1 + footer2)
        t.Fatalf("workspace height mismatch: got %d want 19", g.workspaceH)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ui -run TestLayoutMetrics_Consistency -count=1`
Expected: FAIL with `undefined: calcLayoutMetrics`

**Step 3: Write minimal implementation**

```go
type layoutMetrics struct {
    contentW   int
    innerW     int
    workspaceH int
    innerH     int
}

func calcLayoutMetrics(windowW, windowH int) layoutMetrics {
    workspaceH := windowH - (1 + 1 + 1 + 2)
    if workspaceH < 0 {
        workspaceH = 0
    }
    innerH := workspaceH - sheetVertMargin
    if innerH < 0 {
        innerH = 0
    }
    return layoutMetrics{
        contentW:   contentWidth(windowW),
        innerW:     sheetInnerWidth(windowW),
        workspaceH: workspaceH,
        innerH:     innerH,
    }
}
```

Use `calcLayoutMetrics` in `Model.Update(tea.WindowSizeMsg)` for list/input sizing.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/ui -run TestLayoutMetrics_Consistency -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/layout.go internal/ui/model.go internal/ui/model_test.go
git commit -m "refactor(ui): add shared layout metrics helper"
```

---

### Task 2: Remove Duplicate Empty-State Geometry in UI Views

**Files:**
- Modify: `internal/ui/views.go`
- Modify: `internal/ui/model_test.go`
- Test: `internal/ui/model_test.go`

**Step 1: Write the failing test**

```go
func TestRenderCenteredEmpty_UsesLayoutInnerBox(t *testing.T) {
    m := NewWithDeps(newFakeApp(domain.MustParseDay("2026-03-04"), nil), fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
    um, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
    m = um.(Model)

    out := renderCenteredEmpty(m, "Nothing due today.")
    lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
    if len(lines) != calcLayoutMetrics(80, 24).innerH {
        t.Fatalf("expected centered empty to use innerH")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ui -run TestRenderCenteredEmpty_UsesLayoutInnerBox -count=1`
Expected: FAIL with `undefined: renderCenteredEmpty`

**Step 3: Write minimal implementation**

```go
func renderCenteredEmpty(m Model, msg string) string {
    g := calcLayoutMetrics(m.width, m.height)
    return centerInBox(msg, g.innerW, g.innerH)
}
```

Update `renderTodayBody` and `renderUpcomingBody` to use `renderCenteredEmpty` and remove duplicated width/height calculations.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/ui -run 'TestRenderCenteredEmpty_UsesLayoutInnerBox|TestRenderTodayBody_EmptyCenteredInWorkspace|TestRenderUpcomingBody_EmptyCenteredInWorkspace' -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/views.go internal/ui/model_test.go
git commit -m "refactor(ui): dedupe empty-state geometry"
```

---

### Task 3: Merge Duplicated History Refresh Pipelines

**Files:**
- Modify: `internal/ui/model.go`
- Modify: `internal/ui/model_history_test.go`
- Test: `internal/ui/model_history_test.go`

**Step 1: Write the failing test**

```go
func TestModel_HistoryRefresh_StatsOnlyWhenRequested(t *testing.T) {
    disableTick(t)
    day := domain.MustParseDay("2026-03-07")
    a := newFakeApp(day, nil)
    m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)}, time.UTC)
    m.view = viewHistory
    m.historyFrom = domain.MustParseDay("2026-03-01")
    m.historyTo = day
    m.historyIndex = 6

    m = applyCmd(m, m.cmdRefreshHistory(false))
    if a.statsCalls != 0 {
        t.Fatalf("expected stats not called")
    }

    m = applyCmd(m, m.cmdRefreshHistory(true))
    if a.statsCalls != 1 {
        t.Fatalf("expected stats called once")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ui -run TestModel_HistoryRefresh_StatsOnlyWhenRequested -count=1`
Expected: FAIL with `m.cmdRefreshHistory undefined`

**Step 3: Write minimal implementation**

```go
func (m Model) cmdRefreshHistory(withStats bool) tea.Cmd {
    day := m.historySelectedDay()
    from, to := m.historyFrom, m.historyTo
    return func() tea.Msg {
        ctx := context.Background()
        done, abandoned, activeCreated, err := m.loadHistoryDay(ctx, day)
        if err != nil {
            return historyRefreshMsg{err: err}
        }
        if !withStats {
            return historyRefreshMsg{done: done, abandoned: abandoned, activeCreated: activeCreated}
        }
        stats, err := m.app.Stats(ctx, from, to)
        if err != nil {
            return historyRefreshMsg{err: err}
        }
        return historyRefreshMsg{done: done, abandoned: abandoned, activeCreated: activeCreated, stats: stats, hasStats: true}
    }
}
```

Refactor `cmdRefreshHistorySelectedDay` and `cmdRefreshHistoryWithStats` to call this helper.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/ui -run 'TestModel_HistoryRefresh_StatsOnlyWhenRequested|TestModel_History_.*' -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/model.go internal/ui/model_history_test.go
git commit -m "refactor(ui): unify history refresh command flow"
```

---

### Task 4: Consolidate Repeated SQLite Task Query Loops

**Files:**
- Modify: `internal/store/sqlite/sqlite.go`
- Create: `internal/store/sqlite/sqlite_internal_test.go`
- Test: `internal/store/sqlite/sqlite_internal_test.go`

**Step 1: Write the failing test**

```go
func TestQueryTasks_EmptyResult(t *testing.T) {
    s, err := OpenInMemory()
    if err != nil {
        t.Fatalf("open: %v", err)
    }
    t.Cleanup(func() { _ = s.Close() })

    got, err := s.queryTasks(context.Background(), "test query", `SELECT id, title, status, created_day, due_day, done_day, abandoned_day FROM tasks WHERE 1=0`)
    if err != nil {
        t.Fatalf("queryTasks: %v", err)
    }
    if len(got) != 0 {
        t.Fatalf("expected empty result, got %#v", got)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/store/sqlite -run TestQueryTasks_EmptyResult -count=1`
Expected: FAIL with `s.queryTasks undefined`

**Step 3: Write minimal implementation**

```go
func (s *SQLiteStore) queryTasks(ctx context.Context, op, q string, args ...any) ([]domain.Task, error) {
    rows, err := s.db.QueryContext(ctx, q, args...)
    if err != nil {
        return nil, fmt.Errorf("%s: %w", op, err)
    }
    defer rows.Close()

    out := make([]domain.Task, 0)
    for rows.Next() {
        t, err := scanTask(rows)
        if err != nil {
            return nil, fmt.Errorf("%s: %w", op, err)
        }
        out = append(out, t)
    }
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("%s: %w", op, err)
    }
    return out, nil
}
```

Switch `ListActiveByCreatedDay`, `ListDoneByDay`, and `ListAbandonedByDay` to use this helper.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/store/sqlite -run 'TestQueryTasks_EmptyResult|TestSQLiteStore_BasicFlow|TestSQLiteStore_ListActiveByCreatedDay_FiltersByDay' -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/store/sqlite/sqlite.go internal/store/sqlite/sqlite_internal_test.go
git commit -m "refactor(store): unify sqlite task query loops"
```

---

### Task 5: Consolidate MarkDone/MarkAbandoned Update Logic

**Files:**
- Modify: `internal/store/sqlite/sqlite.go`
- Modify: `internal/store/sqlite/sqlite_test.go`
- Test: `internal/store/sqlite/sqlite_test.go`

**Step 1: Write the failing test**

```go
func TestSetStatusDay_NotFoundWrapsNoRows(t *testing.T) {
    s, err := sqlite.OpenInMemory()
    if err != nil {
        t.Fatalf("open: %v", err)
    }
    t.Cleanup(func() { _ = s.Close() })

    err = s.setStatusDay(context.Background(), "mark done", 999, domain.StatusDone, "done_day", domain.MustParseDay("2026-03-05"), "abandoned_day")
    if !errors.Is(err, sql.ErrNoRows) {
        t.Fatalf("expected sql.ErrNoRows, got %v", err)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/store/sqlite -run TestSetStatusDay_NotFoundWrapsNoRows -count=1`
Expected: FAIL with `s.setStatusDay undefined`

**Step 3: Write minimal implementation**

```go
func (s *SQLiteStore) setStatusDay(ctx context.Context, op string, id int64, status domain.Status, dayColumn string, day domain.Day, clearColumn string) error {
    q := fmt.Sprintf(`UPDATE tasks SET status = ?, %s = ?, %s = NULL WHERE id = ?`, dayColumn, clearColumn)
    res, err := s.db.ExecContext(ctx, q, string(status), day.String(), id)
    if err != nil {
        return fmt.Errorf("%s: %w", op, err)
    }
    n, err := res.RowsAffected()
    if err != nil {
        return fmt.Errorf("%s: %w", op, err)
    }
    if n == 0 {
        return fmt.Errorf("%s: id=%d: %w", op, id, sql.ErrNoRows)
    }
    return nil
}
```

Refactor `MarkDone` and `MarkAbandoned` to call this helper.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/store/sqlite -run 'TestSetStatusDay_NotFoundWrapsNoRows|TestSQLiteStore_MarkDone_NotFound_WrapsNoRows|TestSQLiteStore_MarkAbandoned_NotFound_WrapsNoRows|TestSQLiteStore_BasicFlow' -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/store/sqlite/sqlite.go internal/store/sqlite/sqlite_test.go
git commit -m "refactor(store): unify mark status update logic"
```

---

### Task 6: Simplify App Layer Repeated Flows

**Files:**
- Modify: `internal/app/app.go`
- Modify: `internal/app/outcome_ratios.go`
- Create: `internal/app/app_internal_test.go`
- Test: `internal/app/app_internal_test.go`

**Step 1: Write the failing test**

```go
func TestHistoryByDay_WrapsErrorWithPrefix(t *testing.T) {
    day := domain.MustParseDay("2026-03-07")
    _, err := historyByDay("history done", day, func(context.Context, domain.Day) ([]domain.Task, error) {
        return nil, errors.New("boom")
    })
    if err == nil || !strings.Contains(err.Error(), "history done") {
        t.Fatalf("expected wrapped prefix error, got %v", err)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/app -run TestHistoryByDay_WrapsErrorWithPrefix -count=1`
Expected: FAIL with `undefined: historyByDay`

**Step 3: Write minimal implementation**

```go
type OutcomeRatios store.OutcomeRatios

func historyByDay(prefix string, day domain.Day, fn func(context.Context, domain.Day) ([]domain.Task, error)) ([]domain.Task, error) {
    out, err := fn(context.Background(), day)
    if err != nil {
        return nil, fmt.Errorf("%s: %w", prefix, err)
    }
    return out, nil
}

func (a *App) listActive(ctx context.Context, window store.ActiveWindow) ([]domain.Task, error) {
    return a.store.ListActive(ctx, store.ListActiveParams{CurrentDay: a.currentDay(), Window: window})
}
```

Use `listActive` in `Today`/`Upcoming`, use `historyByDay` in history methods, and return `OutcomeRatios(out)` in `Stats`.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/app -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/app/app.go internal/app/outcome_ratios.go internal/app/app_internal_test.go
git commit -m "refactor(app): collapse repeated query and mapping flows"
```

---

### Task 7: Aggressively Prune Redundant SQLite Tests

**Files:**
- Modify: `internal/store/sqlite/sqlite_test.go`
- Delete: `internal/store/sqlite/sqlite_query_test.go`
- Delete: `internal/store/sqlite/dsn_test.go`
- Test: `internal/store/sqlite/sqlite_test.go`

**Step 1: Write replacement tests before deletion**

Add table-driven coverage inside `sqlite_test.go`:

```go
func TestSQLiteStore_MarkStatus_NotFound_WrapsNoRows(t *testing.T) {
    cases := []struct{
        name string
        call func(*sqlite.SQLiteStore, context.Context) error
        wantSub string
    }{
        {"done", func(s *sqlite.SQLiteStore, ctx context.Context) error { return s.MarkDone(ctx, 12345, domain.MustParseDay("2026-03-05")) }, "mark done"},
        {"abandoned", func(s *sqlite.SQLiteStore, ctx context.Context) error { return s.MarkAbandoned(ctx, 12345, domain.MustParseDay("2026-03-06")) }, "mark abandoned"},
    }
    // assert errors.Is(err, sql.ErrNoRows) and substring present
}

func TestSQLiteStore_DSNForPath_EscapesSpaces(t *testing.T) {
    got := sqlite.DSNForPathForTest("/tmp/dir with spaces/todo.db") // or keep same-package test if preferred
    want := "file:/tmp/dir%20with%20spaces/todo.db"
    if got != want { t.Fatalf("...") }
}
```

If you avoid test-only exports, keep DSN assertion in same-package test file and merge there.

**Step 2: Run tests before deleting old files**

Run: `go test ./internal/store/sqlite -run 'TestSQLiteStore_MarkStatus_NotFound_WrapsNoRows|TestSQLiteStore_DSNForPath_EscapesSpaces' -count=1`
Expected: PASS

**Step 3: Delete redundant tests**

- Remove literal-query string test (`sqlite_query_test.go`)
- Remove standalone duplicate DSN test file if merged elsewhere (`dsn_test.go`)

**Step 4: Run package tests**

Run: `go test ./internal/store/sqlite -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/store/sqlite/sqlite_test.go
git rm internal/store/sqlite/sqlite_query_test.go internal/store/sqlite/dsn_test.go
git commit -m "test(store): remove redundant sqlite tests and merge coverage"
```

---

### Task 8: Aggressively Prune and Consolidate UI Tests

**Files:**
- Create: `internal/ui/testkit_test.go`
- Modify: `internal/ui/model_test.go`
- Modify: `internal/ui/model_tick_test.go`
- Delete: `internal/ui/branding_test.go`
- Test: `internal/ui/model_test.go`
- Test: `internal/ui/model_history_test.go`
- Test: `internal/ui/model_tick_test.go`

**Step 1: Write replacement table-driven tests first**

Add high-signal consolidated tests in `model_test.go`:

```go
func TestModel_TodayActions_Table(t *testing.T) {
    cases := []struct{
        name string
        key tea.KeyMsg
        wantDone []int64
        wantAbandoned []int64
        wantPostponed []int64
    }{
        {"done", keyRune('x'), []int64{1}, nil, nil},
        {"abandon", keyRune('d'), nil, []int64{1}, nil},
        {"postpone", keyRune('p'), nil, nil, []int64{1}},
    }
    // setup once per case, assert behavior per action
}

func TestRenderEmptyStates_Table(t *testing.T) {
    // today/upcoming in one table-driven test
}
```

**Step 2: Run new tests to verify they pass before deletions**

Run: `go test ./internal/ui -run 'TestModel_TodayActions_Table|TestRenderEmptyStates_Table' -count=1`
Expected: PASS

**Step 3: Extract shared testkit and delete redundant tests**

- Move shared fakes/helpers from `model_test.go` into `testkit_test.go`
- Reuse them in `model_tick_test.go`
- Delete `branding_test.go` after ensuring equivalent behavior is covered in existing view/layout tests

**Step 4: Run full UI tests**

Run: `go test ./internal/ui -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/testkit_test.go internal/ui/model_test.go internal/ui/model_tick_test.go internal/ui/model_history_test.go
git rm internal/ui/branding_test.go
git commit -m "test(ui): consolidate helpers and remove redundant cases"
```

---

### Task 9: Remove Dead/Redundant UI Production Code

**Files:**
- Modify: `internal/ui/views.go`
- Modify: `internal/ui/styles.go`
- Modify: `internal/ui/model.go`
- Test: `internal/ui/model_test.go`

**Step 1: Write a smoke test guarding all top-level views**

```go
func TestModel_View_Smoke_AllViews_RenderNonEmpty(t *testing.T) {
    m := NewWithDeps(newFakeApp(domain.MustParseDay("2026-03-04"), nil), fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
    um, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
    m = um.(Model)
    if strings.TrimSpace(m.View()) == "" { t.Fatalf("today view empty") }
    m.view = viewUpcoming
    if strings.TrimSpace(m.View()) == "" { t.Fatalf("upcoming view empty") }
    m.view = viewHistory
    if strings.TrimSpace(m.View()) == "" { t.Fatalf("history view empty") }
}
```

**Step 2: Run test to verify baseline**

Run: `go test ./internal/ui -run TestModel_View_Smoke_AllViews_RenderNonEmpty -count=1`
Expected: PASS

**Step 3: Remove dead helpers and duplicated paths**

Delete functions that add no behavior/value and are no longer needed after previous tasks, e.g.:

- `renderToday`, `renderUpcoming`, `renderHistory` (if unused)
- `Model.frame` (if unused)
- `bodyHeight` (unused)
- `max` (unused)

Then simplify imports accordingly.

**Step 4: Run UI tests**

Run: `go test ./internal/ui -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/views.go internal/ui/styles.go internal/ui/model.go internal/ui/model_test.go
git commit -m "refactor(ui): remove dead helpers and simplify render paths"
```

---

### Task 10: Full Regression Verification and Final Cleanup

**Files:**
- Modify (only if required): `README.md`

**Step 1: Run full test suite**

Run: `go test ./... -count=1`
Expected: PASS (all packages)

**Step 2: Check for accidental behavior/doc drift**

Run: `git diff -- README.md`
Expected: empty diff unless behavior-facing text changed

**Step 3: Optional minimal doc sync**

If key descriptions changed during simplification, update only affected lines in `README.md`.

**Step 4: Re-run full tests**

Run: `go test ./... -count=1`
Expected: PASS

**Step 5: Final commit (or squash later by preference)**

```bash
git add -A
git commit -m "refactor: aggressively simplify code and trim redundant tests"
```

---

## Execution Notes

- Keep behavior stable: no user-visible command/key/flow change.
- Prefer deleting code over introducing new abstractions unless helper removes clear duplication.
- Keep commit cadence frequent (one task = one commit).
- If any task causes unstable failures, stop and isolate by package before proceeding.
