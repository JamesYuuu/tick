package domain

import "testing"

func TestTask_IsDelayed(t *testing.T) {
	t.Run("active task delayed when due day before current day", func(t *testing.T) {
		task := Task{
			ID:     1,
			Title:  "t",
			Status: StatusActive,
			CreatedDay: MustParseDay("2026-03-01"),
			DueDay: MustParseDay("2026-03-01"),
		}

		if !task.IsDelayed(MustParseDay("2026-03-04")) {
			t.Fatalf("expected task to be delayed")
		}
	})

	t.Run("active task not delayed when due day same as current day", func(t *testing.T) {
		task := Task{
			ID:     1,
			Title:  "t",
			Status: StatusActive,
			CreatedDay: MustParseDay("2026-03-01"),
			DueDay: MustParseDay("2026-03-04"),
		}

		if task.IsDelayed(MustParseDay("2026-03-04")) {
			t.Fatalf("expected task to not be delayed")
		}
	})

	t.Run("done task never delayed", func(t *testing.T) {
		task := Task{
			ID:     1,
			Title:  "t",
			Status: StatusDone,
			CreatedDay: MustParseDay("2026-03-01"),
			DueDay: MustParseDay("2026-03-01"),
		}

		if task.IsDelayed(MustParseDay("2026-03-04")) {
			t.Fatalf("expected done task to not be delayed")
		}
	})

	t.Run("abandoned task never delayed", func(t *testing.T) {
		task := Task{
			ID:     1,
			Title:  "t",
			Status: StatusAbandoned,
			CreatedDay: MustParseDay("2026-03-01"),
			DueDay: MustParseDay("2026-03-01"),
		}

		if task.IsDelayed(MustParseDay("2026-03-04")) {
			t.Fatalf("expected abandoned task to not be delayed")
		}
	})
}
