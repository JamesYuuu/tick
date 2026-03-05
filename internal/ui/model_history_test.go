package ui

import (
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/JamesYuuu/tick/internal/domain"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func disableTick(t *testing.T) {
	orig := tickEvery
	tickEvery = 0
	t.Cleanup(func() { tickEvery = orig })
}

func TestModel_Today_ShowsEmptyStateWhenNoTasks(t *testing.T) {
	disableTick(t)

	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	day := domain.MustParseDay("2026-03-04")
	a := newFakeApp(day, nil)

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	um, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = um.(Model)
	m = applyCmd(m, m.Init())

	out := m.View()
	if indexOf(out, "Nothing due today.") < 0 {
		t.Fatalf("expected empty state copy for today view, got:\n%s", out)
	}
}

func TestModel_Upcoming_ShowsEmptyStateWhenNoTasks(t *testing.T) {
	disableTick(t)

	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	day := domain.MustParseDay("2026-03-04")
	a := newFakeApp(day, nil)

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	um, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = um.(Model)
	m = applyCmd(m, m.Init())

	// Switch to Upcoming view and refresh lists.
	um, cmd := m.Update(keyRune('2'))
	m = um.(Model)
	m = applyCmd(m, cmd)

	out := m.View()
	if indexOf(out, "No upcoming tasks.") < 0 {
		t.Fatalf("expected empty state copy for upcoming view, got:\n%s", out)
	}
}

func TestModel_History_ShowsEmptyCopyUnderHeadingsWhenNoOutcomes(t *testing.T) {
	disableTick(t)

	day := domain.MustParseDay("2026-03-04")
	a := newFakeApp(day, nil)

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	um, cmd := m.Update(keyRune('3'))
	m = um.(Model)
	m = applyCmd(m, cmd)

	bodyOnly := renderHistoryBody(m)
	re := regexp.MustCompile(`(?s)Done.*\(none\).*Abandoned.*\(none\)`)
	if !re.MatchString(bodyOnly) {
		t.Fatalf("expected history body to show empty copy under headings, got:\n%s", bodyOnly)
	}
}

func TestModel_History_EnterRefreshesStatsAndSelectedDayLists(t *testing.T) {
	disableTick(t)

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
	disableTick(t)

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
	disableTick(t)

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
	disableTick(t)

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
	disableTick(t)

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
