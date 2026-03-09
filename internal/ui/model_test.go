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
	if g.workspaceH != 19 { // 24 - (header1 + sep1 + sep1 + footer2)
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
			workspaceH := m.height - (1 + 1 + 1 + 2)
			if workspaceH < 0 {
				workspaceH = 0
			}
			innerH := workspaceH - sheetVertMargin
			if innerH < 0 {
				innerH = 0
			}
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
	workspaceH := m.height - (1 + 1 + 1 + 2)
	if workspaceH < 0 {
		workspaceH = 0
	}
	innerH := workspaceH - sheetVertMargin
	if innerH < 0 {
		innerH = 0
	}
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
		{name: "abandon", key: keyRune('d'), id: 2, wantAbandoned: []int64{2}},
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

	if !strings.Contains(lines[len(lines)-1], "q:Quit") {
		t.Fatalf("expected last line to contain q:Quit, got %q", lines[len(lines)-1])
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

	workspaceHeight := h - 5 // header(1) + sep(1) + workspace + sep(1) + status(1) + help(1)
	sepIndex := 2 + workspaceHeight
	if sepIndex >= len(lines) {
		t.Fatalf("test invalid: expected separator index %d within %d lines", sepIndex, len(lines))
	}
	if lines[sepIndex] != sep {
		t.Fatalf("expected separator after workspace at line %d, got %q", sepIndex+1, lines[sepIndex])
	}

	if !strings.Contains(lines[len(lines)-1], "q:Quit") {
		t.Fatalf("expected help at last line to contain q:Quit, got %q", lines[len(lines)-1])
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

	statusLine := lines[len(lines)-2]
	helpLine := lines[len(lines)-1]
	sep := separatorLine(w)
	if lines[len(lines)-3] != sep {
		t.Fatalf("expected separator directly above footer, got %q", lines[len(lines)-3])
	}
	if strings.TrimSpace(statusLine) != "" {
		t.Fatalf("expected blank status line when statusMsg empty, got %q", statusLine)
	}
	if !strings.Contains(helpLine, "q:Quit") {
		t.Fatalf("expected help line in footer, got %q", helpLine)
	}
}

func TestModel_View_Footer_NonEmptyStatusMsg_RendersAboveHelpLine(t *testing.T) {
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

	statusLine := lines[len(lines)-2]
	helpLine := lines[len(lines)-1]
	if !strings.Contains(statusLine, "done: boom") {
		t.Fatalf("expected non-empty status line to include statusMsg, got %q", statusLine)
	}
	if strings.Contains(helpLine, "done: boom") {
		t.Fatalf("expected statusMsg to render above help line, got help=%q", helpLine)
	}
	if !strings.Contains(helpLine, "q:Quit") {
		t.Fatalf("expected help line in footer, got %q", helpLine)
	}
}

func TestModel_View_Footer_HistoryView_RatiosOnStatusLine_StillTwoLineFooter(t *testing.T) {
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

	statusLine := lines[len(lines)-2]
	helpLine := lines[len(lines)-1]
	sep := separatorLine(w)
	if lines[len(lines)-3] != sep {
		t.Fatalf("expected separator directly above footer, got %q", lines[len(lines)-3])
	}
	if !strings.Contains(statusLine, "DoneDelayedRatio") || !strings.Contains(statusLine, "AbandonedDelayedRatio") {
		t.Fatalf("expected ratios on status line, got %q", statusLine)
	}
	if strings.Contains(helpLine, "DoneDelayedRatio") || strings.Contains(helpLine, "AbandonedDelayedRatio") {
		t.Fatalf("expected ratios to be on status line (not help line), got help=%q", helpLine)
	}
	if !strings.Contains(helpLine, "q:Quit") {
		t.Fatalf("expected help line in footer, got %q", helpLine)
	}
}
