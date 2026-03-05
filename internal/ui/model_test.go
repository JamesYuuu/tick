package ui

import (
	"bytes"
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/JamesYuuu/tick/internal/app"
	"github.com/JamesYuuu/tick/internal/domain"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func listLen(m list.Model) int { return len(m.Items()) }

type fakeClock struct{ now time.Time }

func (c fakeClock) Now() time.Time { return c.now }

type fakeApp struct {
	currentDay domain.Day
	nextID     int64
	tasks      []domain.Task

	todayCalls    int
	upcomingCalls int

	historyDoneByDay      map[string][]domain.Task
	historyAbandonedByDay map[string][]domain.Task
	historyDoneCalls      int
	historyAbandonedCalls int
	lastHistoryDay        domain.Day
	historyErr            error

	statsRatios   map[string]float64
	statsCalls    int
	lastStatsFrom domain.Day
	lastStatsTo   domain.Day
	statsErr      error

	addedTitles  []string
	doneIDs      []int64
	abandonedIDs []int64
	postponedIDs []int64

	todayErr    error
	upcomingErr error
	addErr      error
	doneErr     error
	abandonErr  error
	postponeErr error
}

func newFakeApp(currentDay domain.Day, tasks []domain.Task) *fakeApp {
	maxID := int64(0)
	for _, t := range tasks {
		if t.ID > maxID {
			maxID = t.ID
		}
	}
	return &fakeApp{
		currentDay:            currentDay,
		nextID:                maxID + 1,
		tasks:                 tasks,
		historyDoneByDay:      map[string][]domain.Task{},
		historyAbandonedByDay: map[string][]domain.Task{},
		statsRatios:           map[string]float64{},
	}
}

func (a *fakeApp) resetHistoryCounters() {
	a.historyDoneCalls = 0
	a.historyAbandonedCalls = 0
	a.statsCalls = 0
}

func (a *fakeApp) Add(ctx context.Context, title string) (domain.Task, error) {
	_ = ctx
	a.addedTitles = append(a.addedTitles, title)
	if a.addErr != nil {
		return domain.Task{}, a.addErr
	}
	created := a.currentDay
	t := domain.Task{ID: a.nextID, Title: title, Status: domain.StatusActive, CreatedDay: created, DueDay: created}
	a.nextID++
	a.tasks = append(a.tasks, t)
	return t, nil
}

func (a *fakeApp) Today(ctx context.Context) ([]domain.Task, error) {
	_ = ctx
	a.todayCalls++
	if a.todayErr != nil {
		return nil, a.todayErr
	}
	out := make([]domain.Task, 0)
	for _, t := range a.tasks {
		if t.Status != domain.StatusActive {
			continue
		}
		if !a.currentDay.Before(t.DueDay) {
			out = append(out, t)
		}
	}
	return out, nil
}

func (a *fakeApp) Upcoming(ctx context.Context) ([]domain.Task, error) {
	_ = ctx
	a.upcomingCalls++
	if a.upcomingErr != nil {
		return nil, a.upcomingErr
	}
	out := make([]domain.Task, 0)
	for _, t := range a.tasks {
		if t.Status != domain.StatusActive {
			continue
		}
		if a.currentDay.Before(t.DueDay) {
			out = append(out, t)
		}
	}
	return out, nil
}

func applyCmd(m Model, cmd tea.Cmd) Model {
	if cmd == nil {
		return m
	}
	msg := cmd()
	if msg == nil {
		return m
	}
	if bm, ok := msg.(tea.BatchMsg); ok {
		for _, subCmd := range bm {
			if subCmd == nil {
				continue
			}
			subMsg := subCmd()
			if subMsg == nil {
				continue
			}
			um, _ := m.Update(subMsg)
			m = um.(Model)
		}
		return m
	}
	um, _ := m.Update(msg)
	return um.(Model)
}

func (a *fakeApp) Done(ctx context.Context, id int64) error {
	_ = ctx
	a.doneIDs = append(a.doneIDs, id)
	if a.doneErr != nil {
		return a.doneErr
	}
	for i := range a.tasks {
		if a.tasks[i].ID == id {
			a.tasks[i].Status = domain.StatusDone
		}
	}
	return nil
}

func (a *fakeApp) Abandon(ctx context.Context, id int64) error {
	_ = ctx
	a.abandonedIDs = append(a.abandonedIDs, id)
	if a.abandonErr != nil {
		return a.abandonErr
	}
	for i := range a.tasks {
		if a.tasks[i].ID == id {
			a.tasks[i].Status = domain.StatusAbandoned
		}
	}
	return nil
}

func (a *fakeApp) PostponeOneDay(ctx context.Context, id int64) error {
	_ = ctx
	a.postponedIDs = append(a.postponedIDs, id)
	if a.postponeErr != nil {
		return a.postponeErr
	}
	next := domain.DayFromTime(a.currentDay.Time().AddDate(0, 0, 1))
	for i := range a.tasks {
		if a.tasks[i].ID == id {
			a.tasks[i].DueDay = next
		}
	}
	return nil
}

func (a *fakeApp) HistoryDoneByDay(ctx context.Context, day domain.Day) ([]domain.Task, error) {
	_ = ctx
	a.historyDoneCalls++
	a.lastHistoryDay = day
	if a.historyErr != nil {
		return nil, a.historyErr
	}
	return a.historyDoneByDay[day.String()], nil
}

func (a *fakeApp) HistoryAbandonedByDay(ctx context.Context, day domain.Day) ([]domain.Task, error) {
	_ = ctx
	a.historyAbandonedCalls++
	a.lastHistoryDay = day
	if a.historyErr != nil {
		return nil, a.historyErr
	}
	return a.historyAbandonedByDay[day.String()], nil
}

func (a *fakeApp) Stats(ctx context.Context, fromDay, toDay domain.Day) (app.OutcomeRatios, error) {
	_ = ctx
	a.statsCalls++
	a.lastStatsFrom = fromDay
	a.lastStatsTo = toDay
	if a.statsErr != nil {
		return app.OutcomeRatios{}, a.statsErr
	}
	return app.OutcomeRatios{
		DoneDelayedRatio:      a.statsRatios["done"],
		AbandonedDelayedRatio: a.statsRatios["abandoned"],
	}, nil
}

func keyRune(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func keyEnter() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyEnter}
}

