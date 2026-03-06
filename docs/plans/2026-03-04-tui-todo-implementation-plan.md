# TUI Todo Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a terminal TUI todo app with Today/Upcoming/History, 3 outcomes (done/abandoned/active), system-time day rollover, and delayed highlighting (`current_day > due_day`) plus delayed done/abandoned ratios.

**Architecture:** Separate pure domain logic (date rules, delayed classification, stats) from persistence (SQLite repository) and UI (Bubble Tea). UI reads `current_day` from a clock abstraction and refreshes on day rollover.

**Tech Stack:** Go 1.22+, Bubble Tea (TUI), Lip Gloss (styles), SQLite (local storage, no sync), `testing` + `testify/require`.

---

## Notes / Defaults

- Timezone: default to system local timezone; allow override via config/env.
- Storage path: default to `~/.tuitodo/todo.db`.
- "Advance Day" is not a user command; UI detects day rollover via a periodic tick.
- User command "Postpone" increments `due_day` (+1 day).
- Today view shows `active` tasks where `due_day <= current_day`.
- Upcoming view shows `active` tasks where `due_day > current_day`.
- Delayed styling when `due_day < current_day`.

## Task 1: Initialize Repo + Module

**Files:**
- Create: `go.mod`
- Create: `cmd/tuitodo/main.go`
- Create: `internal/app/version.go`
- Create: `README.md`

**Step 1: Initialize module**

Run: `go mod init tuitodo`
Expected: creates `go.mod`.

**Step 2: Create minimal CLI entry**

`cmd/tuitodo/main.go` (minimal):

```go
package main

import "fmt"

func main() {
  fmt.Println("tuitodo")
}
```

**Step 3: Run to verify**

Run: `go run ./cmd/tuitodo`
Expected: prints `tuitodo`.

**Step 4: Commit**

Run:

```bash
git init
git add go.mod cmd/tuitodo/main.go
git commit -m "chore: init go module"
```

## Task 2: Define Domain Types (Task, Status, Day)

**Files:**
- Create: `internal/domain/day.go`
- Create: `internal/domain/task.go`
- Test: `internal/domain/task_test.go`

**Step 1: Write failing tests for delayed classification**

`internal/domain/task_test.go`:

```go
package domain

import "testing"

func TestTask_IsDelayed(t *testing.T) {
  current := MustParseDay("2026-03-04")

  t.Run("not delayed when due_day equals current", func(t *testing.T) {
    task := Task{Status: StatusActive, DueDay: MustParseDay("2026-03-04")}
    if task.IsDelayed(current) {
      t.Fatalf("expected not delayed")
    }
  })

  t.Run("delayed when due_day before current", func(t *testing.T) {
    task := Task{Status: StatusActive, DueDay: MustParseDay("2026-03-03")}
    if !task.IsDelayed(current) {
      t.Fatalf("expected delayed")
    }
  })

  t.Run("not delayed when not active", func(t *testing.T) {
    task := Task{Status: StatusDone, DueDay: MustParseDay("2026-03-03")}
    if task.IsDelayed(current) {
      t.Fatalf("expected not delayed")
    }
  })
}
```

**Step 2: Run tests (should fail)**

Run: `go test ./...`
Expected: FAIL (types/functions not defined).

**Step 3: Implement minimal domain**

`internal/domain/day.go`:

```go
package domain

import (
  "time"
)

// Day is a local date in YYYY-MM-DD.
type Day struct{ y, m, d int }

func MustParseDay(s string) Day {
  day, err := ParseDay(s)
  if err != nil {
    panic(err)
  }
  return day
}

func ParseDay(s string) (Day, error) {
  t, err := time.Parse("2006-01-02", s)
  if err != nil {
    return Day{}, err
  }
  return Day{y: t.Year(), m: int(t.Month()), d: t.Day()}, nil
}

func (d Day) String() string {
  return time.Date(d.y, time.Month(d.m), d.d, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
}

func (d Day) Before(other Day) bool {
  if d.y != other.y {
    return d.y < other.y
  }
  if d.m != other.m {
    return d.m < other.m
  }
  return d.d < other.d
}
```

