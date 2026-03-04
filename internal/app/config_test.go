package app

import (
	"testing"
	"time"

	"github.com/JamesYuuu/tick/internal/store/sqlite"
)

type fakeClock struct {
	now time.Time
}

func (c fakeClock) Now() time.Time { return c.now }

func TestNew_DefaultsNilLocationToUTC(t *testing.T) {
	s, err := sqlite.OpenInMemory()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	a, err := New(Config{Store: s, Clock: fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}})
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	if a.loc == nil {
		t.Fatalf("expected app location to be non-nil")
	}
	if a.loc != time.UTC {
		t.Fatalf("expected app location to default to UTC, got %v", a.loc)
	}
}
