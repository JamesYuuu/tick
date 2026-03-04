package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/JamesYuuu/tick/internal/app"
	"github.com/JamesYuuu/tick/internal/domain"
	"github.com/JamesYuuu/tick/internal/store/sqlite"
)

type fakeClock struct {
	now time.Time
}

func (c fakeClock) Now() time.Time { return c.now }

func TestApp_Add_DefaultsDueDayToCurrentDay(t *testing.T) {
	ctx := context.Background()

	s, err := sqlite.OpenInMemory()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	clk := fakeClock{now: time.Date(2026, 3, 4, 15, 0, 0, 0, time.UTC)}

	a, err := app.New(app.Config{Store: s, Clock: clk, Location: time.UTC})
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	got, err := a.Add(ctx, "hello")
	if err != nil {
		t.Fatalf("add: %v", err)
	}

	if got.DueDay.String() != "2026-03-04" {
		t.Fatalf("expected due_day to default to current day, got %s", got.DueDay.String())
	}
}

func TestApp_Today_ReturnsActiveTasksDueOnOrBeforeCurrentDay(t *testing.T) {
	ctx := context.Background()

	s, err := sqlite.OpenInMemory()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	clk := fakeClock{now: time.Date(2026, 3, 4, 9, 30, 0, 0, time.UTC)}
	a, err := app.New(app.Config{Store: s, Clock: clk, Location: time.UTC})
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	created := domain.MustParseDay("2026-03-01")
	current := domain.MustParseDay("2026-03-04")

	late, err := s.CreateTask(ctx, "late", created, domain.MustParseDay("2026-03-03"))
	if err != nil {
		t.Fatalf("create late: %v", err)
	}
	today, err := s.CreateTask(ctx, "today", created, current)
	if err != nil {
		t.Fatalf("create today: %v", err)
	}
	tomorrow, err := s.CreateTask(ctx, "tomorrow", created, domain.MustParseDay("2026-03-05"))
	if err != nil {
		t.Fatalf("create tomorrow: %v", err)
	}
	if err := s.MarkDone(ctx, tomorrow.ID, domain.MustParseDay("2026-03-04")); err != nil {
		t.Fatalf("mark done: %v", err)
	}

	got, err := a.Today(ctx)
	if err != nil {
		t.Fatalf("today: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 tasks, got %#v", got)
	}
	if got[0].ID != late.ID || got[1].ID != today.ID {
		t.Fatalf("expected tasks [late, today], got %#v", got)
	}
}

func TestApp_Upcoming_ReturnsActiveTasksDueAfterCurrentDay(t *testing.T) {
	ctx := context.Background()

	s, err := sqlite.OpenInMemory()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	clk := fakeClock{now: time.Date(2026, 3, 4, 9, 30, 0, 0, time.UTC)}
	a, err := app.New(app.Config{Store: s, Clock: clk, Location: time.UTC})
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	created := domain.MustParseDay("2026-03-01")
	current := domain.MustParseDay("2026-03-04")

	if _, err := s.CreateTask(ctx, "late", created, domain.MustParseDay("2026-03-03")); err != nil {
		t.Fatalf("create late: %v", err)
	}
	if _, err := s.CreateTask(ctx, "today", created, current); err != nil {
		t.Fatalf("create today: %v", err)
	}
	next, err := s.CreateTask(ctx, "next", created, domain.MustParseDay("2026-03-05"))
	if err != nil {
		t.Fatalf("create next: %v", err)
	}

	got, err := a.Upcoming(ctx)
	if err != nil {
		t.Fatalf("upcoming: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 task, got %#v", got)
	}
	if got[0].ID != next.ID {
		t.Fatalf("expected task next, got %#v", got)
	}
}
