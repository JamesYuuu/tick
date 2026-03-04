package ui

import (
	"github.com/JamesYuuu/tick/internal/app"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type view int

const (
	viewToday view = iota
	viewUpcoming
	viewHistory
)

type Model struct {
	app  *app.App
	view view
	keys keyMap
	styles styles
}

func New(a *app.App) Model {
	return Model{app: a, view: viewToday, keys: defaultKeyMap(), styles: defaultStyles()}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Today):
			m.view = viewToday
			return m, nil
		case key.Matches(msg, m.keys.Upcoming):
			m.view = viewUpcoming
			return m, nil
		case key.Matches(msg, m.keys.History):
			m.view = viewHistory
			return m, nil
		}
	}

	_ = msg
	return m, nil
}

func (m Model) View() string {
	switch m.view {
	case viewToday:
		return renderToday(m)
	case viewUpcoming:
		return renderUpcoming(m)
	case viewHistory:
		return renderHistory(m)
	default:
		return renderToday(m)
	}
}
