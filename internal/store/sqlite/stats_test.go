package sqlite_test

import (
	"context"
	"testing"

	"github.com/JamesYuuu/tick/internal/domain"
	"github.com/JamesYuuu/tick/internal/store/sqlite"
)

func TestSQLiteStore_StatsOutcomeRatios(t *testing.T) {
	ctx := context.Background()

	s, err := sqlite.OpenInMemory()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})

	created := domain.MustParseDay("2026-03-01")

	// Done on time (due_day == done_day).
	onTime, err := s.CreateTask(ctx, "on time", created, domain.MustParseDay("2026-03-05"))
	if err != nil {
		t.Fatalf("create on time: %v", err)
	}
	if err := s.MarkDone(ctx, onTime.ID, domain.MustParseDay("2026-03-05")); err != nil {
		t.Fatalf("mark done on time: %v", err)
	}

	// Done delayed (due_day < done_day).
	delayedDone, err := s.CreateTask(ctx, "delayed done", created, domain.MustParseDay("2026-03-04"))
	if err != nil {
		t.Fatalf("create delayed done: %v", err)
	}
	if err := s.MarkDone(ctx, delayedDone.ID, domain.MustParseDay("2026-03-05")); err != nil {
		t.Fatalf("mark done delayed: %v", err)
	}

	// Abandoned delayed (due_day < abandoned_day).
	delayedAbandoned, err := s.CreateTask(ctx, "delayed abandoned", created, domain.MustParseDay("2026-03-04"))
	if err != nil {
		t.Fatalf("create delayed abandoned: %v", err)
	}
	if err := s.MarkAbandoned(ctx, delayedAbandoned.ID, domain.MustParseDay("2026-03-06")); err != nil {
		t.Fatalf("mark abandoned delayed: %v", err)
	}

	from := domain.MustParseDay("2026-03-05")
	to := domain.MustParseDay("2026-03-06")

	got, err := s.StatsOutcomeRatios(ctx, from, to)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}

	if got.TotalDone != 2 {
		t.Fatalf("expected TotalDone=2, got %d", got.TotalDone)
	}
	if got.DelayedDone != 1 {
		t.Fatalf("expected DelayedDone=1, got %d", got.DelayedDone)
	}
	if got.TotalAbandoned != 1 {
		t.Fatalf("expected TotalAbandoned=1, got %d", got.TotalAbandoned)
	}
	if got.DelayedAbandoned != 1 {
		t.Fatalf("expected DelayedAbandoned=1, got %d", got.DelayedAbandoned)
	}

	if got.DoneDelayedRatio != 0.5 {
		t.Fatalf("expected DoneDelayedRatio=0.5, got %v", got.DoneDelayedRatio)
	}
	if got.AbandonedDelayedRatio != 1.0 {
		t.Fatalf("expected AbandonedDelayedRatio=1.0, got %v", got.AbandonedDelayedRatio)
	}
}
