package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

var errTitleBlank = errors.New("title cannot be blank")

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
	title, err := normalizeTitle(title)
	if err != nil {
		return domain.Task{}, err
	}
	day := a.currentDay()
	return a.store.CreateTask(ctx, title, day, day)
}

func (a *App) EditTitle(ctx context.Context, id int64, title string) error {
	title, err := normalizeTitle(title)
	if err != nil {
		return err
	}
	return a.store.UpdateTitle(ctx, id, title)
}

func (a *App) Delete(ctx context.Context, id int64) error {
	return a.store.DeleteTask(ctx, id)
}

func (a *App) Today(ctx context.Context) ([]domain.Task, error) {
	return a.listActive(ctx, store.ActiveDueLTECurrent)
}

func (a *App) Upcoming(ctx context.Context) ([]domain.Task, error) {
	return a.listActive(ctx, store.ActiveDueGTCurrent)
}

func (a *App) listActive(ctx context.Context, window store.ActiveWindow) ([]domain.Task, error) {
	return a.store.ListActive(ctx, store.ListActiveParams{CurrentDay: a.currentDay(), Window: window})
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
	return mapOutcomeRatios(out), nil
}

func (a *App) HistoryDoneByDay(ctx context.Context, day domain.Day) ([]domain.Task, error) {
	return historyByDay(ctx, "history done", day, a.store.ListDoneByDay)
}

func (a *App) HistoryAbandonedByDay(ctx context.Context, day domain.Day) ([]domain.Task, error) {
	return historyByDay(ctx, "history abandoned", day, a.store.ListAbandonedByDay)
}

func (a *App) HistoryActiveByCreatedDay(ctx context.Context, day domain.Day) ([]domain.Task, error) {
	return historyByDay(ctx, "history active by created day", day, a.store.ListActiveByCreatedDay)
}

func historyByDay(
	ctx context.Context,
	prefix string,
	day domain.Day,
	fn func(context.Context, domain.Day) ([]domain.Task, error),
) ([]domain.Task, error) {
	out, err := fn(ctx, day)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", prefix, err)
	}
	return out, nil
}

func normalizeTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", errTitleBlank
	}
	return title, nil
}
