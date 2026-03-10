package ui

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/JamesYuuu/tick/internal/app"
	"github.com/JamesYuuu/tick/internal/domain"
	tea "github.com/charmbracelet/bubbletea"
)

func disableTick(t interface{ Cleanup(func()) }) {
	orig := tickEvery
	tickEvery = 0
	t.Cleanup(func() { tickEvery = orig })
}

type fakeClock struct{ now time.Time }

func (c fakeClock) Now() time.Time { return c.now }

type fakeApp struct {
	currentDay domain.Day
	nextID     int64
	tasks      []domain.Task

	todayCalls    int
	upcomingCalls int

	historyDoneByDay          map[string][]domain.Task
	historyAbandonedByDay     map[string][]domain.Task
	historyActiveByCreatedDay map[string][]domain.Task
	historyDoneCalls          int
	historyAbandonedCalls     int
	historyActiveCreatedCalls int
	lastHistoryDay            domain.Day
	historyErr                error

	statsRatios   map[string]float64
	statsCalls    int
	lastStatsFrom domain.Day
	lastStatsTo   domain.Day
	statsErr      error

	addedTitles  []string
	editedTasks  []editedTaskCall
	deletedIDs   []int64
	doneIDs      []int64
	abandonedIDs []int64
	postponedIDs []int64

	todayErr    error
	upcomingErr error
	addErr      error
	editErr     error
	deleteErr   error
	doneErr     error
	abandonErr  error
	postponeErr error
}

type editedTaskCall struct {
	id    int64
	title string
}

type inputModalCase struct {
	name                   string
	open                   func(*Model, domain.Task)
	seed                   []domain.Task
	input                  string
	key                    tea.KeyMsg
	wantOpen               bool
	wantAdded              []string
	wantEdit               []editedTaskCall
	wantCmd                bool
	wantInput              string
	wantTodayLen           int
	wantSelectedTitle      string
	wantTodayCallsDelta    int
	wantUpcomingCallsDelta int
}

func newFakeApp(currentDay domain.Day, tasks []domain.Task) *fakeApp {
	maxID := int64(0)
	for _, task := range tasks {
		if task.ID > maxID {
			maxID = task.ID
		}
	}
	return &fakeApp{
		currentDay:                currentDay,
		nextID:                    maxID + 1,
		tasks:                     tasks,
		historyDoneByDay:          map[string][]domain.Task{},
		historyAbandonedByDay:     map[string][]domain.Task{},
		historyActiveByCreatedDay: map[string][]domain.Task{},
		statsRatios:               map[string]float64{},
	}
}

func (a *fakeApp) resetHistoryCounters() {
	a.historyDoneCalls = 0
	a.historyAbandonedCalls = 0
	a.historyActiveCreatedCalls = 0
	a.statsCalls = 0
}

func (a *fakeApp) Add(ctx context.Context, title string) (domain.Task, error) {
	_ = ctx
	a.addedTitles = append(a.addedTitles, title)
	if a.addErr != nil {
		return domain.Task{}, a.addErr
	}
	created := a.currentDay
	task := domain.Task{ID: a.nextID, Title: title, Status: domain.StatusActive, CreatedDay: created, DueDay: created}
	a.nextID++
	a.tasks = append(a.tasks, task)
	return task, nil
}

func (a *fakeApp) EditTitle(ctx context.Context, id int64, title string) error {
	_ = ctx
	a.editedTasks = append(a.editedTasks, editedTaskCall{id: id, title: title})
	if a.editErr != nil {
		return a.editErr
	}
	for i := range a.tasks {
		if a.tasks[i].ID == id {
			a.tasks[i].Title = title
		}
	}
	return nil
}

func (a *fakeApp) Delete(ctx context.Context, id int64) error {
	_ = ctx
	a.deletedIDs = append(a.deletedIDs, id)
	if a.deleteErr != nil {
		return a.deleteErr
	}
	for i := range a.tasks {
		if a.tasks[i].ID == id {
			a.tasks = append(a.tasks[:i], a.tasks[i+1:]...)
			break
		}
	}
	return nil
}

