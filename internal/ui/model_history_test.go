package ui

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/JamesYuuu/tick/internal/domain"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

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
	m.width = 80
	m.height = 24

	bodyOnly := renderHistoryBody(m)
	if indexOf(bodyOnly, "No history tasks.") < 0 {
		t.Fatalf("expected history body to show No history tasks. when there are no rows, got:\n%s", bodyOnly)
	}
	if hasExactLine(bodyOnly, "Done") || hasExactLine(bodyOnly, "Abandoned") {
		t.Fatalf("expected history body to not include Done/Abandoned headings anymore, got:\n%s", bodyOnly)
	}
}

func TestModel_History_EnterRefreshesStatsAndSelectedDayLists(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	overdue := domain.MustParseDay("2026-03-03")
	a := newFakeApp(current, nil)
	a.historyDoneByDay = map[string][]domain.Task{
		current.String(): {{ID: 1, Title: "done-1", Status: domain.StatusDone, CreatedDay: current, DueDay: current}},
	}
	a.historyAbandonedByDay = map[string][]domain.Task{
		current.String(): {{ID: 2, Title: "ab-1", Status: domain.StatusAbandoned, CreatedDay: current, DueDay: current}},
	}
	a.historyActiveByCreatedDay = map[string][]domain.Task{
		current.String(): {{ID: 3, Title: "active-1", Status: domain.StatusActive, CreatedDay: current, DueDay: overdue}},
	}
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
	um, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
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
	if !containsAll(v, []string{"[✓] done-1", "[✗] ab-1", "[ ] active-1", "done: 1", "abandoned: 1", "overdue active: 1"}) {
		t.Fatalf("expected history view to include lists and count stats, got:\n%s", v)
	}
	if indexOf(v, current.String()) >= 0 {
		t.Fatalf("expected history dates to not include year, got:\n%s", v)
	}

	bodyOnly := renderHistoryBody(m)
	if !containsAll(bodyOnly, []string{"done: 1", "abandoned: 1", "overdue active: 1"}) {
		t.Fatalf("expected count stats to be rendered in history body, got:\n%s", bodyOnly)
	}
	if strings.Contains(m.footerStatusLine(), "done:") || strings.Contains(m.footerStatusLine(), "abandoned:") || strings.Contains(m.footerStatusLine(), "overdue active:") {
		t.Fatalf("expected count stats to be absent from footer status, got %q", m.footerStatusLine())
	}
}

