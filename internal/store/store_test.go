package store

import (
	"context"
	"testing"

	"github.com/JamesYuuu/tick/internal/domain"
)

// stubStore is a test-only type used for compile-time interface conformance.
// It intentionally does not implement any real behavior.
type stubStore struct{}

func (s *stubStore) Close() error { return nil }

func (s *stubStore) CreateTask(ctx context.Context, title string, createdDay, dueDay domain.Day) (domain.Task, error) {
	return domain.Task{}, nil
}

func (s *stubStore) ListActive(ctx context.Context, p ListActiveParams) ([]domain.Task, error) {
	return nil, nil
}

func (s *stubStore) MarkDone(ctx context.Context, id int64, doneDay domain.Day) error { return nil }

func (s *stubStore) MarkAbandoned(ctx context.Context, id int64, abandonedDay domain.Day) error { return nil }

func (s *stubStore) Postpone(ctx context.Context, id int64, newDueDay domain.Day) error { return nil }

func (s *stubStore) ListDoneByDay(ctx context.Context, day domain.Day) ([]domain.Task, error) {
	return nil, nil
}

func (s *stubStore) ListAbandonedByDay(ctx context.Context, day domain.Day) ([]domain.Task, error) {
	return nil, nil
}

func (s *stubStore) StatsOutcomeRatios(ctx context.Context, fromDay, toDay domain.Day) (OutcomeRatios, error) {
	return OutcomeRatios{}, nil
}

func TestStore_Interface(t *testing.T) {
	var _ Store = (*stubStore)(nil)
}