`internal/domain/task.go`:

```go
package domain

type Status string

const (
  StatusActive    Status = "active"
  StatusDone      Status = "done"
  StatusAbandoned Status = "abandoned"
)

type Task struct {
  ID           int64
  Title        string
  Status       Status
  CreatedDay   Day
  DueDay       Day
  DoneDay      *Day
  AbandonedDay *Day
}

func (t Task) IsDelayed(currentDay Day) bool {
  return t.Status == StatusActive && t.DueDay.Before(currentDay)
}
```

**Step 4: Run tests (should pass)**

Run: `go test ./...`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/domain/day.go internal/domain/task.go internal/domain/task_test.go
git commit -m "feat: add domain task model and delayed rule"
```

## Task 3: Add Clock Abstraction + Current Day Computation

**Files:**
- Create: `internal/timeutil/clock.go`
- Create: `internal/timeutil/day.go`
- Test: `internal/timeutil/day_test.go`

**Step 1: Write failing tests for day rollover detection**

`internal/timeutil/day_test.go`:

```go
package timeutil

import (
  "testing"
  "time"

  "tuitodo/internal/domain"
)

type fakeClock struct{ t time.Time }

func (f fakeClock) Now() time.Time { return f.t }

func TestCurrentDay_LocalTZ(t *testing.T) {
  loc := time.FixedZone("X", 8*3600)
  c := fakeClock{t: time.Date(2026, 3, 4, 1, 2, 3, 0, loc)}

  got := CurrentDay(c, loc)
  want := domain.MustParseDay("2026-03-04")
  if got.String() != want.String() {
    t.Fatalf("got %s want %s", got.String(), want.String())
  }
}
```

**Step 2: Run tests (should fail)**

Run: `go test ./...`
Expected: FAIL (missing package/functions).

**Step 3: Implement clock + current day**

`internal/timeutil/clock.go`:

```go
package timeutil

import "time"

type Clock interface {
  Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }
```

`internal/timeutil/day.go`:

```go
package timeutil

import (
  "time"

  "tuitodo/internal/domain"
)

func CurrentDay(c Clock, loc *time.Location) domain.Day {
  t := c.Now().In(loc)
  return domain.MustParseDay(t.Format("2006-01-02"))
}
```

**Step 4: Run tests (should pass)**

Run: `go test ./...`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/timeutil/clock.go internal/timeutil/day.go internal/timeutil/day_test.go
git commit -m "feat: add clock abstraction and current day computation"
```

## Task 4: Define Repository Interfaces (Persistence Boundary)

**Files:**
- Create: `internal/store/store.go`
- Create: `internal/store/query.go`
- Create: `internal/store/store_test.go`

**Step 1: Write failing compile-time tests for interface contract**

`internal/store/store_test.go`:

```go
package store

import "testing"

func TestStore_Interface(t *testing.T) {
  var _ Store = (*SQLiteStore)(nil)
}
```

**Step 2: Run tests (should fail)**

Run: `go test ./...`
Expected: FAIL (SQLiteStore not defined yet).

**Step 3: Define interfaces and query structs**

`internal/store/query.go`:

```go
package store

import "tuitodo/internal/domain"

type ActiveWindow int

const (
  ActiveDueLTECurrent ActiveWindow = iota // due_day <= current_day
  ActiveDueGTCurrent                      // due_day > current_day
)

type ListActiveParams struct {
  CurrentDay domain.Day
  Window     ActiveWindow
}
```

`internal/store/store.go`:

```go
package store

import (
  "context"

  "tuitodo/internal/domain"
)

type Store interface {
  Close() error

  CreateTask(ctx context.Context, title string, createdDay, dueDay domain.Day) (domain.Task, error)
  ListActive(ctx context.Context, p ListActiveParams) ([]domain.Task, error)
  MarkDone(ctx context.Context, id int64, doneDay domain.Day) error
  MarkAbandoned(ctx context.Context, id int64, abandonedDay domain.Day) error
  Postpone(ctx context.Context, id int64, newDueDay domain.Day) error

  // History / stats
  ListDoneByDay(ctx context.Context, day domain.Day) ([]domain.Task, error)
  ListAbandonedByDay(ctx context.Context, day domain.Day) ([]domain.Task, error)
  StatsOutcomeRatios(ctx context.Context, fromDay, toDay domain.Day) (OutcomeRatios, error)
}

type OutcomeRatios struct {
  TotalDone          int
  DelayedDone        int
  TotalAbandoned     int
  DelayedAbandoned   int
  DoneDelayedRatio   float64
  AbandonedDelayedRatio float64
}
```