func TestRenderHistoryStats_UsesLoadedCollectionCountsInsteadOfRatios(t *testing.T) {
	disableTick(t)

	today := domain.MustParseDay("2026-03-07")
	overdue := domain.MustParseDay("2026-03-06")
	m := NewWithDeps(newFakeApp(today, nil), fakeClock{now: time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m.view = viewHistory
	m.width = 80
	m.height = 18
	m.historyFrom = addDays(today, -6)
	m.historyTo = today
	m.historyIndex = 6
	m.historyDone = []domain.Task{
		{ID: 1, Title: "done-01", Status: domain.StatusDone, CreatedDay: today, DueDay: today, DoneDay: &today},
		{ID: 2, Title: "done-02", Status: domain.StatusDone, CreatedDay: today, DueDay: today, DoneDay: &today},
	}
	m.historyAbandoned = []domain.Task{{ID: 3, Title: "ab-01", Status: domain.StatusAbandoned, CreatedDay: today, DueDay: today, AbandonedDay: &today}}
	m.historyActiveCreated = []domain.Task{
		{ID: 4, Title: "active-overdue", Status: domain.StatusActive, CreatedDay: today, DueDay: overdue},
		{ID: 5, Title: "active-not-overdue", Status: domain.StatusActive, CreatedDay: today, DueDay: today},
	}
	m.historyStats.DoneDelayedRatio = 0.25
	m.historyStats.AbandonedDelayedRatio = 0.5

	got := renderHistoryStats(m)
	if want := m.styles.Stats.Render("done: 2  abandoned: 1  overdue active: 1"); got != want {
		t.Fatalf("expected stats line %q, got %q", want, got)
	}
	if strings.Contains(got, "DoneDelayedRatio") || strings.Contains(got, "AbandonedDelayedRatio") || strings.Contains(got, "0.25") || strings.Contains(got, "0.5") {
		t.Fatalf("expected ratios to be absent from stats line, got %q", got)
	}
}

func TestRenderHistoryBody_BottomAnchorsStatsBelowViewportPadding(t *testing.T) {
	disableTick(t)

	day := domain.MustParseDay("2026-03-07")
	m := NewWithDeps(newFakeApp(day, nil), fakeClock{now: time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m.view = viewHistory
	m.width = 80
	m.height = 18
	m.historyFrom = addDays(day, -6)
	m.historyTo = day
	m.historyIndex = 6
	m.historyDone = []domain.Task{{ID: 1, Title: "done-01", Status: domain.StatusDone, CreatedDay: day, DueDay: day, DoneDay: &day}}
	m.historyAbandoned = []domain.Task{{ID: 2, Title: "ab-01", Status: domain.StatusAbandoned, CreatedDay: day, DueDay: day, AbandonedDay: &day}}
	m.historyActiveCreated = []domain.Task{{ID: 3, Title: "active-01", Status: domain.StatusActive, CreatedDay: day, DueDay: addDays(day, -1)}}

	body := renderHistoryBody(m)
	statsLine := renderHistoryStats(m)
	if !containsAll(body, []string{"done: 1", "abandoned: 1", "overdue active: 1"}) {
		t.Fatalf("expected stats in history body, got:\n%s", body)
	}

	lines := strings.Split(body, "\n")
	if got := lines[len(lines)-1]; got != statsLine {
		t.Fatalf("expected stats on final line, got %q in:\n%s", got, body)
	}
	if strings.TrimSpace(lines[len(lines)-2]) != "" {
		t.Fatalf("expected blank separator before stats, got %q in:\n%s", lines[len(lines)-2], body)
	}

	dividerIdx := indexOfExactLine(lines, strings.Repeat("-", sheetInnerWidth(m.width)-2))
	if dividerIdx < 0 {
		t.Fatalf("expected divider line in history body, got:\n%s", body)
	}
	detailStart := dividerIdx + 1
	detailEnd := len(lines) - 3
	if got, want := detailEnd-detailStart+1, m.historyDetailViewportHeight(); got != want {
		t.Fatalf("expected detail viewport height %d, got %d in:\n%s", want, got, body)
	}
	if got := lines[detailStart]; !strings.Contains(got, "[✓] done-01") {
		t.Fatalf("expected first detail row to contain task, got %q in:\n%s", got, body)
	}
	if got := lines[detailStart+1]; !strings.Contains(got, "[✗] ab-01") {
		t.Fatalf("expected second detail row to contain task, got %q in:\n%s", got, body)
	}
	if got := lines[detailStart+2]; !strings.Contains(got, "[ ] active-01") {
		t.Fatalf("expected third detail row to contain task, got %q in:\n%s", got, body)
	}
	for i := detailStart + 3; i <= detailEnd; i++ {
		if strings.TrimSpace(lines[i]) != "" {
			t.Fatalf("expected viewport padding before bottom-anchored stats at line %d, got %q in:\n%s", i, lines[i], body)
		}
	}
}

func TestRenderHistoryBody_ScrolledStateKeepsStatsVisibleWhileRowsMove(t *testing.T) {
	disableTick(t)

	day := domain.MustParseDay("2026-03-07")
	m := NewWithDeps(newFakeApp(day, nil), fakeClock{now: time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m.view = viewHistory
	m.width = 80
	m.height = 18
	m.historyFrom = addDays(day, -6)
	m.historyTo = day
	m.historyIndex = 6
	m.historyAbandoned = []domain.Task{{ID: 100, Title: "ab-01", Status: domain.StatusAbandoned, CreatedDay: day, DueDay: day, AbandonedDay: &day}}
	m.historyActiveCreated = []domain.Task{{ID: 101, Title: "active-01", Status: domain.StatusActive, CreatedDay: day, DueDay: addDays(day, -1)}}
	for i := 0; i < 12; i++ {
		m.historyDone = append(m.historyDone, domain.Task{ID: int64(i + 1), Title: fmt.Sprintf("done-%02d", i+1), Status: domain.StatusDone, CreatedDay: day, DueDay: day, DoneDay: &day})
	}

	m.historyScroll = 0
	bodyTop := renderHistoryBody(m)
	m.historyScroll = 5
	bodyScrolled := renderHistoryBody(m)

	statsLine := renderHistoryStats(m)
	topLines := strings.Split(bodyTop, "\n")
	scrolledLines := strings.Split(bodyScrolled, "\n")
	topStatsIdx := indexOfExactLine(topLines, statsLine)
	scrolledStatsIdx := indexOfExactLine(scrolledLines, statsLine)
	if topStatsIdx < 0 || scrolledStatsIdx < 0 {
		t.Fatalf("expected stats line in both history bodies\nTOP:\n%s\nSCROLLED:\n%s", bodyTop, bodyScrolled)
	}
	if topStatsIdx != len(topLines)-1 || scrolledStatsIdx != len(scrolledLines)-1 {
		t.Fatalf("expected stats to remain on final line\nTOP:\n%s\nSCROLLED:\n%s", bodyTop, bodyScrolled)
	}
	if topStatsIdx != scrolledStatsIdx {
		t.Fatalf("expected stats line to stay at same anchored index, got top=%d scrolled=%d", topStatsIdx, scrolledStatsIdx)
	}
	if strings.TrimSpace(topLines[topStatsIdx-1]) != "" || strings.TrimSpace(scrolledLines[scrolledStatsIdx-1]) != "" {
		t.Fatalf("expected blank separator before stats in both states\nTOP:\n%s\nSCROLLED:\n%s", bodyTop, bodyScrolled)
	}

	topDividerIdx := indexOfExactLine(topLines, strings.Repeat("-", sheetInnerWidth(m.width)-2))
	scrolledDividerIdx := indexOfExactLine(scrolledLines, strings.Repeat("-", sheetInnerWidth(m.width)-2))
	if topDividerIdx < 0 || scrolledDividerIdx < 0 {
		t.Fatalf("expected divider line in both history bodies\nTOP:\n%s\nSCROLLED:\n%s", bodyTop, bodyScrolled)
	}
	topDetails := strings.Join(topLines[topDividerIdx+1:topStatsIdx-1], "\n")
	scrolledDetails := strings.Join(scrolledLines[scrolledDividerIdx+1:scrolledStatsIdx-1], "\n")
	if topDetails == scrolledDetails {
		t.Fatalf("expected detail viewport rows to move under scroll\nTOP:\n%s\nSCROLLED:\n%s", bodyTop, bodyScrolled)
	}
	if !strings.Contains(topDetails, "done-01") || strings.Contains(scrolledDetails, "done-01") {
		t.Fatalf("expected top rows to scroll out of detail viewport\nTOP DETAILS:\n%s\nSCROLLED DETAILS:\n%s", topDetails, scrolledDetails)
	}
	if !strings.Contains(scrolledDetails, "done-06") {
		t.Fatalf("expected later rows to scroll into detail viewport, got:\n%s", scrolledDetails)
	}
}

func TestModel_History_UpDownScrollsDetails_NotDate(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, nil)
	a.historyDoneByDay = map[string][]domain.Task{}
	a.historyAbandonedByDay = map[string][]domain.Task{}
	a.historyActiveByCreatedDay = map[string][]domain.Task{}
	a.statsRatios = map[string]float64{}
	rows := make([]domain.Task, 0, 12)
	for i := 0; i < 12; i++ {
		rows = append(rows, domain.Task{ID: int64(i + 1), Title: fmt.Sprintf("done-%02d", i+1), Status: domain.StatusDone, CreatedDay: current, DueDay: current, DoneDay: &current})
	}
	a.historyDoneByDay[current.String()] = rows

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	um, cmd := m.Update(keyTab())
	m = um.(Model)
	um, cmd = m.Update(keyTab())
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected cmd when entering history view")
	}
	m = applyCmd(m, cmd)
	um, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 18})
	m = um.(Model)

	idx0 := m.historyIndex
	mt := reflect.TypeOf(m)
	f, ok := mt.FieldByName("historyScroll")
	if !ok {
		t.Fatalf("expected Model to have historyScroll field")
	}
	if f.Type.Kind() != reflect.Int {
		t.Fatalf("expected historyScroll to be int")
	}

	scroll0 := reflect.ValueOf(m).FieldByName("historyScroll").Int()
	if scroll0 != 0 {
		t.Fatalf("expected initial historyScroll=0, got %d", scroll0)
	}

	a.resetHistoryCounters()
	um, cmd = m.Update(keyRune('j'))
	m = um.(Model)
	if cmd != nil {
		t.Fatalf("expected no refresh cmd when scrolling, got non-nil")
	}
	if m.historyIndex != idx0 {
		t.Fatalf("expected historyIndex unchanged when scrolling, got %d (want %d)", m.historyIndex, idx0)
	}
	scroll1 := reflect.ValueOf(m).FieldByName("historyScroll").Int()
	if scroll1 != scroll0+1 {
		t.Fatalf("expected historyScroll to increment by 1, got %d (want %d)", scroll1, scroll0+1)
	}
	if a.historyDoneCalls != 0 || a.historyAbandonedCalls != 0 || a.historyActiveCreatedCalls != 0 || a.statsCalls != 0 {
		t.Fatalf("expected no history/stats refresh calls when scrolling, got done=%d abandoned=%d activeCreated=%d stats=%d", a.historyDoneCalls, a.historyAbandonedCalls, a.historyActiveCreatedCalls, a.statsCalls)
	}

	um, cmd = m.Update(keyRune('k'))
	m = um.(Model)
	if cmd != nil {
		t.Fatalf("expected no refresh cmd when scrolling up, got non-nil")
	}
	scroll2 := reflect.ValueOf(m).FieldByName("historyScroll").Int()
	if scroll2 != scroll0 {
		t.Fatalf("expected historyScroll to return to %d, got %d", scroll0, scroll2)
	}
}

func TestModel_History_DownDoesNotScrollWhenAllRowsFit(t *testing.T) {
	disableTick(t)

	day := domain.MustParseDay("2026-03-07")
	a := newFakeApp(day, nil)
	a.historyDoneByDay[day.String()] = []domain.Task{{ID: 1, Title: "one", Status: domain.StatusDone, CreatedDay: day, DueDay: day, DoneDay: &day}}

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)}, time.UTC)
	um, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	m = um.(Model)
	m.view = viewHistory
	m.historyFrom = addDays(day, -6)
	m.historyTo = day
	m.historyIndex = 6
	m = applyCmd(m, m.cmdRefreshHistorySelectedDay())

	if m.historyScroll != 0 {
		t.Fatalf("expected initial historyScroll=0, got %d", m.historyScroll)
	}
	um, _ = m.Update(keyRune('j'))
	m = um.(Model)
	if m.historyScroll != 0 {
		t.Fatalf("expected no scrolling when all rows fit, got %d", m.historyScroll)
	}

	um, _ = m.Update(keyRune('k'))
	m = um.(Model)
	if m.historyScroll != 0 {
		t.Fatalf("expected no upward scrolling when all rows fit, got %d", m.historyScroll)
	}
}