func (a *fakeApp) Today(ctx context.Context) ([]domain.Task, error) {
	_ = ctx
	a.todayCalls++
	if a.todayErr != nil {
		return nil, a.todayErr
	}
	out := make([]domain.Task, 0)
	for _, task := range a.tasks {
		if task.Status != domain.StatusActive {
			continue
		}
		if !a.currentDay.Before(task.DueDay) {
			out = append(out, task)
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
	for _, task := range a.tasks {
		if task.Status != domain.StatusActive {
			continue
		}
		if a.currentDay.Before(task.DueDay) {
			out = append(out, task)
		}
	}
	return out, nil
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

func (a *fakeApp) HistoryActiveByCreatedDay(ctx context.Context, day domain.Day) ([]domain.Task, error) {
	_ = ctx
	a.historyActiveCreatedCalls++
	a.lastHistoryDay = day
	if a.historyErr != nil {
		return nil, a.historyErr
	}
	return a.historyActiveByCreatedDay[day.String()], nil
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

func applyCmd(m Model, cmd tea.Cmd) Model {
	if cmd == nil {
		return m
	}
	msg := cmd()
	if msg == nil {
		return m
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, subCmd := range batch {
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

func execBatchCmds(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		out := make([]tea.Msg, 0, len(batch))
		for _, subCmd := range batch {
			if subCmd == nil {
				continue
			}
			out = append(out, subCmd())
		}
		return out
	}
	return []tea.Msg{msg}
}

func keyRune(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func keyTab() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyTab}
}

func keyEnter() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyEnter}
}

func keyEsc() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyEsc}
}

func runModalCase(t *testing.T, tc inputModalCase) {
	t.Helper()

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, tc.seed)
	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m = applyCmd(m, m.Init())

	todayCallsBefore := a.todayCalls
	upcomingCallsBefore := a.upcomingCalls

	var task domain.Task
	if len(tc.seed) > 0 {
		task = tc.seed[0]
	}
	tc.open(&m, task)
	m.addInput.SetValue(tc.input)

	um, cmd := m.Update(tc.key)
	m = um.(Model)

	if got := cmd != nil; got != tc.wantCmd {
		t.Fatalf("command presence mismatch: got %v want %v", got, tc.wantCmd)
	}
	if cmd != nil {
		um, _ = m.Update(cmd())
		m = um.(Model)
	}

	if got := m.modal.kind != modalKindNone; got != tc.wantOpen {
		t.Fatalf("modal open mismatch: got %v want %v", got, tc.wantOpen)
	}
	if !reflect.DeepEqual(a.addedTitles, tc.wantAdded) {
		t.Fatalf("added titles mismatch: got %#v want %#v", a.addedTitles, tc.wantAdded)
	}
	if !reflect.DeepEqual(a.editedTasks, tc.wantEdit) {
		t.Fatalf("edited tasks mismatch: got %#v want %#v", a.editedTasks, tc.wantEdit)
	}
	if m.addInput.Value() != tc.wantInput {
		t.Fatalf("input value mismatch: got %q want %q", m.addInput.Value(), tc.wantInput)
	}
	if tc.wantTodayLen >= 0 && listLen(m.todayList) != tc.wantTodayLen {
		t.Fatalf("today list length mismatch: got %d want %d", listLen(m.todayList), tc.wantTodayLen)
	}
	if tc.wantSelectedTitle != "" {
		selected, ok := m.todayList.SelectedItem().(taskItem)
		if !ok {
			t.Fatalf("expected selected today item")
		}
		if selected.task.Title != tc.wantSelectedTitle {
			t.Fatalf("selected title mismatch: got %q want %q", selected.task.Title, tc.wantSelectedTitle)
		}
	}
	if a.todayCalls != todayCallsBefore+tc.wantTodayCallsDelta {
		t.Fatalf("today refresh calls mismatch: got before=%d after=%d want delta=%d", todayCallsBefore, a.todayCalls, tc.wantTodayCallsDelta)
	}
	if a.upcomingCalls != upcomingCallsBefore+tc.wantUpcomingCallsDelta {
		t.Fatalf("upcoming refresh calls mismatch: got before=%d after=%d want delta=%d", upcomingCallsBefore, a.upcomingCalls, tc.wantUpcomingCallsDelta)
	}
}