**Step 4: Run tests (should still fail)**

Run: `go test ./...`
Expected: FAIL (SQLiteStore not defined).

**Step 5: Commit**

```bash
git add internal/store/store.go internal/store/query.go internal/store/store_test.go
git commit -m "feat: define persistence boundary for tasks and stats"
```

## Task 5: Implement SQLite Store + Schema + Migrations

**Files:**
- Create: `internal/store/sqlite/sqlite.go`
- Create: `internal/store/sqlite/migrate.go`
- Modify: `internal/store/store_test.go`
- Test: `internal/store/sqlite/sqlite_test.go`

**Step 1: Write failing integration tests (create/list/mark/postpone)**

`internal/store/sqlite/sqlite_test.go`:

```go
package sqlite

import (
  "context"
  "testing"

  "tuitodo/internal/domain"
  "tuitodo/internal/store"
)

func TestSQLiteStore_BasicFlow(t *testing.T) {
  ctx := context.Background()
  s, err := OpenInMemory()
  if err != nil { t.Fatal(err) }
  defer s.Close()

  today := domain.MustParseDay("2026-03-04")

  task, err := s.CreateTask(ctx, "a", today, today)
  if err != nil { t.Fatal(err) }

  tasks, err := s.ListActive(ctx, store.ListActiveParams{CurrentDay: today, Window: store.ActiveDueLTECurrent})
  if err != nil { t.Fatal(err) }
  if len(tasks) != 1 { t.Fatalf("expected 1 task") }

  tomorrow := domain.MustParseDay("2026-03-05")
  if err := s.Postpone(ctx, task.ID, tomorrow); err != nil { t.Fatal(err) }

  upcoming, err := s.ListActive(ctx, store.ListActiveParams{CurrentDay: today, Window: store.ActiveDueGTCurrent})
  if err != nil { t.Fatal(err) }
  if len(upcoming) != 1 { t.Fatalf("expected 1 upcoming") }

  if err := s.MarkDone(ctx, task.ID, today); err != nil { t.Fatal(err) }

  tasks, err = s.ListActive(ctx, store.ListActiveParams{CurrentDay: tomorrow, Window: store.ActiveDueLTECurrent})
  if err != nil { t.Fatal(err) }
  if len(tasks) != 0 { t.Fatalf("expected 0 active") }
}
```

**Step 2: Run tests (should fail)**

Run: `go test ./...`
Expected: FAIL (sqlite store not implemented).

**Step 3: Implement SQLite store + schema**

Schema (single table) suggestion:

- `tasks(id INTEGER PRIMARY KEY, title TEXT NOT NULL, status TEXT NOT NULL, created_day TEXT NOT NULL, due_day TEXT NOT NULL, done_day TEXT NULL, abandoned_day TEXT NULL)`
- Indexes: `(status, due_day)`, `(done_day)`, `(abandoned_day)`.

Implementation requirements:

- Open at file path and run migrations.
- Provide `OpenInMemory()` for tests.
- Use transactions for state transitions.
- Enforce:
  - done: set status done, set done_day, clear abandoned_day.
  - abandoned: set status abandoned, set abandoned_day, clear done_day.
  - postpone: only allowed for active tasks.

**Step 4: Update interface compile-time test**

Modify `internal/store/store_test.go` to reference the actual type:

```go
package store

import (
  "testing"

  "tuitodo/internal/store/sqlite"
)

func TestStore_Interface(t *testing.T) {
  var _ Store = (*sqlite.SQLiteStore)(nil)
}
```

**Step 5: Run tests (should pass)**

Run: `go test ./...`
Expected: PASS.

**Step 6: Commit**

