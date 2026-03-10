package ui

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/JamesYuuu/tick/internal/app"
	"github.com/JamesYuuu/tick/internal/domain"
	"github.com/JamesYuuu/tick/internal/timeutil"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

type view int

const (
	viewToday view = iota
	viewUpcoming
	viewHistory
)

type Model struct {
	app          appClient
	view         view
	keys         keyMap
	styles       styles
	clock        clock
	loc          *time.Location
	lastDay      domain.Day
	statusMsg    string
	width        int
	height       int
	modal        modalState
	addInput     textinput.Model
	todayList    list.Model
	upcomingList list.Model

	historyFrom          domain.Day
	historyTo            domain.Day
	historyIndex         int
	historyScroll        int
	historyDone          []domain.Task
	historyAbandoned     []domain.Task
	historyActiveCreated []domain.Task
	historyStats         app.OutcomeRatios
}

type appClient interface {
	Add(ctx context.Context, title string) (domain.Task, error)
	EditTitle(ctx context.Context, id int64, title string) error
	Delete(ctx context.Context, id int64) error
	Today(ctx context.Context) ([]domain.Task, error)
	Upcoming(ctx context.Context) ([]domain.Task, error)
	Done(ctx context.Context, id int64) error
	Abandon(ctx context.Context, id int64) error
	PostponeOneDay(ctx context.Context, id int64) error
	HistoryDoneByDay(ctx context.Context, day domain.Day) ([]domain.Task, error)
	HistoryAbandonedByDay(ctx context.Context, day domain.Day) ([]domain.Task, error)
	HistoryActiveByCreatedDay(ctx context.Context, day domain.Day) ([]domain.Task, error)
	Stats(ctx context.Context, fromDay, toDay domain.Day) (app.OutcomeRatios, error)
}

type clock interface {
	Now() time.Time
}

type modalKind int

const (
	modalKindNone modalKind = iota
	modalKindAdd
	modalKindEdit
	modalKindDelete
)

type modalState struct {
	kind       modalKind
	taskID     int64
	taskTitle  string
	submitting bool
}

var tickEvery = 10 * time.Second

func New(a *app.App) Model {
	return NewWithDeps(a, timeutil.SystemClock{}, time.Local)
}

func NewWithDeps(a appClient, clk clock, loc *time.Location) Model {
	m := Model{app: a, view: viewToday, keys: defaultKeyMap(), styles: defaultStyles(), clock: clk, loc: loc}

	t := textinput.New()
	t.Placeholder = "Add task"
	t.Prompt = "> "
	t.CharLimit = 200
	m.addInput = t

	m.todayList = newTaskList(todayItemDelegate{styles: m.styles, currentDay: m.currentDay()})
	m.upcomingList = newTaskList(simpleItemDelegate{styles: m.styles})

	m.lastDay = m.currentDay()

	return m
}

func addDays(d domain.Day, delta int) domain.Day {
	return domain.DayFromTime(d.Time().AddDate(0, 0, delta))
}

func newTaskList(d list.ItemDelegate) list.Model {
	l := list.New(nil, d, 0, 0)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	l.SetShowStatusBar(false)
	return l
}

type taskItem struct{ task domain.Task }

func (i taskItem) Title() string       { return i.task.Title }
func (i taskItem) Description() string { return "" }
func (i taskItem) FilterValue() string { return i.task.Title }

type todayItemDelegate struct {
	styles     styles
	currentDay domain.Day
}

func (d todayItemDelegate) Height() int                               { return 1 }
func (d todayItemDelegate) Spacing() int                              { return 0 }
func (d todayItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d todayItemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	it, ok := item.(taskItem)
	if !ok {
		return
	}

	selected := index == m.Index()
	line := it.task.Title

	if it.task.IsDelayed(d.currentDay) {
		if selected {
			line = d.styles.Reverse.Render(d.styles.Delayed.Render(line))
		} else {
			line = d.styles.Delayed.Render(line)
		}
	} else if selected {
		line = d.styles.Reverse.Render(line)
	}
	_, _ = fmt.Fprint(w, line)
}

type simpleItemDelegate struct{ styles styles }

func (d simpleItemDelegate) Height() int                               { return 1 }
func (d simpleItemDelegate) Spacing() int                              { return 0 }
func (d simpleItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d simpleItemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	it, ok := item.(taskItem)
	if !ok {
		return
	}
	selected := index == m.Index()
	line := it.task.Title
	if selected {
		line = d.styles.Reverse.Render(line)
	}
	_, _ = fmt.Fprint(w, line)
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.cmdRefreshActive(), m.tickCmd())
}

