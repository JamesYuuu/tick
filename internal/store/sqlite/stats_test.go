package sqlite_test

import (
	"context"
	"math"
	"testing"

	"github.com/JamesYuuu/tick/internal/domain"
	"github.com/JamesYuuu/tick/internal/store/sqlite"
)

func assertFloatWithin(t *testing.T, got, want, eps float64) {
	t.Helper()
	if math.Abs(got-want) > eps {
		t.Fatalf("expected %v (±%v), got %v", want, eps, got)
	}
}

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

	assertFloatWithin(t, got.DoneDelayedRatio, 0.5, 1e-9)
	assertFloatWithin(t, got.AbandonedDelayedRatio, 1.0, 1e-9)
}

func TestSQLiteStore_StatsOutcomeRatios_InvertedRangeReturnsError(t *testing.T) {
	ctx := context.Background()

	s, err := sqlite.OpenInMemory()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})

	from := domain.MustParseDay("2026-03-06")
	to := domain.MustParseDay("2026-03-05")

	if _, err := s.StatsOutcomeRatios(ctx, from, to); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestSQLiteStore_StatsOutcomeRatios_EmptyRangeReturnsZerosAndZeroRatios(t *testing.T) {
	ctx := context.Background()

	s, err := sqlite.OpenInMemory()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})

	from := domain.MustParseDay("2026-04-01")
	to := domain.MustParseDay("2026-04-02")

	got, err := s.StatsOutcomeRatios(ctx, from, to)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}

	if got.TotalDone != 0 {
		t.Fatalf("expected TotalDone=0, got %d", got.TotalDone)
	}
	if got.DelayedDone != 0 {
		t.Fatalf("expected DelayedDone=0, got %d", got.DelayedDone)
	}
	if got.TotalAbandoned != 0 {
		t.Fatalf("expected TotalAbandoned=0, got %d", got.TotalAbandoned)
	}
	if got.DelayedAbandoned != 0 {
		t.Fatalf("expected DelayedAbandoned=0, got %d", got.DelayedAbandoned)
	}
	assertFloatWithin(t, got.DoneDelayedRatio, 0, 1e-9)
	assertFloatWithin(t, got.AbandonedDelayedRatio, 0, 1e-9)
}
