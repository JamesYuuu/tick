package ui

import (
	"regexp"
	"testing"
	"time"

	"github.com/JamesYuuu/tick/internal/domain"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func setImmediateTick(t *testing.T) {
	orig := tickEvery
	tickEvery = 0
	t.Cleanup(func() { tickEvery = orig })
}

func execBatchCmds(cmd tea.Cmd) ([]tea.Msg, bool) {
	if cmd == nil {
		return nil, false
	}
	m := cmd()
	s, ok := m.(tea.BatchMsg)
	if !ok {
		return []tea.Msg{m}, false
	}
	out := make([]tea.Msg, 0, len(s))
	for _, c := range s {
		if c == nil {
			continue
		}
		out = append(out, c())
	}
	return out, true
}

func TestModel_Tick_ReschedulesAndDoesNotRefreshWhenDayUnchanged(t *testing.T) {
	setImmediateTick(t)

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, nil)
	clk := &fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}

	m := NewWithDeps(a, clk, time.UTC)
	m = applyCmd(m, m.Init())

	a.todayCalls = 0
	a.upcomingCalls = 0

	_, cmd := m.Update(tickMsg{})
	if cmd == nil {
		t.Fatalf("expected tick to return a command")
	}
	msgs, _ := execBatchCmds(cmd)
	if len(msgs) != 1 {
		t.Fatalf("expected tick to only schedule one next tick, got %d msgs", len(msgs))
	}
	if _, ok := msgs[0].(tickMsg); !ok {
		t.Fatalf("expected scheduled tick to yield tickMsg, got %T", msgs[0])
	}
	if a.todayCalls != 0 || a.upcomingCalls != 0 {
		t.Fatalf("expected no refresh calls when day unchanged, got today=%d upcoming=%d", a.todayCalls, a.upcomingCalls)
	}
}

func TestModel_Tick_RefreshesActiveListsWhenDayChanged(t *testing.T) {
	setImmediateTick(t)

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, nil)
	clk := &fakeClock{now: time.Date(2026, 3, 4, 23, 59, 0, 0, time.UTC)}

	m := NewWithDeps(a, clk, time.UTC)
	m = applyCmd(m, m.Init())

	a.todayCalls = 0
	a.upcomingCalls = 0

	// Simulate day rollover.
	clk.now = time.Date(2026, 3, 5, 0, 1, 0, 0, time.UTC)

	um, cmd := m.Update(tickMsg{})
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected tick to return a command")
	}
	msg := cmd()
	bm, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected tea.BatchMsg, got %T", msg)
	}
	if len(bm) != 2 {
		t.Fatalf("expected tick to batch refresh+reschedule on day change, got batch len=%d", len(bm))
	}
	// Apply refresh; ignore rescheduled tick cmd.
	msgs, _ := execBatchCmds(cmd)
	var gotRefresh bool
	var gotTick bool
	for _, msg := range msgs {
		switch m2 := msg.(type) {
		case refreshMsg:
			gotRefresh = true
			um, _ = m.Update(m2)
			m = um.(Model)
		case tickMsg:
			gotTick = true
		}
	}
	if !gotRefresh || !gotTick {
		t.Fatalf("expected both refreshMsg and tickMsg, got refresh=%v tick=%v", gotRefresh, gotTick)
	}

	if a.todayCalls == 0 || a.upcomingCalls == 0 {
		t.Fatalf("expected refresh calls when day changed, got today=%d upcoming=%d", a.todayCalls, a.upcomingCalls)
	}
}

func TestModel_Tick_InHistoryViewUpdatesWindowToEndAtNewDay(t *testing.T) {
	setImmediateTick(t)

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, nil)
	clk := &fakeClock{now: time.Date(2026, 3, 4, 23, 59, 0, 0, time.UTC)}

	m := NewWithDeps(a, clk, time.UTC)
	// Enter history view.
	um, cmd := m.Update(keyTab())
	m = um.(Model)
	um, cmd = m.Update(keyTab())
	m = um.(Model)
	m = applyCmd(m, cmd)

	// Simulate day rollover.
	clk.now = time.Date(2026, 3, 5, 0, 1, 0, 0, time.UTC)

	a.resetHistoryCounters()

	um, cmd = m.Update(tickMsg{})
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected cmd")
	}
	msgs, _ := execBatchCmds(cmd)
	var gotRefresh bool
	var gotTick bool
	for _, msg := range msgs {
		switch m2 := msg.(type) {
		case historyRefreshMsg:
			gotRefresh = true
			um, _ = m.Update(m2)
			m = um.(Model)
		case tickMsg:
			gotTick = true
		}
	}
	if !gotRefresh || !gotTick {
		t.Fatalf("expected both historyRefreshMsg and tickMsg, got refresh=%v tick=%v", gotRefresh, gotTick)
	}

	newDay := domain.MustParseDay("2026-03-05")
	wantFrom := domain.DayFromTime(newDay.Time().AddDate(0, 0, -6))
	if a.lastStatsFrom.String() != wantFrom.String() || a.lastStatsTo.String() != newDay.String() {
		t.Fatalf("expected stats called with %s..%s, got %s..%s", wantFrom.String(), newDay.String(), a.lastStatsFrom.String(), a.lastStatsTo.String())
	}
}

func TestModel_Tick_UpdatesDelayedHighlightingAfterRollover(t *testing.T) {
	setImmediateTick(t)

	lipgloss.SetColorProfile(termenv.ANSI256)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })
	red := regexp.MustCompile("\\x1b\\[[0-9;]*(31|91|38;5;9)[0-9;]*m")

	// Task due today should become delayed after day rolls over.
	day1 := domain.MustParseDay("2026-03-04")
	task := domain.Task{ID: 1, Title: "t1", Status: domain.StatusActive, CreatedDay: day1, DueDay: day1}
	a := newFakeApp(day1, []domain.Task{task})
	clk := &fakeClock{now: time.Date(2026, 3, 4, 23, 59, 0, 0, time.UTC)}

	m := NewWithDeps(a, clk, time.UTC)
	um, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = um.(Model)
	m = applyCmd(m, m.Init())
	if red.MatchString(m.View()) {
		t.Fatalf("expected no delayed highlighting before rollover")
	}

	// Before rollover: due day equals current day => not delayed.
	if task.IsDelayed(day1) {
		t.Fatalf("expected task to not be delayed on day1")
	}

	// Roll to next day and apply refresh.
	clk.now = time.Date(2026, 3, 5, 0, 1, 0, 0, time.UTC)
	um, cmd := m.Update(tickMsg{})
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected cmd")
	}
	msgs, _ := execBatchCmds(cmd)
	for _, msg := range msgs {
		if r, ok := msg.(refreshMsg); ok {
			um, _ = m.Update(r)
			m = um.(Model)
		}
	}
	if !red.MatchString(m.View()) {
		t.Fatalf("expected delayed highlighting after rollover")
	}
}