func TestModel_History_DownClampsAtLastViewportStart(t *testing.T) {
	disableTick(t)

	day := domain.MustParseDay("2026-03-07")
	a := newFakeApp(day, nil)
	rows := make([]domain.Task, 0, 12)
	for i := 0; i < 12; i++ {
		rows = append(rows, domain.Task{ID: int64(i + 1), Title: fmt.Sprintf("done-%02d", i+1), Status: domain.StatusDone, CreatedDay: day, DueDay: day, DoneDay: &day})
	}
	a.historyDoneByDay[day.String()] = rows

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)}, time.UTC)
	um, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 18})
	m = um.(Model)
	m.view = viewHistory
	m.historyFrom = addDays(day, -6)
	m.historyTo = day
	m.historyIndex = 6
	m = applyCmd(m, m.cmdRefreshHistorySelectedDay())

	for i := 0; i < 20; i++ {
		um, _ = m.Update(keyRune('j'))
		m = um.(Model)
	}

	if m.historyScroll != 8 {
		t.Fatalf("expected historyScroll clamped to 8, got %d", m.historyScroll)
	}

	body := renderHistoryBody(m)
	if !strings.Contains(body, "done-09") {
		t.Fatalf("expected body to start at the last full viewport, got:\n%s", body)
	}
	if !containsAll(body, []string{"done-10", "done-12"}) {
		t.Fatalf("expected body to show the final viewport rows, got:\n%s", body)
	}
}

