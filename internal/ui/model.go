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
	adding       bool
	addInput     textinput.Model
	todayList    list.Model
	upcomingList list.Model

	historyFrom      domain.Day
	historyTo        domain.Day
	historyIndex     int
	historyDone      []domain.Task
	historyAbandoned []domain.Task
	historyStats     app.OutcomeRatios
}

type appClient interface {
	Add(ctx context.Context, title string) (domain.Task, error)
	Today(ctx context.Context) ([]domain.Task, error)
	Upcoming(ctx context.Context) ([]domain.Task, error)
	Done(ctx context.Context, id int64) error
	Abandon(ctx context.Context, id int64) error
	PostponeOneDay(ctx context.Context, id int64) error
	HistoryDoneByDay(ctx context.Context, day domain.Day) ([]domain.Task, error)
	HistoryAbandonedByDay(ctx context.Context, day domain.Day) ([]domain.Task, error)
	Stats(ctx context.Context, fromDay, toDay domain.Day) (app.OutcomeRatios, error)
}

type clock interface {
	Now() time.Time
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
	prefix := "  "
	if selected {
		prefix = "> "
	}
	line := prefix + it.task.Title

	if it.task.IsDelayed(d.currentDay) {
		if selected {
			line = d.styles.RowSelDl.Render(line)
		} else {
			line = d.styles.Delayed.Render(line)
		}
	} else if selected {
		line = d.styles.RowSel.Render(line)
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
	prefix := "  "
	if selected {
		prefix = "> "
	}
	line := prefix + it.task.Title
	if selected {
		line = d.styles.RowSel.Render(line)
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

func (m Model) cmdRefreshActive() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		today, err := m.app.Today(ctx)
		if err != nil {
			return refreshMsg{err: fmt.Errorf("today: %w", err)}
		}
		up, err := m.app.Upcoming(ctx)
		if err != nil {
			return refreshMsg{err: fmt.Errorf("upcoming: %w", err)}
		}
		return refreshMsg{today: today, upcoming: up}
	}
}

func (m Model) cmdActThenRefresh(prefix string, act func(ctx context.Context) error) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if err := act(ctx); err != nil {
			return refreshMsg{err: fmt.Errorf("%s: %w", prefix, err)}
		}
		today, err := m.app.Today(ctx)
		if err != nil {
			return refreshMsg{err: fmt.Errorf("today: %w", err)}
		}
		up, err := m.app.Upcoming(ctx)
		if err != nil {
			return refreshMsg{err: fmt.Errorf("upcoming: %w", err)}
		}
		return refreshMsg{today: today, upcoming: up}
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

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		// Fullscreen layout has a fixed 2-line footer (status + help).
		workspaceHeight := msg.Height - (1 + 1 + 1 + 2)
		if workspaceHeight < 0 {
			workspaceHeight = 0
		}
		innerHeight := workspaceHeight - sheetVertMargin
		if innerHeight < 0 {
			innerHeight = 0
		}
		workspaceWidth := sheetInnerWidth(msg.Width)
		m.todayList.SetSize(workspaceWidth, innerHeight)
		m.upcomingList.SetSize(workspaceWidth, innerHeight)
		return m, nil
	case refreshMsg:
		if msg.err != nil {
			m.statusMsg = msg.err.Error()
			return m, nil
		}
		m.statusMsg = ""
		m.todayList.SetItems(tasksToItems(msg.today))
		today := m.currentDay()
		m.lastDay = today
		m.todayList.SetDelegate(todayItemDelegate{styles: m.styles, currentDay: today})
		m.upcomingList.SetItems(tasksToItems(msg.upcoming))
		return m, nil
	case historyRefreshMsg:
		if msg.err != nil {
			m.statusMsg = msg.err.Error()
			return m, nil
		}
		m.statusMsg = ""
		m.historyDone = msg.done
		m.historyAbandoned = msg.abandoned
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
		if m.adding {
			if key.Matches(msg, m.keys.Quit) {
				return m, tea.Quit
			}
			switch msg.Type {
			case tea.KeyEsc:
				m.adding = false
				m.addInput.Blur()
				return m, nil
			case tea.KeyEnter:
				title := m.addInput.Value()
				m.adding = false
				m.addInput.Blur()
				m.addInput.SetValue("")
				if title == "" {
					return m, nil
				}
				return m, m.cmdActThenRefresh("add", func(ctx context.Context) error {
					_, err := m.app.Add(ctx, title)
					return err
				})
			}
			var cmd tea.Cmd
			m.addInput, cmd = m.addInput.Update(msg)
			return m, cmd
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Today):
			m.view = viewToday
			return m, m.cmdRefreshActive()
		case key.Matches(msg, m.keys.Upcoming):
			m.view = viewUpcoming
			return m, m.cmdRefreshActive()
		case key.Matches(msg, m.keys.History):
			m.view = viewHistory
			m.historyTo = m.currentDay()
			m.historyFrom = addDays(m.historyTo, -6)
			m.historyIndex = 6
			return m, m.cmdRefreshHistoryWithStats()
		case key.Matches(msg, m.keys.Add):
			if m.view != viewToday {
				return m, nil
			}
			m.adding = true
			m.addInput.Focus()
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
				if m.historyIndex > 0 {
					m.historyIndex--
					return m, m.cmdRefreshHistorySelectedDay()
				}
				return m, nil
			case key.Matches(msg, m.keys.HistoryDown):
				if m.historyIndex < 6 {
					m.historyIndex++
					return m, m.cmdRefreshHistorySelectedDay()
				}
				return m, nil
			case key.Matches(msg, m.keys.HistoryLeft):
				m.historyFrom = addDays(m.historyFrom, -1)
				m.historyTo = addDays(m.historyTo, -1)
				return m, m.cmdRefreshHistoryWithStats()
			case key.Matches(msg, m.keys.HistoryRight):
				m.historyFrom = addDays(m.historyFrom, 1)
				m.historyTo = addDays(m.historyTo, 1)
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
	done      []domain.Task
	abandoned []domain.Task
	stats     app.OutcomeRatios
	hasStats  bool
	err       error
}

