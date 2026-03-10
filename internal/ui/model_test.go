package ui

import (
	"bytes"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/JamesYuuu/tick/internal/domain"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

func TestSeparatorLine_LeavesOneColumnToAvoidWrap(t *testing.T) {
	w := 80
	cw := contentWidth(w)

	got := separatorLine(w)
	if got == "" {
		t.Fatalf("expected non-empty separatorLine(%d)", w)
	}
	// Global separators: keep left aligned, right -2 cells.
	want := strings.Repeat("-", cw-2)
	if got != want {
		t.Fatalf("expected separatorLine(%d)=%q, got %q", w, want, got)
	}
}

func TestLayoutMetrics_Consistency(t *testing.T) {
	g := calcLayoutMetrics(80, 24)
	if g.contentW != contentWidth(80) {
		t.Fatalf("content width mismatch: got %d want %d", g.contentW, contentWidth(80))
	}
	if g.innerW != sheetInnerWidth(80) {
		t.Fatalf("inner width mismatch: got %d want %d", g.innerW, sheetInnerWidth(80))
	}
	if g.workspaceH != 19 { // 24 - (header1 + sep2 + help2)
		t.Fatalf("workspace height mismatch: got %d want 19", g.workspaceH)
	}
}

func TestLayoutMetrics_ClampAtSmallWindowSizes(t *testing.T) {
	g := calcLayoutMetrics(0, 0)
	if g.contentW != 0 {
		t.Fatalf("expected content width clamp at 0, got %d", g.contentW)
	}
	if g.innerW != 0 {
		t.Fatalf("expected inner width clamp at 0, got %d", g.innerW)
	}
	if g.workspaceH != 0 {
		t.Fatalf("expected workspace height clamp at 0, got %d", g.workspaceH)
	}
	if g.innerH != 0 {
		t.Fatalf("expected inner height clamp at 0, got %d", g.innerH)
	}

	g = calcLayoutMetrics(3, 4)
	if g.innerW != 0 {
		t.Fatalf("expected inner width clamp at 0 for narrow width, got %d", g.innerW)
	}
	if g.workspaceH != 0 || g.innerH != 0 {
		t.Fatalf("expected height clamps at 0 for short window, got workspace=%d inner=%d", g.workspaceH, g.innerH)
	}
}

func TestModel_View_UsesSameSizingAsWindowSizeUpdate(t *testing.T) {
	day := domain.MustParseDay("2026-03-04")
	a := newFakeApp(day, nil)

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	const w = 80
	const h = 24
	um, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = um.(Model)

	g := calcLayoutMetrics(w, h)
	if m.todayList.Width() != g.innerW || m.todayList.Height() != g.innerH {
		t.Fatalf("expected today list size from Update to be %dx%d, got %dx%d", g.innerW, g.innerH, m.todayList.Width(), m.todayList.Height())
	}
	if m.upcomingList.Width() != g.innerW || m.upcomingList.Height() != g.innerH {
		t.Fatalf("expected upcoming list size from Update to be %dx%d, got %dx%d", g.innerW, g.innerH, m.upcomingList.Width(), m.upcomingList.Height())
	}
	if m.addInput.Width != g.innerW {
		t.Fatalf("expected add input width from Update to be %d, got %d", g.innerW, m.addInput.Width)
	}

	_ = m.View()

	if m.todayList.Width() != g.innerW || m.todayList.Height() != g.innerH {
		t.Fatalf("expected today list size from View to match Update path (%dx%d), got %dx%d", g.innerW, g.innerH, m.todayList.Width(), m.todayList.Height())
	}
	if m.upcomingList.Width() != g.innerW || m.upcomingList.Height() != g.innerH {
		t.Fatalf("expected upcoming list size from View to match Update path (%dx%d), got %dx%d", g.innerW, g.innerH, m.upcomingList.Width(), m.upcomingList.Height())
	}
	if m.addInput.Width != g.innerW {
		t.Fatalf("expected add input width from View to match Update path (%d), got %d", g.innerW, m.addInput.Width)
	}
}

func TestRenderEmptyStates_Table(t *testing.T) {
	disableTick(t)

	day := domain.MustParseDay("2026-03-04")
	tests := []struct {
		name    string
		render  func(Model) string
		wantMsg string
	}{
		{name: "today", render: renderTodayBody, wantMsg: "Nothing due today."},
		{name: "upcoming", render: renderUpcomingBody, wantMsg: "No upcoming tasks."},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := NewWithDeps(newFakeApp(day, nil), fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
			um, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
			m = um.(Model)

			body := tc.render(m)
			lines := strings.Split(strings.TrimRight(body, "\n"), "\n")
			innerH := calcLayoutMetrics(m.width, m.height).innerH
			innerW := sheetInnerWidth(m.width)
			if len(lines) != innerH {
				t.Fatalf("expected body to have %d lines, got %d", innerH, len(lines))
			}

			topPad := (innerH - 1) / 2
			if topPad < 0 {
				topPad = 0
			}
			if !strings.Contains(lines[topPad], tc.wantMsg) {
				t.Fatalf("expected %q on centered line %d, got %q", tc.wantMsg, topPad, lines[topPad])
			}
			leftPad := (innerW - ansi.StringWidth(tc.wantMsg)) / 2
			if leftPad < 0 {
				leftPad = 0
			}
			if got := len(lines[topPad]) - len(strings.TrimLeft(lines[topPad], " ")); got != leftPad {
				t.Fatalf("expected left padding %d, got %d (line=%q)", leftPad, got, lines[topPad])
			}
		})
	}
}

func TestModel_View_EmptyStateCopyAcrossTabs(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	day := domain.MustParseDay("2026-03-04")
	a := newFakeApp(day, nil)

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	um, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = um.(Model)
	m = applyCmd(m, m.Init())

	out := m.View()
	if !strings.Contains(out, "Nothing due today.") {
		t.Fatalf("expected today empty copy in Today view, got %q", out)
	}

	um, cmd := m.Update(keyTab())
	m = um.(Model)
	m = applyCmd(m, cmd)

	out = m.View()
	if !strings.Contains(out, "No upcoming tasks.") {
		t.Fatalf("expected upcoming empty copy in Upcoming view, got %q", out)
	}
}