func TestModel_History_ScrollResetsOnEnteringHistoryView(t *testing.T) {
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
	if cmd != nil {
		m = applyCmd(m, cmd)
	}

	m.historyScroll = 3

	um, cmd = m.Update(keyTab())
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected cmd when entering history view")
	}
	if m.historyScroll != 0 {
		t.Fatalf("expected historyScroll reset to 0 when entering history view, got %d", m.historyScroll)
	}
}

func TestModel_History_ScrollResetsOnHistoryRefreshMsg(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, nil)

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	um, cmd := m.Update(keyTab())
	m = um.(Model)
	if cmd != nil {
		m = applyCmd(m, cmd)
	}
	um, cmd = m.Update(keyTab())
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected cmd when entering history view")
	}
	m = applyCmd(m, cmd)

	m.historyScroll = 5
	um, _ = m.Update(historyRefreshMsg{done: nil, abandoned: nil, activeCreated: nil})
	m = um.(Model)
	if m.historyScroll != 0 {
		t.Fatalf("expected historyScroll reset to 0 on history refresh, got %d", m.historyScroll)
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

func TestModel_HistoryRefresh_StatsOnlyWhenRequested(t *testing.T) {
	disableTick(t)
	day := domain.MustParseDay("2026-03-07")
	a := newFakeApp(day, nil)
	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m.view = viewHistory
	m.historyFrom = domain.MustParseDay("2026-03-01")
	m.historyTo = day
	m.historyIndex = 6

	m = applyCmd(m, m.cmdRefreshHistory(false))
	if a.statsCalls != 0 {
		t.Fatalf("expected stats not called")
	}

	m = applyCmd(m, m.cmdRefreshHistory(true))
	if a.statsCalls != 1 {
		t.Fatalf("expected stats called once")
	}
}

func TestModel_History_HNoLongerShiftsWindow(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, nil)
	a.historyActiveByCreatedDay = map[string][]domain.Task{}

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	um, cmd := m.Update(keyTab())
	m = um.(Model)
	if cmd != nil {
		m = applyCmd(m, cmd)
	}
	um, cmd = m.Update(keyTab())
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected cmd when entering history view")
	}
	m = applyCmd(m, cmd)

	idx0 := m.historyIndex
	a.resetHistoryCounters()

	um, cmd = m.Update(keyRune('h'))
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected cmd when moving left with h")
	}
	m = applyCmd(m, cmd)
	if m.historyIndex != idx0-1 {
		t.Fatalf("expected historyIndex=%d after h, got %d", idx0-1, m.historyIndex)
	}
	if a.statsCalls != 0 {
		t.Fatalf("expected no stats refresh on within-window move, got %d", a.statsCalls)
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
	if cmd != nil {
		m = applyCmd(m, cmd)
	}
	um, cmd = m.Update(keyTab())
	m = um.(Model)
	m = applyCmd(m, cmd)

	idx0 := m.historyIndex

	// 'h' and Left are equivalent.
	um, cmd = m.Update(keyRune('h'))
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected cmd for h")
	}
	m = applyCmd(m, cmd)
	idxH := m.historyIndex

	m.historyIndex = idx0
	um, cmd = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected cmd for left")
	}
	m = applyCmd(m, cmd)
	idxLeft := m.historyIndex

	if idxH != idxLeft {
		t.Fatalf("expected h and left to behave the same, got %d vs %d", idxH, idxLeft)
	}

	// 'l' and Right are equivalent (use a non-edge index so it always moves).
	m.historyIndex = 0
	um, cmd = m.Update(keyRune('l'))
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected cmd for l")
	}
	m = applyCmd(m, cmd)
	idxL := m.historyIndex

	m.historyIndex = 0
	um, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected cmd for right")
	}
	m = applyCmd(m, cmd)
	idxRight := m.historyIndex

	if idxL != idxRight {
		t.Fatalf("expected l and right to behave the same, got %d vs %d", idxL, idxRight)
	}
}

