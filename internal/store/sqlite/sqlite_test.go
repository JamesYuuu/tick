package sqlite_test

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/JamesYuuu/tick/internal/domain"
	"github.com/JamesYuuu/tick/internal/store"
	"github.com/JamesYuuu/tick/internal/store/sqlite"
)

func TestSQLiteStore_BasicFlow(t *testing.T) {
	ctx := context.Background()

	s, err := sqlite.OpenInMemory()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})

	current := domain.MustParseDay("2026-03-04")
	created := current

	soon, err := s.CreateTask(ctx, "due today", created, current)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	later, err := s.CreateTask(ctx, "due later", created, domain.MustParseDay("2026-03-10"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	activeLTE, err := s.ListActive(ctx, store.ListActiveParams{CurrentDay: current, Window: store.ActiveDueLTECurrent})
	if err != nil {
		t.Fatalf("list active lte: %v", err)
	}
	if len(activeLTE) != 1 || activeLTE[0].ID != soon.ID {
		t.Fatalf("expected only soon in lte window, got %#v", activeLTE)
	}

	activeGT, err := s.ListActive(ctx, store.ListActiveParams{CurrentDay: current, Window: store.ActiveDueGTCurrent})
	if err != nil {
		t.Fatalf("list active gt: %v", err)
	}
	if len(activeGT) != 1 || activeGT[0].ID != later.ID {
		t.Fatalf("expected only later in gt window, got %#v", activeGT)
	}

	postponedDay := domain.MustParseDay("2026-03-12")
	if err := s.Postpone(ctx, later.ID, postponedDay); err != nil {
		t.Fatalf("postpone: %v", err)
	}
	activeGT2, err := s.ListActive(ctx, store.ListActiveParams{CurrentDay: current, Window: store.ActiveDueGTCurrent})
	if err != nil {
		t.Fatalf("list active gt2: %v", err)
	}
	if len(activeGT2) != 1 || activeGT2[0].DueDay.String() != postponedDay.String() {
		t.Fatalf("expected postponed task due_day updated, got %#v", activeGT2)
	}

	doneDay := domain.MustParseDay("2026-03-05")
	if err := s.MarkDone(ctx, soon.ID, doneDay); err != nil {
		t.Fatalf("mark done: %v", err)
	}
	doneTasks, err := s.ListDoneByDay(ctx, doneDay)
	if err != nil {
		t.Fatalf("list done: %v", err)
	}
	if len(doneTasks) != 1 || doneTasks[0].ID != soon.ID || doneTasks[0].Status != domain.StatusDone {
		t.Fatalf("expected done task, got %#v", doneTasks)
	}
	if doneTasks[0].DoneDay == nil || doneTasks[0].DoneDay.String() != doneDay.String() {
		t.Fatalf("expected done_day set, got %#v", doneTasks[0])
	}
	if doneTasks[0].AbandonedDay != nil {
		t.Fatalf("expected abandoned_day cleared, got %#v", doneTasks[0])
	}

	if err := s.Postpone(ctx, soon.ID, domain.MustParseDay("2026-03-20")); err == nil {
		t.Fatalf("expected postpone on non-active task to error")
	}

	abandonedDay := domain.MustParseDay("2026-03-06")
	if err := s.MarkAbandoned(ctx, soon.ID, abandonedDay); err != nil {
		t.Fatalf("mark abandoned: %v", err)
	}
	abandonedTasks, err := s.ListAbandonedByDay(ctx, abandonedDay)
	if err != nil {
		t.Fatalf("list abandoned: %v", err)
	}
	if len(abandonedTasks) != 1 || abandonedTasks[0].ID != soon.ID || abandonedTasks[0].Status != domain.StatusAbandoned {
		t.Fatalf("expected abandoned task, got %#v", abandonedTasks)
	}
	if abandonedTasks[0].AbandonedDay == nil || abandonedTasks[0].AbandonedDay.String() != abandonedDay.String() {
		t.Fatalf("expected abandoned_day set, got %#v", abandonedTasks[0])
	}
	if abandonedTasks[0].DoneDay != nil {
		t.Fatalf("expected done_day cleared, got %#v", abandonedTasks[0])
	}

	// Transition back to done should clear abandoned_day and remove it from abandoned list.
	doneDay2 := domain.MustParseDay("2026-03-07")
	if err := s.MarkDone(ctx, soon.ID, doneDay2); err != nil {
		t.Fatalf("mark done 2: %v", err)
	}
	abandonedTasks2, err := s.ListAbandonedByDay(ctx, abandonedDay)
	if err != nil {
		t.Fatalf("list abandoned 2: %v", err)
	}
	if len(abandonedTasks2) != 0 {
		t.Fatalf("expected abandoned list cleared after done transition, got %#v", abandonedTasks2)
	}
}

func TestSQLiteStore_Postpone_NotFound_WrapsNoRows(t *testing.T) {
	ctx := context.Background()

	s, err := sqlite.OpenInMemory()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})

	id := int64(12345)
	if err := s.Postpone(ctx, id, domain.MustParseDay("2026-03-12")); err == nil {
		t.Fatalf("expected error")
	} else if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected errors.Is(err, sql.ErrNoRows) true, got %v", err)
	} else if err == sql.ErrNoRows {
		t.Fatalf("expected wrapped error, got %v", err)
	} else if !strings.Contains(err.Error(), "postpone") {
		t.Fatalf("expected context in error, got %v", err)
	}
}

