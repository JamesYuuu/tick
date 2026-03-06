package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/JamesYuuu/tick/internal/domain"
	"github.com/JamesYuuu/tick/internal/store"
)

func TestHistoryByDay_WrapsErrorWithPrefix(t *testing.T) {
	day := domain.MustParseDay("2026-03-07")

	_, err := historyByDay(context.Background(), "history done", day, func(context.Context, domain.Day) ([]domain.Task, error) {
		return nil, errors.New("boom")
	})
	if err == nil || !strings.Contains(err.Error(), "history done") {
		t.Fatalf("expected wrapped prefix error, got %v", err)
	}
}

func TestHistoryByDay_ReturnsTasksOnSuccess(t *testing.T) {
	day := domain.MustParseDay("2026-03-07")
	want := []domain.Task{{ID: 42, Title: "task"}}

	got, err := historyByDay(context.Background(), "history done", day, func(context.Context, domain.Day) ([]domain.Task, error) {
		return want, nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(got) != len(want) || got[0].ID != want[0].ID || got[0].Title != want[0].Title {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestApp_HistoryDoneByDay_WrapsStoreErrorWithMethodPrefix(t *testing.T) {
	a := &App{store: methodStubStore{
		listDoneByDay: func(context.Context, domain.Day) ([]domain.Task, error) {
			return nil, errors.New("boom")
		},
	}}

	_, err := a.HistoryDoneByDay(context.Background(), domain.MustParseDay("2026-03-07"))
	if err == nil || !strings.Contains(err.Error(), "history done: boom") {
		t.Fatalf("expected wrapped history done error, got %v", err)
	}
}

func TestApp_Stats_ReturnsMappedOutcomeRatiosOnSuccess(t *testing.T) {
	a := &App{store: methodStubStore{
		statsOutcomeRatios: func(context.Context, domain.Day, domain.Day) (store.OutcomeRatios, error) {
			return store.OutcomeRatios{
				TotalDone:             10,
				DelayedDone:           3,
				TotalAbandoned:        4,
				DelayedAbandoned:      1,
				DoneDelayedRatio:      0.3,
				AbandonedDelayedRatio: 0.25,
			}, nil
		},
	}}

	got, err := a.Stats(context.Background(), domain.MustParseDay("2026-03-01"), domain.MustParseDay("2026-03-07"))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got.TotalDone != 10 || got.DelayedDone != 3 || got.TotalAbandoned != 4 || got.DelayedAbandoned != 1 {
		t.Fatalf("unexpected totals in stats: %+v", got)
	}
	if got.DoneDelayedRatio != 0.3 || got.AbandonedDelayedRatio != 0.25 {
		t.Fatalf("unexpected ratios in stats: %+v", got)
	}
}

func TestApp_Stats_WrapsStoreError(t *testing.T) {
	a := &App{store: methodStubStore{
		statsOutcomeRatios: func(context.Context, domain.Day, domain.Day) (store.OutcomeRatios, error) {
			return store.OutcomeRatios{}, errors.New("boom")
		},
	}}

	_, err := a.Stats(context.Background(), domain.MustParseDay("2026-03-01"), domain.MustParseDay("2026-03-07"))
	if err == nil || !strings.Contains(err.Error(), "stats: boom") {
		t.Fatalf("expected wrapped stats error, got %v", err)
	}
}

func TestMapOutcomeRatios_MapsAllFields(t *testing.T) {
	in := store.OutcomeRatios{
		TotalDone:             9,
		DelayedDone:           2,
		TotalAbandoned:        3,
		DelayedAbandoned:      1,
		DoneDelayedRatio:      2.0 / 9.0,
		AbandonedDelayedRatio: 1.0 / 3.0,
	}

	got := mapOutcomeRatios(in)

	if got.TotalDone != in.TotalDone || got.DelayedDone != in.DelayedDone || got.TotalAbandoned != in.TotalAbandoned || got.DelayedAbandoned != in.DelayedAbandoned {
		t.Fatalf("unexpected mapped totals: %+v", got)
	}
	if got.DoneDelayedRatio != in.DoneDelayedRatio || got.AbandonedDelayedRatio != in.AbandonedDelayedRatio {
		t.Fatalf("unexpected mapped ratios: %+v", got)
	}
}

type methodStubStore struct {
	listDoneByDay      func(context.Context, domain.Day) ([]domain.Task, error)
	statsOutcomeRatios func(context.Context, domain.Day, domain.Day) (store.OutcomeRatios, error)
}

func (s methodStubStore) Close() error { return nil }

func (s methodStubStore) CreateTask(context.Context, string, domain.Day, domain.Day) (domain.Task, error) {
	return domain.Task{}, errors.New("not implemented")
}

func (s methodStubStore) ListActive(context.Context, store.ListActiveParams) ([]domain.Task, error) {
	return nil, errors.New("not implemented")
}

func (s methodStubStore) MarkDone(context.Context, int64, domain.Day) error {
	return errors.New("not implemented")
}

func (s methodStubStore) MarkAbandoned(context.Context, int64, domain.Day) error {
	return errors.New("not implemented")
}

func (s methodStubStore) Postpone(context.Context, int64, domain.Day) error {
	return errors.New("not implemented")
}

func (s methodStubStore) ListActiveByCreatedDay(context.Context, domain.Day) ([]domain.Task, error) {
	return nil, errors.New("not implemented")
}

func (s methodStubStore) ListDoneByDay(ctx context.Context, day domain.Day) ([]domain.Task, error) {
	if s.listDoneByDay != nil {
		return s.listDoneByDay(ctx, day)
	}
	return nil, errors.New("not implemented")
}

func (s methodStubStore) ListAbandonedByDay(context.Context, domain.Day) ([]domain.Task, error) {
	return nil, errors.New("not implemented")
}

func (s methodStubStore) StatsOutcomeRatios(ctx context.Context, fromDay, toDay domain.Day) (store.OutcomeRatios, error) {
	if s.statsOutcomeRatios != nil {
		return s.statsOutcomeRatios(ctx, fromDay, toDay)
	}
	return store.OutcomeRatios{}, errors.New("not implemented")
}