func (m Model) historySelectedDay() domain.Day {
	return addDays(m.historyFrom, m.historyIndex)
}

func (m Model) cmdRefreshHistorySelectedDay() tea.Cmd {
	day := m.historySelectedDay()
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
		return historyRefreshMsg{done: done, abandoned: ab}
	}
}

func (m Model) cmdRefreshHistoryWithStats() tea.Cmd {
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
		stats, err := m.app.Stats(ctx, from, to)
		if err != nil {
			return historyRefreshMsg{err: err}
		}
		return historyRefreshMsg{done: done, abandoned: ab, stats: stats, hasStats: true}
	}
}

func bodyHeight(windowHeight int) int {
	// header(1) + blank(2) + blank(2) + status(1) + help(1) + trailing(1)
	h := windowHeight - 8
	if h < 0 {
		return 0
	}
	return h
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
	status := m.footerStatusLine()
	help := m.help()

	// Clamp variable-height blocks so zone line positions are stable.
	header = forceHeight(header, 1)
	status = forceHeight(status, 1)
	help = forceHeight(help, 1)

	// Fullscreen layout has a fixed 2-line footer (status + help).
	workspaceHeight := m.height - (1 + 1 + 1 + 2)
	if workspaceHeight < 0 {
		workspaceHeight = 0
	}
	innerHeight := workspaceHeight - sheetVertMargin
	if innerHeight < 0 {
		innerHeight = 0
	}
	workspaceWidth := sheetInnerWidth(m.width)
	if workspaceWidth > 0 {
		m.todayList.SetSize(workspaceWidth, innerHeight)
		m.upcomingList.SetSize(workspaceWidth, innerHeight)
	}
	frameBody := forceHeight(body, innerHeight)
	workspace := m.frame(active, frameBody)
	workspace = forceHeight(workspace, workspaceHeight)

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(sep)
	b.WriteString("\n")
	b.WriteString(workspace)
	b.WriteString("\n")
	b.WriteString(sep)
	b.WriteString("\n")
	// Status line is always present to keep footer height stable.
	b.WriteString(status)
	b.WriteString("\n")
	b.WriteString(help)

	out := b.String()
	out = forceHeight(out, m.height)
	out = padLeftToWidth(out, m.width)
	return out
}