func TestModel_View_Smoke_AllViews_RenderNonEmpty(t *testing.T) {
	m := NewWithDeps(newFakeApp(domain.MustParseDay("2026-03-04"), nil), fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	um, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = um.(Model)

	if strings.TrimSpace(m.View()) == "" {
		t.Fatalf("today view empty")
	}

	m.view = viewUpcoming
	if strings.TrimSpace(m.View()) == "" {
		t.Fatalf("upcoming view empty")
	}

	m.view = viewHistory
	if strings.TrimSpace(m.View()) == "" {
		t.Fatalf("history view empty")
	}
}

func TestRenderCenteredEmpty_UsesLayoutInnerBox(t *testing.T) {
	m := NewWithDeps(newFakeApp(domain.MustParseDay("2026-03-04"), nil), fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	um, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = um.(Model)

	out := renderCenteredEmpty(m, "Nothing due today.")
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	innerH := calcLayoutMetrics(m.width, m.height).innerH
	if len(lines) != innerH {
		t.Fatalf("expected centered empty to use innerH=%d, got %d", innerH, len(lines))
	}
}

func listLen(m list.Model) int { return len(m.Items()) }

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

func TestModel_TodayActions_Table(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	tests := []struct {
		name          string
		key           tea.KeyMsg
		id            int64
		wantDone      []int64
		wantAbandoned []int64
		wantPostponed []int64
		wantUpcoming  int
	}{
		{name: "done", key: keyRune('x'), id: 1, wantDone: []int64{1}},
		{name: "abandon", key: keyRune('b'), id: 2, wantAbandoned: []int64{2}},
		{name: "postpone", key: keyRune('p'), id: 3, wantPostponed: []int64{3}, wantUpcoming: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := newFakeApp(current, []domain.Task{{ID: tc.id, Title: "task", Status: domain.StatusActive, CreatedDay: current, DueDay: current}})
			m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
			m = applyCmd(m, m.Init())

			um, cmd := m.Update(tc.key)
			m = um.(Model)
			if cmd == nil {
				t.Fatalf("expected command from action")
			}
			um, _ = m.Update(cmd())
			m = um.(Model)

			if !equalInt64s(a.doneIDs, tc.wantDone) {
				t.Fatalf("done ids mismatch: got %#v want %#v", a.doneIDs, tc.wantDone)
			}
			if !equalInt64s(a.abandonedIDs, tc.wantAbandoned) {
				t.Fatalf("abandoned ids mismatch: got %#v want %#v", a.abandonedIDs, tc.wantAbandoned)
			}
			if !equalInt64s(a.postponedIDs, tc.wantPostponed) {
				t.Fatalf("postponed ids mismatch: got %#v want %#v", a.postponedIDs, tc.wantPostponed)
			}
			if listLen(m.todayList) != 0 {
				t.Fatalf("expected today list cleared after action, got %d", listLen(m.todayList))
			}

			if tc.wantUpcoming > 0 {
				um, cmd = m.Update(keyTab())
				m = um.(Model)
				m = applyCmd(m, cmd)
				if listLen(m.upcomingList) != tc.wantUpcoming {
					t.Fatalf("expected upcoming list size %d, got %d", tc.wantUpcoming, listLen(m.upcomingList))
				}
			}
		})
	}
}

func equalInt64s(got, want []int64) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestModel_Today_AddOpensAddModal(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, nil)

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m = applyCmd(m, m.Init())

	um, _ := m.Update(keyRune('a'))
	m = um.(Model)

	if m.modal.kind != modalKindAdd {
		t.Fatalf("expected add modal, got %v", m.modal.kind)
	}
	if m.modal.taskID != 0 {
		t.Fatalf("expected add modal task id to be empty, got %d", m.modal.taskID)
	}
	if m.modal.taskTitle != "" {
		t.Fatalf("expected add modal task title to be empty, got %q", m.modal.taskTitle)
	}
	if m.addInput.Value() != "" {
		t.Fatalf("expected add modal input to start empty, got %q", m.addInput.Value())
	}
}

func TestModel_Upcoming_EditOpensEditModalWithPrefilledTitle(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	upcoming := domain.Task{ID: 7, Title: "write tests", Status: domain.StatusActive, CreatedDay: current, DueDay: addDays(current, 1)}
	a := newFakeApp(current, []domain.Task{upcoming})

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m = applyCmd(m, m.Init())
	um, cmd := m.Update(keyTab())
	m = um.(Model)
	m = applyCmd(m, cmd)

	um, _ = m.Update(keyRune('e'))
	m = um.(Model)

	if m.modal.kind != modalKindEdit {
		t.Fatalf("expected edit modal, got %v", m.modal.kind)
	}
	if m.modal.taskID != upcoming.ID {
		t.Fatalf("expected edit modal task id %d, got %d", upcoming.ID, m.modal.taskID)
	}
	if m.modal.taskTitle != upcoming.Title {
		t.Fatalf("expected edit modal task title %q, got %q", upcoming.Title, m.modal.taskTitle)
	}
	if m.addInput.Value() != upcoming.Title {
		t.Fatalf("expected edit input prefilled with %q, got %q", upcoming.Title, m.addInput.Value())
	}
}

func TestModel_History_EditDoesNothing(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, nil)

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m = applyCmd(m, m.Init())
	um, cmd := m.Update(keyTab())
	m = um.(Model)
	m = applyCmd(m, cmd)
	um, cmd = m.Update(keyTab())
	m = um.(Model)
	m = applyCmd(m, cmd)

	um, nextCmd := m.Update(keyRune('e'))
	m = um.(Model)

	if m.view != viewHistory {
		t.Fatalf("expected history view, got %v", m.view)
	}
	if m.modal.kind != modalKindNone {
		t.Fatalf("expected no modal in history view, got %v", m.modal.kind)
	}
	if nextCmd != nil {
		t.Fatalf("expected no command when editing in history view")
	}
}

func TestModel_Upcoming_DeleteOpensDeleteModal(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	upcoming := domain.Task{ID: 11, Title: "archive docs", Status: domain.StatusActive, CreatedDay: current, DueDay: addDays(current, 2)}
	a := newFakeApp(current, []domain.Task{upcoming})

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m = applyCmd(m, m.Init())
	um, cmd := m.Update(keyTab())
	m = um.(Model)
	m = applyCmd(m, cmd)

	um, _ = m.Update(keyRune('d'))
	m = um.(Model)

	if m.modal.kind != modalKindDelete {
		t.Fatalf("expected delete modal, got %v", m.modal.kind)
	}
	if m.modal.taskID != upcoming.ID {
		t.Fatalf("expected delete modal task id %d, got %d", upcoming.ID, m.modal.taskID)
	}
	if m.modal.taskTitle != upcoming.Title {
		t.Fatalf("expected delete modal task title %q, got %q", upcoming.Title, m.modal.taskTitle)
	}
	if m.addInput.Value() != "" {
		t.Fatalf("expected delete modal to leave text input empty, got %q", m.addInput.Value())
	}
}

func TestModel_AddModal_EnterCreatesTask(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, nil)

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m = applyCmd(m, m.Init())
	todayCallsBefore := a.todayCalls
	upcomingCallsBefore := a.upcomingCalls

	um, _ := m.Update(keyRune('a'))
	m = um.(Model)
	m.addInput.SetValue("new task")
	um, cmd := m.Update(keyEnter())
	m = um.(Model)

	if cmd == nil {
		t.Fatalf("expected add submit command")
	}
	um, _ = m.Update(cmd())
	m = um.(Model)

	if len(a.addedTitles) != 1 || a.addedTitles[0] != "new task" {
		t.Fatalf("expected add call with title %q, got %#v", "new task", a.addedTitles)
	}
	if m.modal.kind != modalKindNone {
		t.Fatalf("expected add modal to close after submit, got %v", m.modal.kind)
	}
	if listLen(m.todayList) != 1 {
		t.Fatalf("expected today list refreshed with new task, got %d items", listLen(m.todayList))
	}
	if a.todayCalls != todayCallsBefore+1 {
		t.Fatalf("expected one refresh today call after add, got before=%d after=%d", todayCallsBefore, a.todayCalls)
	}
	if a.upcomingCalls != upcomingCallsBefore+1 {
		t.Fatalf("expected one refresh upcoming call after add, got before=%d after=%d", upcomingCallsBefore, a.upcomingCalls)
	}
}

func TestModel_AddModal_ClosesWhenRefreshFailsAfterSuccessfulAdd(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, nil)

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m = applyCmd(m, m.Init())
	a.todayErr = errors.New("refresh boom")

	um, _ := m.Update(keyRune('a'))
	m = um.(Model)
	m.addInput.SetValue("new task")
	um, cmd := m.Update(keyEnter())
	m = um.(Model)

	if cmd == nil {
		t.Fatalf("expected add submit command")
	}
	um, _ = m.Update(cmd())
	m = um.(Model)

	if len(a.addedTitles) != 1 || a.addedTitles[0] != "new task" {
		t.Fatalf("expected add call with title %q, got %#v", "new task", a.addedTitles)
	}
	if m.modal.kind != modalKindNone {
		t.Fatalf("expected add modal to close after successful add even when refresh fails, got %v", m.modal.kind)
	}
	if m.statusMsg != "today: refresh boom" {
		t.Fatalf("expected refresh error status after successful add, got %q", m.statusMsg)
	}
	if listLen(m.todayList) != 0 {
		t.Fatalf("expected list to remain stale when refresh fails, got %d items", listLen(m.todayList))
	}

	um, followupCmd := m.Update(keyEnter())
	m = um.(Model)
	if followupCmd != nil {
		t.Fatalf("expected no follow-up submit command after modal closes")
	}
	if len(a.addedTitles) != 1 {
		t.Fatalf("expected no duplicate add after refresh failure, got %#v", a.addedTitles)
	}
}