func TestModel_Init_LoadsTodayList(t *testing.T) {
	orig := tickEvery
	tickEvery = 0
	t.Cleanup(func() { tickEvery = orig })

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, []domain.Task{{ID: 1, Title: "t1", Status: domain.StatusActive, CreatedDay: current, DueDay: current}})

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m = applyCmd(m, m.Init())

	if listLen(m.todayList) != 1 {
		t.Fatalf("expected 1 item in today list, got %d", listLen(m.todayList))
	}
}

func TestModel_Today_XMarksDoneAndRemovesFromToday(t *testing.T) {
	orig := tickEvery
	tickEvery = 0
	t.Cleanup(func() { tickEvery = orig })

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, []domain.Task{{ID: 1, Title: "t1", Status: domain.StatusActive, CreatedDay: current, DueDay: current}})

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m = applyCmd(m, m.Init())

	um, cmd := m.Update(keyRune('x'))
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected command from done action")
	}
	msg := cmd()
	if len(a.doneIDs) != 1 || a.doneIDs[0] != 1 {
		t.Fatalf("expected Done called with id=1, got %#v", a.doneIDs)
	}

	um, _ = m.Update(msg)
	m = um.(Model)
	if listLen(m.todayList) != 0 {
		t.Fatalf("expected task removed from today after done, got %d", listLen(m.todayList))
	}
}

func TestModel_Today_DMarksAbandonedAndRemovesFromToday(t *testing.T) {
	orig := tickEvery
	tickEvery = 0
	t.Cleanup(func() { tickEvery = orig })

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, []domain.Task{{ID: 2, Title: "t2", Status: domain.StatusActive, CreatedDay: current, DueDay: current}})

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m = applyCmd(m, m.Init())

	um, cmd := m.Update(keyRune('d'))
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected command from abandon action")
	}
	msg := cmd()
	if len(a.abandonedIDs) != 1 || a.abandonedIDs[0] != 2 {
		t.Fatalf("expected Abandon called with id=2, got %#v", a.abandonedIDs)
	}

	um, _ = m.Update(msg)
	m = um.(Model)
	if listLen(m.todayList) != 0 {
		t.Fatalf("expected task removed from today after abandon, got %d", listLen(m.todayList))
	}
}

func TestModel_Today_PPostponesAndMovesTaskToUpcoming(t *testing.T) {
	orig := tickEvery
	tickEvery = 0
	t.Cleanup(func() { tickEvery = orig })

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, []domain.Task{{ID: 3, Title: "t3", Status: domain.StatusActive, CreatedDay: current, DueDay: current}})

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m = applyCmd(m, m.Init())

	um, cmd := m.Update(keyRune('p'))
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected command from postpone action")
	}
	msg := cmd()
	if len(a.postponedIDs) != 1 || a.postponedIDs[0] != 3 {
		t.Fatalf("expected PostponeOneDay called with id=3, got %#v", a.postponedIDs)
	}

	um, _ = m.Update(msg)
	m = um.(Model)
	if listLen(m.todayList) != 0 {
		t.Fatalf("expected task removed from today after postpone, got %d", listLen(m.todayList))
	}

	um, cmd = m.Update(keyRune('2'))
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected command when switching to upcoming")
	}
	um, _ = m.Update(cmd())
	m = um.(Model)
	if listLen(m.upcomingList) != 1 {
		t.Fatalf("expected 1 item in upcoming after postpone, got %d", listLen(m.upcomingList))
	}
}