type tickMsg struct{}

func (m Model) tickCmd() tea.Cmd {
	return tea.Tick(tickEvery, func(time.Time) tea.Msg { return tickMsg{} })
}

type refreshMsg struct {
	today    []domain.Task
	upcoming []domain.Task
	err      error
}

type activeListsResult struct {
	today    []domain.Task
	upcoming []domain.Task
}

type modalSubmitMsg struct {
	today    []domain.Task
	upcoming []domain.Task
	err      error
	close    bool
}

type deleteModalSubmitMsg struct {
	view  view
	tasks []domain.Task
	err   error
	close bool
}

func (m Model) loadActiveLists(ctx context.Context) (activeListsResult, error) {
	today, err := m.app.Today(ctx)
	if err != nil {
		return activeListsResult{}, fmt.Errorf("today: %w", err)
	}
	upcoming, err := m.app.Upcoming(ctx)
	if err != nil {
		return activeListsResult{}, fmt.Errorf("upcoming: %w", err)
	}
	return activeListsResult{today: today, upcoming: upcoming}, nil
}

func (m Model) cmdRefreshActive() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		lists, err := m.loadActiveLists(ctx)
		if err != nil {
			return refreshMsg{err: err}
		}
		return refreshMsg{today: lists.today, upcoming: lists.upcoming}
	}
}

func (m Model) cmdActThenRefresh(prefix string, act func(ctx context.Context) error) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if err := act(ctx); err != nil {
			return refreshMsg{err: fmt.Errorf("%s: %w", prefix, err)}
		}
		lists, err := m.loadActiveLists(ctx)
		if err != nil {
			return refreshMsg{err: err}
		}
		return refreshMsg{today: lists.today, upcoming: lists.upcoming}
	}
}

func (m Model) cmdSubmitModal() tea.Cmd {
	modal := m.modal
	title := strings.TrimSpace(m.addInput.Value())

	return func() tea.Msg {
		ctx := context.Background()
		switch modal.kind {
		case modalKindAdd:
			if _, err := m.app.Add(ctx, title); err != nil {
				return modalSubmitMsg{err: fmt.Errorf("add: %w", err)}
			}
		case modalKindEdit:
			if err := m.app.EditTitle(ctx, modal.taskID, title); err != nil {
				return modalSubmitMsg{err: fmt.Errorf("edit: %w", err)}
			}
		default:
			return nil
		}

		lists, err := m.loadActiveLists(ctx)
		if err != nil {
			return modalSubmitMsg{err: err, close: true}
		}
		return modalSubmitMsg{today: lists.today, upcoming: lists.upcoming, close: true}
	}
}

func (m Model) cmdConfirmDelete() tea.Cmd {
	modal := m.modal
	currentView := m.view

	return func() tea.Msg {
		ctx := context.Background()
		if err := m.app.Delete(ctx, modal.taskID); err != nil {
			return deleteModalSubmitMsg{view: currentView, err: fmt.Errorf("delete: %w", err)}
		}

		switch currentView {
		case viewToday:
			today, err := m.app.Today(ctx)
			if err != nil {
				return deleteModalSubmitMsg{view: currentView, err: fmt.Errorf("today: %w", err), close: true}
			}
			return deleteModalSubmitMsg{view: currentView, tasks: today, close: true}
		case viewUpcoming:
			upcoming, err := m.app.Upcoming(ctx)
			if err != nil {
				return deleteModalSubmitMsg{view: currentView, err: fmt.Errorf("upcoming: %w", err), close: true}
			}
			return deleteModalSubmitMsg{view: currentView, tasks: upcoming, close: true}
		default:
			return deleteModalSubmitMsg{view: currentView, close: true}
		}
	}
}

func (m Model) currentDay() domain.Day {
	return timeutil.CurrentDay(m.clock, m.loc)
}

func tasksToItems(ts []domain.Task) []list.Item {
	items := make([]list.Item, 0, len(ts))
	for _, t := range ts {
		items = append(items, taskItem{task: t})
	}
	return items
}

func (m *Model) applyActiveRefresh(todayTasks, upcomingTasks []domain.Task) {
	m.statusMsg = ""
	m.todayList.SetItems(tasksToItems(todayTasks))
	today := m.currentDay()
	m.lastDay = today
	m.todayList.SetDelegate(todayItemDelegate{styles: m.styles, currentDay: today})
	m.upcomingList.SetItems(tasksToItems(upcomingTasks))
}

