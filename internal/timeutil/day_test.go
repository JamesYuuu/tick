package timeutil

import (
	"testing"
	"time"

	"github.com/JamesYuuu/tick/internal/domain"
)

type fakeClock struct {
	now time.Time
}

func (f fakeClock) Now() time.Time {
	return f.now
}

func TestCurrentDay_UsesProvidedLocationCalendarDay(t *testing.T) {
	// 2026-03-05T07:30:00Z is still 2026-03-04 in UTC-08.
	now := time.Date(2026, 3, 5, 7, 30, 0, 0, time.UTC)
	loc := time.FixedZone("X", -8*60*60)

	got := CurrentDay(fakeClock{now: now}, loc)
	want := domain.MustParseDay("2026-03-04")

	if got.String() != want.String() {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestCurrentDay_PositiveOffsetUsesLocalCalendarDay(t *testing.T) {
	// 2026-03-04T16:30:00Z is already 2026-03-05 in UTC+08.
	now := time.Date(2026, 3, 4, 16, 30, 0, 0, time.UTC)
	loc := time.FixedZone("UTC+8", 8*60*60)

	got := CurrentDay(fakeClock{now: now}, loc)
	want := domain.MustParseDay("2026-03-05")

	if got.String() != want.String() {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestCurrentDay_NilLocationDefaultsToUTC(t *testing.T) {
	now := time.Date(2026, 3, 4, 23, 59, 0, 0, time.UTC)

	got := CurrentDay(fakeClock{now: now}, nil)
	want := domain.MustParseDay("2026-03-04")

	if got.String() != want.String() {
		t.Fatalf("expected %s, got %s", want, got)
	}
}