func TestModel_History_LeftRightMovesSelectedDate(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, nil)
	a.historyActiveByCreatedDay = map[string][]domain.Task{}

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	// Today -> Upcoming -> History
	um, cmd := m.Update(keyTab())
	m = um.(Model)
	if cmd != nil {
		m = applyCmd(m, cmd)
	}
	um, cmd = m.Update(keyTab())
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected cmd when entering history view")
	}
	m = applyCmd(m, cmd)

	idx0 := m.historyIndex
	if idx0 != 6 {
		t.Fatalf("expected starting at index 6 (today), got %d", idx0)
	}

	// Left should move to index 5 and refresh selected day.
	a.resetHistoryCounters()
	um, cmd = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected cmd after moving selection left")
	}
	m = applyCmd(m, cmd)
	if m.historyIndex != idx0-1 {
		t.Fatalf("expected historyIndex=%d, got %d", idx0-1, m.historyIndex)
	}
	if a.statsCalls != 0 {
		t.Fatalf("expected no stats refresh when moving selection within window")
	}

	// Right should move back to index 6 and refresh selected day.
	a.resetHistoryCounters()
	um, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected cmd after moving selection right")
	}
	m = applyCmd(m, cmd)
	if m.historyIndex != idx0 {
		t.Fatalf("expected historyIndex=%d, got %d", idx0, m.historyIndex)
	}
	if a.statsCalls != 0 {
		t.Fatalf("expected no stats refresh when moving selection within window")
	}
}