func TestModel_AddModal_EscCancels(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, nil)

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m = applyCmd(m, m.Init())

	um, _ := m.Update(keyRune('a'))
	m = um.(Model)
	um, _ = m.Update(keyRune('x'))
	m = um.(Model)
	um, cmd := m.Update(keyEsc())
	m = um.(Model)

	if cmd != nil {
		t.Fatalf("expected no command when canceling add modal")
	}
	if len(a.addedTitles) != 0 {
		t.Fatalf("expected add modal cancel to skip add, got %#v", a.addedTitles)
	}
	if m.modal.kind != modalKindNone {
		t.Fatalf("expected add modal closed after esc, got %v", m.modal.kind)
	}
	if m.addInput.Value() != "" {
		t.Fatalf("expected add input cleared after esc, got %q", m.addInput.Value())
	}
}

func TestModel_AddModal_EnterWithEmptyTitleDoesNothing(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, nil)

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m = applyCmd(m, m.Init())

	um, _ := m.Update(keyRune('a'))
	m = um.(Model)
	m.addInput.SetValue("   ")
	um, cmd := m.Update(keyEnter())
	m = um.(Model)

	if cmd != nil {
		t.Fatalf("expected no command for empty add title")
	}
	if len(a.addedTitles) != 0 {
		t.Fatalf("expected no add call for empty title, got %#v", a.addedTitles)
	}
	if m.modal.kind != modalKindAdd {
		t.Fatalf("expected add modal to stay open for empty title, got %v", m.modal.kind)
	}
}

func TestModel_AddModal_AllowsBoundRunesAsInput(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, nil)

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m = applyCmd(m, m.Init())

	um, _ := m.Update(keyRune('a'))
	m = um.(Model)

	letters := "abdepqx"
	for _, r := range letters {
		um, _ = m.Update(keyRune(r))
		m = um.(Model)
	}

	if m.addInput.Value() != letters {
		t.Fatalf("expected add modal to accept bound runes as input %q, got %q", letters, m.addInput.Value())
	}
	if m.modal.kind != modalKindAdd {
		t.Fatalf("expected add modal to remain open, got %v", m.modal.kind)
	}
}

func TestModel_EditModal_EnterUpdatesTitle(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	task := domain.Task{ID: 7, Title: "write tests", Status: domain.StatusActive, CreatedDay: current, DueDay: current}
	a := newFakeApp(current, []domain.Task{task})

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m = applyCmd(m, m.Init())
	todayCallsBefore := a.todayCalls
	upcomingCallsBefore := a.upcomingCalls

	um, _ := m.Update(keyRune('e'))
	m = um.(Model)
	m.addInput.SetValue("beta")
	um, cmd := m.Update(keyEnter())
	m = um.(Model)

	if cmd == nil {
		t.Fatalf("expected edit submit command")
	}
	um, _ = m.Update(cmd())
	m = um.(Model)

	if len(a.editedTasks) != 1 || a.editedTasks[0] != (editedTaskCall{id: task.ID, title: "beta"}) {
		t.Fatalf("expected edit call %#v, got %#v", editedTaskCall{id: task.ID, title: "beta"}, a.editedTasks)
	}
	if m.modal.kind != modalKindNone {
		t.Fatalf("expected edit modal to close after submit, got %v", m.modal.kind)
	}
	selected, ok := m.todayList.SelectedItem().(taskItem)
	if !ok {
		t.Fatalf("expected selected today item after edit")
	}
	if selected.task.Title != "beta" {
		t.Fatalf("expected refreshed title %q, got %q", "beta", selected.task.Title)
	}
	if a.todayCalls != todayCallsBefore+1 {
		t.Fatalf("expected one refresh today call after edit, got before=%d after=%d", todayCallsBefore, a.todayCalls)
	}
	if a.upcomingCalls != upcomingCallsBefore+1 {
		t.Fatalf("expected one refresh upcoming call after edit, got before=%d after=%d", upcomingCallsBefore, a.upcomingCalls)
	}
}

func TestModel_EditModal_EscCancels(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	task := domain.Task{ID: 7, Title: "write tests", Status: domain.StatusActive, CreatedDay: current, DueDay: current}
	a := newFakeApp(current, []domain.Task{task})

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m = applyCmd(m, m.Init())

	um, _ := m.Update(keyRune('e'))
	m = um.(Model)
	um, _ = m.Update(keyRune('x'))
	m = um.(Model)
	um, cmd := m.Update(keyEsc())
	m = um.(Model)

	if cmd != nil {
		t.Fatalf("expected no command when canceling edit modal")
	}
	if len(a.editedTasks) != 0 {
		t.Fatalf("expected edit modal cancel to skip edit, got %#v", a.editedTasks)
	}
	if m.modal.kind != modalKindNone {
		t.Fatalf("expected edit modal closed after esc, got %v", m.modal.kind)
	}
	if m.addInput.Value() != "" {
		t.Fatalf("expected edit input cleared after esc, got %q", m.addInput.Value())
	}
}

func TestModel_EditModal_EnterWithEmptyTitleDoesNothing(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	task := domain.Task{ID: 7, Title: "write tests", Status: domain.StatusActive, CreatedDay: current, DueDay: current}
	a := newFakeApp(current, []domain.Task{task})

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m = applyCmd(m, m.Init())

	um, _ := m.Update(keyRune('e'))
	m = um.(Model)
	m.addInput.SetValue("")
	um, cmd := m.Update(keyEnter())
	m = um.(Model)

	if cmd != nil {
		t.Fatalf("expected no command for empty edit title")
	}
	if len(a.editedTasks) != 0 {
		t.Fatalf("expected no edit call for empty title, got %#v", a.editedTasks)
	}
	if m.modal.kind != modalKindEdit {
		t.Fatalf("expected edit modal to stay open for empty title, got %v", m.modal.kind)
	}
}

func TestModel_EditModal_BackspaceEditsInput(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	task := domain.Task{ID: 7, Title: "write tests", Status: domain.StatusActive, CreatedDay: current, DueDay: current}
	a := newFakeApp(current, []domain.Task{task})

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m = applyCmd(m, m.Init())

	um, _ := m.Update(keyRune('e'))
	m = um.(Model)
	before := m.addInput.Value()
	if before == "" {
		t.Fatalf("expected prefilled edit input")
	}

	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = um.(Model)
	after := m.addInput.Value()
	if after != before[:len(before)-1] {
		t.Fatalf("expected backspace to edit input from %q to %q, got %q", before, before[:len(before)-1], after)
	}
	if m.modal.kind != modalKindEdit {
		t.Fatalf("expected edit modal to remain open, got %v", m.modal.kind)
	}
}