func (m Model) selectedTaskID() (int64, bool) {
	if m.view != viewToday {
		return 0, false
	}
	it := m.todayList.SelectedItem()
	ti, ok := it.(taskItem)
	if !ok {
		return 0, false
	}
	return ti.task.ID, true
}

func (m Model) selectedActiveTask() (domain.Task, bool) {
	var it list.Item
	switch m.view {
	case viewToday:
		it = m.todayList.SelectedItem()
	case viewUpcoming:
		it = m.upcomingList.SelectedItem()
	default:
		return domain.Task{}, false
	}
	ti, ok := it.(taskItem)
	if !ok {
		return domain.Task{}, false
	}
	return ti.task, true
}

func (m *Model) openAddModal() {
	m.modal = modalState{kind: modalKindAdd}
	m.addInput.SetValue("")
	m.addInput.Focus()
}

func (m *Model) openTaskModal(kind modalKind, task domain.Task) {
	m.modal = modalState{kind: kind, taskID: task.ID, taskTitle: task.Title}
	if kind == modalKindEdit {
		m.addInput.SetValue(task.Title)
		m.addInput.Focus()
		return
	}

	m.addInput.Blur()
	m.addInput.SetValue("")
	if kind == modalKindAdd {
		m.addInput.Focus()
	}
}

func (m *Model) closeModal() {
	m.modal = modalState{}
	m.addInput.Blur()
	m.addInput.SetValue("")
}

func modalBlocksKey(keys keyMap, msg tea.KeyMsg) bool {
	return key.Matches(msg, keys.Done) ||
		key.Matches(msg, keys.Abandon) ||
		key.Matches(msg, keys.Postpone) ||
		key.Matches(msg, keys.Edit) ||
		key.Matches(msg, keys.Delete) ||
		key.Matches(msg, keys.Add) ||
		key.Matches(msg, keys.NextView)
}