func TestModel_Today_AAddTaskPromptsAndAddsToList(t *testing.T) {
	orig := tickEvery
	tickEvery = 0
	t.Cleanup(func() { tickEvery = orig })

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, nil)

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m = applyCmd(m, m.Init())

	um, _ := m.Update(keyRune('a'))
	m = um.(Model)
	if !m.adding {
		t.Fatalf("expected to enter add mode")
	}

	m.addInput.SetValue("hello")
	um, cmd := m.Update(keyEnter())
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected command from add submit")
	}
	msg := cmd()
	if len(a.addedTitles) != 1 || a.addedTitles[0] != "hello" {
		t.Fatalf("expected Add called with title=hello, got %#v", a.addedTitles)
	}

	um, _ = m.Update(msg)
	m = um.(Model)
	if listLen(m.todayList) != 1 {
		t.Fatalf("expected 1 item in today after add, got %d", listLen(m.todayList))
	}
}

func TestModel_AddMode_QQuits(t *testing.T) {
	orig := tickEvery
	tickEvery = 0
	t.Cleanup(func() { tickEvery = orig })

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, nil)

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m = applyCmd(m, m.Init())

	um, _ := m.Update(keyRune('a'))
	m = um.(Model)
	if !m.adding {
		t.Fatalf("expected to enter add mode")
	}

	_, cmd := m.Update(keyRune('q'))
	if cmd == nil {
		t.Fatalf("expected quit command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg")
	}
}

func TestModel_ShowsStatusMessageOnActionError(t *testing.T) {
	orig := tickEvery
	tickEvery = 0
	t.Cleanup(func() { tickEvery = orig })

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, []domain.Task{{ID: 9, Title: "t9", Status: domain.StatusActive, CreatedDay: current, DueDay: current}})
	a.doneErr = errors.New("boom")

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m = applyCmd(m, m.Init())

	um, cmd := m.Update(keyRune('x'))
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected command")
	}
	um, _ = m.Update(cmd())
	m = um.(Model)
	if m.statusMsg == "" {
		t.Fatalf("expected status message to be set on error")
	}
	if listLen(m.todayList) != 1 {
		t.Fatalf("expected today list unchanged on error")
	}
}

func TestTodayDelegate_RendersDelayedTaskInRed(t *testing.T) {
	orig := tickEvery
	tickEvery = 0
	t.Cleanup(func() { tickEvery = orig })

	lipgloss.SetColorProfile(termenv.ANSI256)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	current := domain.MustParseDay("2026-03-04")
	delayed := domain.Task{ID: 10, Title: "late", Status: domain.StatusActive, CreatedDay: current, DueDay: domain.MustParseDay("2026-03-03")}

	s := defaultStyles()
	d := todayItemDelegate{styles: s, currentDay: current}

	l := list.New([]list.Item{taskItem{task: delayed}}, d, 80, 10)
	l.SetShowHelp(false)
	l.SetShowTitle(false)

	var buf bytes.Buffer
	d.Render(&buf, l, 0, taskItem{task: delayed})
	got := buf.String()

	red := regexp.MustCompile("\\x1b\\[[0-9;]*(31|91|38;5;9)[0-9;]*m")
	if !red.MatchString(got) {
		t.Fatalf("expected delayed task to include red ANSI color, got %q", got)
	}
}

func TestModel_View_ShowsSelectedRowWithDistinctHighlight(t *testing.T) {
	orig := tickEvery
	tickEvery = 0
	t.Cleanup(func() { tickEvery = orig })

	// Force ANSI output so we can assert selection styling deterministically.
	lipgloss.SetColorProfile(termenv.ANSI256)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	day := domain.MustParseDay("2026-03-04")
	a := newFakeApp(day, []domain.Task{
		{ID: 1, Title: "first", Status: domain.StatusActive, CreatedDay: day, DueDay: day},
		{ID: 2, Title: "second", Status: domain.StatusActive, CreatedDay: day, DueDay: day},
	})

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	um, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = um.(Model)
	m = applyCmd(m, m.Init())

	out := m.View()

	// Selected row should use a calm background highlight.
	bg := regexp.MustCompile("\\x1b\\[[0-9;]*48;[0-9;]*m")
	if !bg.MatchString(out) {
		t.Fatalf("expected View to include ANSI background highlight for selected row, got: %q", out)
	}
}