func TestModel_DeleteModal_YConfirmsDelete(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	task := domain.Task{ID: 7, Title: "write tests", Status: domain.StatusActive, CreatedDay: current, DueDay: addDays(current, 1)}
	a := newFakeApp(current, []domain.Task{task})

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m = applyCmd(m, m.Init())
	um, cmd := m.Update(keyTab())
	m = um.(Model)
	m = applyCmd(m, cmd)
	upcomingCallsBefore := a.upcomingCalls

	um, _ = m.Update(keyRune('d'))
	m = um.(Model)
	um, cmd = m.Update(keyRune('y'))
	m = um.(Model)

	if cmd == nil {
		t.Fatalf("expected delete confirm command")
	}
	if m.modal.submitting != true {
		t.Fatalf("expected delete modal to enter submitting state")
	}

	um, _ = m.Update(cmd())
	m = um.(Model)

	if len(a.deletedIDs) != 1 || a.deletedIDs[0] != task.ID {
		t.Fatalf("expected delete call for id %d, got %#v", task.ID, a.deletedIDs)
	}
	if m.modal.kind != modalKindNone {
		t.Fatalf("expected delete modal to close after confirm, got %v", m.modal.kind)
	}
	if m.view != viewUpcoming {
		t.Fatalf("expected to remain in upcoming view, got %v", m.view)
	}
	if listLen(m.upcomingList) != 0 {
		t.Fatalf("expected upcoming list refreshed after delete, got %d items", listLen(m.upcomingList))
	}
	if a.upcomingCalls != upcomingCallsBefore+1 {
		t.Fatalf("expected one refresh upcoming call after delete, got before=%d after=%d", upcomingCallsBefore, a.upcomingCalls)
	}
}

func TestModel_DeleteModal_NCancels(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	task := domain.Task{ID: 7, Title: "write tests", Status: domain.StatusActive, CreatedDay: current, DueDay: current}
	a := newFakeApp(current, []domain.Task{task})

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m = applyCmd(m, m.Init())

	um, _ := m.Update(keyRune('d'))
	m = um.(Model)
	um, cmd := m.Update(keyRune('n'))
	m = um.(Model)

	if cmd != nil {
		t.Fatalf("expected no command when canceling delete modal")
	}
	if len(a.deletedIDs) != 0 {
		t.Fatalf("expected delete modal cancel to skip delete, got %#v", a.deletedIDs)
	}
	if m.modal.kind != modalKindNone {
		t.Fatalf("expected delete modal closed after n, got %v", m.modal.kind)
	}
	if listLen(m.todayList) != 1 {
		t.Fatalf("expected today list unchanged after cancel, got %d items", listLen(m.todayList))
	}
}

func TestModel_DeleteModal_EscCancels(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	task := domain.Task{ID: 7, Title: "write tests", Status: domain.StatusActive, CreatedDay: current, DueDay: current}
	a := newFakeApp(current, []domain.Task{task})

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m = applyCmd(m, m.Init())

	um, _ := m.Update(keyRune('d'))
	m = um.(Model)
	um, cmd := m.Update(keyEsc())
	m = um.(Model)

	if cmd != nil {
		t.Fatalf("expected no command when canceling delete modal with esc")
	}
	if len(a.deletedIDs) != 0 {
		t.Fatalf("expected delete modal esc to skip delete, got %#v", a.deletedIDs)
	}
	if m.modal.kind != modalKindNone {
		t.Fatalf("expected delete modal closed after esc, got %v", m.modal.kind)
	}
	if listLen(m.todayList) != 1 {
		t.Fatalf("expected today list unchanged after esc, got %d items", listLen(m.todayList))
	}
}

func TestModel_View_RendersCenteredAddModal(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	m := NewWithDeps(newFakeApp(current, nil), fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	const w = 80
	const h = 24
	um, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = um.(Model)
	m = applyCmd(m, m.Init())

	um, _ = m.Update(keyRune('a'))
	m = um.(Model)

	out := m.View()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	var modalLine string
	for _, line := range lines {
		if strings.Contains(line, "| Add task") {
			modalLine = line
			break
		}
	}
	if modalLine == "" {
		t.Fatalf("expected centered add modal header in view, got %q", out)
	}

	leftPad := strings.Index(modalLine, "| Add task")
	if leftPad < 18 {
		t.Fatalf("expected add modal to be visibly centered, got left pad %d in line %q", leftPad, modalLine)
	}
	if !strings.Contains(out, "| >") {
		t.Fatalf("expected add modal input row in view, got %q", out)
	}
	if !strings.Contains(out, "enter:save  esc:cancel") {
		t.Fatalf("expected lowercase add modal actions in view, got %q", out)
	}
}

func TestModel_View_RendersDeleteModalCopy(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	task := domain.Task{ID: 7, Title: "write tests", Status: domain.StatusActive, CreatedDay: current, DueDay: current}
	m := NewWithDeps(newFakeApp(current, []domain.Task{task}), fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	const w = 80
	const h = 24
	um, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = um.(Model)
	m = applyCmd(m, m.Init())

	um, _ = m.Update(keyRune('d'))
	m = um.(Model)

	out := m.View()
	for _, want := range []string{"Delete task?", "write tests", "y:delete  n:cancel"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected delete modal copy %q in view, got %q", want, out)
		}
	}
}

func TestModel_Help_IncludesEditDeleteKeys(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	current := domain.MustParseDay("2026-03-04")
	m := NewWithDeps(newFakeApp(current, nil), fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)

	todayHelp := m.help()
	for _, want := range []string{"e:Edit", "d:Delete"} {
		if !strings.Contains(todayHelp, strings.ToLower(want)) {
			t.Fatalf("expected today help to include %q, got %q", want, todayHelp)
		}
	}

	m.view = viewUpcoming
	upcomingHelp := m.help()
	for _, want := range []string{"e:Edit", "d:Delete"} {
		if !strings.Contains(upcomingHelp, strings.ToLower(want)) {
			t.Fatalf("expected upcoming help to include %q, got %q", want, upcomingHelp)
		}
	}
}

func TestModel_Help_UsesLowercaseLabelsAndStyledKeys(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI256)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	m := NewWithDeps(newFakeApp(domain.MustParseDay("2026-03-04"), nil), fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)

	todayHelp := m.help()
	if strings.Contains(todayHelp, "+1 day") {
		t.Fatalf("expected old postpone wording to be gone, got %q", todayHelp)
	}
	if strings.Contains(todayHelp, "plus one day") {
		t.Fatalf("expected one-word postpone wording, got %q", todayHelp)
	}
	if !strings.Contains(todayHelp, "postpone") {
		t.Fatalf("expected postpone wording, got %q", todayHelp)
	}
	if !regexp.MustCompile(`\x1b\[[0-9;]*1[0-9;]*m[a-z/:]+:\x1b\[[0-9;]*m\x1b\[[0-9;]*m[a-z ]+`).MatchString(todayHelp) {
		t.Fatalf("expected help output to visually emphasize full key token including colon, got %q", todayHelp)
	}

	m.view = viewHistory
	historyHelp := m.help()
	if !strings.Contains(historyHelp, "prev day") || !strings.Contains(historyHelp, "next day") {
		t.Fatalf("expected lowercase nav wording, got %q", historyHelp)
	}
	if strings.Contains(historyHelp, "Prev day") || strings.Contains(historyHelp, "Next day") {
		t.Fatalf("expected nav wording to stay lowercase, got %q", historyHelp)
	}
}

func TestModel_Help_UsesReadableFooterToneWithoutPaleGray242(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI256)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	m := NewWithDeps(newFakeApp(domain.MustParseDay("2026-03-04"), nil), fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)

	help := m.help()
	if regexp.MustCompile(`\x1b\[[0-9;]*38;5;242[0-9;]*m`).MatchString(help) {
		t.Fatalf("expected footer help to avoid pale gray 242 styling, got %q", help)
	}
	if !regexp.MustCompile(`\x1b\[[0-9;]*1[0-9;]*m[a-z/:]+:`).MatchString(help) {
		t.Fatalf("expected footer help keys to remain emphasized, got %q", help)
	}
}