func (m Model) handleModalKey(msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	if m.modal.kind == modalKindNone {
		return m, nil, false
	}

	if msg.Type == tea.KeyEsc {
		m.closeModal()
		return m, nil, true
	}

	switch m.modal.kind {
	case modalKindAdd, modalKindEdit:
		if msg.Type == tea.KeyEnter {
			if m.modal.submitting {
				return m, nil, true
			}
			if strings.TrimSpace(m.addInput.Value()) == "" {
				return m, nil, true
			}
			m.modal.submitting = true
			return m, m.cmdSubmitModal(), true
		}
		var cmd tea.Cmd
		m.addInput, cmd = m.addInput.Update(msg)
		return m, cmd, true
	case modalKindDelete:
		if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 {
			switch strings.ToLower(string(msg.Runes[0])) {
			case "y":
				if m.modal.submitting {
					return m, nil, true
				}
				m.modal.submitting = true
				return m, m.cmdConfirmDelete(), true
			case "n":
				m.closeModal()
				return m, nil, true
			}
		}
		if modalBlocksKey(m.keys, msg) {
			return m, nil, true
		}
		return m, nil, true
	default:
		return m, nil, true
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		g := calcLayoutMetrics(msg.Width, msg.Height)
		m.todayList.SetSize(g.innerW, g.innerH)
		m.upcomingList.SetSize(g.innerW, g.innerH)
		m.addInput.Width = g.innerW
		m.clampHistoryScroll()
		return m, nil
	case refreshMsg:
		if msg.err != nil {
			m.statusMsg = msg.err.Error()
			return m, nil
		}
		m.applyActiveRefresh(msg.today, msg.upcoming)
		return m, nil
	case modalSubmitMsg:
		m.modal.submitting = false
		if msg.close {
			m.closeModal()
		}
		if msg.err != nil {
			m.statusMsg = msg.err.Error()
			return m, nil
		}
		m.applyActiveRefresh(msg.today, msg.upcoming)
		return m, nil
	case deleteModalSubmitMsg:
		m.modal.submitting = false
		if msg.close {
			m.closeModal()
		}
		if msg.err != nil {
			m.statusMsg = msg.err.Error()
			return m, nil
		}
		m.statusMsg = ""
		switch msg.view {
		case viewToday:
			m.todayList.SetItems(tasksToItems(msg.tasks))
			today := m.currentDay()
			m.lastDay = today
			m.todayList.SetDelegate(todayItemDelegate{styles: m.styles, currentDay: today})
		case viewUpcoming:
			m.upcomingList.SetItems(tasksToItems(msg.tasks))
		}
		return m, nil
	case historyRefreshMsg:
		if msg.err != nil {
			m.statusMsg = msg.err.Error()
			return m, nil
		}
		m.statusMsg = ""
		m.historyScroll = 0
		m.historyDone = msg.done
		m.historyAbandoned = msg.abandoned
		m.historyActiveCreated = msg.activeCreated
		if msg.hasStats {
			m.historyStats = msg.stats
		}
		return m, nil
	case tickMsg:
		day := m.currentDay()
		if day.Time().Equal(m.lastDay.Time()) {
			return m, tea.Batch(m.tickCmd())
		}
		m.lastDay = day
		if m.view == viewHistory {
			m.historyTo = day
			m.historyFrom = addDays(m.historyTo, -6)
			m.historyIndex = 6
			return m, tea.Batch(m.cmdRefreshHistoryWithStats(), m.tickCmd())
		}
		return m, tea.Batch(m.cmdRefreshActive(), m.tickCmd())
	case tea.KeyMsg:
		if next, cmd, handled := m.handleModalKey(msg); handled {
			return next, cmd
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.NextView):
			switch m.view {
			case viewToday:
				m.view = viewUpcoming
				return m, m.cmdRefreshActive()
			case viewUpcoming:
				m.view = viewHistory
				m.historyTo = m.currentDay()
				m.historyFrom = addDays(m.historyTo, -6)
				m.historyIndex = 6
				m.historyScroll = 0
				return m, m.cmdRefreshHistoryWithStats()
			case viewHistory:
				m.view = viewToday
				return m, m.cmdRefreshActive()
			default:
				m.view = viewToday
				return m, m.cmdRefreshActive()
			}
		case key.Matches(msg, m.keys.Add):
			if m.view != viewToday {
				return m, nil
			}
			m.openAddModal()
			return m, nil
		case key.Matches(msg, m.keys.Edit):
			task, ok := m.selectedActiveTask()
			if !ok {
				return m, nil
			}
			m.openTaskModal(modalKindEdit, task)
			return m, nil
		case key.Matches(msg, m.keys.Delete):
			task, ok := m.selectedActiveTask()
			if !ok {
				return m, nil
			}
			m.openTaskModal(modalKindDelete, task)
			return m, nil
		case key.Matches(msg, m.keys.Done):
			id, ok := m.selectedTaskID()
			if !ok {
				return m, nil
			}
			return m, m.cmdActThenRefresh("done", func(ctx context.Context) error { return m.app.Done(ctx, id) })
		case key.Matches(msg, m.keys.Abandon):
			id, ok := m.selectedTaskID()
			if !ok {
				return m, nil
			}
			return m, m.cmdActThenRefresh("abandon", func(ctx context.Context) error { return m.app.Abandon(ctx, id) })
		case key.Matches(msg, m.keys.Postpone):
			id, ok := m.selectedTaskID()
			if !ok {
				return m, nil
			}
			return m, m.cmdActThenRefresh("postpone", func(ctx context.Context) error { return m.app.PostponeOneDay(ctx, id) })
		}

		if m.view == viewHistory {
			switch {
			case key.Matches(msg, m.keys.HistoryUp):
				m.clampHistoryScroll()
				if m.historyScroll > 0 {
					m.historyScroll--
				}
				return m, nil
			case key.Matches(msg, m.keys.HistoryDown):
				m.clampHistoryScroll()
				if m.historyScroll < m.maxHistoryScroll() {
					m.historyScroll++
				}
				return m, nil
			case key.Matches(msg, m.keys.HistoryLeft):
				m.historyScroll = 0
				if m.historyIndex > 0 {
					m.historyIndex--
					return m, m.cmdRefreshHistorySelectedDay()
				}
				m.historyFrom = addDays(m.historyFrom, -1)
				m.historyTo = addDays(m.historyTo, -1)
				m.historyIndex = 0
				return m, m.cmdRefreshHistoryWithStats()
			case key.Matches(msg, m.keys.HistoryRight):
				m.historyScroll = 0
				if m.historyIndex < 6 {
					m.historyIndex++
					return m, m.cmdRefreshHistorySelectedDay()
				}
				today := m.currentDay()
				if m.historyTo.Time().Equal(today.Time()) {
					return m, nil
				}
				nextTo := addDays(m.historyTo, 1)
				if today.Before(nextTo) {
					nextTo = today
				}
				m.historyTo = nextTo
				m.historyFrom = addDays(m.historyTo, -6)
				m.historyIndex = 6
				return m, m.cmdRefreshHistoryWithStats()
			}
		}
	}

	var cmd tea.Cmd
	switch m.view {
	case viewToday:
		m.todayList, cmd = m.todayList.Update(msg)
	case viewUpcoming:
		m.upcomingList, cmd = m.upcomingList.Update(msg)
	}
	return m, cmd
}

