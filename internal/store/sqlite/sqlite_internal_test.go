package sqlite

import (
	"context"
	"strings"
	"testing"
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