func TestModel_Help_UsesTwoGroupedLines(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	current := domain.MustParseDay("2026-03-04")
	m := NewWithDeps(newFakeApp(current, nil), fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)

	tests := []struct {
		name        string
		view        view
		wantLine1   []string
		wantLine2   []string
		forbidLine1 []string
		forbidLine2 []string
	}{
		{
			name:        "today",
			view:        viewToday,
			wantLine1:   []string{"a:add", "e:edit", "d:delete", "x:done", "b:abandon", "p:postpone"},
			wantLine2:   []string{"tab:next", "q:quit"},
			forbidLine1: []string{"tab:next", "q:quit"},
		},
		{
			name:        "upcoming",
			view:        viewUpcoming,
			wantLine1:   []string{"e:edit", "d:delete"},
			wantLine2:   []string{"tab:next", "q:quit"},
			forbidLine1: []string{"tab:next", "q:quit"},
		},
		{
			name:        "history",
			view:        viewHistory,
			wantLine1:   []string{"left/h:prev day", "right/l:next day", "up/k:scroll up", "down/j:scroll down"},
			wantLine2:   []string{"tab:next", "q:quit"},
			forbidLine1: []string{"tab:next", "q:quit"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m.view = tc.view
			help := m.help()
			lines := strings.Split(help, "\n")
			if len(lines) != 2 {
				t.Fatalf("expected two help lines, got %d in %q", len(lines), help)
			}

			for _, want := range tc.wantLine1 {
				if !strings.Contains(lines[0], want) {
					t.Fatalf("expected first help line to include %q, got %q", want, lines[0])
				}
			}
			for _, want := range tc.wantLine2 {
				if !strings.Contains(lines[1], want) {
					t.Fatalf("expected second help line to include %q, got %q", want, lines[1])
				}
			}
			for _, forbid := range tc.forbidLine1 {
				if strings.Contains(lines[0], forbid) {
					t.Fatalf("expected first help line to exclude %q, got %q", forbid, lines[0])
				}
			}
			for _, forbid := range tc.forbidLine2 {
				if strings.Contains(lines[1], forbid) {
					t.Fatalf("expected second help line to exclude %q, got %q", forbid, lines[1])
				}
			}
		})
	}
}

func TestModel_Help_TodayActionsRightAlignCompletionGroup(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	current := domain.MustParseDay("2026-03-04")
	m := NewWithDeps(newFakeApp(current, nil), fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)

	const w = 80
	const h = 24
	um, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = um.(Model)
	m = applyCmd(m, m.Init())

	out := m.View()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != h {
		t.Fatalf("expected View to return exactly %d lines, got %d", h, len(lines))
	}

	line1 := lines[len(lines)-2]
	left := "a:add  e:edit  d:delete"
	right := "x:done  b:abandon  p:postpone"

	if !strings.HasPrefix(line1, left) {
		t.Fatalf("expected first footer line to start with %q, got %q", left, line1)
	}
	rightStart := strings.Index(line1, right)
	if rightStart == -1 {
		t.Fatalf("expected first footer line to include right action group %q, got %q", right, line1)
	}

	wantRightStart := contentWidth(w) - len(right)
	if rightStart != wantRightStart {
		t.Fatalf("expected right action group to start at column %d, got %d in %q", wantRightStart, rightStart, line1)
	}

	between := line1[len(left):rightStart]
	if strings.Trim(between, " ") != "" {
		t.Fatalf("expected only padding between Today action groups, got %q in %q", between, line1)
	}
	if len(between) < 2 {
		t.Fatalf("expected visible separation between Today action groups, got %d spaces in %q", len(between), line1)
	}
}

func TestModel_Help_HistoryScrollActionsRightAlignGroup(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	current := domain.MustParseDay("2026-03-04")
	m := NewWithDeps(newFakeApp(current, nil), fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)

	const w = 80
	const h = 24
	um, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = um.(Model)
	m.view = viewHistory

	help := m.help()
	lines := strings.Split(help, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected two help lines, got %d in %q", len(lines), help)
	}
	line1 := lines[0]
	left := "left/h:prev day  right/l:next day"
	right := "up/k:scroll up  down/j:scroll down"
	if !strings.HasPrefix(line1, left) {
		t.Fatalf("expected history first help line to start with %q, got %q", left, line1)
	}
	rightStart := strings.Index(line1, right)
	if rightStart == -1 {
		t.Fatalf("expected history first help line to include right group %q, got %q", right, line1)
	}
	wantRightStart := contentWidth(w) - len(right)
	if rightStart != wantRightStart {
		t.Fatalf("expected history right action group to start at column %d, got %d in %q", wantRightStart, rightStart, line1)
	}
}

func TestModel_TaskModal_IgnoresDuplicateSubmitWhileInFlight(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	baseClock := fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}

	t.Run("add", func(t *testing.T) {
		a := newFakeApp(current, nil)

		m := NewWithDeps(a, baseClock, time.UTC)
		m = applyCmd(m, m.Init())

		um, _ := m.Update(keyRune('a'))
		m = um.(Model)
		m.addInput.SetValue("new task")

		um, firstCmd := m.Update(keyEnter())
		m = um.(Model)
		if firstCmd == nil {
			t.Fatalf("expected first add submit command")
		}

		um, secondCmd := m.Update(keyEnter())
		m = um.(Model)
		if secondCmd != nil {
			t.Fatalf("expected duplicate add submit to be ignored")
		}

		um, _ = m.Update(firstCmd())
		m = um.(Model)

		if len(a.addedTitles) != 1 || a.addedTitles[0] != "new task" {
			t.Fatalf("expected exactly one add call with title %q, got %#v", "new task", a.addedTitles)
		}
	})

	t.Run("edit", func(t *testing.T) {
		task := domain.Task{ID: 7, Title: "write tests", Status: domain.StatusActive, CreatedDay: current, DueDay: current}
		a := newFakeApp(current, []domain.Task{task})

		m := NewWithDeps(a, baseClock, time.UTC)
		m = applyCmd(m, m.Init())

		um, _ := m.Update(keyRune('e'))
		m = um.(Model)
		m.addInput.SetValue("beta")

		um, firstCmd := m.Update(keyEnter())
		m = um.(Model)
		if firstCmd == nil {
			t.Fatalf("expected first edit submit command")
		}

		um, secondCmd := m.Update(keyEnter())
		m = um.(Model)
		if secondCmd != nil {
			t.Fatalf("expected duplicate edit submit to be ignored")
		}

		um, _ = m.Update(firstCmd())
		m = um.(Model)

		if len(a.editedTasks) != 1 || a.editedTasks[0] != (editedTaskCall{id: task.ID, title: "beta"}) {
			t.Fatalf("expected exactly one edit call %#v, got %#v", editedTaskCall{id: task.ID, title: "beta"}, a.editedTasks)
		}
	})

	t.Run("delete", func(t *testing.T) {
		task := domain.Task{ID: 8, Title: "cleanup", Status: domain.StatusActive, CreatedDay: current, DueDay: current}
		a := newFakeApp(current, []domain.Task{task})

		m := NewWithDeps(a, baseClock, time.UTC)
		m = applyCmd(m, m.Init())

		um, _ := m.Update(keyRune('d'))
		m = um.(Model)

		um, firstCmd := m.Update(keyRune('y'))
		m = um.(Model)
		if firstCmd == nil {
			t.Fatalf("expected first delete confirm command")
		}

		um, secondCmd := m.Update(keyRune('y'))
		m = um.(Model)
		if secondCmd != nil {
			t.Fatalf("expected duplicate delete confirm to be ignored")
		}

		um, _ = m.Update(firstCmd())
		m = um.(Model)

		if len(a.deletedIDs) != 1 || a.deletedIDs[0] != task.ID {
			t.Fatalf("expected exactly one delete call for id %d, got %#v", task.ID, a.deletedIDs)
		}
	})
}