type historyRefreshMsg struct {
	done          []domain.Task
	abandoned     []domain.Task
	activeCreated []domain.Task
	stats         app.OutcomeRatios
	hasStats      bool
	err           error
}

func (m Model) historySelectedDay() domain.Day {
	return addDays(m.historyFrom, m.historyIndex)
}

func (m Model) cmdRefreshHistory(withStats bool) tea.Cmd {
	day := m.historySelectedDay()
	from, to := m.historyFrom, m.historyTo
	return func() tea.Msg {
		ctx := context.Background()
		done, err := m.app.HistoryDoneByDay(ctx, day)
		if err != nil {
			return historyRefreshMsg{err: err}
		}
		ab, err := m.app.HistoryAbandonedByDay(ctx, day)
		if err != nil {
			return historyRefreshMsg{err: err}
		}
		activeCreated, err := m.app.HistoryActiveByCreatedDay(ctx, day)
		if err != nil {
			return historyRefreshMsg{err: err}
		}
		if !withStats {
			return historyRefreshMsg{done: done, abandoned: ab, activeCreated: activeCreated}
		}
		stats, err := m.app.Stats(ctx, from, to)
		if err != nil {
			return historyRefreshMsg{err: err}
		}
		return historyRefreshMsg{done: done, abandoned: ab, activeCreated: activeCreated, stats: stats, hasStats: true}
	}
}

func (m Model) cmdRefreshHistorySelectedDay() tea.Cmd {
	return m.cmdRefreshHistory(false)
}

func (m Model) cmdRefreshHistoryWithStats() tea.Cmd {
	return m.cmdRefreshHistory(true)
}

func (m Model) View() string {
	active := "Today"
	body := ""
	switch m.view {
	case viewToday:
		active = "Today"
		body = renderTodayBody(m)
	case viewUpcoming:
		active = "Upcoming"
		body = renderUpcomingBody(m)
	case viewHistory:
		active = "History"
		body = renderHistoryBody(m)
	default:
		active = "Today"
		body = renderTodayBody(m)
	}

	header := m.header(active)
	sep := separatorLine(m.width)
	footerLine1, footerLine2 := m.footerLines()

	// Clamp variable-height blocks so zone line positions are stable.
	header = forceHeight(header, 1)
	footer := forceHeight(strings.Join([]string{footerLine1, footerLine2}, "\n"), footerHelpHeight)

	// Fullscreen layout has a fixed 2-line footer.
	g := calcLayoutMetrics(m.width, m.height)
	if g.innerW > 0 {
		m.todayList.SetSize(g.innerW, g.innerH)
		m.upcomingList.SetSize(g.innerW, g.innerH)
		// Prefer sizing in WindowSizeMsg, but keep addInput stable if View runs first.
		m.addInput.Width = g.innerW
	}
	frameBody := forceHeight(body, g.innerH)
	workspace := m.sheetFrame(frameBody, g.contentW)
	workspace = forceHeight(workspace, g.workspaceH)

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(sep)
	b.WriteString("\n")
	b.WriteString(workspace)
	b.WriteString("\n")
	b.WriteString(sep)
	b.WriteString("\n")
	b.WriteString(footer)

	out := b.String()
	out = renderOverlay(out, renderModal(m), g.contentW, m.height)
	out = forceHeight(out, m.height)
	out = clipLinesToWidth(out, g.contentW)
	out = padLeftToWidth(out, m.width)
	return out
}

func (m Model) sheetFrame(body string, contentW int) string {
	sheet := m.styles.Sheet
	if contentW > 0 {
		// Clamp to content width by accounting for the sheet's own frame size.
		w := contentW - sheet.GetHorizontalFrameSize()
		if w < 0 {
			w = 0
		}
		sheet = sheet.Width(w)
	}
	return sheet.Render(body)
}

func clipLinesToWidth(s string, w int) string {
	if w <= 0 || s == "" {
		return s
	}
	lines := strings.Split(s, "\n")
	for i := range lines {
		if ansi.StringWidth(lines[i]) > w {
			lines[i] = ansi.Truncate(lines[i], w, "")
		}
	}
	return strings.Join(lines, "\n")
}
