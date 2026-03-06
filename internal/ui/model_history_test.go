package ui

import (
	"errors"
	"regexp"
	"strings"
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
	um, cmd := m.Update(keyTab())
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
	um, cmd := m.Update(keyTab())
	m = um.(Model)
	um, cmd = m.Update(keyTab())
	m = um.(Model)
	m = applyCmd(m, cmd)

	bodyOnly := renderHistoryBody(m)
	if indexOf(bodyOnly, "None") < 0 {
		t.Fatalf("expected history body to show None when there are no rows, got:\n%s", bodyOnly)
	}
	if indexOf(bodyOnly, "Done") >= 0 || indexOf(bodyOnly, "Abandoned") >= 0 {
		t.Fatalf("expected history body to not include Done/Abandoned headings anymore, got:\n%s", bodyOnly)
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
	a.historyActiveByCreatedDay = map[string][]domain.Task{}
	a.statsRatios = map[string]float64{"done": 0.25, "abandoned": 0.50}

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	um, cmd := m.Update(keyTab())
	m = um.(Model)
	um, cmd = m.Update(keyTab())
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected cmd when entering history view")
	}
	um, _ = m.Update(cmd())
	m = um.(Model)

	if m.view != viewHistory {
		t.Fatalf("expected viewHistory")
	}
	if a.historyDoneCalls != 1 || a.historyAbandonedCalls != 1 || a.historyActiveCreatedCalls != 1 {
		t.Fatalf("expected done/abandoned/activeCreated called once, got done=%d abandoned=%d activeCreated=%d", a.historyDoneCalls, a.historyAbandonedCalls, a.historyActiveCreatedCalls)
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
	if !containsAll(v, []string{"> 03-04", "[✓] done-1", "[✗] ab-1", "DoneDelayedRatio", "AbandonedDelayedRatio"}) {
		t.Fatalf("expected history view to include lists and ratios, got:\n%s", v)
	}
	if indexOf(v, current.String()) >= 0 {
		t.Fatalf("expected history dates to not include year, got:\n%s", v)
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
	a.historyActiveByCreatedDay = map[string][]domain.Task{}
	a.statsRatios = map[string]float64{}

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	um, cmd := m.Update(keyTab())
	m = um.(Model)
	um, cmd = m.Update(keyTab())
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
	if !containsAll(v, []string{"> 03-03", "[✓] dprev"}) {
		t.Fatalf("expected selection marker and content updated, got:\n%s", v)
	}
}

func TestModel_History_KDoesNotRefreshStats(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	prev := domain.DayFromTime(current.Time().AddDate(0, 0, -1))
	a := newFakeApp(current, nil)
	a.historyDoneByDay = map[string][]domain.Task{prev.String(): {{ID: 10, Title: "dprev", Status: domain.StatusDone, CreatedDay: prev, DueDay: prev}}}
	a.historyAbandonedByDay = map[string][]domain.Task{}
	a.historyActiveByCreatedDay = map[string][]domain.Task{}
	a.statsRatios = map[string]float64{"done": 0.25, "abandoned": 0.50}

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	um, cmd := m.Update(keyTab())
	m = um.(Model)
	um, cmd = m.Update(keyTab())
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
	a.historyActiveByCreatedDay = map[string][]domain.Task{}

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	um, cmd := m.Update(keyTab())
	m = um.(Model)
	um, cmd = m.Update(keyTab())
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

func TestModel_History_HNoLongerShiftsWindow(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, nil)
	a.historyDoneByDay = map[string][]domain.Task{}
	a.historyAbandonedByDay = map[string][]domain.Task{}
	a.historyActiveByCreatedDay = map[string][]domain.Task{}
	a.statsRatios = map[string]float64{}

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	um, cmd := m.Update(keyTab())
	m = um.(Model)
	um, cmd = m.Update(keyTab())
	m = um.(Model)
	um, _ = m.Update(cmd())
	m = um.(Model)

	from0, to0, idx0 := m.historyFrom, m.historyTo, m.historyIndex
	a.resetHistoryCounters()

	um, cmd = m.Update(keyRune('h'))
	m = um.(Model)
	if cmd != nil {
		m = applyCmd(m, cmd)
	}

	if m.historyFrom.String() != from0.String() || m.historyTo.String() != to0.String() || m.historyIndex != idx0 {
		t.Fatalf("expected h to do nothing, got from=%s to=%s idx=%d", m.historyFrom.String(), m.historyTo.String(), m.historyIndex)
	}
	if a.statsCalls != 0 {
		t.Fatalf("expected no stats refresh from h, got %d", a.statsCalls)
	}
}

func TestModel_Keymap_TabCyclesBetweenTopViews(t *testing.T) {
	disableTick(t)

	day := domain.MustParseDay("2026-03-04")
	a := newFakeApp(day, nil)

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m = applyCmd(m, m.Init())

	if m.view != viewToday {
		t.Fatalf("expected initial viewToday")
	}

	// Today -> Upcoming
	um, cmd := m.Update(keyTab())
	m = um.(Model)
	m = applyCmd(m, cmd)
	if m.view != viewUpcoming {
		t.Fatalf("expected Tab to switch to viewUpcoming, got %v", m.view)
	}

	// Upcoming -> History
	um, cmd = m.Update(keyTab())
	m = um.(Model)
	m = applyCmd(m, cmd)
	if m.view != viewHistory {
		t.Fatalf("expected Tab to switch to viewHistory, got %v", m.view)
	}

	// History -> Today
	um, cmd = m.Update(keyTab())
	m = um.(Model)
	m = applyCmd(m, cmd)
	if m.view != viewToday {
		t.Fatalf("expected Tab to cycle back to viewToday, got %v", m.view)
	}
}

func TestModel_History_LeftRightKeysNoLongerShiftWindow(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, nil)
	a.historyActiveByCreatedDay = map[string][]domain.Task{}

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	um, cmd := m.Update(keyTab())
	m = um.(Model)
	um, cmd = m.Update(keyTab())
	m = um.(Model)
	m = applyCmd(m, cmd)

	from0, to0, idx0 := m.historyFrom, m.historyTo, m.historyIndex
	a.resetHistoryCounters()

	// h/left and l/right used to shift the 7-day window; now they do nothing.
	keys := []tea.KeyMsg{
		keyRune('h'),
		{Type: tea.KeyLeft},
		keyRune('l'),
		{Type: tea.KeyRight},
	}
	for _, km := range keys {
		um, cmd = m.Update(km)
		m = um.(Model)
		if cmd != nil {
			m = applyCmd(m, cmd)
		}
	}

	if m.historyFrom.String() != from0.String() || m.historyTo.String() != to0.String() || m.historyIndex != idx0 {
		t.Fatalf("expected history window/index unchanged, got from=%s to=%s idx=%d", m.historyFrom.String(), m.historyTo.String(), m.historyIndex)
	}
	if a.statsCalls != 0 {
		t.Fatalf("expected no stats refresh from left/right keys, got %d", a.statsCalls)
	}
}

func TestModel_History_UpAtTopAutoRollsWindowBackOneDay(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, nil)
	a.historyActiveByCreatedDay = map[string][]domain.Task{}

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	um, cmd := m.Update(keyTab())
	m = um.(Model)
	um, cmd = m.Update(keyTab())
	m = um.(Model)
	m = applyCmd(m, cmd)

	// Move selection to top of window.
	for i := 0; i < 6; i++ {
		um, cmd = m.Update(keyRune('k'))
		m = um.(Model)
		m = applyCmd(m, cmd)
	}
	if m.historyIndex != 0 {
		t.Fatalf("expected historyIndex at top (0), got %d", m.historyIndex)
	}

	from0, to0 := m.historyFrom, m.historyTo
	a.resetHistoryCounters()

	// One more up should auto-roll the window back 1 day.
	um, cmd = m.Update(keyRune('k'))
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected cmd after auto-roll")
	}
	m = applyCmd(m, cmd)

	if m.historyIndex != 0 {
		t.Fatalf("expected index to remain at 0 after auto-roll, got %d", m.historyIndex)
	}
	if m.historyFrom.String() != addDays(from0, -1).String() || m.historyTo.String() != addDays(to0, -1).String() {
		t.Fatalf("expected window to roll back by 1 day, got from=%s to=%s", m.historyFrom.String(), m.historyTo.String())
	}
	if a.statsCalls != 1 {
		t.Fatalf("expected stats refresh on window roll, got %d", a.statsCalls)
	}
}