func TestModel_Today_EditModal_DoesNotReplaceBodyYet(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, []domain.Task{{ID: 1, Title: "first", Status: domain.StatusActive, CreatedDay: current, DueDay: current}})

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	um, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = um.(Model)
	m = applyCmd(m, m.Init())

	um, _ = m.Update(keyRune('e'))
	m = um.(Model)

	if m.modal.kind != modalKindEdit {
		t.Fatalf("expected edit modal open, got %v", m.modal.kind)
	}
	body := renderTodayBody(m)
	if strings.Contains(body, "Add task") {
		t.Fatalf("expected edit modal state to not reuse add body rendering, got %q", body)
	}
	if !strings.Contains(body, "first") {
		t.Fatalf("expected today list body to remain visible before overlay rendering, got %q", body)
	}
}

func TestModel_ModalOpen_SuspendsDoneAction(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, []domain.Task{{ID: 9, Title: "t9", Status: domain.StatusActive, CreatedDay: current, DueDay: current}})

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m = applyCmd(m, m.Init())
	um, _ := m.Update(keyRune('a'))
	m = um.(Model)

	if m.modal.kind != modalKindAdd {
		t.Fatalf("expected add modal open, got %v", m.modal.kind)
	}
	um, _ = m.Update(keyRune('x'))
	m = um.(Model)
	if len(a.doneIDs) != 0 {
		t.Fatalf("expected done action suspended while modal is open, got %#v", a.doneIDs)
	}
	if m.addInput.Value() != "x" {
		t.Fatalf("expected key rune to go to modal input, got %q", m.addInput.Value())
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

	terracotta := regexp.MustCompile("\\x1b\\[[0-9;]*38;5;131[0-9;]*m")
	if !terracotta.MatchString(got) {
		t.Fatalf("expected delayed task to include terracotta ANSI color, got %q", got)
	}
}

func TestTodayItemDelegate_SelectedDelayed_KeepsRedTextAndSelectedBackground(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI256)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	day := domain.MustParseDay("2026-03-04")
	s := defaultStyles()
	d := todayItemDelegate{styles: s, currentDay: day}
	l := newTaskList(d)
	l.SetItems([]list.Item{taskItem{task: domain.Task{ID: 1, Title: "late", Status: domain.StatusActive, CreatedDay: day, DueDay: addDays(day, -1)}}})
	l.Select(0)

	var buf bytes.Buffer
	d.Render(&buf, l, 0, l.Items()[0])
	got := buf.String()

	if strings.Contains(got, "> ") {
		t.Fatalf("expected selected row to not use > marker, got %q", got)
	}

	if !strings.Contains(got, "late") {
		t.Fatalf("expected selected delayed row to contain title, got %q", got)
	}
	if !strings.Contains(got, "\x1b[7m") {
		t.Fatalf("expected selected delayed row to use reverse-style selected background, got %q", got)
	}
	terracotta := regexp.MustCompile("\\x1b\\[[0-9;]*38;5;131[0-9;]*m")
	if !terracotta.MatchString(got) {
		t.Fatalf("expected selected delayed row to keep terracotta foreground, got %q", got)
	}
}

func TestTodayItemDelegate_Unselected_DoesNotIndentRow(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	day := domain.MustParseDay("2026-03-04")
	d := todayItemDelegate{styles: defaultStyles(), currentDay: day}
	l := newTaskList(d)
	l.SetItems([]list.Item{
		taskItem{task: domain.Task{ID: 1, Title: "selected", Status: domain.StatusActive, CreatedDay: day, DueDay: day}},
		taskItem{task: domain.Task{ID: 2, Title: "plain", Status: domain.StatusActive, CreatedDay: day, DueDay: day}},
	})
	l.Select(0)

	var buf bytes.Buffer
	d.Render(&buf, l, 1, l.Items()[1])
	got := buf.String()

	if strings.HasPrefix(got, "  ") {
		t.Fatalf("expected unselected row to start at column 0, got %q", got)
	}
	if got != "plain" {
		t.Fatalf("expected unselected row to render plain title, got %q", got)
	}
}

func TestSimpleItemDelegate_Selected_UsesSharedSelectedStyle(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI256)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	s := defaultStyles()
	d := simpleItemDelegate{styles: s}
	l := newTaskList(d)
	l.SetItems([]list.Item{taskItem{task: domain.Task{ID: 1, Title: "soon", Status: domain.StatusActive}}})
	l.Select(0)

	var buf bytes.Buffer
	d.Render(&buf, l, 0, l.Items()[0])
	got := buf.String()

	if strings.Contains(got, "> ") {
		t.Fatalf("expected selected row to not use > marker, got %q", got)
	}

	if !strings.Contains(got, "soon") {
		t.Fatalf("expected selected row to contain title, got %q", got)
	}
	if !strings.Contains(got, "\x1b[7m") {
		t.Fatalf("expected selected row to use reverse-style selected treatment, got %q", got)
	}
}

func TestSimpleItemDelegate_Unselected_DoesNotIndentRow(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	d := simpleItemDelegate{styles: defaultStyles()}
	l := newTaskList(d)
	l.SetItems([]list.Item{
		taskItem{task: domain.Task{ID: 1, Title: "selected", Status: domain.StatusActive}},
		taskItem{task: domain.Task{ID: 2, Title: "plain", Status: domain.StatusActive}},
	})
	l.Select(0)

	var buf bytes.Buffer
	d.Render(&buf, l, 1, l.Items()[1])
	got := buf.String()

	if strings.HasPrefix(got, "  ") {
		t.Fatalf("expected unselected row to start at column 0, got %q", got)
	}
	if got != "plain" {
		t.Fatalf("expected unselected row to render plain title, got %q", got)
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

	// Selected row should use the shared reverse selection treatment.
	if !strings.Contains(out, "\x1b[7m") {
		t.Fatalf("expected View to include reverse-video highlight for selected row, got: %q", out)
	}
}

func TestModel_View_FillsWindowHeightAndCentersWhenWide(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	day := domain.MustParseDay("2026-03-04")
	a := newFakeApp(day, nil)

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	const w = 120
	const h = 24
	um, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = um.(Model)

	out := m.View()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != h {
		t.Fatalf("expected View to return exactly %d lines, got %d", h, len(lines))
	}

	// When the terminal is wider than the max content width (96), the view is centered.
	wantPad := (w - maxContentWidth) / 2
	if wantPad <= 0 {
		t.Fatalf("test setup invalid: expected width %d to require centering", w)
	}
	pad := strings.Repeat(" ", wantPad)
	for i, ln := range lines {
		if !strings.HasPrefix(ln, pad) {
			t.Fatalf("expected line %d to have left padding of %d spaces when width=%d, got: %q", i+1, wantPad, w, ln)
		}
	}
}

func TestModel_View_RendersThreeZonesAndFooterHelp(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	day := domain.MustParseDay("2026-03-04")
	a := newFakeApp(day, nil)

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	const w = 80
	const h = 24
	um, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = um.(Model)

	out := m.View()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) == 0 {
		t.Fatalf("expected View to return at least 1 line")
	}
	if !strings.Contains(lines[0], appLogo) {
		t.Fatalf("expected first line to contain %q, got %q", appLogo, lines[0])
	}
	if strings.Contains(lines[0], "tuitodo") {
		t.Fatalf("expected header to not contain old app name, got %q", lines[0])
	}
	if !strings.Contains(lines[0], "tick") {
		t.Fatalf("expected header to contain tick wordmark, got %q", lines[0])
	}
	if strings.Contains(lines[0], "{Upcoming}") || strings.Contains(lines[0], "{History}") {
		t.Fatalf("expected only active tab to use marker, got %q", lines[0])
	}
	if !strings.Contains(lines[0], "[Today]") {
		t.Fatalf("expected active tab to use visible selected-label fallback, got %q", lines[0])
	}
	if strings.Contains(lines[0], "[Upcoming]") || strings.Contains(lines[0], "[History]") {
		t.Fatalf("expected only active tab to use selected-label fallback, got %q", lines[0])
	}

	// Separators are inset by two spaces on both left and right.
	sep := separatorLine(w)
	if sep == "" {
		t.Fatalf("expected non-empty separatorLine for width %d", w)
	}
	sepCount := 0
	for _, ln := range lines {
		if ln == sep {
			sepCount++
		}
	}
	if sepCount != 2 {
		t.Fatalf("expected exactly 2 separator lines equal to separatorLine(%d), got %d", w, sepCount)
	}

	if !strings.Contains(lines[len(lines)-1], "q:quit") {
		t.Fatalf("expected last line to contain q:quit, got %q", lines[len(lines)-1])
	}
}

