package sqlite_test

import (
	"context"
	"testing"

	"github.com/JamesYuuu/tick/internal/domain"
	"github.com/JamesYuuu/tick/internal/store"
	"github.com/JamesYuuu/tick/internal/store/sqlite"
)

func TestOpenInMemory_DoesNotShareStateAcrossCalls(t *testing.T) {
	ctx := context.Background()

	s1, err := sqlite.OpenInMemory()
	if err != nil {
		t.Fatalf("open 1: %v", err)
	}
	t.Cleanup(func() { _ = s1.Close() })

	s2, err := sqlite.OpenInMemory()
	if err != nil {
		t.Fatalf("open 2: %v", err)
	}
	t.Cleanup(func() { _ = s2.Close() })

	day := domain.MustParseDay("2026-03-04")
	if _, err := s1.CreateTask(ctx, "only in store 1", day, day); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := s2.ListActive(ctx, store.ListActiveParams{CurrentDay: day, Window: store.ActiveDueLTECurrent})
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected store 2 to be empty, got %#v", got)
	}
}
