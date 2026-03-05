package store

import (
	"context"

	"github.com/JamesYuuu/tick/internal/domain"
)

type Store interface {
	Close() error

	CreateTask(ctx context.Context, title string, createdDay, dueDay domain.Day) (domain.Task, error)
	ListActive(ctx context.Context, p ListActiveParams) ([]domain.Task, error)
	MarkDone(ctx context.Context, id int64, doneDay domain.Day) error
	MarkAbandoned(ctx context.Context, id int64, abandonedDay domain.Day) error
	Postpone(ctx context.Context, id int64, newDueDay domain.Day) error
	ListActiveByCreatedDay(ctx context.Context, day domain.Day) ([]domain.Task, error)

	ListDoneByDay(ctx context.Context, day domain.Day) ([]domain.Task, error)
	ListAbandonedByDay(ctx context.Context, day domain.Day) ([]domain.Task, error)
	StatsOutcomeRatios(ctx context.Context, fromDay, toDay domain.Day) (OutcomeRatios, error)
}

type OutcomeRatios struct {
	TotalDone             int
	DelayedDone           int
	TotalAbandoned        int
	DelayedAbandoned      int
	DoneDelayedRatio      float64
	AbandonedDelayedRatio float64
}