func TestModel_View_Header_UsesReverseSelectedTabAndStyledTickWordmark(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI256)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	day := domain.MustParseDay("2026-03-04")
	m := NewWithDeps(newFakeApp(day, nil), fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	um, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = um.(Model)

	out := m.View()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if !strings.Contains(lines[0], "tick") {
		t.Fatalf("expected tick wordmark in header, got %q", lines[0])
	}
	if !strings.Contains(lines[0], "\x1b[1;") && !strings.Contains(lines[0], "\x1b[1m") {
		t.Fatalf("expected bold styling on tick wordmark, got %q", lines[0])
	}
	if !strings.Contains(lines[0], "\x1b[1;3m") && !strings.Contains(lines[0], "\x1b[3;") && !strings.Contains(lines[0], "\x1b[3m") {
		t.Fatalf("expected italic styling on tick wordmark, got %q", lines[0])
	}
	if strings.Contains(lines[0], "{Today}") || strings.Contains(lines[0], "{Upcoming}") || strings.Contains(lines[0], "{History}") {
		t.Fatalf("expected no brace-selected tabs, got %q", lines[0])
	}
}

func TestModel_View_UsesFixedZoneLinePositions_80x24(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	day := domain.MustParseDay("2026-03-04")
	a := newFakeApp(day, nil)

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	const w = 80
	const h = 24
	um, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = um.(Model)

	// Simulate sheet rendering producing multi-line header content (e.g. wrapping).
	m.styles.Tab = m.styles.Tab.Width(1)
	m.styles.Reverse = m.styles.Reverse.Width(1)

	out := m.View()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != h {
		t.Fatalf("expected View to return exactly %d lines, got %d", h, len(lines))
	}
	if !strings.Contains(lines[0], appLogo) {
		t.Fatalf("expected header at line 1 to contain %q, got %q", appLogo, lines[0])
	}

	sep := separatorLine(w)
	if lines[1] != sep {
		t.Fatalf("expected separator at line 2, got %q", lines[1])
	}

	workspaceHeight := h - (headerHeight + separatorHeights + footerHeight)
	sepIndex := 2 + workspaceHeight
	if sepIndex >= len(lines) {
		t.Fatalf("test invalid: expected separator index %d within %d lines", sepIndex, len(lines))
	}
	if lines[sepIndex] != sep {
		t.Fatalf("expected separator after workspace at line %d, got %q", sepIndex+1, lines[sepIndex])
	}

	if !strings.Contains(lines[len(lines)-1], "q:quit") {
		t.Fatalf("expected help at last line to contain q:quit, got %q", lines[len(lines)-1])
	}
}

func TestModel_View_NarrowWidth_DoesNotOverflowContentWidth(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	day := domain.MustParseDay("2026-03-04")
	a := newFakeApp(day, []domain.Task{{ID: 1, Title: "a very very very very very long task title", Status: domain.StatusActive, CreatedDay: day, DueDay: day}})

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	const w = 50
	const h = 20
	um, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = um.(Model)
	m = applyCmd(m, m.Init())

	out := m.View()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	cw := contentWidth(w)
	pad := (w - cw) / 2
	prefix := strings.Repeat(" ", pad)

	for i, ln := range lines {
		if pad > 0 && strings.HasPrefix(ln, prefix) {
			ln = ln[pad:]
		}
		if ansi.StringWidth(ln) > cw {
			t.Fatalf("expected line %d to be <= %d cells (after removing centering pad), got %d: %q", i+1, cw, ansi.StringWidth(ln), ln)
		}
	}
}

func TestModel_View_UsesTwoLineFooterAcrossViews(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	day := domain.MustParseDay("2026-03-04")
	a := newFakeApp(day, nil)
	a.statsRatios = map[string]float64{"done": 0.25, "abandoned": 0.50}

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	const w = 80
	const h = 24
	um, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = um.(Model)
	m = applyCmd(m, m.Init())

	tests := []struct {
		name      string
		advance   int
		wantLine1 []string
		wantLine2 []string
	}{
		{
			name:      "today",
			wantLine1: []string{"a:add", "x:done"},
			wantLine2: []string{"tab:next", "q:quit"},
		},
		{
			name:      "upcoming",
			advance:   1,
			wantLine1: []string{"e:edit", "d:delete"},
			wantLine2: []string{"tab:next", "q:quit"},
		},
		{
			name:      "history",
			advance:   2,
			wantLine1: []string{"left/h:prev day", "down/j:scroll down"},
			wantLine2: []string{"tab:next", "q:quit"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			viewModel := m
			for i := 0; i < tc.advance; i++ {
				um, cmd := viewModel.Update(keyTab())
				viewModel = um.(Model)
				viewModel = applyCmd(viewModel, cmd)
			}

			out := viewModel.View()
			lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
			if len(lines) != h {
				t.Fatalf("expected View to return exactly %d lines, got %d", h, len(lines))
			}

			sep := separatorLine(w)
			if lines[len(lines)-3] != sep {
				t.Fatalf("expected separator directly above two-line footer, got %q", lines[len(lines)-3])
			}

			for _, want := range tc.wantLine1 {
				if !strings.Contains(lines[len(lines)-2], want) {
					t.Fatalf("expected first footer line to include %q, got %q", want, lines[len(lines)-2])
				}
			}
			for _, want := range tc.wantLine2 {
				if !strings.Contains(lines[len(lines)-1], want) {
					t.Fatalf("expected second footer line to include %q, got %q", want, lines[len(lines)-1])
				}
			}
		})
	}
}

func TestModel_View_RendersASCIISheetBorderFramePattern(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	day := domain.MustParseDay("2026-03-04")
	a := newFakeApp(day, nil)

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	const w = 80
	const h = 24
	um, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = um.(Model)

	out := m.View()

	topOrBottomFrame := regexp.MustCompile(`(?m)^\+[-]+\+$`)
	if !topOrBottomFrame.MatchString(out) {
		t.Fatalf("expected View to contain ASCII sheet top/bottom frame line (+---+), got %q", out)
	}

	sideFrame := regexp.MustCompile(`(?m)^\|.*\|$`)
	if !sideFrame.MatchString(out) {
		t.Fatalf("expected View to contain ASCII sheet side frame lines (|...|), got %q", out)
	}
}

