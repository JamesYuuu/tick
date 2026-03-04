package sqlite_test

import (
	"context"
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
