package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/JamesYuuu/tick/internal/domain"
	"github.com/JamesYuuu/tick/internal/store"
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

func TestExecAffectingOne_NotFoundWrapsNoRows(t *testing.T) {
	s, err := OpenInMemory()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	err = s.execAffectingOne(context.Background(), "delete task", 999, `DELETE FROM tasks WHERE id = ?`, 999)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
	if !strings.Contains(err.Error(), "delete task: id=999") {
		t.Fatalf("expected contextual id in error, got %v", err)
	}
}

func TestActiveDueOperator_RejectsUnknownWindow(t *testing.T) {
	_, err := activeDueOperator(store.ActiveWindow(99))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "99") {
		t.Fatalf("expected window value in error, got %v", err)
	}
}

func TestListActive_ScanErrorIsNotPrefixedWithListActive(t *testing.T) {
	s, err := OpenInMemory()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	_, err = s.db.ExecContext(context.Background(),
		`INSERT INTO tasks(title, status, created_day, due_day, done_day, abandoned_day)
		 VALUES(?, ?, ?, ?, NULL, NULL)`,
		"broken", string(domain.StatusActive), "not-a-day", "2026-03-04",
	)
	if err != nil {
		t.Fatalf("insert malformed row: %v", err)
	}

	_, err = s.ListActive(context.Background(), store.ListActiveParams{
		CurrentDay: domain.MustParseDay("2026-03-04"),
		Window:     store.ActiveDueLTECurrent,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "scan task:") {
		t.Fatalf("expected scan task error, got %v", err)
	}
	if strings.HasPrefix(err.Error(), "list active: ") {
		t.Fatalf("expected unprefixed scan error, got %v", err)
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

func TestSQLiteStore_DSNForPath_EscapesSpaces(t *testing.T) {
	path := "/tmp/dir with spaces/todo.db"

	got := dsnForPath(path)
	want := "file:/tmp/dir%20with%20spaces/todo.db"
	if got != want {
		t.Fatalf("dsnForPath(%q) = %q, want %q", path, got, want)
	}
}
