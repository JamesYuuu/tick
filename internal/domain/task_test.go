package domain

import "testing"

func TestTask_IsDelayed(t *testing.T) {
	t.Run("active task delayed when due day before current day", func(t *testing.T) {
		task := Task{
			Status: StatusActive,
			DueDay: MustParseDay("2026-03-01"),
		}

		if !task.IsDelayed(MustParseDay("2026-03-04")) {
			t.Fatalf("expected task to be delayed")
		}
	})

	t.Run("active task not delayed when due day same as current day", func(t *testing.T) {
		task := Task{
			Status: StatusActive,
			DueDay: MustParseDay("2026-03-04"),
		}

		if task.IsDelayed(MustParseDay("2026-03-04")) {
			t.Fatalf("expected task to not be delayed")
		}
	})

	t.Run("done task never delayed", func(t *testing.T) {
		task := Task{
			Status: StatusDone,
			DueDay: MustParseDay("2026-03-01"),
		}

		if task.IsDelayed(MustParseDay("2026-03-04")) {
			t.Fatalf("expected done task to not be delayed")
		}
	})
}

func TestDay_ParseAndStringAndBefore(t *testing.T) {
	d, err := ParseDay("2026-03-04")
	if err != nil {
		t.Fatalf("expected parse to succeed: %v", err)
	}

	if got := d.String(); got != "2026-03-04" {
		t.Fatalf("expected String() to match input, got %q", got)
	}

	if !MustParseDay("2026-03-03").Before(MustParseDay("2026-03-04")) {
		t.Fatalf("expected Before to be true")
	}
	if MustParseDay("2026-03-04").Before(MustParseDay("2026-03-04")) {
		t.Fatalf("expected Before to be false when equal")
	}
}
