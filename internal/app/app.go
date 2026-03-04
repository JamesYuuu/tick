package app

import (
	"context"
	"fmt"
	"time"

	"github.com/JamesYuuu/tick/internal/domain"
	"github.com/JamesYuuu/tick/internal/store"
	"github.com/JamesYuuu/tick/internal/timeutil"
)

type App struct {
	store store.Store
	clock timeutil.Clock
	loc   *time.Location
}

func New(cfg Config) (*App, error) {
	if err := (&cfg).validate(); err != nil {
		return nil, err
	}
	return &App{store: cfg.Store, clock: cfg.Clock, loc: cfg.Location}, nil
}

func (a *App) currentDay() domain.Day {
	return timeutil.CurrentDay(a.clock, a.loc)
}

func (a *App) Add(ctx context.Context, title string) (domain.Task, error) {
	day := a.currentDay()
	return a.store.CreateTask(ctx, title, day, day)
}

func (a *App) Today(ctx context.Context) ([]domain.Task, error) {
	return a.store.ListActive(ctx, store.ListActiveParams{CurrentDay: a.currentDay(), Window: store.ActiveDueLTECurrent})
}

func (a *App) Upcoming(ctx context.Context) ([]domain.Task, error) {
	return a.store.ListActive(ctx, store.ListActiveParams{CurrentDay: a.currentDay(), Window: store.ActiveDueGTCurrent})
}

func (a *App) Done(ctx context.Context, id int64) error {
	return a.store.MarkDone(ctx, id, a.currentDay())
}

func (a *App) Abandon(ctx context.Context, id int64) error {
	return a.store.MarkAbandoned(ctx, id, a.currentDay())
}

func (a *App) PostponeOneDay(ctx context.Context, id int64) error {
	current := a.currentDay()
	newDue := domain.DayFromTime(current.Time().AddDate(0, 0, 1))
	return a.store.Postpone(ctx, id, newDue)
}


func (a *App) Stats(ctx context.Context, fromDay, toDay domain.Day) (OutcomeRatios, error) {
	out, err := a.store.StatsOutcomeRatios(ctx, fromDay, toDay)
	if err != nil {
		return OutcomeRatios{}, fmt.Errorf("stats: %w", err)
	}
	return OutcomeRatios{
		TotalDone:             out.TotalDone,
		DelayedDone:           out.DelayedDone,
		TotalAbandoned:        out.TotalAbandoned,
		DelayedAbandoned:      out.DelayedAbandoned,
		DoneDelayedRatio:      out.DoneDelayedRatio,
		AbandonedDelayedRatio: out.AbandonedDelayedRatio,
	}, nil
}

func (a *App) HistoryDoneByDay(ctx context.Context, day domain.Day) ([]domain.Task, error) {
	out, err := a.store.ListDoneByDay(ctx, day)
	if err != nil {
		return nil, fmt.Errorf("history done: %w", err)
	}
	return out, nil
}

func (a *App) HistoryAbandonedByDay(ctx context.Context, day domain.Day) ([]domain.Task, error) {
	out, err := a.store.ListAbandonedByDay(ctx, day)
	if err != nil {
		return nil, fmt.Errorf("history abandoned: %w", err)
	}
	return out, nil
}