func TestModel_History_LeftAtEdgeRollsWindowBackOneDay(t *testing.T) {
	disableTick(t)

	current := domain.MustParseDay("2026-03-04")
	a := newFakeApp(current, nil)
	a.historyActiveByCreatedDay = map[string][]domain.Task{}

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	um, cmd := m.Update(keyTab())
	m = um.(Model)
	if cmd != nil {
		m = applyCmd(m, cmd)
	}
	um, cmd = m.Update(keyTab())
	m = um.(Model)
	m = applyCmd(m, cmd)

	// Move to left edge (index 0) using left navigation within window.
	for i := 0; i < 6; i++ {
		um, cmd = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
		m = um.(Model)
		if cmd == nil {
			t.Fatalf("expected cmd while moving selection left")
		}
		m = applyCmd(m, cmd)
	}
	if m.historyIndex != 0 {
		t.Fatalf("expected historyIndex=0, got %d", m.historyIndex)
	}

	from0, to0 := m.historyFrom, m.historyTo
	a.resetHistoryCounters()

	// Left at edge should roll window back by one day, keep index 0, and refresh stats.
	um, cmd = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected cmd after left edge roll")
	}
	m = applyCmd(m, cmd)

	if m.historyIndex != 0 {
		t.Fatalf("expected historyIndex to remain 0 after roll, got %d", m.historyIndex)
	}
	if m.historyFrom.String() != addDays(from0, -1).String() || m.historyTo.String() != addDays(to0, -1).String() {
		t.Fatalf("expected window rolled back by 1 day, got from=%s to=%s", m.historyFrom.String(), m.historyTo.String())
	}
	if a.statsCalls != 1 {
		t.Fatalf("expected stats refresh on window roll, got %d", a.statsCalls)
	}
}