func TestModel_History_DownAtBottomAutoRollsWindowForwardOneDay_ClampedAtToday(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, nil)
	a.historyActiveByCreatedDay = map[string][]domain.Task{}

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	um, cmd := m.Update(keyTab())
	m = um.(Model)
	um, cmd = m.Update(keyTab())
	m = um.(Model)
	m = applyCmd(m, cmd)

	// Move selection to bottom (today).
	if m.historyIndex != 6 {
		t.Fatalf("expected initial historyIndex at 6, got %d", m.historyIndex)
	}

	from0, to0 := m.historyFrom, m.historyTo
	a.resetHistoryCounters()

	// Down at bottom should attempt to roll forward, but it's already clamped at today.
	um, cmd = m.Update(keyRune('j'))
	m = um.(Model)
	if cmd != nil {
		m = applyCmd(m, cmd)
	}

	if m.historyFrom.String() != from0.String() || m.historyTo.String() != to0.String() {
		t.Fatalf("expected window clamped at today (no forward roll), got from=%s to=%s", m.historyFrom.String(), m.historyTo.String())
	}
	if m.historyIndex != 6 {
		t.Fatalf("expected index to remain at 6, got %d", m.historyIndex)
	}
	if a.statsCalls != 0 {
		t.Fatalf("expected no stats refresh when clamped, got %d", a.statsCalls)
	}

	// Roll back one day by auto-roll at top, then roll forward by one day and ensure it stops at today.
	for i := 0; i < 6; i++ {
		um, cmd = m.Update(keyRune('k'))
		m = um.(Model)
		m = applyCmd(m, cmd)
	}
	um, cmd = m.Update(keyRune('k'))
	m = um.(Model)
	m = applyCmd(m, cmd)
	if m.historyTo.String() != addDays(to0, -1).String() {
		t.Fatalf("expected window shifted back one day")
	}

	// Now at least one forward roll is possible.
	for i := 0; i < 6; i++ {
		um, cmd = m.Update(keyRune('j'))
		m = um.(Model)
		m = applyCmd(m, cmd)
	}
	a.resetHistoryCounters()
	um, cmd = m.Update(keyRune('j'))
	m = um.(Model)
	m = applyCmd(m, cmd)
	if m.historyTo.String() != to0.String() {
		t.Fatalf("expected forward roll back to today, got to=%s", m.historyTo.String())
	}
	if a.statsCalls != 1 {
		t.Fatalf("expected stats refresh on forward roll, got %d", a.statsCalls)
	}
}

