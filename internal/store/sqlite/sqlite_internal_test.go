package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/JamesYuuu/tick/internal/domain"
)

func TestQueryTasks_EmptyResult(t *testing.T) {
	s, err := OpenInMemory()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	got, err := s.queryTasks(context.Background(), "test query", `SELECT id, title, status, created_day, due_day, done_day, abandoned_day FROM tasks WHERE 1=0`)
	if err != nil {
		t.Fatalf("queryTasks: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty result, got %#v", got)
	}
	if got != nil {
		t.Fatalf("expected nil slice for empty result, got non-nil len=%d", len(got))
	}
}

func TestQueryTasks_PrefixesQueryErrorWithOp(t *testing.T) {
	s, err := OpenInMemory()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	_, err = s.queryTasks(context.Background(), "test query", "SELECT nope FROM tasks")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.HasPrefix(err.Error(), "test query: ") {
		t.Fatalf("expected error to be prefixed with op, got %q", err.Error())
	}
}

func TestSetStatusDay_NotFoundWrapsNoRows(t *testing.T) {
	s, err := OpenInMemory()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	err = s.setStatusDay(context.Background(), "mark done", 999, domain.StatusDone, "done_day", domain.MustParseDay("2026-03-05"), "abandoned_day")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestSetStatusDay_InvalidDayColumnReturnsExplicitError(t *testing.T) {
	s, err := OpenInMemory()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	err = s.setStatusDay(context.Background(), "mark done", 1, domain.StatusDone, "created_day", domain.MustParseDay("2026-03-05"), "abandoned_day")
	if !errors.Is(err, ErrInvalidStatusDayColumn) {
		t.Fatalf("expected ErrInvalidStatusDayColumn, got %v", err)
	}
}

func TestSetStatusDay_InvalidClearColumnReturnsExplicitError(t *testing.T) {
	s, err := OpenInMemory()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	err = s.setStatusDay(context.Background(), "mark done", 1, domain.StatusDone, "done_day", domain.MustParseDay("2026-03-05"), "created_day")
	if !errors.Is(err, ErrInvalidStatusDayColumn) {
		t.Fatalf("expected ErrInvalidStatusDayColumn, got %v", err)
	}
}