func TestSQLiteStore_Postpone_NotActive_ReturnsInvalidTransition(t *testing.T) {
	ctx := context.Background()

	s, err := sqlite.OpenInMemory()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})

	created := domain.MustParseDay("2026-03-04")
	due := domain.MustParseDay("2026-03-10")
	newDue := domain.MustParseDay("2026-03-12")

	t.Run("done", func(t *testing.T) {
		task, err := s.CreateTask(ctx, "x", created, due)
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if err := s.MarkDone(ctx, task.ID, domain.MustParseDay("2026-03-05")); err != nil {
			t.Fatalf("mark done: %v", err)
		}
		if err := s.Postpone(ctx, task.ID, newDue); err == nil {
			t.Fatalf("expected error")
		} else if !errors.Is(err, sqlite.ErrInvalidTransition) {
			t.Fatalf("expected ErrInvalidTransition, got %v", err)
		} else if errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("expected not to wrap sql.ErrNoRows, got %v", err)
		}
	})

	t.Run("abandoned", func(t *testing.T) {
		task, err := s.CreateTask(ctx, "y", created, due)
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if err := s.MarkAbandoned(ctx, task.ID, domain.MustParseDay("2026-03-06")); err != nil {
			t.Fatalf("mark abandoned: %v", err)
		}
		if err := s.Postpone(ctx, task.ID, newDue); err == nil {
			t.Fatalf("expected error")
		} else if !errors.Is(err, sqlite.ErrInvalidTransition) {
			t.Fatalf("expected ErrInvalidTransition, got %v", err)
		} else if errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("expected not to wrap sql.ErrNoRows, got %v", err)
		}
	})
}

func TestSQLiteStore_ListActiveByCreatedDay_FiltersByDay(t *testing.T) {
	ctx := context.Background()

	s, err := sqlite.OpenInMemory()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})

	day1 := domain.MustParseDay("2026-03-04")
	day2 := domain.MustParseDay("2026-03-05")

	active1, err := s.CreateTask(ctx, "a", day1, day1)
	if err != nil {
		t.Fatalf("create active1: %v", err)
	}
	tDone, err := s.CreateTask(ctx, "done", day1, day1)
	if err != nil {
		t.Fatalf("create done: %v", err)
	}
	tAbandoned, err := s.CreateTask(ctx, "abandoned", day1, day1)
	if err != nil {
		t.Fatalf("create abandoned: %v", err)
	}
	active2, err := s.CreateTask(ctx, "b", day1, day1)
	if err != nil {
		t.Fatalf("create active2: %v", err)
	}
	_, err = s.CreateTask(ctx, "other day", day2, day2)
	if err != nil {
		t.Fatalf("create other day: %v", err)
	}

	if err := s.MarkDone(ctx, tDone.ID, day1); err != nil {
		t.Fatalf("mark done: %v", err)
	}
	if err := s.MarkAbandoned(ctx, tAbandoned.ID, day1); err != nil {
		t.Fatalf("mark abandoned: %v", err)
	}

	out, err := s.ListActiveByCreatedDay(ctx, day1)
	if err != nil {
		t.Fatalf("list active by created day: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected only day1 active tasks, got %#v", out)
	}
	if out[0].ID != active1.ID || out[1].ID != active2.ID {
		t.Fatalf("expected id ASC order [%d %d], got %#v", active1.ID, active2.ID, out)
	}
}

func TestSQLiteStore_MarkStatus_NotFound_WrapsNoRows(t *testing.T) {
	ctx := context.Background()

	s, err := sqlite.OpenInMemory()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})

	tests := []struct {
		name    string
		call    func(*sqlite.SQLiteStore, context.Context) error
		wantSub string
	}{
		{
			name: "done",
			call: func(s *sqlite.SQLiteStore, ctx context.Context) error {
				return s.MarkDone(ctx, 12345, domain.MustParseDay("2026-03-05"))
			},
			wantSub: "mark done",
		},
		{
			name: "abandoned",
			call: func(s *sqlite.SQLiteStore, ctx context.Context) error {
				return s.MarkAbandoned(ctx, 12345, domain.MustParseDay("2026-03-06"))
			},
			wantSub: "mark abandoned",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call(s, ctx)
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("expected errors.Is(err, sql.ErrNoRows) true, got %v", err)
			}
			if err == sql.ErrNoRows {
				t.Fatalf("expected wrapped error, got %v", err)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("expected context substring %q in error, got %v", tc.wantSub, err)
			}
		})
	}
}