func TestModel_History_DownAtBottom_WhenWindowLagsToday_MovesForwardOnlyOneDay(t *testing.T) {
	disableTick(t)

	today := domain.MustParseDay("2026-03-10")
	a := newFakeApp(today, nil)
	a.historyActiveByCreatedDay = map[string][]domain.Task{}

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)}, time.UTC)
	um, cmd := m.Update(keyTab())
	m = um.(Model)
	um, cmd = m.Update(keyTab())
	m = um.(Model)
	m = applyCmd(m, cmd)

	// Put the selection at the bottom, but set the visible 7-day window to lag behind today.
	m.historyTo = addDays(today, -3)
	m.historyFrom = addDays(m.historyTo, -6)
	m.historyIndex = 6

	from0, to0 := m.historyFrom, m.historyTo
	a.resetHistoryCounters()

	um, cmd = m.Update(keyRune('j'))
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected cmd after auto-roll")
	}
	m = applyCmd(m, cmd)

	if m.historyIndex != 6 {
		t.Fatalf("expected index to remain at 6, got %d", m.historyIndex)
	}
	if m.historyFrom.String() != addDays(from0, 1).String() {
		t.Fatalf("expected historyFrom to move forward by 1 day, got %s", m.historyFrom.String())
	}
	if m.historyTo.String() != addDays(to0, 1).String() {
		t.Fatalf("expected historyTo to move forward by 1 day, got %s", m.historyTo.String())
	}
	if m.historyTo.String() != addDays(m.historyFrom, 6).String() {
		t.Fatalf("expected 7-day window preserved, got from=%s to=%s", m.historyFrom.String(), m.historyTo.String())
	}
	if a.statsCalls != 1 {
		t.Fatalf("expected stats refresh on window roll, got %d", a.statsCalls)
	}
	if a.lastStatsFrom.String() != m.historyFrom.String() || a.lastStatsTo.String() != m.historyTo.String() {
		t.Fatalf("expected stats called with %s..%s, got %s..%s", m.historyFrom.String(), m.historyTo.String(), a.lastStatsFrom.String(), a.lastStatsTo.String())
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

func TestRenderHistoryBody_FormatsRowsAndHighlightsDelayed(t *testing.T) {
	disableTick(t)

	lipgloss.SetColorProfile(termenv.ANSI256)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	today := domain.MustParseDay("2026-03-04")
	selected := domain.MustParseDay("2026-03-03")
	doneOnTime := domain.MustParseDay("2026-03-02")
	doneLate := domain.MustParseDay("2026-03-03")

	m := NewWithDeps(newFakeApp(today, nil), fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m.view = viewHistory
	m.historyFrom = selected
	m.historyIndex = 0

	m.historyDone = []domain.Task{
		{ID: 1, Title: "done-late", Status: domain.StatusDone, CreatedDay: selected, DueDay: doneOnTime, DoneDay: &doneLate},
		// selectedDay is after due, but completion day is not: should NOT be marked delayed.
		{ID: 2, Title: "done-on-time", Status: domain.StatusDone, CreatedDay: selected, DueDay: doneOnTime, DoneDay: &doneOnTime},
		// Be conservative: missing DoneDay should not be marked delayed.
		{ID: 3, Title: "done-missing-day", Status: domain.StatusDone, CreatedDay: selected, DueDay: doneOnTime, DoneDay: nil},
	}
	m.historyAbandoned = []domain.Task{
		{ID: 4, Title: "ab-late", Status: domain.StatusAbandoned, CreatedDay: selected, DueDay: doneOnTime, AbandonedDay: &doneLate},
		// selectedDay is after due, but abandonment day is not: should NOT be marked delayed.
		{ID: 5, Title: "ab-on-time", Status: domain.StatusAbandoned, CreatedDay: selected, DueDay: doneOnTime, AbandonedDay: &doneOnTime},
	}
	m.historyActiveCreated = []domain.Task{
		{ID: 6, Title: "active-late", Status: domain.StatusActive, CreatedDay: selected, DueDay: domain.MustParseDay("2026-03-03")},
		{ID: 7, Title: "active-not-late", Status: domain.StatusActive, CreatedDay: selected, DueDay: today},
	}

	body := renderHistoryBody(m)
	if !containsAll(body, []string{"> 03-03", "[✓] done-late", "[✓] done-on-time", "[✓] done-missing-day", "[✗] ab-late", "[✗] ab-on-time", "[ ] active-late"}) {
		t.Fatalf("expected formatted rows, got:\n%s", body)
	}
	if indexOf(body, "active-not-late") >= 0 {
		t.Fatalf("expected non-overdue active task to be filtered out, got:\n%s", body)
	}

	red := regexp.MustCompile("\\x1b\\[[0-9;]*(31|91|38;5;1|38;5;9)[0-9;]*m")
	if !red.MatchString(lineContaining(body, "[✓] done-late")) {
		t.Fatalf("expected delayed done row to be red, got %q", body)
	}
	if red.MatchString(lineContaining(body, "[✓] done-on-time")) {
		t.Fatalf("expected on-time done row to not be red, got %q", body)
	}
	if red.MatchString(lineContaining(body, "[✓] done-missing-day")) {
		t.Fatalf("expected done row with missing DoneDay to not be red, got %q", body)
	}
	if !red.MatchString(lineContaining(body, "[✗] ab-late")) {
		t.Fatalf("expected delayed abandoned row to be red, got %q", body)
	}
	if red.MatchString(lineContaining(body, "[✗] ab-on-time")) {
		t.Fatalf("expected on-time abandoned row to not be red, got %q", body)
	}
	if !red.MatchString(lineContaining(body, "[ ] active-late")) {
		t.Fatalf("expected overdue active row to be red, got %q", body)
	}

	if !(indexOf(body, "[✓] done-late") < indexOf(body, "[✗] ab-late") && indexOf(body, "[✗] ab-late") < indexOf(body, "[ ] active-late")) {
		t.Fatalf("expected group order done -> abandoned -> overdue active, got:\n%s", body)
	}
}

func lineContaining(s, sub string) string {
	for _, ln := range strings.Split(s, "\n") {
		if strings.Contains(ln, sub) {
			return ln
		}
	}
	return ""
}