func TestModel_History_RightAtEdgeRollsWindowForwardOneDay_ClampedToToday(t *testing.T) {
	disableTick(t)

	today := domain.MustParseDay("2026-03-04")
	a := newFakeApp(today, nil)
	a.historyActiveByCreatedDay = map[string][]domain.Task{}

	m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
	um, cmd := m.Update(keyTab())
	m = um.(Model)
	if cmd != nil {
		m = applyCmd(m, cmd)
	}
	um, cmd = m.Update(keyTab())
	m = um.(Model)
	m = applyCmd(m, cmd)

	// At bottom edge (index 6) and clamped at today, right should no-op.
	if m.historyIndex != 6 {
		t.Fatalf("expected historyIndex=6 at entry, got %d", m.historyIndex)
	}
	from0, to0 := m.historyFrom, m.historyTo
	a.resetHistoryCounters()
	um, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = um.(Model)
	if cmd != nil {
		m = applyCmd(m, cmd)
	}
	if m.historyFrom.String() != from0.String() || m.historyTo.String() != to0.String() {
		t.Fatalf("expected window unchanged when clamped at today")
	}
	if m.historyIndex != 6 {
		t.Fatalf("expected historyIndex unchanged when clamped at today")
	}
	if a.statsCalls != 0 {
		t.Fatalf("expected no stats refresh when clamped at today")
	}

	// If the window lags behind today, right at edge should roll forward by one day.
	m.historyTo = addDays(today, -3)
	m.historyFrom = addDays(m.historyTo, -6)
	m.historyIndex = 6

	from1, to1 := m.historyFrom, m.historyTo
	a.resetHistoryCounters()
	um, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = um.(Model)
	if cmd == nil {
		t.Fatalf("expected cmd after forward roll")
	}
	m = applyCmd(m, cmd)
	if m.historyIndex != 6 {
		t.Fatalf("expected historyIndex to remain 6 after roll")
	}
	if m.historyFrom.String() != addDays(from1, 1).String() || m.historyTo.String() != addDays(to1, 1).String() {
		t.Fatalf("expected window rolled forward by 1 day")
	}
	if a.statsCalls != 1 {
		t.Fatalf("expected stats refresh on window roll, got %d", a.statsCalls)
	}

	// Roll forward until clamped at today.
	for !m.historyTo.Time().Equal(today.Time()) {
		um, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRight})
		m = um.(Model)
		m = applyCmd(m, cmd)
	}
	from2, to2 := m.historyFrom, m.historyTo
	a.resetHistoryCounters()
	um, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = um.(Model)
	if cmd != nil {
		m = applyCmd(m, cmd)
	}
	if m.historyFrom.String() != from2.String() || m.historyTo.String() != to2.String() {
		t.Fatalf("expected clamped window to remain unchanged")
	}
	if a.statsCalls != 0 {
		t.Fatalf("expected no stats refresh once clamped")
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
	m.width = 80
	m.height = 24

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
	if !containsAll(body, []string{"03-03", "[✓] done-late", "[✓] done-on-time", "[✓] done-missing-day", "[✗] ab-late", "[✗] ab-on-time", "[ ] active-late"}) {
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

func TestRenderHistoryBody_DateTableAndDividerAndViewportScroll(t *testing.T) {
	disableTick(t)

	lipgloss.SetColorProfile(termenv.ANSI256)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	today := domain.MustParseDay("2026-03-07")
	from := domain.MustParseDay("2026-03-01")

	m := NewWithDeps(newFakeApp(today, nil), fakeClock{now: time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m.view = viewHistory
	m.width = 80
	m.height = 18
	m.historyFrom = from
	m.historyTo = today
	m.historyIndex = 6

	// Populate enough rows to require scrolling.
	m.historyDone = []domain.Task{}
	for i := 0; i < 12; i++ {
		m.historyDone = append(m.historyDone, domain.Task{ID: int64(i + 1), Title: fmt.Sprintf("done-%02d", i+1), Status: domain.StatusDone, CreatedDay: today, DueDay: today, DoneDay: &today})
	}

	body0 := renderHistoryBody(m)
	// Date table should include the 7 MM-DD values.
	for i := 0; i < 7; i++ {
		d := addDays(from, i)
		if indexOf(body0, fmtMMDD(d)) < 0 {
			t.Fatalf("expected date %s in date table, got:\n%s", fmtMMDD(d), body0)
		}
	}
	// Should always include an inset divider line between selector and details.
	innerW := sheetInnerWidth(m.width)
	if innerW <= 0 {
		t.Fatalf("expected innerW > 0")
	}
	// Internal divider: right -2 cells.
	divider := strings.Repeat("-", innerW-2)
	if indexOf(body0, divider) < 0 {
		t.Fatalf("expected divider of length %d, got:\n%s", innerW, body0)
	}
	// Selected date should stay reverse video when ANSI styling is available.
	if !strings.Contains(body0, "\x1b[7m") {
		t.Fatalf("expected reverse-video selected date, got:\n%s", body0)
	}
	if !strings.Contains(body0, "\x1b[7m 03-07 \x1b[0m|") {
		t.Fatalf("expected selected ANSI date cell to preserve padded width, got:\n%s", body0)
	}

	// Scrolling should change which rows are visible.
	m.historyScroll = 0
	bodyTop := renderHistoryBody(m)
	m.historyScroll = 5
	bodyScrolled := renderHistoryBody(m)
	if indexOf(bodyTop, "done-01") < 0 {
		t.Fatalf("expected top viewport to include done-01, got:\n%s", bodyTop)
	}
	if indexOf(bodyScrolled, "done-01") >= 0 {
		t.Fatalf("expected scrolled viewport to not include done-01, got:\n%s", bodyScrolled)
	}
}

func TestRenderHistoryBody_DateTable_UsesAsciiSelectedLabelFallback(t *testing.T) {
	disableTick(t)

	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	today := domain.MustParseDay("2026-03-07")
	from := domain.MustParseDay("2026-03-01")

	m := NewWithDeps(newFakeApp(today, nil), fakeClock{now: time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m.view = viewHistory
	m.width = 80
	m.height = 18
	m.historyFrom = from
	m.historyTo = today
	m.historyIndex = 6

	body := renderHistoryBody(m)
	if !strings.Contains(body, "[03-07]") {
		t.Fatalf("expected selected date to use visible ASCII fallback, got:\n%s", body)
	}
	if strings.Contains(body, "[03-06]") {
		t.Fatalf("expected unselected dates to render differently from selected date, got:\n%s", body)
	}
}

func TestRenderHistoryBody_DoesNotDoubleHorizontalRulesBetweenSelectorAndDetails(t *testing.T) {
	disableTick(t)

	today := domain.MustParseDay("2026-03-07")
	from := domain.MustParseDay("2026-03-01")

	m := NewWithDeps(newFakeApp(today, nil), fakeClock{now: time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m.view = viewHistory
	m.width = 80
	m.height = 18
	m.historyFrom = from
	m.historyTo = today
	m.historyIndex = 6
	m.historyDone = []domain.Task{{ID: 1, Title: "done-01", Status: domain.StatusDone, CreatedDay: today, DueDay: today, DoneDay: &today}}

	body := renderHistoryBody(m)
	innerW := sheetInnerWidth(m.width)
	divider := strings.Repeat("-", innerW-2)

	lines := strings.Split(body, "\n")
	// We want exactly one divider line.
	count := 0
	for _, ln := range lines {
		if ln == divider {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one divider line, got %d:\n%s", count, body)
	}
}

func TestRenderHistoryBody_StylesStatsLineDistinctly(t *testing.T) {
	disableTick(t)

	lipgloss.SetColorProfile(termenv.ANSI256)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	day := domain.MustParseDay("2026-03-07")
	m := NewWithDeps(newFakeApp(day, nil), fakeClock{now: time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m.view = viewHistory
	m.width = 80
	m.height = 18
	m.historyFrom = addDays(day, -6)
	m.historyTo = day
	m.historyIndex = 6
	m.historyDone = []domain.Task{{ID: 1, Title: "done-01", Status: domain.StatusDone, CreatedDay: day, DueDay: day, DoneDay: &day}}
	m.historyAbandoned = []domain.Task{{ID: 2, Title: "ab-01", Status: domain.StatusAbandoned, CreatedDay: day, DueDay: day, AbandonedDay: &day}}
	m.historyActiveCreated = []domain.Task{{ID: 3, Title: "active-01", Status: domain.StatusActive, CreatedDay: day, DueDay: addDays(day, -1)}}

	body := renderHistoryBody(m)
	statsLine := lineContaining(body, "done:")
	if statsLine == "" {
		t.Fatalf("expected stats line in history body, got:\n%s", body)
	}
	if statsLine == "done: 1  abandoned: 1  overdue active: 1" {
		t.Fatalf("expected styled stats line, got plain text %q", statsLine)
	}
	if !strings.Contains(statsLine, "\x1b[") {
		t.Fatalf("expected ANSI styling on stats line, got %q", statsLine)
	}
	if strings.Contains(lineContaining(body, "No history tasks."), "\x1b[") {
		t.Fatalf("expected ordinary body text to remain unstyled, got:\n%s", body)
	}
	if want := m.styles.Stats.Render("done: 1  abandoned: 1  overdue active: 1"); statsLine != want {
		t.Fatalf("expected stats line to use dedicated Stats style\nwant: %q\n got: %q", want, statsLine)
	}
}

func TestRenderHistoryBody_DividerEndsWithSpaceToAvoidWrapArtifacts(t *testing.T) {
	disableTick(t)

	today := domain.MustParseDay("2026-03-07")
	from := domain.MustParseDay("2026-03-01")

	m := NewWithDeps(newFakeApp(today, nil), fakeClock{now: time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)}, time.UTC)
	m.view = viewHistory
	m.width = 80
	m.height = 18
	m.historyFrom = from
	m.historyTo = today
	m.historyIndex = 6
	m.historyDone = []domain.Task{{ID: 1, Title: "done-01", Status: domain.StatusDone, CreatedDay: today, DueDay: today, DoneDay: &today}}

	body := renderHistoryBody(m)
	innerW := sheetInnerWidth(m.width)
	if innerW <= 0 {
		t.Fatalf("expected innerW > 0")
	}

	divider := strings.Repeat("-", innerW-2)
	if indexOf(body, divider) < 0 {
		t.Fatalf("expected inset divider %q to be present, got:\n%s", divider, body)
	}
}

func TestSheetInnerWidth_MatchesLipglossSheetFrameSize(t *testing.T) {
	disableTick(t)

	m := NewWithDeps(newFakeApp(domain.MustParseDay("2026-03-07"), nil), fakeClock{now: time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)}, time.UTC)
	windowW := 123

	cw := contentWidth(windowW)
	got := sheetInnerWidth(windowW)
	want := cw - m.styles.Sheet.GetHorizontalFrameSize()
	if want < 0 {
		want = 0
	}
	if got != want {
		t.Fatalf("expected sheetInnerWidth(%d)=%d to match contentWidth(%d)-Sheet.GetHorizontalFrameSize()=%d (cw=%d, frame=%d)", windowW, got, windowW, want, cw, m.styles.Sheet.GetHorizontalFrameSize())
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

func hasExactLine(s, want string) bool {
	for _, ln := range strings.Split(s, "\n") {
		if strings.TrimSpace(ln) == want {
			return true
		}
	}
	return false
}

func indexOfExactLine(lines []string, want string) int {
	for i, line := range lines {
		if line == want {
			return i
		}
	}
	return -1
}
