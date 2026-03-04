package domain

import (
	"testing"
	"time"
)

func TestDayFromTime_NormalizesToMidnightUTC(t *testing.T) {
	t.Run("converts input time to UTC day midnight", func(t *testing.T) {
		inLoc := time.FixedZone("X", -8*60*60)
		in := time.Date(2026, 3, 4, 23, 45, 0, 0, inLoc) // 2026-03-05T07:45:00Z

		d := DayFromTime(in)

		want := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)
		if !d.Time().Equal(want) {
			t.Fatalf("expected %v, got %v", want, d.Time())
		}
		if d.Time().Location() != time.UTC {
			t.Fatalf("expected UTC location, got %v", d.Time().Location())
		}
	})
}

func TestParseDay_NormalizedMidnightUTC(t *testing.T) {
	d, err := ParseDay("2026-03-04")
	if err != nil {
		t.Fatalf("expected parse to succeed: %v", err)
	}

	if d.Time().Location() != time.UTC {
		t.Fatalf("expected UTC location, got %v", d.Time().Location())
	}
	if d.Time().Hour() != 0 || d.Time().Minute() != 0 || d.Time().Second() != 0 || d.Time().Nanosecond() != 0 {
		t.Fatalf("expected midnight UTC, got %v", d.Time())
	}

	d2 := MustParseDay(d.String())
	if !d2.Time().Equal(d.Time()) {
		t.Fatalf("expected round-trip parse to preserve day, got %v then %v", d.Time(), d2.Time())
	}
}