func TestModel_WindowSize_SetsAddInputWidthToSheetInnerWidth(t *testing.T) {
	day := domain.MustParseDay("2026-03-04")
	a := newFakeApp(day, nil)

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	const w = 50
	const h = 20
	um, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = um.(Model)

	want := sheetInnerWidth(w)
	if m.addInput.Width != want {
		t.Fatalf("expected addInput width to be %d, got %d", want, m.addInput.Width)
	}
}

func TestModel_View_Footer_EmptyStatusMsg_StillRendersTwoLineFooter(t *testing.T) {
	orig := tickEvery
	tickEvery = 0
	t.Cleanup(func() { tickEvery = orig })

	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	day := domain.MustParseDay("2026-03-04")
	a := newFakeApp(day, nil)

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	const w = 80
	const h = 24
	um, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = um.(Model)
	m = applyCmd(m, m.Init())

	if m.view == viewHistory {
		t.Fatalf("test setup invalid: expected non-history view")
	}
	if m.statusMsg != "" {
		t.Fatalf("test setup invalid: expected empty statusMsg, got %q", m.statusMsg)
	}

	out := m.View()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != h {
		t.Fatalf("expected View to return exactly %d lines, got %d", h, len(lines))
	}

	helpLine1 := lines[len(lines)-2]
	helpLine2 := lines[len(lines)-1]
	sep := separatorLine(w)
	if lines[len(lines)-3] != sep {
		t.Fatalf("expected separator directly above footer, got %q", lines[len(lines)-3])
	}
	if !strings.Contains(helpLine2, "q:quit") {
		t.Fatalf("expected second help line in footer, got %q", helpLine2)
	}
	if strings.TrimSpace(helpLine1) == "" {
		t.Fatalf("expected first help line to contain grouped actions, got %q", helpLine1)
	}
}

func TestModel_View_Footer_NonEmptyStatusMsg_RendersInlineWithFirstHelpLine(t *testing.T) {
	orig := tickEvery
	tickEvery = 0
	t.Cleanup(func() { tickEvery = orig })

	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	day := domain.MustParseDay("2026-03-04")
	a := newFakeApp(day, nil)

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	const w = 80
	const h = 24
	um, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = um.(Model)
	m = applyCmd(m, m.Init())

	m.statusMsg = "done: boom"
	out := m.View()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != h {
		t.Fatalf("expected View to return exactly %d lines, got %d", h, len(lines))
	}

	helpLine1 := lines[len(lines)-2]
	helpLine2 := lines[len(lines)-1]
	if !strings.Contains(helpLine1, "done: boom") {
		t.Fatalf("expected first footer line to include statusMsg, got %q", helpLine1)
	}
	if strings.Contains(helpLine2, "done: boom") {
		t.Fatalf("expected statusMsg to stay off second footer line, got %q", helpLine2)
	}
	if !strings.Contains(helpLine2, "q:quit") {
		t.Fatalf("expected second help line in footer, got %q", helpLine2)
	}
}

func TestModel_View_Footer_TodayNonEmptyStatus_KeepsCompletionGroupRightAligned(t *testing.T) {
	orig := tickEvery
	tickEvery = 0
	t.Cleanup(func() { tickEvery = orig })

	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	day := domain.MustParseDay("2026-03-04")
	m := NewWithDeps(newFakeApp(day, nil), fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)

	const w = 80
	const h = 24
	um, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = um.(Model)
	m = applyCmd(m, m.Init())
	m.statusMsg = "done: boom"

	out := m.View()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != h {
		t.Fatalf("expected View to return exactly %d lines, got %d", h, len(lines))
	}

	line1 := lines[len(lines)-2]
	leftPrefix := "done: boom  a:add  e:edit  d:delete"
	right := "x:done  b:abandon  p:postpone"

	if !strings.HasPrefix(line1, leftPrefix) {
		t.Fatalf("expected first footer line to start with %q, got %q", leftPrefix, line1)
	}
	rightStart := strings.Index(line1, right)
	if rightStart == -1 {
		t.Fatalf("expected first footer line to include right action group %q, got %q", right, line1)
	}

	wantRightStart := contentWidth(w) - len(right)
	if rightStart != wantRightStart {
		t.Fatalf("expected right action group to start at column %d, got %d in %q", wantRightStart, rightStart, line1)
	}

	between := line1[len(leftPrefix):rightStart]
	if strings.Trim(between, " ") != "" {
		t.Fatalf("expected only padding between status/left prefix and right group, got %q in %q", between, line1)
	}
	if len(between) < 2 {
		t.Fatalf("expected visible separation between status/left prefix and right group, got %d spaces in %q", len(between), line1)
	}
}

func TestModel_View_Footer_TodayNarrowWidth_DropsCompletionGroupBeforeClipping(t *testing.T) {
	orig := tickEvery
	tickEvery = 0
	t.Cleanup(func() { tickEvery = orig })

	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	day := domain.MustParseDay("2026-03-04")
	m := NewWithDeps(newFakeApp(day, nil), fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)

	const w = 40
	const h = 24
	um, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = um.(Model)
	m = applyCmd(m, m.Init())

	out := m.View()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != h {
		t.Fatalf("expected View to return exactly %d lines, got %d", h, len(lines))
	}

	line1 := lines[len(lines)-2]
	want := "a:add  e:edit  d:delete"
	if line1 != want {
		t.Fatalf("expected narrow Today footer line to intentionally keep only left group %q, got %q", want, line1)
	}
	if strings.Contains(line1, "x:done") || strings.Contains(line1, "b:abandon") || strings.Contains(line1, "p:postpone") {
		t.Fatalf("expected narrow Today footer line to omit completion group entirely, got %q", line1)
	}
}

func TestModel_View_Footer_HistoryView_KeepsStatusLineEmptyAndRatiosOutOfFooter(t *testing.T) {
	orig := tickEvery
	tickEvery = 0
	t.Cleanup(func() { tickEvery = orig })

	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	day := domain.MustParseDay("2026-03-04")
	a := newFakeApp(day, nil)
	a.statsRatios = map[string]float64{"done": 0.25, "abandoned": 0.50}

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	const w = 80
	const h = 24
	um, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = um.(Model)

	// Enter history view (loads selected day + stats).
	um, cmd := m.Update(keyTab())
	m = um.(Model)
	um, cmd = m.Update(keyTab())
	m = um.(Model)
	m = applyCmd(m, cmd)

	if m.view != viewHistory {
		t.Fatalf("expected history view")
	}
	if m.statusMsg != "" {
		t.Fatalf("test setup invalid: expected empty statusMsg, got %q", m.statusMsg)
	}

	out := m.View()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != h {
		t.Fatalf("expected View to return exactly %d lines, got %d", h, len(lines))
	}

	helpLine1 := lines[len(lines)-2]
	helpLine2 := lines[len(lines)-1]
	sep := separatorLine(w)
	if lines[len(lines)-3] != sep {
		t.Fatalf("expected separator directly above footer, got %q", lines[len(lines)-3])
	}
	if strings.Contains(helpLine1, "done:") || strings.Contains(helpLine1, "abandoned:") || strings.Contains(helpLine1, "overdue active:") || strings.Contains(helpLine2, "done:") || strings.Contains(helpLine2, "abandoned:") || strings.Contains(helpLine2, "overdue active:") {
		t.Fatalf("expected history count stats to be absent from footer, got help1=%q help2=%q", helpLine1, helpLine2)
	}
	if !strings.Contains(helpLine2, "q:quit") {
		t.Fatalf("expected second help line in footer, got %q", helpLine2)
	}
	if !strings.Contains(out, "done: 0") || !strings.Contains(out, "abandoned: 0") || !strings.Contains(out, "overdue active: 0") {
		t.Fatalf("expected count stats in history content area, got:\n%s", out)
	}
}
