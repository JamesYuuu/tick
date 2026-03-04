package ui

import (
	"errors"
	"testing"
	"time"

	"github.com/JamesYuuu/tick/internal/domain"
)

func TestModel_History_EnterRefreshesStatsAndSelectedDayLists(t *testing.T) {
	orig := tickEvery
	tickEvery = 0
	t.Cleanup(func() { tickEvery = orig })

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, nil)
	a.historyDoneByDay = map[string][]domain.Task{
		current.String(): {{ID: 1, Title: "done-1", Status: domain.StatusDone, CreatedDay: current, DueDay: current}},
	}
	a.historyAbandonedByDay = map[string][]domain.Task{
		current.String(): {{ID: 2, Title: "ab-1", Status: domain.StatusAbandoned, CreatedDay: current, DueDay: current}},
	}
	a.statsRatios = map[string]float64{"done": 0.25, "abandoned": 0.50}

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	um, cmd := m.Update(keyRune('3'))
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected cmd when entering history view")
	}
	um, _ = m.Update(cmd())
	m = um.(Model)

	if m.view != viewHistory {
		t.Fatalf("expected viewHistory")
	}
	if a.historyDoneCalls != 1 || a.historyAbandonedCalls != 1 {
		t.Fatalf("expected done/abandoned called once, got done=%d abandoned=%d", a.historyDoneCalls, a.historyAbandonedCalls)
	}
	if a.statsCalls != 1 {
		t.Fatalf("expected stats called once, got %d", a.statsCalls)
	}
	if a.lastHistoryDay.String() != current.String() {
		t.Fatalf("expected history called with day=%s, got %s", current.String(), a.lastHistoryDay.String())
	}
	wantFrom := domain.DayFromTime(current.Time().AddDate(0, 0, -6))
	if a.lastStatsFrom.String() != wantFrom.String() || a.lastStatsTo.String() != current.String() {
		t.Fatalf("expected stats called with %s..%s, got %s..%s", wantFrom.String(), current.String(), a.lastStatsFrom.String(), a.lastStatsTo.String())
	}

	v := m.View()
	if !containsAll(v, []string{"Done", "done-1", "Abandoned", "ab-1", "DoneDelayedRatio", "AbandonedDelayedRatio"}) {
		t.Fatalf("expected history view to include lists and ratios, got:\n%s", v)
	}

	bodyOnly := renderHistoryBody(m)
	if containsAll(bodyOnly, []string{"DoneDelayedRatio", "AbandonedDelayedRatio"}) {
		t.Fatalf("expected ratios to be in footer/status area, not body")
	}
}

func TestModel_History_KMovesSelectionUpOneDayAndRefreshes(t *testing.T) {
	orig := tickEvery
	tickEvery = 0
	t.Cleanup(func() { tickEvery = orig })

	current := domain.MustParseDay("2026-03-04")
	prev := domain.DayFromTime(current.Time().AddDate(0, 0, -1))
	a := newFakeApp(current, nil)
	a.historyDoneByDay = map[string][]domain.Task{prev.String(): {{ID: 10, Title: "dprev", Status: domain.StatusDone, CreatedDay: prev, DueDay: prev}}}
	a.historyAbandonedByDay = map[string][]domain.Task{}
	a.statsRatios = map[string]float64{}

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	um, cmd := m.Update(keyRune('3'))
	m = um.(Model)
	um, _ = m.Update(cmd())
	m = um.(Model)

	a.resetHistoryCounters()
	um, cmd = m.Update(keyRune('k'))
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected cmd after moving selection")
	}
	um, _ = m.Update(cmd())
	m = um.(Model)

	if a.lastHistoryDay.String() != prev.String() {
		t.Fatalf("expected refresh for day=%s, got %s", prev.String(), a.lastHistoryDay.String())
	}
	v := m.View()
	if !containsAll(v, []string{"> " + prev.String(), "dprev"}) {
		t.Fatalf("expected selection marker on %s and content updated, got:\n%s", prev.String(), v)
	}
}

func TestModel_History_KDoesNotRefreshStats(t *testing.T) {
	orig := tickEvery
	tickEvery = 0
	t.Cleanup(func() { tickEvery = orig })

	current := domain.MustParseDay("2026-03-04")
	prev := domain.DayFromTime(current.Time().AddDate(0, 0, -1))
	a := newFakeApp(current, nil)
	a.historyDoneByDay = map[string][]domain.Task{prev.String(): {{ID: 10, Title: "dprev", Status: domain.StatusDone, CreatedDay: prev, DueDay: prev}}}
	a.historyAbandonedByDay = map[string][]domain.Task{}
	a.statsRatios = map[string]float64{"done": 0.25, "abandoned": 0.50}

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	um, cmd := m.Update(keyRune('3'))
	m = um.(Model)
	um, _ = m.Update(cmd())
	m = um.(Model)

	if a.statsCalls != 1 {
		t.Fatalf("expected stats called once on entering history, got %d", a.statsCalls)
	}

	um, cmd = m.Update(keyRune('k'))
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected cmd after moving selection")
	}
	um, _ = m.Update(cmd())
	_ = um.(Model)

	if a.statsCalls != 1 {
		t.Fatalf("expected k selection move to not refresh stats, got statsCalls=%d", a.statsCalls)
	}
}

func TestModel_History_RefreshPassesThroughHistoryDoneError(t *testing.T) {
	orig := tickEvery
	tickEvery = 0
	t.Cleanup(func() { tickEvery = orig })

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, nil)
	a.historyErr = errors.New("history done: boom")

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	um, cmd := m.Update(keyRune('3'))
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected cmd when entering history view")
	}
	um, _ = m.Update(cmd())
	m = um.(Model)

	if m.statusMsg != "history done: boom" {
		t.Fatalf("expected status message passed through unchanged, got %q", m.statusMsg)
	}
}

func TestModel_History_HShiftsWindowBackOneDayAndRefreshesStats(t *testing.T) {
	orig := tickEvery
	tickEvery = 0
	t.Cleanup(func() { tickEvery = orig })

	current := domain.MustParseDay("2026-03-04")
	shiftedTo := domain.DayFromTime(current.Time().AddDate(0, 0, -1))
	a := newFakeApp(current, nil)
	a.historyDoneByDay = map[string][]domain.Task{}
	a.historyAbandonedByDay = map[string][]domain.Task{}
	a.statsRatios = map[string]float64{}

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	um, cmd := m.Update(keyRune('3'))
	m = um.(Model)
	um, _ = m.Update(cmd())
	m = um.(Model)

	a.resetHistoryCounters()
	um, cmd = m.Update(keyRune('h'))
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected cmd after shifting window")
	}
	um, _ = m.Update(cmd())
	_ = um.(Model)

	wantFrom := domain.DayFromTime(shiftedTo.Time().AddDate(0, 0, -6))
	if a.lastStatsFrom.String() != wantFrom.String() || a.lastStatsTo.String() != shiftedTo.String() {
		t.Fatalf("expected stats called with %s..%s, got %s..%s", wantFrom.String(), shiftedTo.String(), a.lastStatsFrom.String(), a.lastStatsTo.String())
	}
	if a.lastHistoryDay.String() != shiftedTo.String() {
		t.Fatalf("expected history lists refreshed for selected day=%s, got %s", shiftedTo.String(), a.lastHistoryDay.String())
	}
}

func containsAll(s string, subs []string) bool {
	for _, sub := range subs {
		if indexOf(s, sub) < 0 {
			return false
		}
	}
	return true
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