```bash
git add internal/store/sqlite internal/store/store_test.go
git commit -m "feat: add sqlite persistence and migrations"
```

## Task 6: Implement Outcome Stats (Delayed Ratios)

**Files:**
- Modify: `internal/store/sqlite/sqlite.go`
- Test: `internal/store/sqlite/stats_test.go`

**Step 1: Write failing tests for ratios**

`internal/store/sqlite/stats_test.go`:

```go
package sqlite

import (
  "context"
  "testing"

  "tuitodo/internal/domain"
)

func TestSQLiteStore_StatsOutcomeRatios(t *testing.T) {
  ctx := context.Background()
  s, err := OpenInMemory()
  if err != nil { t.Fatal(err) }
  defer s.Close()

  d1 := domain.MustParseDay("2026-03-04")
  d2 := domain.MustParseDay("2026-03-05")

  // on-time done
  t1, _ := s.CreateTask(ctx, "t1", d1, d1)
  _ = s.MarkDone(ctx, t1.ID, d1)

  // delayed done: due_day d1, done on d2
  t2, _ := s.CreateTask(ctx, "t2", d1, d1)
  _ = s.MarkDone(ctx, t2.ID, d2)

  // delayed abandoned: due_day d1, abandoned on d2
  t3, _ := s.CreateTask(ctx, "t3", d1, d1)
  _ = s.MarkAbandoned(ctx, t3.ID, d2)

  stats, err := s.StatsOutcomeRatios(ctx, d1, d2)
  if err != nil { t.Fatal(err) }

  if stats.TotalDone != 2 || stats.DelayedDone != 1 {
    t.Fatalf("unexpected done counts: %+v", stats)
  }
  if stats.TotalAbandoned != 1 || stats.DelayedAbandoned != 1 {
    t.Fatalf("unexpected abandoned counts: %+v", stats)
  }
  if stats.DoneDelayedRatio != 0.5 {
    t.Fatalf("unexpected done ratio: %+v", stats)
  }
  if stats.AbandonedDelayedRatio != 1.0 {
    t.Fatalf("unexpected abandoned ratio: %+v", stats)
  }
}
```

**Step 2: Run tests (should fail)**

Run: `go test ./...`
Expected: FAIL (StatsOutcomeRatios not implemented).

**Step 3: Implement StatsOutcomeRatios in sqlite store**

Definition (per design):

- delayed done: `done_day` in range AND `due_day < done_day`.
- delayed abandoned: `abandoned_day` in range AND `due_day < abandoned_day`.

Compute ratios as float64; guard division by zero by returning 0.

**Step 4: Run tests (should pass)**

Run: `go test ./...`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/store/sqlite/sqlite.go internal/store/sqlite/stats_test.go
git commit -m "feat: compute delayed outcome ratios for history"
```

## Task 7: Add Application Layer (Use-Cases)

**Files:**
- Create: `internal/app/app.go`
- Create: `internal/app/config.go`
- Test: `internal/app/app_test.go`

**Step 1: Write failing tests for Today/Upcoming queries and transitions**

`internal/app/app_test.go` should validate:

- Create defaults to `due_day = current_day`.
- Today returns `active` tasks with `due_day <= current_day`.
- Upcoming returns `active` tasks with `due_day > current_day`.

**Step 2: Run tests (should fail)**

Run: `go test ./...`
Expected: FAIL.

**Step 3: Implement App struct**

App responsibilities:

- Hold `store.Store`, `timeutil.Clock`, timezone.
- Provide methods:
  - `Add(title)`
  - `Today()`
  - `Upcoming()`
  - `Done(id)` / `Abandon(id)`
  - `PostponeOneDay(id)`
  - `Stats(from,to)`

**Step 4: Run tests (should pass)**

Run: `go test ./...`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/app/app.go internal/app/config.go internal/app/app_test.go
git commit -m "feat: add app layer for task use-cases"
```

## Task 8: TUI Model Skeleton (Bubble Tea) + Navigation

**Files:**
- Create: `internal/ui/model.go`
- Create: `internal/ui/views.go`
- Create: `internal/ui/styles.go`
- Modify: `cmd/tuitodo/main.go`

**Step 1: Add Bubble Tea dependencies**

Run: `go get github.com/charmbracelet/bubbletea@latest github.com/charmbracelet/lipgloss@latest`
Expected: `go.mod` updated.

**Step 2: Implement minimal TUI that boots and shows Today**

`cmd/tuitodo/main.go` should:

- open store at default path (create dirs)
- create `app.App`
- start bubbletea program

**Step 3: Manual verify**

Run: `go run ./cmd/tuitodo`
Expected: TUI window renders.

**Step 4: Commit**

```bash
git add cmd/tuitodo/main.go internal/ui go.mod go.sum
git commit -m "feat: add TUI skeleton with view navigation"
```

## Task 9: Today View Interactions (Add/Done/Abandon/Postpone)

**Files:**
- Modify: `internal/ui/model.go`
- Modify: `internal/ui/views.go`
- Modify: `internal/ui/styles.go`

**Step 1: Implement key bindings**

Suggested bindings (adjust later):

- `a`: add task (prompt input)
- `x`: mark done
- `d`: mark abandoned
- `p`: postpone +1 day
- `u`: switch to Upcoming
- `h`: switch to History
- `q`: quit

**Step 2: Implement delayed styling**

- When rendering task rows in Today, call `task.IsDelayed(currentDay)`; if true apply red lipgloss style.

**Step 3: Manual verify**

Run: `go run ./cmd/tuitodo`
Expected:

- new tasks show in Today
- done/abandon removes from Today
- postpone moves to Upcoming
- tasks with `due_day` in the past appear red

**Step 4: Commit**

```bash
git add internal/ui
git commit -m "feat: add Today interactions and delayed highlighting"
```

## Task 10: Upcoming View

**Files:**
- Modify: `internal/ui/views.go`

**Step 1: Render upcoming list**

- Query app for Upcoming tasks.
- Display without delayed styling (by definition `due_day > current_day`).

**Step 2: Manual verify**

Run: `go run ./cmd/tuitodo`
Expected: postponed tasks appear in Upcoming.

**Step 3: Commit**

```bash
git add internal/ui/views.go
git commit -m "feat: add Upcoming view"
```

## Task 11: History View + Delayed Ratios

**Files:**
- Modify: `internal/ui/views.go`
- Modify: `internal/ui/model.go`

**Step 1: Display history day and counts**

- Choose a default range (e.g., last 7 days) and allow moving cursor/range.
- Show done/abandoned totals per day.

**Step 2: Display delayed outcome ratios (required)**

- For selected range, fetch `StatsOutcomeRatios(from,to)` and display:
  - delayed done ratio
  - delayed abandoned ratio

**Step 3: Manual verify**

Run: `go run ./cmd/tuitodo`
Expected: ratios update as tasks are completed/abandoned on later days.

**Step 4: Commit**

```bash
git add internal/ui
git commit -m "feat: add History view with delayed outcome ratios"
```

## Task 12: System Day Rollover Refresh

**Files:**
- Modify: `internal/ui/model.go`

**Step 1: Add periodic tick**

- Every N seconds (e.g., 10s), recompute `current_day` via clock.
- If day changed since last render, refresh lists and update styles.

**Step 2: Manual verify**

- With a fake clock (optional dev flag) or by temporarily lowering comparison threshold, verify refresh triggers.

**Step 3: Commit**

```bash
git add internal/ui/model.go
git commit -m "feat: refresh UI on system day rollover"
```

## Task 13: Packaging Polish

**Files:**
- Modify: `README.md`
- Create: `internal/app/paths.go`

**Step 1: Document keybindings and storage path**

- Add a concise README with install/run steps.

**Step 2: Ensure storage dir creation is robust**

- Create `~/.tuitodo/` if missing.
- Handle DB open errors with user-friendly message.

**Step 3: Final verification**

Run:

```bash
go test ./...
go run ./cmd/tuitodo
```

Expected: tests pass; app runs.

**Step 4: Commit**

```bash
git add README.md internal/app/paths.go
git commit -m "docs: add usage and improve storage path handling"
```
